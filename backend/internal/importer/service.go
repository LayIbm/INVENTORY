package importer

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/layssagonzalez/device-inventory/backend/internal/normalize"
)

// Service orchestrates the full import pipeline: parse → map → validate →
// deduplicate → insert/update. It is used by both the CLI and HTTP handler.
type Service struct {
	pool *pgxpool.Pool
}

// NewService creates a new Service backed by the given connection pool.
func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

// ─── Preview (dry-run) ────────────────────────────────────────────────────────

// Preview parses and validates the file without modifying the database.
func (s *Service) Preview(ctx context.Context, filePath string, opts Options) (*PreviewResult, error) {
	opts.DryRun = true
	rows, sheets, err := s.prepare(ctx, filePath)
	if err != nil {
		return nil, err
	}

	// Check DB duplicates BEFORE tallying so counts are accurate.
	// In dry-run, upsert=false so existing serials are always marked SKIP.
	s.checkDBDuplicates(ctx, rows, opts)

	// Build result
	result := &PreviewResult{
		Sheets: sheets,
	}
	result.Summary.Files = 1
	result.Summary.Sheets = len(sheets)

	// Tally empty rows skipped during parsing.
	for _, si := range sheets {
		result.Summary.RowsIgnored += si.EmptyRows
	}

	for i := range rows {
		r := &rows[i]
		result.Summary.RowsRead++
		switch r.ProposedAction {
		case ActionSkip:
			result.Summary.Duplicates++
		case ActionInsert:
			result.Summary.Insertable++
			result.Summary.Valid++
		case ActionUpdate:
			result.Summary.Updatable++
			result.Summary.Valid++
		case ActionError:
			result.Summary.WithErrors++
		}
		if r.HasWarnings() {
			result.Summary.WithWarnings++
		}
		switch r.Entity {
		case EntityLaptop:
			result.Summary.LaptopRows++
		case EntityEquipment:
			result.Summary.EquipmentRows++
		default:
			result.Summary.UnknownRows++
		}
	}

	// Sample: first 20 rows
	sampleCount := 0
	for i := range rows {
		r := &rows[i]
		if sampleCount < 20 {
			result.Sample = append(result.Sample, *r)
			sampleCount++
		}
		if r.HasErrors() {
			result.Errors = append(result.Errors, *r)
		} else if r.HasWarnings() {
			result.Warnings = append(result.Warnings, *r)
		}
	}

	result.CanApply = result.Summary.WithErrors == 0 ||
		result.Summary.Insertable > 0 || result.Summary.Updatable > 0

	return result, nil
}

// ─── Apply ────────────────────────────────────────────────────────────────────

// Apply runs the full import: parse → validate → insert/update in batches.
// Returns an ApplyResult with final counts and the path to the error CSV.
func (s *Service) Apply(ctx context.Context, filePath string, opts Options) (*ApplyResult, error) {
	if opts.BatchSize <= 0 {
		opts.BatchSize = DefaultBatchSize
	}

	start := time.Now()
	rows, sheetInfos, err := s.prepare(ctx, filePath)
	if err != nil {
		return nil, err
	}

	// Check DB for existing serials and adjust actions
	s.checkDBDuplicates(ctx, rows, opts)

	// Batch processing
	var insertedTotal, updatedTotal, skippedTotal, erroredTotal int

	if opts.Atomic {
		// Single transaction; rollback everything on any write error.
		ins, upd, skp, er, err2 := s.writeBatch(ctx, rows, opts)
		if err2 != nil {
			return nil, errDB("atomic batch failed", err2)
		}
		insertedTotal = ins
		updatedTotal = upd
		skippedTotal = skp
		erroredTotal = er
	} else {
		for start2 := 0; start2 < len(rows); start2 += opts.BatchSize {
			end := start2 + opts.BatchSize
			if end > len(rows) {
				end = len(rows)
			}
			batch := rows[start2:end]
			ins, upd, skp, er, err2 := s.writeBatch(ctx, batch, opts)
			if err2 != nil {
				slog.Error("importer batch failed, continuing", "batch_start", start2, "err", err2)
				erroredTotal += len(batch)
				continue
			}
			insertedTotal += ins
			updatedTotal += upd
			skippedTotal += skp
			erroredTotal += er
		}
	}

	elapsed := time.Since(start)

	result := &ApplyResult{
		FinishedAt: time.Now(),
	}
	result.Summary.Files = 1
	result.Summary.Sheets = len(sheetInfos)
	result.Summary.RowsRead = len(rows)
	for _, si := range sheetInfos {
		result.Summary.RowsIgnored += si.EmptyRows
	}
	result.Summary.Inserted = insertedTotal
	result.Summary.Updated = updatedTotal
	result.Summary.Skipped = skippedTotal
	result.Summary.Errored = erroredTotal
	result.Summary.DurationMs = elapsed.Milliseconds()

	for _, r := range rows {
		if r.HasWarnings() {
			result.Summary.WithWarnings++
		}
		if r.HasErrors() {
			result.Summary.WithErrors++
		}
	}

	return result, nil
}

// ─── Internals ────────────────────────────────────────────────────────────────

// prepare parses the file, maps columns, validates rows, and returns the
// annotated rows together with sheet metadata. This is shared between Preview
// and Apply so the validation logic is identical.
func (s *Service) prepare(ctx context.Context, filePath string) ([]ImportRow, []SheetInfo, error) {
	sheets, err := ParseFile(filePath)
	if err != nil {
		return nil, nil, err
	}

	var allRows []ImportRow
	var sheetInfos []SheetInfo
	seenSerials := map[string]int{} // tracks serials across ALL sheets in the file

	for _, ps := range sheets {
		if len(ps.Headers) == 0 && len(ps.Rows) == 0 {
			continue
		}

		cols := make([]string, 0, len(ps.Headers))
		for i, h := range ps.Headers {
			if ps.FieldKeys[i] != "" && ps.FieldKeys[i] != "IGNORE" {
				cols = append(cols, h)
			}
		}
		sheetInfos = append(sheetInfos, SheetInfo{
			File:      ps.File,
			Sheet:     ps.Sheet,
			RowCount:  len(ps.Rows),
			EmptyRows: ps.EmptyRows,
			Columns:   cols,
		})

		rowNum := 2 // Excel rows are 1-based; header is row 1
		if ps.MergedTitle != "" {
			rowNum = 3 // title row bumped everything
		}

		// Infer a sheet-level default device type when the sheet has no
		// device_type column. Used for sheets like "TinyPCs ya asignadas"
		// that only contain one type of device.
		sheetDefaultType := inferSheetDeviceType(ps.Sheet, ps.FieldKeys)

		for _, rawRow := range ps.Rows {
			// Build the fields map from resolved column names
			fields := make(map[string]any, len(rawRow))
			for k, v := range rawRow {
				fields[k] = v
			}

			// Apply sheet-level device_type inference when the row has none.
			if sheetDefaultType != "" {
				if existing := str(fields, "device_type"); existing == "" {
					fields["device_type"] = sheetDefaultType
				}
			}

			row := ImportRow{
				File:           ps.File,
				Sheet:          ps.Sheet,
				Row:            rowNum,
				Raw:            rawRow,
				Fields:         fields,
				ProposedAction: ActionUnknown,
			}
			rowNum++

			validateRow(&row, seenSerials)
			allRows = append(allRows, row)
		}
	}

	return allRows, sheetInfos, nil
}

// checkDBDuplicates queries the database for each serial that has a proposed
// INSERT action. If the serial already exists, it adjusts the action to SKIP
// or UPDATE depending on the Upsert option.
func (s *Service) checkDBDuplicates(ctx context.Context, rows []ImportRow, opts Options) {
	if s.pool == nil {
		return
	}
	for i := range rows {
		r := &rows[i]
		if r.ProposedAction != ActionInsert {
			continue
		}
		serial := str(r.Fields, "serial")
		if serial == "" {
			continue
		}
		existsIn := s.serialExists(ctx, serial)
		if existsIn != "" {
			if opts.Upsert {
				addWarning(r, "serial", serial, serial,
					fmt.Sprintf("serial already exists in %s table — will update", existsIn))
				r.ProposedAction = ActionUpdate
			} else {
				addWarning(r, "serial", serial, serial,
					fmt.Sprintf("serial already exists in %s table", existsIn))
				r.ProposedAction = ActionSkip
			}
		}
	}
}

// serialExists returns "laptops", "equipment", or "" depending on which table
// contains the serial.
func (s *Service) serialExists(ctx context.Context, serial string) string {
	var dummy string
	if err := s.pool.QueryRow(ctx, `SELECT serial FROM laptops WHERE serial = $1`, serial).
		Scan(&dummy); err == nil {
		return "laptops"
	}
	if err := s.pool.QueryRow(ctx, `SELECT serial FROM equipment WHERE serial = $1`, serial).
		Scan(&dummy); err == nil {
		return "equipment"
	}
	return ""
}

// writeBatch writes a slice of rows in a single transaction.
func (s *Service) writeBatch(ctx context.Context, rows []ImportRow, opts Options) (inserted, updated, skipped, errored int, err error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, 0, 0, len(rows), errDB("begin transaction", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	for i := range rows {
		r := &rows[i]
		switch r.ProposedAction {
		case ActionSkip:
			r.ExecutedAction = ActionSkip
			skipped++
		case ActionError:
			r.ExecutedAction = ActionError
			errored++
		case ActionInsert:
			if writeErr := s.insertRow(ctx, tx, r, opts); writeErr != nil {
				addError(r, "db_insert", "", writeErr.Error())
				r.ExecutedAction = ActionError
				if opts.Atomic {
					return 0, 0, 0, len(rows), writeErr
				}
				errored++
				continue
			}
			r.ExecutedAction = ActionInsert
			inserted++
		case ActionUpdate:
			if writeErr := s.updateRow(ctx, tx, r, opts); writeErr != nil {
				addError(r, "db_update", "", writeErr.Error())
				r.ExecutedAction = ActionError
				if opts.Atomic {
					return 0, 0, 0, len(rows), writeErr
				}
				errored++
				continue
			}
			r.ExecutedAction = ActionUpdate
			updated++
		default:
			r.ExecutedAction = ActionSkip
			skipped++
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return 0, 0, 0, len(rows), errDB("commit transaction", err)
	}
	return inserted, updated, skipped, errored, nil
}

// ─── Row writers ─────────────────────────────────────────────────────────────

func (s *Service) insertRow(ctx context.Context, tx pgx.Tx, row *ImportRow, opts Options) error {
	f := row.Fields
	serial := str(f, "serial")
	model := str(f, "model")

	if row.Entity == EntityLaptop {
		histJSON := `[{"date":"` + todayShort() + `","type":"Registro","tone":"neutral","employee":"` +
			opts.Actor + `","notes":"Importado desde Excel."}]`
		_, err := tx.Exec(ctx, `
			INSERT INTO laptops (
				serial, model, variant, brand, condition,
				availability, prep, comodato,
				powers_on, os, win11_ready, bios_password,
				wifi, bluetooth, charger_included, last_format_date,
				employee_name, employee_email, employee_talent_id, employee_manager_email,
				lcd_ok, expected_return_date, usage, notes, history,
				owner, hostname, geography
			) VALUES (
				$1,$2,$3,$4,$5,
				$6,$7,$8,
				$9,$10,$11,$12,
				$13,$14,$15,$16,
				$17,$18,$19,$20,
				$21,$22,$23,$24,$25,
				$26,$27,$28
			) ON CONFLICT (serial) DO NOTHING`,
			serial, model, str(f, "model_variant"), coalesce(str(f, "brand"), "Lenovo"),
			coalesce(str(f, "condition"), "Unknown"),
			toLaptopAvailability(coalesce(str(f, "availability"), "AVAILABLE")),
			toPrepStatus(coalesce(str(f, "assignability"), "NEEDS_PREP")),
			toComodatoStatus(str(f, "comodato")),
			coalesce(str(f, "powers_on"), "OK"),
			coalesce(str(f, "os"), ""),
			parseBoolField(str(f, "win11_ready")),
			str(f, "bios_password"),
			coalesce(str(f, "wifi"), "UNKNOWN"),
			coalesce(str(f, "bluetooth"), "UNKNOWN"),
			parseBoolField(str(f, "charger_included")),
			str(f, "last_format_date"),
			nullableStr(normalize.Name(str(f, "employee_name"))),
			str(f, "employee_email"),
			str(f, "employee_talent_id"),
			str(f, "employee_manager_email"),
			coalesce(str(f, "lcd_ok"), "OK"),
			str(f, "expected_return_date"),
			str(f, "usage"),
			str(f, "notes"),
			histJSON,
			coalesce(str(f, "owner"), "IBM"),
			str(f, "hostname"),
			str(f, "geography"),
		)
		return err
	}

	// Equipment row
	histJSON := `[{"date":"` + todayShort() + `","type":"Registro","tone":"neutral","employee":"` +
		opts.Actor + `","notes":"Importado desde Excel."}]`

	deviceCost := nullableCostStr(str(f, "device_cost"))
	contractCost := nullableCostStr(str(f, "contract_cost"))

	_, err := tx.Exec(ctx, `
		INSERT INTO equipment (
			serial, device_type, brand, model, variant, condition,
			availability, assignability, comodato,
			powers_on, os, bios_password, wifi, bluetooth,
			charger_included, last_format_date,
			employee_name, employee_email, employee_talent_id,
			transaction_date, expected_return_date,
			monitor_included, monitor_serial,
			notes, history,
			product_id, net_type,
			end_of_sale, end_of_life, end_contract_support,
			device_cost, contract_cost,
			room, eol_status, contract_status,
			device_company, contact_name, contact_phone, contact_email,
			ibm_network_email_support, ibm_local_email_support,
			owner, hostname, geography
		) VALUES (
			$1,$2,$3,$4,$5,$6,
			$7,$8,$9,
			$10,$11,$12,$13,$14,
			$15,$16,
			$17,$18,$19,
			$20,$21,
			$22,$23,
			$24,$25,
			$26,$27,
			$28,$29,$30,
			$31,$32,
			$33,$34,$35,
			$36,$37,$38,$39,
			$40,$41,
			$42,$43,$44
		) ON CONFLICT (serial) DO NOTHING`,
		serial, str(f, "device_type"), coalesce(str(f, "brand"), "Unknown"),
		model, str(f, "model_variant"), coalesce(str(f, "condition"), "Unknown"),
		toEquipmentAvailability(coalesce(str(f, "availability"), "AVAILABLE")),
		toPrepStatus(coalesce(str(f, "assignability"), "READY")),
		toComodatoStatus(str(f, "comodato")),
		coalesce(str(f, "powers_on"), "OK"),
		str(f, "os"), str(f, "bios_password"),
		coalesce(str(f, "wifi"), "UNKNOWN"),
		coalesce(str(f, "bluetooth"), "UNKNOWN"),
		parseBoolField(str(f, "charger_included")),
		str(f, "last_format_date"),
		nullableStr(normalize.Name(str(f, "employee_name"))),
		str(f, "employee_email"), str(f, "employee_talent_id"),
		str(f, "transaction_date"), str(f, "expected_return_date"),
		parseBoolField(str(f, "monitor_included")),
		str(f, "monitor_serial"),
		str(f, "notes"), histJSON,
		str(f, "product_id"), coalesce(str(f, "net_type"), "N/A"),
		nullableDate(str(f, "end_of_sale")),
		nullableDate(str(f, "end_of_life")),
		nullableDate(str(f, "end_contract_support")),
		deviceCost, contractCost,
		str(f, "room"), str(f, "eol_status"), str(f, "contract_status"),
		str(f, "device_company"), str(f, "contact_name"),
		str(f, "contact_phone"), str(f, "contact_email"),
		str(f, "ibm_network_email_support"), str(f, "ibm_local_email_support"),
		coalesce(str(f, "owner"), "IBM"),
		str(f, "hostname"),
		str(f, "geography"),
	)
	return err
}

func (s *Service) updateRow(ctx context.Context, tx pgx.Tx, row *ImportRow, opts Options) error {
	// Upsert: update the row keeping existing values for any blank imported fields.
	f := row.Fields
	serial := str(f, "serial")

	if row.Entity == EntityLaptop {
		// Availability and prep are enum columns; cast the parameter to TEXT first so
		// PostgreSQL can match the COALESCE branches. NULLIF and COALESCE then work
		// correctly: an empty string becomes NULL which falls back to the existing enum value.
		_, err := tx.Exec(ctx, `
			UPDATE laptops SET
				model                  = COALESCE(NULLIF($2,''), model),
				condition              = COALESCE(NULLIF($3,''), condition),
				availability           = COALESCE(NULLIF($4,'')::availability_status, availability),
				prep                   = COALESCE(NULLIF($5,'')::prep_status, prep),
				comodato               = COALESCE(NULLIF($6,'')::comodato_status, comodato),
				os                     = COALESCE(NULLIF($7,''), os),
				bios_password          = COALESCE(NULLIF($8,''), bios_password),
				wifi                   = COALESCE(NULLIF($9,''), wifi),
				bluetooth              = COALESCE(NULLIF($10,''), bluetooth),
				employee_name          = NULLIF($11,''),
				employee_email         = COALESCE(NULLIF($12,''), employee_email),
				employee_talent_id     = COALESCE(NULLIF($13,''), employee_talent_id),
				employee_manager_email = COALESCE(NULLIF($14,''), employee_manager_email),
				expected_return_date   = COALESCE(NULLIF($15,''), expected_return_date),
				hostname               = COALESCE(NULLIF($16,''), hostname),
				geography              = COALESCE(NULLIF($17,''), geography),
				owner                  = COALESCE(NULLIF($18,''), owner),
				usage                  = COALESCE(NULLIF($19,''), usage),
				notes                  = COALESCE(NULLIF($20,''), notes)
			WHERE serial = $1`,
			serial,
				str(f, "model"), str(f, "condition"),
				toLaptopAvailabilityUpdate(str(f, "availability")),
					inferAssignability(str(f, "assignability"), str(f, "availability")),
				toComodatoStatusUpdate(str(f, "comodato")),
				str(f, "os"), str(f, "bios_password"),
				str(f, "wifi"), str(f, "bluetooth"),
				nullableStr(normalize.Name(str(f, "employee_name"))),
				str(f, "employee_email"), str(f, "employee_talent_id"),
				str(f, "employee_manager_email"),
				str(f, "expected_return_date"),
				str(f, "hostname"), str(f, "geography"),
				str(f, "owner"), str(f, "usage"), str(f, "notes"),
			)
			return err
		}
	
		_, err := tx.Exec(ctx, `
			UPDATE equipment SET
				model                = COALESCE(NULLIF($2,''), model),
				condition            = COALESCE(NULLIF($3,''), condition),
				availability         = COALESCE(NULLIF($4,'')::availability_status, availability),
				assignability        = COALESCE(NULLIF($5,'')::prep_status, assignability),
				comodato             = COALESCE(NULLIF($6,'')::comodato_status, comodato),
				employee_name        = NULLIF($7,''),
				employee_email       = COALESCE(NULLIF($8,''), employee_email),
				employee_talent_id   = COALESCE(NULLIF($9,''), employee_talent_id),
				expected_return_date = COALESCE(NULLIF($10,''), expected_return_date),
				eol_status           = COALESCE(NULLIF($11,''), eol_status),
				contract_status      = COALESCE(NULLIF($12,''), contract_status),
				owner                = COALESCE(NULLIF($13,''), owner),
				hostname             = COALESCE(NULLIF($14,''), hostname),
				geography            = COALESCE(NULLIF($15,''), geography),
				notes                = COALESCE(NULLIF($16,''), notes)
			WHERE serial = $1`,
			serial,
			str(f, "model"), str(f, "condition"),
			toEquipmentAvailabilityUpdate(str(f, "availability")),
			inferAssignability(str(f, "assignability"), str(f, "availability")),
			toComodatoStatusUpdate(str(f, "comodato")),
			nullableStr(normalize.Name(str(f, "employee_name"))),
			str(f, "employee_email"), str(f, "employee_talent_id"),
			str(f, "expected_return_date"),
			str(f, "eol_status"), str(f, "contract_status"),
			str(f, "owner"), str(f, "hostname"), str(f, "geography"),
			str(f, "notes"),
		)
		return err
}

// ─── Small helpers ────────────────────────────────────────────────────────────

func coalesce(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func nullableStr(s string) *string {
	if s == "" || blankOrNA(s) {
		return nil
	}
	return &s
}

func nullableDate(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func nullableCostStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func parseBoolField(s string) bool {
	b, _ := NormalizeBool(s)
	return b
}

func todayShort() string {
	return time.Now().Format("02/01/2006")
}

// toLaptopAvailability converts the importer availability value to the
// laptops table enum: DISPONIBLE / ASIGNADA / NO_DISPONIBLE / SCRAP.
// Returns "DISPONIBLE" for inserts (caller provides a default via coalesce).
// For updates, use toLaptopAvailabilityUpdate which returns "" on unknown input
// so COALESCE in SQL can preserve the existing DB value.
func toLaptopAvailability(s string) string {
	switch strings.ToUpper(s) {
	case "AVAILABLE", "DISPONIBLE":
		return "DISPONIBLE"
	case "NOT_AVAILABLE", "ASIGNADA":
		return "ASIGNADA"
	case "NO_DISPONIBLE":
		return "NO_DISPONIBLE"
	case "SCRAP":
		return "SCRAP"
	default:
		return "DISPONIBLE"
	}
}

// toLaptopAvailabilityUpdate is like toLaptopAvailability but returns ""
// for unrecognised/empty values so the SQL COALESCE keeps the existing row value.
func toLaptopAvailabilityUpdate(s string) string {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "AVAILABLE", "DISPONIBLE":
		return "DISPONIBLE"
	case "NOT_AVAILABLE", "ASIGNADA":
		return "ASIGNADA"
	case "NO_DISPONIBLE":
		return "NO_DISPONIBLE"
	case "SCRAP":
		return "SCRAP"
	default:
		return ""
	}
}

// toEquipmentAvailability maps importer availability values to the equipment
// table enum. Equipment uses the same availability_status enum.
func toEquipmentAvailability(s string) string {
	switch strings.ToUpper(s) {
	case "AVAILABLE", "DISPONIBLE":
		return "DISPONIBLE"
	case "NOT_AVAILABLE", "ASIGNADA":
		return "ASIGNADA"
	case "NO_DISPONIBLE":
		return "NO_DISPONIBLE"
	case "SCRAP":
		return "SCRAP"
	default:
		return "DISPONIBLE"
	}
}

// toEquipmentAvailabilityUpdate is like toEquipmentAvailability but returns ""
// for unrecognised/empty values so the SQL COALESCE keeps the existing row value.
func toEquipmentAvailabilityUpdate(s string) string {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "AVAILABLE", "DISPONIBLE":
		return "DISPONIBLE"
	case "NOT_AVAILABLE", "ASIGNADA":
		return "ASIGNADA"
	case "NO_DISPONIBLE":
		return "NO_DISPONIBLE"
	case "SCRAP":
		return "SCRAP"
	default:
		return ""
	}
}

// toPrepStatus converts assignability strings to the prep_status enum:
// LISTA / NECESITA_PREP / NO_FUNCIONAL.
func toPrepStatus(s string) string {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "READY", "LISTA", "LIST", "LISTO":
		return "LISTA"
	case "NEEDS_PREP", "NECESITA_PREP", "NEEDS PREP", "NEED PREP", "PREP":
		return "NECESITA_PREP"
	case "NOT_FUNCTIONAL", "NO_FUNCIONAL", "NOT FUNCTIONAL", "NO FUNCIONAL":
		return "NO_FUNCIONAL"
	default:
		return "NECESITA_PREP"
	}
}

// toPrepStatusUpdate is like toPrepStatus but returns "" on unknown/empty input
// so COALESCE in SQL keeps the existing DB value.
// Special case: when the device is already assigned (availability=ASIGNADA),
// an empty assignability is inferred as LISTA.
func toPrepStatusUpdate(s string) string {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "READY", "LISTA", "LIST", "LISTO":
		return "LISTA"
	case "NEEDS_PREP", "NECESITA_PREP", "NEEDS PREP", "NEED PREP", "PREP":
		return "NECESITA_PREP"
	case "NOT_FUNCTIONAL", "NO_FUNCIONAL", "NOT FUNCTIONAL", "NO FUNCIONAL":
		return "NO_FUNCIONAL"
	default:
		return ""
	}
}

// inferAssignability returns LISTA when assignability is blank and the device
// is either assigned or available — only NO_FUNCIONAL requires an explicit value.
// Recognises both DB enum values and raw importer strings (typos included).
// Returns "" (preserve DB value) only when assignability is blank AND availability
// is also unrecognised.
func inferAssignability(assignability, availability string) string {
	prep := toPrepStatusUpdate(assignability)
	if prep == "" {
		switch strings.ToUpper(strings.TrimSpace(availability)) {
		case "ASIGNADA", "NOT_AVAILABLE", "ASSIGNED", "ASIGNED":
			return "LISTA"
		case "DISPONIBLE", "AVAILABLE", "AVAIBLE", "AVAIALBLE":
			return "LISTA"
		}
	}
	return prep
}

// toComodatoStatus converts comodato field values to the comodato_status enum:
// FIRMADO / PENDIENTE / N/A.
func toComodatoStatus(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "true", "firmado", "signed", "1", "yes", "si", "sí":
		return "FIRMADO"
	case "pending", "pendiente":
		return "PENDIENTE"
	default:
		return "N/A"
	}
}

// toComodatoStatusUpdate is like toComodatoStatus but returns "" on unknown/empty
// input so COALESCE in SQL keeps the existing DB value.
func toComodatoStatusUpdate(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "true", "firmado", "signed", "1", "yes", "si", "sí":
		return "FIRMADO"
	case "pending", "pendiente":
		return "PENDIENTE"
	case "n/a", "na":
		return "N/A"
	default:
		return ""
	}
}

// inferSheetDeviceType returns a canonical device type string to use as the
// default for rows in this sheet when no device_type column is present.
// Returns "" when no inference can be made.
func inferSheetDeviceType(sheetName string, fieldKeys []string) string {
	// Check whether a device_type column already exists in this sheet.
	hasDeviceType := false
	hasSerial := false
	hasEmployee := false
	hasHostname := false
	for _, k := range fieldKeys {
		switch k {
		case "device_type":
			hasDeviceType = true
		case "serial":
			hasSerial = true
		case "employee_name", "employee_email", "employee_talent_id":
			hasEmployee = true
		case "hostname":
			hasHostname = true
		}
	}
	if hasDeviceType {
		return "" // sheet has its own device_type column
	}

	// Infer from sheet name.
	lower := strings.ToLower(strings.TrimSpace(sheetName))
	switch {
	case strings.Contains(lower, "tinypc") || strings.Contains(lower, "tiny pc"):
		return "Desktop"
	case strings.Contains(lower, "laptop"):
		return "Laptop"
	case strings.Contains(lower, "monitor"):
		return "Monitor"
	case strings.Contains(lower, "switch"):
		return "Switch"
	case strings.Contains(lower, "router"):
		return "Router"
	}

	// EPD-style sheets (e.g. "Sheet1") that have serial + employee info but no
	// device_type column are always laptops — IBM EPD lists only contain laptops.
	if hasSerial && hasEmployee && hasHostname {
		return "Laptop"
	}

	return ""
}
