package equipment

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Store handles all database operations for equipment.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore creates a new Store.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

const selectCols = `
	id, serial, device_type, brand, model, variant, condition,
	availability, assignability, comodato,
	powers_on, os, bios_password, wifi, bluetooth,
	charger_included, last_format_date,
	employee_name, employee_email, employee_talent_id,
	transaction_date, expected_return_date,
	monitor_included, monitor_serial,
	COALESCE(owner,''), COALESCE(hostname,''), COALESCE(geography,''),
	notes, history, created_at, updated_at,
	COALESCE(product_id,''), COALESCE(net_type,''),
	end_of_sale::text, end_of_life::text, end_contract_support::text,
	device_cost::text, contract_cost::text,
	COALESCE(room,''), COALESCE(eol_status,''), COALESCE(contract_status,''),
	COALESCE(device_company,''), COALESCE(contact_name,''),
	COALESCE(contact_phone,''), COALESCE(contact_email,''),
	COALESCE(ibm_network_email_support,''), COALESCE(ibm_local_email_support,'')`

func scanEquipment(row interface {
	Scan(...any) error
}) (*Equipment, error) {
	var e Equipment
	var histRaw []byte
	if err := row.Scan(
		&e.ID, &e.Serial, &e.DeviceType, &e.Brand, &e.Model, &e.Variant, &e.Condition,
		&e.Availability, &e.Assignability, &e.Comodato,
		&e.PowersOn, &e.OS, &e.BIOSPassword, &e.WiFi, &e.Bluetooth,
		&e.ChargerIncluded, &e.LastFormatDate,
		&e.EmployeeName, &e.EmployeeEmail, &e.EmployeeTalentID,
		&e.TransactionDate, &e.ExpectedReturnDate,
		&e.MonitorIncluded, &e.MonitorSerial,
		&e.Owner, &e.Hostname, &e.Geography,
		&e.Notes, &histRaw, &e.CreatedAt, &e.UpdatedAt,
		&e.ProductID, &e.NetType,
		&e.EndOfSale, &e.EndOfLife, &e.EndContractSupport,
		&e.DeviceCost, &e.ContractCost,
		&e.Room, &e.EOLStatus, &e.ContractStatus,
		&e.DeviceCompany, &e.ContactName,
		&e.ContactPhone, &e.ContactEmail,
		&e.IBMNetworkEmailSupport, &e.IBMLocalEmailSupport,
	); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(histRaw, &e.History); err != nil {
		e.History = []HistoryEntry{}
	}
	return &e, nil
}

// List returns all equipment ordered by created_at desc.
func (s *Store) List(ctx context.Context) ([]Equipment, error) {
	rows, err := s.pool.Query(ctx, `SELECT`+selectCols+` FROM equipment ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("equipment list: %w", err)
	}
	defer rows.Close()

	var items []Equipment
	for rows.Next() {
		e, err := scanEquipment(rows)
		if err != nil {
			return nil, fmt.Errorf("equipment list scan: %w", err)
		}
		items = append(items, *e)
	}
	if items == nil {
		items = []Equipment{}
	}
	return items, nil
}

// GetByID returns a single equipment record by UUID.
func (s *Store) GetByID(ctx context.Context, id string) (*Equipment, error) {
	row := s.pool.QueryRow(ctx, `SELECT`+selectCols+` FROM equipment WHERE id = $1`, id)
	e, err := scanEquipment(row)
	if err != nil {
		return nil, fmt.Errorf("equipment get: %w", err)
	}
	return e, nil
}

// Create inserts a new equipment record and returns it.
func (s *Store) Create(ctx context.Context, req CreateRequest) (*Equipment, error) {
	entry := HistoryEntry{
		Date:     todayStr(),
		Type:     "Registro",
		Tone:     "neutral",
		Employee: req.Actor,
		Notes:    "Alta inicial en el sistema.",
	}
	hJSON, err := historyJSON([]HistoryEntry{entry})
	if err != nil {
		return nil, err
	}

	// Convert empty string cost pointers to nil so NUMERIC column receives NULL
	deviceCost := nullableNumericStr(req.DeviceCost)
	contractCost := nullableNumericStr(req.ContractCost)

	var id string
	err = s.pool.QueryRow(ctx, `
		INSERT INTO equipment (
			serial, device_type, brand, model, variant, condition,
			availability, assignability, comodato,
			powers_on, os, bios_password, wifi, bluetooth,
			charger_included, last_format_date,
			employee_name, employee_email, employee_talent_id,
			transaction_date, expected_return_date,
			monitor_included, monitor_serial,
			owner, hostname, geography,
			notes, history,
			product_id, net_type,
			end_of_sale, end_of_life, end_contract_support,
			device_cost, contract_cost,
			room, eol_status, contract_status,
			device_company, contact_name, contact_phone, contact_email,
			ibm_network_email_support, ibm_local_email_support
			) VALUES (
				$1,$2,$3,$4,$5,$6,
				$7,$8,$9,
				$10,$11,$12,$13,$14,
				$15,$16,
				$17,$18,$19,
				$20,$21,
				$22,$23,
				$24,$25,$26,
				$27,$28,
				$29,$30,
				$31,$32,$33,
				$34,$35,
				$36,$37,$38,
				$39,$40,$41,$42,
				$43,$44
			) RETURNING id`,
		req.Serial, req.DeviceType, req.Brand, req.Model, req.Variant, req.Condition,
		req.Availability, req.Assignability, req.Comodato,
		req.PowersOn, req.OS, req.BIOSPassword, req.WiFi, req.Bluetooth,
		req.ChargerIncluded, req.LastFormatDate,
		req.EmployeeName, req.EmployeeEmail, req.EmployeeTalentID,
		req.TransactionDate, req.ExpectedReturnDate,
		req.MonitorIncluded, req.MonitorSerial,
		req.Owner, req.Hostname, req.Geography,
		req.Notes, hJSON,
		req.ProductID, req.NetType,
		req.EndOfSale, req.EndOfLife, req.EndContractSupport,
		deviceCost, contractCost,
		req.Room, req.EOLStatus, req.ContractStatus,
		req.DeviceCompany, req.ContactName, req.ContactPhone, req.ContactEmail,
		req.IBMNetworkEmailSupport, req.IBMLocalEmailSupport,
	).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("equipment create: %w", err)
	}
	return s.GetByID(ctx, id)
}

// diffNoteEquip builds a human-readable summary of what changed.
func diffNoteEquip(existing *Equipment, req UpdateRequest) string {
	type change struct{ field, from, to string }
	var changes []change

	ptrStr := func(s *string) string {
		if s == nil {
			return "Sin asignar"
		}
		return *s
	}

	if existing.Availability != req.Availability {
		changes = append(changes, change{"Disponibilidad", existing.Availability, req.Availability})
	}
	if existing.Assignability != req.Assignability {
		changes = append(changes, change{"Preparación", existing.Assignability, req.Assignability})
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
	if existing.Condition != req.Condition {
		changes = append(changes, change{"Condición", existing.Condition, req.Condition})
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
		if note != "" {
			note += " | "
		}
		note += c.field + ": " + c.from + " → " + c.to
	}
	return note
}

// Update modifies an existing equipment record and appends a history entry.
func (s *Store) Update(ctx context.Context, id string, req UpdateRequest) (*Equipment, error) {
	existing, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	tone := "neutral"
	if existing.Availability != req.Availability {
		if req.Availability == "ASIGNADA" {
			tone = "ok"
		}
		if req.Availability == "NO_DISPONIBLE" {
			tone = "danger"
		}
	}

	entry := HistoryEntry{
		Date:     todayStr(),
		Type:     "Actualización",
		Tone:     tone,
		Employee: req.Actor,
		Notes:    diffNoteEquip(existing, req),
	}
	newHistory := append(existing.History, entry)
	hJSON, err := historyJSON(newHistory)
	if err != nil {
		return nil, err
	}

	deviceCost := nullableNumericStr(req.DeviceCost)
	contractCost := nullableNumericStr(req.ContractCost)

	_, err = s.pool.Exec(ctx, `
		UPDATE equipment SET
			device_type=$1, brand=$2, model=$3, variant=$4, condition=$5,
			availability=$6, assignability=$7, comodato=$8,
			powers_on=$9, os=$10, bios_password=$11, wifi=$12, bluetooth=$13,
			charger_included=$14, last_format_date=$15,
			employee_name=$16, employee_email=$17, employee_talent_id=$18,
			transaction_date=$19, expected_return_date=$20,
			monitor_included=$21, monitor_serial=$22,
			owner=$23, hostname=$24, geography=$25,
			notes=$26, history=$27,
			product_id=$28, net_type=$29,
			end_of_sale=$30, end_of_life=$31, end_contract_support=$32,
			device_cost=$33, contract_cost=$34,
			room=$35, eol_status=$36, contract_status=$37,
			device_company=$38, contact_name=$39, contact_phone=$40, contact_email=$41,
			ibm_network_email_support=$42, ibm_local_email_support=$43
		WHERE id=$44`,
		req.DeviceType, req.Brand, req.Model, req.Variant, req.Condition,
		req.Availability, req.Assignability, req.Comodato,
		req.PowersOn, req.OS, req.BIOSPassword, req.WiFi, req.Bluetooth,
		req.ChargerIncluded, req.LastFormatDate,
		req.EmployeeName, req.EmployeeEmail, req.EmployeeTalentID,
		req.TransactionDate, req.ExpectedReturnDate,
		req.MonitorIncluded, req.MonitorSerial,
		req.Owner, req.Hostname, req.Geography,
		req.Notes, hJSON,
		req.ProductID, req.NetType,
		req.EndOfSale, req.EndOfLife, req.EndContractSupport,
		deviceCost, contractCost,
		req.Room, req.EOLStatus, req.ContractStatus,
		req.DeviceCompany, req.ContactName, req.ContactPhone, req.ContactEmail,
		req.IBMNetworkEmailSupport, req.IBMLocalEmailSupport,
		id,
	)
	if err != nil {
		return nil, fmt.Errorf("equipment update: %w", err)
	}
	return s.GetByID(ctx, id)
}

// GetBySerial returns a single equipment record by its serial number.
func (s *Store) GetBySerial(ctx context.Context, serial string) (*Equipment, error) {
	row := s.pool.QueryRow(ctx, `SELECT`+selectCols+` FROM equipment WHERE serial = $1`, serial)
	e, err := scanEquipment(row)
	if err != nil {
		return nil, fmt.Errorf("equipment get by serial: %w", err)
	}
	return e, nil
}

// Delete removes an equipment record by ID.
func (s *Store) Delete(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx, "DELETE FROM equipment WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("equipment delete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("equipment not found")
	}
	return nil
}

// nullableNumericStr converts an empty or nil string pointer to nil (so that
// the NUMERIC(15,2) column receives NULL), or returns the pointer unchanged
// when the value is a non-empty string.
func nullableNumericStr(s *string) *string {
	if s == nil || *s == "" {
		return nil
	}
	return s
}
