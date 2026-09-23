package laptop

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Store handles all database operations for laptops.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore creates a new Store.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// List returns all laptops ordered by created_at desc.
func (s *Store) List(ctx context.Context) ([]Laptop, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, serial, model, variant, brand, condition,
		       availability, prep, comodato, powers_on, os, win11_ready,
		       bios_password, wifi, bluetooth, charger_included,
		       last_format_date, employee_name,
		       employee_email, employee_talent_id, employee_manager_email,
		       lcd_ok, expected_return_date,
		       owner, hostname, geography,
		       last_bios_update, bios_details, bluetooth_disabled_bios,
		       epd_status, ipv6, usage, notes, history,
		       created_at, updated_at
		FROM laptops ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("laptop list: %w", err)
	}
	defer rows.Close()

	var laptops []Laptop
	for rows.Next() {
		var l Laptop
		var histRaw []byte
		if err := rows.Scan(
			&l.ID, &l.Serial, &l.Model, &l.Variant, &l.Brand, &l.Condition,
			&l.Availability, &l.Prep, &l.Comodato, &l.PowersOn, &l.OS, &l.Win11Ready,
			&l.BIOSPassword, &l.WiFi, &l.Bluetooth, &l.ChargerIncluded,
			&l.LastFormatDate, &l.EmployeeName,
			&l.EmployeeEmail, &l.EmployeeTalentID, &l.EmployeeManagerEmail,
			&l.LcdOk, &l.ExpectedReturnDate,
			&l.Owner, &l.Hostname, &l.Geography,
			&l.LastBIOSUpdate, &l.BIOSDetails, &l.BluetoothDisabledBIOS,
			&l.EpdStatus, &l.IPv6, &l.Usage, &l.Notes, &histRaw,
			&l.CreatedAt, &l.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("laptop list scan: %w", err)
		}
		if err := json.Unmarshal(histRaw, &l.History); err != nil {
			l.History = []HistoryEntry{}
		}
		laptops = append(laptops, l)
	}
	if laptops == nil {
		laptops = []Laptop{}
	}
	return laptops, nil
}

// GetByID returns a single laptop by its UUID.
func (s *Store) GetByID(ctx context.Context, id string) (*Laptop, error) {
	var l Laptop
	var histRaw []byte
	err := s.pool.QueryRow(ctx, `
		SELECT id, serial, model, variant, brand, condition,
		       availability, prep, comodato, powers_on, os, win11_ready,
		       bios_password, wifi, bluetooth, charger_included,
		       last_format_date, employee_name,
		       employee_email, employee_talent_id, employee_manager_email,
		       lcd_ok, expected_return_date,
		       owner, hostname, geography,
		       last_bios_update, bios_details, bluetooth_disabled_bios,
		       epd_status, ipv6, usage, notes, history,
		       created_at, updated_at
		FROM laptops WHERE id = $1`, id).Scan(
		&l.ID, &l.Serial, &l.Model, &l.Variant, &l.Brand, &l.Condition,
		&l.Availability, &l.Prep, &l.Comodato, &l.PowersOn, &l.OS, &l.Win11Ready,
		&l.BIOSPassword, &l.WiFi, &l.Bluetooth, &l.ChargerIncluded,
		&l.LastFormatDate, &l.EmployeeName,
		&l.EmployeeEmail, &l.EmployeeTalentID, &l.EmployeeManagerEmail,
		&l.LcdOk, &l.ExpectedReturnDate,
		&l.Owner, &l.Hostname, &l.Geography,
		&l.LastBIOSUpdate, &l.BIOSDetails, &l.BluetoothDisabledBIOS,
		&l.EpdStatus, &l.IPv6, &l.Usage, &l.Notes, &histRaw,
		&l.CreatedAt, &l.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("laptop get: %w", err)
	}
	if err := json.Unmarshal(histRaw, &l.History); err != nil {
		l.History = []HistoryEntry{}
	}
	return &l, nil
}

// Create inserts a new laptop and returns it.
func (s *Store) Create(ctx context.Context, req CreateRequest) (*Laptop, error) {
	entry := HistoryEntry{
		Date:     todayStr(),
		Type:     "Registro",
		Tone:     "neutral",
		Employee: req.Actor,
		Notes:    "Alta inicial en el sistema.",
	}
	histJSON, err := historyJSON([]HistoryEntry{entry})
	if err != nil {
		return nil, err
	}

	var id string
	err = s.pool.QueryRow(ctx, `
		INSERT INTO laptops
		  (serial, model, variant, brand, condition, availability, prep, comodato,
		   powers_on, os, win11_ready, bios_password, wifi, bluetooth,
		   charger_included, last_format_date, employee_name,
		   employee_email, employee_talent_id, employee_manager_email,
		   lcd_ok, expected_return_date,
		   owner, hostname, geography,
		   last_bios_update, bios_details, bluetooth_disabled_bios,
		   epd_status, ipv6, usage, notes, history)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,
		        $18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31,$32,$33)
		RETURNING id`,
		req.Serial, req.Model, req.Variant, req.Brand, req.Condition,
		req.Availability, req.Prep, req.Comodato,
		req.PowersOn, req.OS, req.Win11Ready, req.BIOSPassword,
		req.WiFi, req.Bluetooth, req.ChargerIncluded, req.LastFormatDate,
		req.EmployeeName,
		req.EmployeeEmail, req.EmployeeTalentID, req.EmployeeManagerEmail,
		req.LcdOk, req.ExpectedReturnDate,
		req.Owner, req.Hostname, req.Geography,
		req.LastBIOSUpdate, req.BIOSDetails, req.BluetoothDisabledBIOS,
		req.EpdStatus, req.IPv6, req.Usage, req.Notes, histJSON,
	).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("laptop create: %w", err)
	}
	return s.GetByID(ctx, id)
}

// diffNote builds a human-readable summary of what changed between existing and new values.
func diffNote(existing *Laptop, req UpdateRequest) string {
	type change struct{ field, from, to string }
	var changes []change

	boolStr := func(b bool) string {
		if b { return "Sí" }
		return "No"
	}
	ptrStr := func(s *string) string {
		if s == nil { return "Sin asignar" }
		return *s
	}

	if existing.Availability != req.Availability {
		changes = append(changes, change{"Disponibilidad", existing.Availability, req.Availability})
	}
	if existing.Prep != req.Prep {
		changes = append(changes, change{"Preparación", existing.Prep, req.Prep})
	}
	if existing.Comodato != req.Comodato {
		changes = append(changes, change{"Comodato", existing.Comodato, req.Comodato})
	}
	if ptrStr(existing.EmployeeName) != ptrStr(req.EmployeeName) {
		changes = append(changes, change{"Empleado", ptrStr(existing.EmployeeName), ptrStr(req.EmployeeName)})
	}
	if existing.EmployeeEmail != req.EmployeeEmail {
		changes = append(changes, change{"Correo empleado", existing.EmployeeEmail, req.EmployeeEmail})
	}
	if existing.EmployeeManagerEmail != req.EmployeeManagerEmail {
		changes = append(changes, change{"Correo manager", existing.EmployeeManagerEmail, req.EmployeeManagerEmail})
	}
	if existing.Condition != req.Condition {
		changes = append(changes, change{"Condición", existing.Condition, req.Condition})
	}
	if existing.OS != req.OS {
		changes = append(changes, change{"Sistema operativo", existing.OS, req.OS})
	}
	if existing.BIOSPassword != req.BIOSPassword {
		changes = append(changes, change{"BIOS", existing.BIOSPassword, req.BIOSPassword})
	}
	if existing.LastBIOSUpdate != req.LastBIOSUpdate {
		changes = append(changes, change{"Último cambio BIOS", existing.LastBIOSUpdate, req.LastBIOSUpdate})
	}
	if existing.BIOSDetails != req.BIOSDetails {
		changes = append(changes, change{"Detalles BIOS", existing.BIOSDetails, req.BIOSDetails})
	}
	if existing.BluetoothDisabledBIOS != req.BluetoothDisabledBIOS {
		changes = append(changes, change{"BT deshabilitado BIOS", boolStr(existing.BluetoothDisabledBIOS), boolStr(req.BluetoothDisabledBIOS)})
	}
	if existing.PowersOn != req.PowersOn {
		changes = append(changes, change{"Enciende", existing.PowersOn, req.PowersOn})
	}
	if existing.ChargerIncluded != req.ChargerIncluded {
		changes = append(changes, change{"Cargador", boolStr(existing.ChargerIncluded), boolStr(req.ChargerIncluded)})
	}
	if existing.ExpectedReturnDate != req.ExpectedReturnDate {
		changes = append(changes, change{"Fecha devolución", existing.ExpectedReturnDate, req.ExpectedReturnDate})
	}
	if existing.Owner != req.Owner {
		changes = append(changes, change{"Dueño", existing.Owner, req.Owner})
	}
	if existing.Hostname != req.Hostname {
		changes = append(changes, change{"Hostname", existing.Hostname, req.Hostname})
	}
	if existing.Geography != req.Geography {
		changes = append(changes, change{"Geografía", existing.Geography, req.Geography})
	}

	if len(changes) == 0 {
		return "Datos del equipo actualizados."
	}

	note := ""
	for _, c := range changes {
		if note != "" { note += " | " }
		note += c.field + ": " + c.from + " → " + c.to
	}
	return note
}

// Update modifies an existing laptop and appends a history entry.
func (s *Store) Update(ctx context.Context, id string, req UpdateRequest) (*Laptop, error) {
	existing, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Determine tone based on availability change
	tone := "neutral"
	if existing.Availability != req.Availability {
		if req.Availability == "ASIGNADA" { tone = "ok" }
		if req.Availability == "DISPONIBLE" { tone = "neutral" }
		if req.Availability == "NO_DISPONIBLE" { tone = "danger" }
	}

	entry := HistoryEntry{
		Date:     todayStr(),
		Type:     "Actualización",
		Tone:     tone,
		Employee: req.Actor,
		Notes:    diffNote(existing, req),
	}
	newHistory := append(existing.History, entry)
	histJSON, err := historyJSON(newHistory)
	if err != nil {
		return nil, err
	}

	_, err = s.pool.Exec(ctx, `
		UPDATE laptops SET
		  model=$1, variant=$2, brand=$3, condition=$4,
		  availability=$5, prep=$6, comodato=$7,
		  powers_on=$8, os=$9, win11_ready=$10, bios_password=$11,
		  wifi=$12, bluetooth=$13, charger_included=$14,
		  last_format_date=$15, employee_name=$16,
		  employee_email=$17, employee_talent_id=$18, employee_manager_email=$19,
		  lcd_ok=$20, expected_return_date=$21,
		  owner=$22, hostname=$23, geography=$24,
		  last_bios_update=$25, bios_details=$26, bluetooth_disabled_bios=$27,
		  epd_status=$28, ipv6=$29, usage=$30, notes=$31, history=$32
		WHERE id=$33`,
		req.Model, req.Variant, req.Brand, req.Condition,
		req.Availability, req.Prep, req.Comodato,
		req.PowersOn, req.OS, req.Win11Ready, req.BIOSPassword,
		req.WiFi, req.Bluetooth, req.ChargerIncluded,
		req.LastFormatDate, req.EmployeeName,
		req.EmployeeEmail, req.EmployeeTalentID, req.EmployeeManagerEmail,
		req.LcdOk, req.ExpectedReturnDate,
		req.Owner, req.Hostname, req.Geography,
		req.LastBIOSUpdate, req.BIOSDetails, req.BluetoothDisabledBIOS,
		req.EpdStatus, req.IPv6, req.Usage, req.Notes, histJSON, id,
	)
	if err != nil {
		return nil, fmt.Errorf("laptop update: %w", err)
	}
	return s.GetByID(ctx, id)
}

// GetBySerial returns a single laptop by its serial number.
func (s *Store) GetBySerial(ctx context.Context, serial string) (*Laptop, error) {
	var l Laptop
	var histRaw []byte
	err := s.pool.QueryRow(ctx, `
		SELECT id, serial, model, variant, brand, condition,
		       availability, prep, comodato, powers_on, os, win11_ready,
		       bios_password, wifi, bluetooth, charger_included,
		       last_format_date, employee_name,
		       employee_email, employee_talent_id, employee_manager_email,
		       lcd_ok, expected_return_date,
		       owner, hostname, geography,
		       last_bios_update, bios_details, bluetooth_disabled_bios,
		       epd_status, ipv6, usage, notes, history,
		       created_at, updated_at
		FROM laptops WHERE serial = $1`, serial).Scan(
		&l.ID, &l.Serial, &l.Model, &l.Variant, &l.Brand, &l.Condition,
		&l.Availability, &l.Prep, &l.Comodato, &l.PowersOn, &l.OS, &l.Win11Ready,
		&l.BIOSPassword, &l.WiFi, &l.Bluetooth, &l.ChargerIncluded,
		&l.LastFormatDate, &l.EmployeeName,
		&l.EmployeeEmail, &l.EmployeeTalentID, &l.EmployeeManagerEmail,
		&l.LcdOk, &l.ExpectedReturnDate,
		&l.Owner, &l.Hostname, &l.Geography,
		&l.LastBIOSUpdate, &l.BIOSDetails, &l.BluetoothDisabledBIOS,
		&l.EpdStatus, &l.IPv6, &l.Usage, &l.Notes, &histRaw,
		&l.CreatedAt, &l.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("laptop get by serial: %w", err)
	}
	if err := json.Unmarshal(histRaw, &l.History); err != nil {
		l.History = []HistoryEntry{}
	}
	return &l, nil
}

// Delete removes a laptop by ID.
func (s *Store) Delete(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx, "DELETE FROM laptops WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("laptop delete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("laptop not found")
	}
	return nil
}
