package importexport

import (
	"fmt"
	"strings"

	"github.com/layssagonzalez/device-inventory/backend/internal/enlace"
	"github.com/layssagonzalez/device-inventory/backend/internal/equipment"
	"github.com/layssagonzalez/device-inventory/backend/internal/laptop"
	"github.com/layssagonzalez/device-inventory/backend/internal/normalize"
	"github.com/xuri/excelize/v2"
)

// laptopHeaders defines the column order for the Laptops sheet.
var laptopHeaders = []string{
	"serial", "model", "variant", "brand", "condition",
	"availability", "prep", "comodato",
	"powers_on", "os", "win11_ready", "bios_password",
	"wifi", "bluetooth", "charger_included", "last_format_date",
	"employee_name", "employee_email", "employee_talent_id",
	"lcd_ok", "expected_return_date", "usage", "notes",
}

// equipmentHeaders defines the column order for the Equipment sheet.
var equipmentHeaders = []string{
	"serial", "device_type", "brand", "model", "variant", "condition",
	"availability", "assignability", "comodato",
	"powers_on", "os", "bios_password", "wifi", "bluetooth",
	"charger_included", "last_format_date",
	"employee_name", "employee_email", "employee_talent_id",
	"transaction_date", "expected_return_date",
	"monitor_included", "monitor_serial", "notes",
}

// ── Per-view column definitions ───────────────────────────────────────────────
//
// Each view exports exactly the columns needed to round-trip through import.
// "computer" and "peripherals" share the same equipment schema but the template
// pre-fills device_type with different default values.

// computerLaptopHeaders: columns for the Laptops sheet in Computer Inventory.
var computerLaptopHeaders = []string{
	"serial", "brand", "model", "variant", "owner", "condition",
	"availability", "prep", "comodato",
	"powers_on", "os", "win11_ready", "bios_password",
	"wifi", "bluetooth", "charger_included", "last_format_date",
	"employee_name", "employee_email", "employee_talent_id",
	"lcd_ok", "expected_return_date", "hostname", "geography", "ipv6", "usage", "notes",
}

// computerEquipHeaders: Desktops + Monitors sheet for Computer Inventory.
var computerEquipHeaders = []string{
	"serial", "device_type", "brand", "model", "variant", "owner", "condition",
	"availability", "assignability", "comodato",
	"powers_on", "os", "bios_password", "wifi", "bluetooth",
	"charger_included", "last_format_date",
	"employee_name", "employee_email", "employee_talent_id",
	"transaction_date", "expected_return_date",
	"monitor_included", "monitor_serial", "notes",
}

// peripheralsHeaders: columns for Peripherals (Mouse/Keyboard/Headset/Cable/Adapter).
var peripheralsHeaders = []string{
	"serial", "device_type", "brand", "model", "variant", "owner", "condition",
	"availability", "assignability", "comodato",
	"employee_name", "employee_email", "employee_talent_id",
	"transaction_date", "expected_return_date", "notes",
}

// networkHeaders: columns for Network Infrastructure.
var networkHeaders = []string{
	"serial", "device_type", "brand", "model", "variant", "product_id", "owner",
	"condition", "availability", "net_type", "room",
	"end_of_sale", "end_of_life", "end_contract_support",
	"device_cost", "contract_cost",
	"eol_status", "contract_status",
	"device_company", "contact_name", "contact_phone", "contact_email",
	"ibm_network_email_support", "ibm_local_email_support",
	"notes",
}

// epdHeaders: columns for EPD by Employee view.
var epdHeaders = []string{
	"serial", "brand", "model", "variant", "owner",
	"availability", "prep", "comodato",
	"employee_name", "employee_email", "employee_talent_id", "employee_manager_email",
	"hostname", "geography", "epd_status", "ipv6",
	"expected_return_date", "notes",
}

// biosHeaders: columns for BIOS Control view.
var biosHeaders = []string{
	"serial", "brand", "model", "variant", "owner",
	"bios_password", "last_bios_update", "bios_details", "bluetooth_disabled_bios",
	"employee_name", "employee_email", "employee_talent_id",
	"geography", "ipv6", "notes",
}

var linkHeaders = []string{
	"type", "company", "public_ip", "ip", "velocity",
	"end_date_contract", "months_of_contract", "additional_service",
	"contract_number", "client_number", "identifier_link",
	"name_contact", "phone_contact", "email_contact",
	"support_phone", "support_clave", "current_po", "comments",
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func strPtr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// ── Excel styling helpers ─────────────────────────────────────────────────────

// colWidths maps canonical field keys to a preferred column width (characters).
// Columns not listed here get the default width of 18.
var colWidths = map[string]float64{
	"serial":                   20,
	"brand":                    14,
	"model":                    14,
	"variant":                  22,
	"owner":                    10,
	"condition":                14,
	"availability":             16,
	"prep":                     16,
	"assignability":            16,
	"comodato":                 14,
	"powers_on":                14,
	"os":                       28,
	"win11_ready":              14,
	"bios_password":            20,
	"wifi":                     16,
	"bluetooth":                16,
	"charger_included":         18,
	"last_format_date":         18,
	"employee_name":            32,
	"employee_email":           34,
	"employee_talent_id":       18,
	"employee_manager_email":   34,
	"lcd_ok":                   14,
	"expected_return_date":     20,
	"geography":                20,
	"ipv6":                     28,
	"hostname":                 26,
	"epd_status":               16,
	"last_bios_update":         20,
	"bios_details":             30,
	"bluetooth_disabled_bios":  24,
	"device_type":              18,
	"transaction_date":         20,
	"monitor_included":         18,
	"monitor_serial":           20,
	"product_id":               20,
	"net_type":                 14,
	"room":                     18,
	"end_of_sale":              16,
	"end_of_life":              16,
	"end_contract_support":     22,
	"device_cost":              16,
	"contract_cost":            16,
	"eol_status":               16,
	"contract_status":          18,
	"device_company":           24,
	"contact_name":             26,
	"contact_phone":            20,
	"contact_email":            30,
	"ibm_network_email_support": 34,
	"ibm_local_email_support":   34,
	"notes":                    36,
	"type":                     14,
	"company":                  24,
	"public_ip":                18,
	"ip":                       18,
	"velocity":                 16,
	"end_date_contract":        20,
	"months_of_contract":       18,
	"additional_service":       24,
	"contract_number":          20,
	"client_number":            20,
	"identifier_link":          22,
	"name_contact":             26,
	"phone_contact":            20,
	"email_contact":            30,
	"support_phone":            20,
	"support_clave":            20,
	"current_po":               20,
	"comments":                 36,
}

// applySheetStyle creates a bold header style, applies it to row 1, and sets
// column widths based on the colWidths map (or a default of 18).
func applySheetStyle(f *excelize.File, sheet string, headers []string) error {
	// Bold + light-grey fill for header row
	style, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 11, Color: "1F2328"},
		Fill: excelize.Fill{
			Type:    "pattern",
			Pattern: 1,
			Color:   []string{"D9E1F2"}, // soft blue-grey
		},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: false},
		Border: []excelize.Border{
			{Type: "bottom", Color: "4472C4", Style: 2},
		},
	})
	if err != nil {
		return fmt.Errorf("applySheetStyle NewStyle: %w", err)
	}

	// Apply style to each header cell in row 1
	for col := range headers {
		cell, err := excelize.CoordinatesToCellName(col+1, 1)
		if err != nil {
			return err
		}
		if err = f.SetCellStyle(sheet, cell, cell, style); err != nil {
			return err
		}
	}

	// Freeze the header row so it stays visible when scrolling
	if err = f.SetPanes(sheet, &excelize.Panes{
		Freeze:      true,
		Split:       false,
		XSplit:      0,
		YSplit:      1,
		TopLeftCell: "A2",
		ActivePane:  "bottomLeft",
	}); err != nil {
		return err
	}

	// Set column widths
	for col, key := range headers {
		colName, err := excelize.ColumnNumberToName(col + 1)
		if err != nil {
			return err
		}
		w, ok := colWidths[key]
		if !ok {
			w = 18
		}
		if err = f.SetColWidth(sheet, colName, colName, w); err != nil {
			return err
		}
	}

	// Set row 1 height to 20 for better readability
	return f.SetRowHeight(sheet, 1, 20)
}

// BuildExportFile creates a two-sheet Excel file populated with current DB data.
func BuildExportFile(laptops []laptop.Laptop, equip []equipment.Equipment) (*excelize.File, error) {
	f := excelize.NewFile()

	// ── Laptops sheet ──────────────────────────────────────────────────────
	f.SetSheetName("Sheet1", "Laptops")
	if err := setSheetHeaders(f, "Laptops", laptopHeaders); err != nil {
		return nil, err
	}
	if err := writeLaptopRows(f, "Laptops", laptopHeaders, laptops); err != nil {
		return nil, err
	}

	// ── Equipment sheet ────────────────────────────────────────────────────
	f.NewSheet("Equipment")
	if err := setSheetHeaders(f, "Equipment", equipmentHeaders); err != nil {
		return nil, err
	}
	if err := writeEquipRows(f, "Equipment", equipmentHeaders, equip); err != nil {
		return nil, err
	}

	return f, nil
}

// BuildTemplateFile creates a blank two-sheet Excel file with correct headers only.
func BuildTemplateFile() (*excelize.File, error) {
	return BuildExportFile(nil, nil)
}

// ── View-specific export/template helpers ─────────────────────────────────────

// setSheetHeaders writes a header row to the given sheet.
func setSheetHeaders(f *excelize.File, sheet string, headers []string) error {
	for col, h := range headers {
		cell, err := excelize.CoordinatesToCellName(col+1, 1)
		if err != nil {
			return fmt.Errorf("header cell [%s] col %d: %w", sheet, col, err)
		}
		f.SetCellValue(sheet, cell, h)
	}
	return applySheetStyle(f, sheet, headers)
}

// laptopFieldValue returns the value for a given header key from a Laptop record.
func laptopFieldValue(l laptop.Laptop, key string) any {
	switch key {
	case "serial":                  return l.Serial
	case "model":                   return l.Model
	case "variant":                 return l.Variant
	case "brand":                   return l.Brand
	case "condition":               return l.Condition
	case "availability":            return l.Availability
	case "prep":                    return l.Prep
	case "comodato":                return l.Comodato
	case "powers_on":               return l.PowersOn
	case "os":                      return l.OS
	case "win11_ready":             return boolStr(l.Win11Ready)
	case "bios_password":           return l.BIOSPassword
	case "wifi":                    return l.WiFi
	case "bluetooth":               return l.Bluetooth
	case "charger_included":        return boolStr(l.ChargerIncluded)
	case "last_format_date":        return l.LastFormatDate
	case "employee_name":           return strPtr(l.EmployeeName)
	case "employee_email":          return l.EmployeeEmail
	case "employee_talent_id":      return l.EmployeeTalentID
	case "employee_manager_email":  return l.EmployeeManagerEmail
	case "lcd_ok":                  return l.LcdOk
	case "expected_return_date":    return l.ExpectedReturnDate
	case "owner":                   return l.Owner
	case "hostname":                return l.Hostname
	case "geography":               return l.Geography
	case "last_bios_update":        return l.LastBIOSUpdate
	case "bios_details":            return l.BIOSDetails
	case "bluetooth_disabled_bios": return boolStr(l.BluetoothDisabledBIOS)
	case "epd_status":              return l.EpdStatus
	case "ipv6":                    return l.IPv6
	case "usage":                   return l.Usage
	case "notes":                   return l.Notes
	}
	return ""
}

// equipFieldValue returns the value for a given header key from an Equipment record.
func equipFieldValue(e equipment.Equipment, key string) any {
	switch key {
	case "serial":                      return e.Serial
	case "device_type":                 return e.DeviceType
	case "brand":                       return e.Brand
	case "model":                       return e.Model
	case "variant":                     return e.Variant
	case "condition":                   return e.Condition
	case "availability":                return e.Availability
	case "assignability":               return e.Assignability
	case "comodato":                    return e.Comodato
	case "powers_on":                   return e.PowersOn
	case "os":                          return e.OS
	case "bios_password":               return e.BIOSPassword
	case "wifi":                        return e.WiFi
	case "bluetooth":                   return e.Bluetooth
	case "charger_included":            return boolStr(e.ChargerIncluded)
	case "last_format_date":            return e.LastFormatDate
	case "employee_name":               return strPtr(e.EmployeeName)
	case "employee_email":              return e.EmployeeEmail
	case "employee_talent_id":          return e.EmployeeTalentID
	case "transaction_date":            return e.TransactionDate
	case "expected_return_date":        return e.ExpectedReturnDate
	case "monitor_included":            return boolStr(e.MonitorIncluded)
	case "monitor_serial":              return e.MonitorSerial
	case "owner":                       return e.Owner
	case "hostname":                    return e.Hostname
	case "geography":                   return e.Geography
	case "notes":                       return e.Notes
	case "product_id":                  return e.ProductID
	case "net_type":                    return e.NetType
	case "end_of_sale":                 return strPtr(e.EndOfSale)
	case "end_of_life":                 return strPtr(e.EndOfLife)
	case "end_contract_support":        return strPtr(e.EndContractSupport)
	case "device_cost":                 return strPtr(e.DeviceCost)
	case "contract_cost":               return strPtr(e.ContractCost)
	case "room":                        return e.Room
	case "eol_status":                  return e.EOLStatus
	case "contract_status":             return e.ContractStatus
	case "device_company":              return e.DeviceCompany
	case "contact_name":                return e.ContactName
	case "contact_phone":               return e.ContactPhone
	case "contact_email":               return e.ContactEmail
	case "ibm_network_email_support":   return e.IBMNetworkEmailSupport
	case "ibm_local_email_support":     return e.IBMLocalEmailSupport
	}
	return ""
}

// writeLaptopRows writes data rows for a slice of laptops using the given columns.
func writeLaptopRows(f *excelize.File, sheet string, headers []string, rows []laptop.Laptop) error {
	for ri, l := range rows {
		rowN := ri + 2
		for col, key := range headers {
			cell, err := excelize.CoordinatesToCellName(col+1, rowN)
			if err != nil {
				return fmt.Errorf("laptop data cell: %w", err)
			}
			f.SetCellValue(sheet, cell, laptopFieldValue(l, key))
		}
	}
	return nil
}

// writeEquipRows writes data rows for equipment using the given columns.
func writeEquipRows(f *excelize.File, sheet string, headers []string, rows []equipment.Equipment) error {
	for ri, e := range rows {
		rowN := ri + 2
		for col, key := range headers {
			cell, err := excelize.CoordinatesToCellName(col+1, rowN)
			if err != nil {
				return fmt.Errorf("equipment data cell: %w", err)
			}
			f.SetCellValue(sheet, cell, equipFieldValue(e, key))
		}
	}
	return nil
}

func linkFieldValue(l enlace.Link, key string) any {
	switch key {
	case "type":
		return l.Type
	case "company":
		return l.Company
	case "public_ip":
		return l.PublicIP
	case "ip":
		return l.IP
	case "velocity":
		return l.Velocity
	case "end_date_contract":
		return l.EndDateContract
	case "months_of_contract":
		return l.MonthsOfContract
	case "additional_service":
		return l.AdditionalService
	case "contract_number":
		return l.ContractNumber
	case "client_number":
		return l.ClientNumber
	case "identifier_link":
		return l.IdentifierLink
	case "name_contact":
		return l.NameContact
	case "phone_contact":
		return l.PhoneContact
	case "email_contact":
		return l.EmailContact
	case "support_phone":
		return l.SupportPhone
	case "support_clave":
		return l.SupportClave
	case "current_po":
		return l.CurrentPO
	case "comments":
		return l.Comments
	}
	return ""
}

func writeLinkRows(f *excelize.File, sheet string, headers []string, rows []enlace.Link) error {
	for ri, l := range rows {
		rowN := ri + 2
		for col, key := range headers {
			cell, err := excelize.CoordinatesToCellName(col+1, rowN)
			if err != nil {
				return fmt.Errorf("link data cell: %w", err)
			}
			f.SetCellValue(sheet, cell, linkFieldValue(l, key))
		}
	}
	return nil
}

// BuildViewExportFile creates a view-specific Excel file with current DB data.
// Supported views: "computer", "peripherals", "network", "links", "epd", "bios".
// Falls back to the full two-sheet export for unknown views.
func BuildViewExportFile(view string, laptopRows []laptop.Laptop, equipRows []equipment.Equipment, linkRows []enlace.Link) (*excelize.File, error) {
	f := excelize.NewFile()

	switch view {
	case "computer":
		// Sheet 1: Laptops
		f.SetSheetName("Sheet1", "Laptops")
		if err := setSheetHeaders(f, "Laptops", computerLaptopHeaders); err != nil {
			return nil, err
		}
		if err := writeLaptopRows(f, "Laptops", computerLaptopHeaders, laptopRows); err != nil {
			return nil, err
		}
		// Sheet 2: Equipment (Desktops + Monitors)
		f.NewSheet("Equipment")
		if err := setSheetHeaders(f, "Equipment", computerEquipHeaders); err != nil {
			return nil, err
		}
		if err := writeEquipRows(f, "Equipment", computerEquipHeaders, equipRows); err != nil {
			return nil, err
		}

	case "peripherals":
		f.SetSheetName("Sheet1", "Peripherals")
		if err := setSheetHeaders(f, "Peripherals", peripheralsHeaders); err != nil {
			return nil, err
		}
		if err := writeEquipRows(f, "Peripherals", peripheralsHeaders, equipRows); err != nil {
			return nil, err
		}

	case "network":
		f.SetSheetName("Sheet1", "Network")
		if err := setSheetHeaders(f, "Network", networkHeaders); err != nil {
			return nil, err
		}
		if err := writeEquipRows(f, "Network", networkHeaders, equipRows); err != nil {
			return nil, err
		}

	case "links":
		f.SetSheetName("Sheet1", "Links")
		if err := setSheetHeaders(f, "Links", linkHeaders); err != nil {
			return nil, err
		}
		if err := writeLinkRows(f, "Links", linkHeaders, linkRows); err != nil {
			return nil, err
		}

	case "epd":
		f.SetSheetName("Sheet1", "EPD")
		if err := setSheetHeaders(f, "EPD", epdHeaders); err != nil {
			return nil, err
		}
		if err := writeLaptopRows(f, "EPD", epdHeaders, laptopRows); err != nil {
			return nil, err
		}

	case "bios":
		f.SetSheetName("Sheet1", "BIOS")
		if err := setSheetHeaders(f, "BIOS", biosHeaders); err != nil {
			return nil, err
		}
		if err := writeLaptopRows(f, "BIOS", biosHeaders, laptopRows); err != nil {
			return nil, err
		}

	default:
		// Full generic export — two sheets
		return BuildExportFile(laptopRows, equipRows)
	}

	return f, nil
}

// BuildViewTemplateFile creates a blank view-specific template (headers only, no data).
func BuildViewTemplateFile(view string) (*excelize.File, error) {
	return BuildViewExportFile(view, nil, nil, nil)
}

// strToPtr converts an empty string to nil, otherwise to a *string.
func strToPtr(s string) *string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return &s
}

func parseBool(s string) bool {
	return strings.EqualFold(s, "true") || s == "1"
}

// ParseImportFile parses both sheets and returns laptop/equipment create requests
// plus a slice of error strings for skipped rows.
func ParseImportFile(f *excelize.File) ([]laptop.CreateRequest, []equipment.CreateRequest, []string) {
	var laptopRows []laptop.CreateRequest
	var equipRows  []equipment.CreateRequest
	var errs []string

	// parseLaptopSheet reads any sheet that holds laptop rows (Laptops, EPD, BIOS).
	parseLaptopSheet := func(sheetName string) {
		rows, err := f.GetRows(sheetName)
		if err != nil || len(rows) < 2 {
			return
		}
		idx := headerIndex(rows[0])
		for i, row := range rows[1:] {
			rowN := i + 2
			serial := strings.TrimSpace(strings.ToUpper(get(row, idx["serial"])))
			if serial == "" {
				errs = append(errs, fmt.Sprintf("%s row %d: serial is required", sheetName, rowN))
				continue
			}
			model := get(row, idx["model"])
			// EPD and BIOS sheets may omit model — use "Unknown" so upsert
			// falls through to the keep() path and preserves the existing value.
			if strings.TrimSpace(model) == "" {
				model = "Unknown"
			}
			laptopRows = append(laptopRows, laptop.CreateRequest{
				Serial:                serial,
				Model:                 model,
				Variant:               get(row, idx["variant"]),
				Brand:                 get(row, idx["brand"]),
				Condition:             get(row, idx["condition"]),
				Availability:          get(row, idx["availability"]),
				Prep:                  get(row, idx["prep"]),
				Comodato:              get(row, idx["comodato"]),
				PowersOn:              get(row, idx["powers_on"]),
				OS:                    get(row, idx["os"]),
				Win11Ready:            parseBool(get(row, idx["win11_ready"])),
				BIOSPassword:          get(row, idx["bios_password"]),
				WiFi:                  get(row, idx["wifi"]),
				Bluetooth:             get(row, idx["bluetooth"]),
				ChargerIncluded:       parseBool(get(row, idx["charger_included"])),
				LastFormatDate:        get(row, idx["last_format_date"]),
				EmployeeName:          strToPtr(normalize.Name(get(row, idx["employee_name"]))),
				EmployeeEmail:         normalize.Email(get(row, idx["employee_email"])),
				EmployeeTalentID:      get(row, idx["employee_talent_id"]),
				EmployeeManagerEmail:  get(row, idx["employee_manager_email"]),
				LcdOk:                 get(row, idx["lcd_ok"]),
				ExpectedReturnDate:    get(row, idx["expected_return_date"]),
				Owner:                 get(row, idx["owner"]),
				Hostname:              get(row, idx["hostname"]),
				Geography:             get(row, idx["geography"]),
				LastBIOSUpdate:        get(row, idx["last_bios_update"]),
				BIOSDetails:           get(row, idx["bios_details"]),
				BluetoothDisabledBIOS: parseBool(get(row, idx["bluetooth_disabled_bios"])),
				EpdStatus:             get(row, idx["epd_status"]),
					IPv6:                  get(row, idx["ipv6"]),
					Usage:                 get(row, idx["usage"]),
					Notes:                 get(row, idx["notes"]),
			})
		}
	}

	// parseEquipSheet reads any sheet that holds equipment rows (Equipment, Peripherals, Network).
	parseEquipSheet := func(sheetName string) {
		rows, err := f.GetRows(sheetName)
		if err != nil || len(rows) < 2 {
			return
		}
		idx := headerIndex(rows[0])
		for i, row := range rows[1:] {
			rowN := i + 2
			serial := strings.TrimSpace(strings.ToUpper(get(row, idx["serial"])))
			if serial == "" {
				errs = append(errs, fmt.Sprintf("%s row %d: serial is required", sheetName, rowN))
				continue
			}
			model := get(row, idx["model"])
			if strings.TrimSpace(model) == "" {
				model = "Unknown"
			}
			equipRows = append(equipRows, equipment.CreateRequest{
				Serial:             serial,
				DeviceType:         get(row, idx["device_type"]),
				Brand:              get(row, idx["brand"]),
				Model:              model,
				Variant:            get(row, idx["variant"]),
				Condition:          get(row, idx["condition"]),
				Availability:       get(row, idx["availability"]),
				Assignability:      get(row, idx["assignability"]),
				Comodato:           get(row, idx["comodato"]),
				PowersOn:           get(row, idx["powers_on"]),
				OS:                 get(row, idx["os"]),
				BIOSPassword:       get(row, idx["bios_password"]),
				WiFi:               get(row, idx["wifi"]),
				Bluetooth:          get(row, idx["bluetooth"]),
				ChargerIncluded:    parseBool(get(row, idx["charger_included"])),
				LastFormatDate:     get(row, idx["last_format_date"]),
				EmployeeName:       strToPtr(normalize.Name(get(row, idx["employee_name"]))),
				EmployeeEmail:      normalize.Email(get(row, idx["employee_email"])),
				EmployeeTalentID:   get(row, idx["employee_talent_id"]),
				TransactionDate:    get(row, idx["transaction_date"]),
				ExpectedReturnDate: get(row, idx["expected_return_date"]),
				MonitorIncluded:    parseBool(get(row, idx["monitor_included"])),
				MonitorSerial:      get(row, idx["monitor_serial"]),
				Owner:              get(row, idx["owner"]),
				Geography:          get(row, idx["geography"]),
				Notes:              get(row, idx["notes"]),
				// Network fields
				ProductID:              get(row, idx["product_id"]),
				NetType:                get(row, idx["net_type"]),
				Room:                   get(row, idx["room"]),
				EOLStatus:              get(row, idx["eol_status"]),
				ContractStatus:         get(row, idx["contract_status"]),
				DeviceCompany:          get(row, idx["device_company"]),
				ContactName:            get(row, idx["contact_name"]),
				ContactPhone:           get(row, idx["contact_phone"]),
				ContactEmail:           get(row, idx["contact_email"]),
				IBMNetworkEmailSupport: get(row, idx["ibm_network_email_support"]),
				IBMLocalEmailSupport:   get(row, idx["ibm_local_email_support"]),
			})
		}
	}

	// ── Laptop sheets: standard + view-specific ────────────────────────────
	for _, name := range []string{"Laptops", "EPD", "BIOS"} {
		parseLaptopSheet(name)
	}

	// ── Equipment sheets: standard + view-specific ─────────────────────────
	for _, name := range []string{"Equipment", "Peripherals", "Network"} {
		parseEquipSheet(name)
	}

	return laptopRows, equipRows, errs
}

func IsLinksFormat(f *excelize.File) bool {
	for _, name := range f.GetSheetList() {
		rows, err := f.GetRows(name)
		if err != nil || len(rows) == 0 {
			continue
		}
		idx := linkHeaderIndex(rows[0])
		if _, ok := idx["company"]; ok {
			if _, ok := idx["type"]; ok {
				if _, ok := idx["identifier_link"]; ok {
					return true
				}
			}
		}
	}
	return false
}

func ParseLinksFile(f *excelize.File) ([]enlace.CreateRequest, []string) {
	var rows [][]string
	var err error
	for _, name := range f.GetSheetList() {
		rows, err = f.GetRows(name)
		if err != nil || len(rows) == 0 {
			continue
		}
		idx := linkHeaderIndex(rows[0])
		if _, ok := idx["company"]; ok {
			if _, ok := idx["type"]; ok {
				if _, ok := idx["identifier_link"]; ok {
					break
				}
			}
		}
		rows = nil
	}
	if len(rows) < 2 {
		return nil, nil
	}
	idx := linkHeaderIndex(rows[0])
	var links []enlace.CreateRequest
	var errs []string
	for i, row := range rows[1:] {
		rowN := i + 2
		company := get(row, idx["company"])
		if strings.TrimSpace(company) == "" {
			errs = append(errs, fmt.Sprintf("Links row %d: company is required", rowN))
			continue
		}
		linkType := get(row, idx["type"])
		if linkType == "" {
			linkType = "Primary"
		}
		links = append(links, enlace.CreateRequest{
			Type:              linkType,
			Company:           company,
			PublicIP:          get(row, idx["public_ip"]),
			IP:                get(row, idx["ip"]),
			Velocity:          get(row, idx["velocity"]),
			EndDateContract:   get(row, idx["end_date_contract"]),
			MonthsOfContract:  get(row, idx["months_of_contract"]),
			AdditionalService: get(row, idx["additional_service"]),
			ContractNumber:    get(row, idx["contract_number"]),
			ClientNumber:      get(row, idx["client_number"]),
			IdentifierLink:    get(row, idx["identifier_link"]),
			NameContact:       normalize.Name(get(row, idx["name_contact"])),
			PhoneContact:      get(row, idx["phone_contact"]),
			EmailContact:      normalize.Email(get(row, idx["email_contact"])),
			SupportPhone:      get(row, idx["support_phone"]),
			SupportClave:      get(row, idx["support_clave"]),
			CurrentPO:         get(row, idx["current_po"]),
			Comments:          get(row, idx["comments"]),
		})
	}
	return links, errs
}

func linkHeaderIndex(headers []string) map[string]int {
	m := make(map[string]int, len(headers))
	for i, h := range headers {
		key := strings.ToLower(strings.TrimSpace(h))
		switch key {
		case "type":
			m["type"] = i
		case "company":
			m["company"] = i
		case "ip publica", "ip pública", "public ip", "public_ip":
			m["public_ip"] = i
		case "ip":
			m["ip"] = i
		case "velocity":
			m["velocity"] = i
		case "end date contract", "end_date_contract":
			m["end_date_contract"] = i
		case "months of contract", "months_of_contract":
			m["months_of_contract"] = i
		case "additional services", "additional service", "additional_service":
			m["additional_service"] = i
		case "contract number", "contract_number":
			m["contract_number"] = i
		case "client number", "client_number":
			m["client_number"] = i
		case "identifier link", "identifier_link":
			m["identifier_link"] = i
		case "name contact", "name_contact":
			m["name_contact"] = i
		case "phone contact", "phone_contact":
			m["phone_contact"] = i
		case "email contact", "email_contact":
			m["email_contact"] = i
		case "support phone", "support_phone":
			m["support_phone"] = i
		case "support clave", "support_clave":
			m["support_clave"] = i
		case "current po", "current_po":
			m["current_po"] = i
		case "comments":
			m["comments"] = i
		}
	}
	return m
}

func headerIndex(headers []string) map[string]int {
	m := make(map[string]int, len(headers))
	for i, h := range headers {
		m[strings.TrimSpace(h)] = i
	}
	return m
}

func get(row []string, idx int) string {
	if idx < 0 || idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx])
}

// ── Legacy USAA format parser ─────────────────────────────────────────────────
//
// Recognises the original operational spreadsheet with three sheets:
//   "Equipo Prestado"     — transaction log (Proceso, Serial No., Name, …)
//   "Equipo USAA Bodega"  — storage inventory with technical fields
//   "TinyPCs ya asignadas"— desktops with employee assignment
//
// Returns the same types as ParseImportFile so the caller is identical.
// IsLegacyFormat returns true when the workbook contains "Equipo Prestado"
// (the primary sheet of the legacy file).

// IsLegacyFormat reports whether the uploaded file looks like the legacy USAA spreadsheet.
func IsLegacyFormat(f *excelize.File) bool {
	for _, name := range f.GetSheetList() {
		if strings.TrimSpace(name) == "Equipo Prestado" {
			return true
		}
	}
	return false
}

// ParseLegacyFile converts the 3-sheet USAA operational spreadsheet into
// laptop and equipment create-requests compatible with the upsert pipeline.
func ParseLegacyFile(f *excelize.File) ([]laptop.CreateRequest, []equipment.CreateRequest, []string) {
	var laptops []laptop.CreateRequest
	var equips  []equipment.CreateRequest
	var errs    []string

	// ── helpers ────────────────────────────────────────────────────────────
	naStr := func(v string) bool {
		t := strings.TrimSpace(strings.ToUpper(v))
		return t == "" || t == "N/A" || t == "NA" || t == "NULL"
	}
	cleanStr := func(v string) string {
		if naStr(v) { return "" }
		return strings.TrimSpace(v)
	}
	cleanSerial := func(v string) string {
		s := strings.TrimSpace(strings.ToUpper(v))
		s = strings.ReplaceAll(s, "-", "")
		s = strings.ReplaceAll(s, " ", "")
		if len(s) > 1 {
			// Strip spurious leading S before PF / PC / MJ serials
			if (s[0] == 'S') && len(s) > 2 {
				rest := s[1:]
				if strings.HasPrefix(rest, "PF") || strings.HasPrefix(rest, "PC") || strings.HasPrefix(rest, "MJ") {
					s = rest
				}
			}
		}
		return s
	}
	mapComodato := func(v string) string {
		t := strings.ToUpper(strings.TrimSpace(v))
		if t == "YES" || t == "DONE" || t == "SIGNED" || t == "FIRMADO" { return "FIRMADO" }
		if t == "PENDING TO SIGN" || t == "PENDING" || t == "PENDIENTE"  { return "PENDIENTE" }
		return "N/A"
	}
	mapPowersOn := func(v string) string {
		if strings.EqualFold(strings.TrimSpace(v), "NOT WORKING") { return "NO ENCIENDE" }
		return "OK"
	}
	mapWifi := func(v string) string {
		t := strings.ToUpper(strings.TrimSpace(v))
		if strings.Contains(t, "ENABLED") || strings.Contains(t, "HABILITADO") { return "Habilitado" }
		if strings.Contains(t, "DISABLED") || strings.Contains(t, "DESHABILITADO") { return "Deshabilitado" }
		return "Desconocido"
	}
	mapCondition := func(v string) string {
		t := strings.ToUpper(strings.TrimSpace(v))
		if strings.Contains(t, "NEW") || strings.Contains(t, "NUEVO") { return "Nuevo" }
		if strings.Contains(t, "DAMAGE") || strings.Contains(t, "DAÑA") { return "Dañado" }
		if strings.Contains(t, "FAIR") || strings.Contains(t, "REGULAR") { return "Regular" }
		if v == "" { return "Bueno" }
		return "Bueno"
	}

	// ── read sheets ────────────────────────────────────────────────────────
	type rowMap = map[string]string
	readSheet := func(name string) []rowMap {
		rows, err := f.GetRows(name)
		if err != nil || len(rows) < 2 { return nil }
		headers := rows[0]
		var out []rowMap
		for _, row := range rows[1:] {
			m := make(rowMap, len(headers))
			for i, h := range headers {
				k := strings.TrimSpace(h)
				if k == "" { continue }
				v := ""
				if i < len(row) { v = strings.TrimSpace(row[i]) }
				m[k] = v
			}
			out = append(out, m)
		}
		return out
	}

	prestado := readSheet("Equipo Prestado")
	bodega   := readSheet("Equipo USAA Bodega")
	tinypcs  := readSheet("TinyPCs ya asignadas")

	// ── build lookup maps ──────────────────────────────────────────────────
	techBySerial := make(map[string]rowMap)
	for _, r := range bodega {
		s := cleanSerial(r["Serial No."])
		if s == "" { continue }
		techBySerial[s] = r
	}
	tinyBySerial := make(map[string]rowMap)
	for _, r := range tinypcs {
		s := cleanSerial(r["Serial Number"])
		if s == "" { continue }
		tinyBySerial[s] = r
	}

	// ── group prestado by serial ───────────────────────────────────────────
	type serialInfo struct {
		rows      []rowMap
		desc      string
		bodegaRow rowMap
		tinyRow   rowMap
	}
	bySerial := make(map[string]*serialInfo)
	ensure := func(serial, desc string) {
		if _, ok := bySerial[serial]; !ok {
			bySerial[serial] = &serialInfo{desc: strings.ToLower(desc)}
		}
	}

	for _, r := range prestado {
		serial := cleanSerial(r["Serial No."])
		if serial == "" || naStr(serial) { continue }
		desc := r["Description"]
		ensure(serial, desc)
		bySerial[serial].rows = append(bySerial[serial].rows, r)
	}
	for _, r := range bodega {
		serial := cleanSerial(r["Serial No."])
		if serial == "" || naStr(serial) { continue }
		ensure(serial, r["Description"])
		bySerial[serial].bodegaRow = r
	}

	// ── bodega rows with no serial (e.g. Steren adapters) get a synthetic serial ──
	// Format: ADAPTER-{sanitised-desc}-{1-based-index}
	synthIdx := 0
	for _, r := range bodega {
		rawSerial := r["Serial No."]
		if !naStr(rawSerial) { continue } // already handled above
		desc := strings.TrimSpace(r["Description"])
		if desc == "" { continue }
		descLower := strings.ToLower(desc)
		// Only synthesise serials for equipment, not laptops
		if !strings.Contains(descLower, "adaptador") && !strings.Contains(descLower, "adapter") &&
			!strings.Contains(descLower, "monitor") && !strings.Contains(descLower, "cable") &&
			!strings.Contains(descLower, "mouse") && !strings.Contains(descLower, "keyboard") &&
			!strings.Contains(descLower, "headset") {
			continue
		}
		synthIdx++
		// Build deterministic serial from description + index so re-imports are idempotent
		safeName := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(desc), " ", "-"))
		safeName = strings.Map(func(ru rune) rune {
			if (ru >= 'A' && ru <= 'Z') || (ru >= '0' && ru <= '9') || ru == '-' {
				return ru
			}
			return -1
		}, safeName)
		synthetic := fmt.Sprintf("SYN-%s-%03d", safeName, synthIdx)
		ensure(synthetic, desc)
		bySerial[synthetic].bodegaRow = r
	}
	for _, r := range tinypcs {
		serial := cleanSerial(r["Serial Number"])
		if serial == "" || naStr(serial) { continue }
		ensure(serial, "tiny desktop")
		bySerial[serial].tinyRow = r
	}

	// ── build payloads ─────────────────────────────────────────────────────
	for serial, info := range bySerial {
		// Determine device type from description / serial prefix
		desc := info.desc
		deviceType := "Laptop"
		if info.tinyRow != nil || strings.HasPrefix(serial, "MJ") {
			deviceType = "Desktop"
		} else if strings.Contains(desc, "monitor") {
			deviceType = "Monitor"
		} else if strings.Contains(desc, "adaptador") || strings.Contains(desc, "adapter") {
			deviceType = "Adapter"
		}

		// Last transaction row determines current state
		var lastRow rowMap
		if len(info.rows) > 0 {
			lastRow = info.rows[len(info.rows)-1]
		}

		// ── availability / employee ────────────────────────────────────────
		availability := "DISPONIBLE"
		employeeName := ""
		employeeEmail := ""
		employeeTalentID := ""
		comodato := "N/A"
		transactionDate := ""

		if tr := info.tinyRow; tr != nil {
			loc := strings.ToUpper(cleanStr(tr["Location"]))
			if loc == "ASSIGNED" { availability = "ASIGNADA" }
			employeeName  = cleanStr(tr["Name"])
			employeeEmail = strings.ToLower(cleanStr(tr["Employee Email"]))
			employeeTalentID = cleanStr(tr["Employee TalentID"])
			comodato = mapComodato(tr["Comodatum"])
		} else if lastRow != nil {
			proc := strings.ToUpper(cleanStr(lastRow["Proceso"]))
			if proc == "ASIGNADA" || proc == "ENTREGA" { availability = "ASIGNADA" }
			employeeName = cleanStr(lastRow["Name"])
			comodato     = mapComodato(lastRow["Firmó Comodato"])
			transactionDate = cleanStr(lastRow["Dia de entrega/Recibo"])
		}
		// Regla universal: si hay empleado asignado, la availability siempre es ASIGNADA.
		// El archivo USAA no siempre tiene "ASIGNADA" en Proceso pero sí tiene el nombre.
		if employeeName != "" && employeeName != "TBD" && !naStr(employeeName) {
			availability = "ASIGNADA"
		}

		// Bodega overrides assignability
		prep := "NECESITA_PREP"
		if br := info.bodegaRow; br != nil {
			a := strings.ToUpper(cleanStr(br["Se puede asignar?"]))
			if a == "NO ASIGNABLE"          { availability = "NO_DISPONIBLE"; prep = "NO_FUNCIONAL" }
			if a == "ASIGNABLE" || a == "ASSIGNED" { prep = "LISTA" }
		} else if availability == "ASIGNADA" {
			prep = "LISTA"
		}

		// ── technical fields ───────────────────────────────────────────────
		tech := info.bodegaRow
		if tech == nil { tech = rowMap{} }
		powersOn  := mapPowersOn(firstNonEmpty(tech["Carga electrica/enciende"], tech["Powers On"]))
		lcdOk     := mapPowersOn(firstNonEmpty(tech["Monitor LCD"], "OK"))
		biosPass  := cleanStr(firstNonEmpty(tech["BIOS PASSWORD"], tech["BIOS"]))
		os        := cleanStr(firstNonEmpty(tech["OS"], tech["Operating System"]))
		wifi      := mapWifi(firstNonEmpty(tech["WIFI"], tech["WiFi Status"]))
		bt        := mapWifi(firstNonEmpty(tech["BLUETOOTH"], tech["Bluetooth Status"]))
		condition := mapCondition(tech["Condition"])
		notes     := cleanStr(firstNonEmpty(tech["Observaciones"], tech["Comments"], tech["FIXED?"]))

		// model / variant — try in priority order
		model := ""
		variant := ""
		if lastRow != nil {
			model   = cleanStr(firstNonEmpty(lastRow["Model"], ""))
			variant = cleanStr(firstNonEmpty(lastRow["Type"],  ""))
		}
		if model == "" && info.tinyRow != nil   { model   = cleanStr(info.tinyRow["Model"]) }
		if model == "" && info.bodegaRow != nil { model   = cleanStr(info.bodegaRow["Model"]) }
		if variant == "" && info.bodegaRow != nil { variant = cleanStr(info.bodegaRow["Type"]) }
		if model == "" {
			switch deviceType {
			case "Desktop": model = "M920Q"
			case "Laptop":  model = "T490"
			default:        model = deviceType
			}
		}

		win11Ready := strings.Contains(strings.ToLower(os), "windows 11") && prep == "LISTA"

		if biosPass == "" { biosPass = "SIN CONTRASEÑA" }
		if wifi == "" { wifi = "Desconocido" }
		if bt == "" { bt = "Desconocido" }

		// monitor
		monitorIncluded := false
		monitorSerial   := ""
		if tr := info.tinyRow; tr != nil {
			monitorIncluded = strings.ToUpper(cleanStr(tr["Monitor"])) == "YES"
			monitorSerial   = cleanStr(tr["Monitor serial"])
		}

		if deviceType == "Laptop" {
			if naStr(biosPass) { biosPass = "SIN CONTRASEÑA" }
			laptops = append(laptops, laptop.CreateRequest{
					Serial:           serial,
					Model:            model,
					Variant:          variant,
					Brand:            "Lenovo",
					Condition:        condition,
					Availability:     availability,
					Prep:             prep,
					Comodato:         comodato,
					PowersOn:         powersOn,
					OS:               os,
					Win11Ready:       win11Ready,
					BIOSPassword:     biosPass,
					WiFi:             wifi,
					Bluetooth:        bt,
					ChargerIncluded:  true,
					EmployeeName:     strToPtr(normalize.Name(employeeName)),
					EmployeeEmail:    normalize.Email(employeeEmail),
					EmployeeTalentID: employeeTalentID,
					LcdOk:            lcdOk,
					Notes:            notes,
					Owner:            "USAA",
				})
		} else {
			assignability := prep
			if assignability == "" { assignability = "NECESITA_PREP" }
			equips = append(equips, equipment.CreateRequest{
					Serial:          serial,
					DeviceType:      deviceType,
					Brand:           "Lenovo",
					Model:           model,
					Variant:         variant,
					Condition:       condition,
					Availability:    availability,
					Assignability:   assignability,
					Comodato:        comodato,
					PowersOn:        powersOn,
					OS:              os,
					BIOSPassword:    biosPass,
					WiFi:            wifi,
					Bluetooth:       bt,
					ChargerIncluded: false,
					EmployeeName:    strToPtr(normalize.Name(employeeName)),
					EmployeeEmail:   normalize.Email(employeeEmail),
					EmployeeTalentID: employeeTalentID,
					TransactionDate:  transactionDate,
					MonitorIncluded:  monitorIncluded,
					MonitorSerial:    monitorSerial,
					Notes:            notes,
					Owner:            "USAA",
				})
		}
	}

	if len(laptops) == 0 && len(equips) == 0 {
		errs = append(errs, "no rows could be parsed from the legacy sheets — check that the file has 'Equipo Prestado', 'Equipo USAA Bodega', or 'TinyPCs ya asignadas' sheets with data")
	}
	return laptops, equips, errs
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		t := strings.TrimSpace(v)
		u := strings.ToUpper(t)
		if t != "" && u != "N/A" && u != "NA" && u != "NULL" {
			return t
		}
	}
	return ""
}

// ── IBM EPD list format parser ────────────────────────────────────────────────
//
// Recognises the IBM EPD employee-device list format:
//   Columns: Resource: Resource, Full Name, Email, Machine Serial Number,
//            Machine Host Name, Country, …
// All rows are laptops assigned to IBM employees.

// IsEPDFormat reports whether the uploaded file looks like the IBM EPD list.
func IsEPDFormat(f *excelize.File) bool {
	for _, name := range f.GetSheetList() {
		rows, err := f.GetRows(name)
		if err != nil || len(rows) == 0 {
			continue
		}
		headers := rows[0]
		hasMachineSerial := false
		hasFullName := false
		for _, h := range headers {
			lh := strings.ToLower(strings.TrimSpace(h))
			if lh == "machine serial number" || lh == "machine serial" {
				hasMachineSerial = true
			}
			if lh == "full name" {
				hasFullName = true
			}
		}
		if hasMachineSerial && hasFullName {
			return true
		}
	}
	return false
}

// ParseEPDFile parses the IBM EPD employee-device list. All rows produce
// laptop upsert requests: the serial is used to match existing records and
// update employee assignment fields.
func ParseEPDFile(f *excelize.File) ([]laptop.CreateRequest, []equipment.CreateRequest, []string) {
	var laptops []laptop.CreateRequest
	var errs    []string

	colIdx := func(headers []string, names ...string) int {
		for _, name := range names {
			for i, h := range headers {
				if strings.EqualFold(strings.TrimSpace(h), name) {
					return i
				}
			}
		}
		return -1
	}

	for _, sheetName := range f.GetSheetList() {
		rows, err := f.GetRows(sheetName)
		if err != nil || len(rows) < 2 {
			continue
		}
		headers := rows[0]

		// Locate columns by name
		idxSerial   := colIdx(headers, "Machine Serial Number", "Machine Serial")
		idxName     := colIdx(headers, "Full Name", "Name")
		idxEmail    := colIdx(headers, "Email")
		idxTalentID := colIdx(headers, "Resource: Resource", "Resource")
		idxHostname := colIdx(headers, "Machine Host Name", "Host Name", "Hostname")
		idxCountry  := colIdx(headers, "Country", "Geography", "Geographic Location")

		if idxSerial < 0 {
			continue // not the EPD sheet
		}

		for i, row := range rows[1:] {
			rowN := i + 2
			serial := strings.TrimSpace(strings.ToUpper(get(row, idxSerial)))
			if serial == "" {
				errs = append(errs, fmt.Sprintf("EPD row %d: serial is empty, skipping", rowN))
				continue
			}
			employeeName  := ""
			if idxName >= 0 {
				employeeName = strings.TrimSpace(get(row, idxName))
			}
			employeeEmail := ""
			if idxEmail >= 0 {
				employeeEmail = strings.ToLower(strings.TrimSpace(get(row, idxEmail)))
			}
			talentID := ""
			if idxTalentID >= 0 {
				talentID = strings.TrimSpace(get(row, idxTalentID))
			}
			hostname := ""
			if idxHostname >= 0 {
				hostname = strings.TrimSpace(get(row, idxHostname))
			}
			geography := ""
			if idxCountry >= 0 {
				geography = strings.TrimSpace(get(row, idxCountry))
			}

			laptops = append(laptops, laptop.CreateRequest{
				Serial:           serial,
				Model:            "Unknown", // EPD format has no model — preserved on UPDATE if serial exists
				Brand:            "IBM",
				Condition:        "Good",
				Availability:     "ASIGNADA",
				Prep:             "LISTA",
				Comodato:         "N/A",
				PowersOn:         "OK",
				BIOSPassword:     "SIN CONTRASEÑA",
				WiFi:             "Desconocido",
				Bluetooth:        "Desconocido",
				EmployeeName:     strToPtr(normalize.Name(employeeName)),
				EmployeeEmail:    normalize.Email(employeeEmail),
				EmployeeTalentID: talentID,
				Hostname:         hostname,
				Geography:        geography,
				Owner:            "IBM",
			})
		}
	}

	return laptops, nil, errs
}

// ── BIOS annual change format parser ─────────────────────────────────────────
//
// Recognises the "Cambio de BIOS Anual" spreadsheet with a single sheet "Data"
// and columns: Name for Mailing, IBM SN, IBM Mail, Serial Number Laptop,
// Type Number Laptop, Cambio Password Done?, When, Location, Comments.
//
// Maps to laptop fields:
//   serial              ← Serial Number Laptop   (cleaned, N/A rows skipped)
//   bios_password       ← derived: "Done" → "CAMBIADO", "N/A"/"N" → keep existing
//   last_bios_update    ← When (date formatted DD/MM/YYYY)
//   bios_details        ← Comments
//   bluetooth_disabled_bios ← Apply BIOS password change: Y → true
//   employee_name       ← Name for Mailing
//   employee_email      ← IBM Mail
//   employee_talent_id  ← IBM SN
//   geography           ← Location
//   variant             ← Type Number Laptop

// IsBIOSFormat reports whether the workbook is the BIOS annual change file.
func IsBIOSFormat(f *excelize.File) bool {
	for _, name := range f.GetSheetList() {
		rows, err := f.GetRows(name)
		if err != nil || len(rows) == 0 {
			continue
		}
		hasSerial := false
		hasBIOSCol := false
		for _, h := range rows[0] {
			lh := strings.ToLower(strings.TrimSpace(h))
			if lh == "serial number laptop" {
				hasSerial = true
			}
			if lh == "cambio password done?" || lh == "apply bios password change" {
				hasBIOSCol = true
			}
		}
		if hasSerial && hasBIOSCol {
			return true
		}
	}
	return false
}

// ParseBIOSFile parses the BIOS annual change spreadsheet and returns laptop
// update requests (keyed by serial). Only rows with a valid serial are included.
func ParseBIOSFile(f *excelize.File) ([]laptop.CreateRequest, []equipment.CreateRequest, []string) {
	var laptops []laptop.CreateRequest
	var errs    []string

	colIdx := func(headers []string, names ...string) int {
		for _, name := range names {
			for i, h := range headers {
				if strings.EqualFold(strings.TrimSpace(h), name) {
					return i
				}
			}
		}
		return -1
	}

	naVal := func(v string) bool {
		u := strings.ToUpper(strings.TrimSpace(v))
		return u == "" || u == "N/A" || u == "NA" || u == "NULL" || u == "N"
	}

	formatDate := func(raw string) string {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return ""
		}
		// excelize returns dates as ISO strings like "2023-05-02T00:00:00.000Z"
		// Strip the time part and convert to DD/MM/YYYY
		if len(raw) >= 10 {
			datePart := raw[:10] // YYYY-MM-DD
			parts := strings.Split(datePart, "-")
			if len(parts) == 3 {
				return parts[2] + "/" + parts[1] + "/" + parts[0]
			}
		}
		return raw
	}

	for _, sheetName := range f.GetSheetList() {
		rows, err := f.GetRows(sheetName)
		if err != nil || len(rows) < 2 {
			continue
		}
		headers := rows[0]

		idxSerial   := colIdx(headers, "Serial Number Laptop")
		idxDone     := colIdx(headers, "Cambio Password Done?")
		idxWhen     := colIdx(headers, "When")
		idxApply    := colIdx(headers, "Apply BIOS password change")
		idxType     := colIdx(headers, "Type Number Laptop")
		idxName     := colIdx(headers, "Name for Mailing")
		idxEmail    := colIdx(headers, "IBM Mail")
		idxSN       := colIdx(headers, "IBM SN")
		idxLocation := colIdx(headers, "Location")
		idxComments := colIdx(headers, "Comments")

		if idxSerial < 0 {
			continue
		}

		for i, row := range rows[1:] {
			rowN := i + 2

			rawSerial := get(row, idxSerial)
			// Clean serial: remove dashes, spaces, leading S
			serial := strings.ToUpper(strings.TrimSpace(rawSerial))
			serial = strings.ReplaceAll(serial, "-", "")
			serial = strings.ReplaceAll(serial, " ", "")
			if len(serial) > 1 && serial[0] == 'S' {
				rest := serial[1:]
				if strings.HasPrefix(rest, "PF") || strings.HasPrefix(rest, "PC") {
					serial = rest
				}
			}
			if naVal(serial) {
				continue
			}

			// BIOS password done status
			doneRaw := strings.TrimSpace(get(row, idxDone))
			biosPass := ""
			switch strings.ToUpper(doneRaw) {
			case "DONE":
				biosPass = "CAMBIADO"
			case "BAJA":
				biosPass = "BAJA"
			default:
				// N/A, empty, or other — don't set; keep() will preserve existing
			}

			// Date of BIOS change
			lastBIOS := ""
			if idxWhen >= 0 {
				lastBIOS = formatDate(get(row, idxWhen))
			}

			// Apply BIOS password change: Y → bluetooth_disabled_bios = true
			btDisabled := false
			if idxApply >= 0 {
				btDisabled = strings.EqualFold(strings.TrimSpace(get(row, idxApply)), "y")
			}

			variant := ""
			if idxType >= 0 {
				variant = strings.TrimSpace(get(row, idxType))
				if naVal(variant) { variant = "" }
			}

			employeeName := ""
			if idxName >= 0 {
				employeeName = strings.TrimSpace(get(row, idxName))
			}
			employeeEmail := ""
			if idxEmail >= 0 {
				employeeEmail = strings.ToLower(strings.TrimSpace(get(row, idxEmail)))
				if naVal(employeeEmail) { employeeEmail = "" }
			}
			talentID := ""
			if idxSN >= 0 {
				talentID = strings.TrimSpace(get(row, idxSN))
			}
			geography := ""
			if idxLocation >= 0 {
				geography = strings.TrimSpace(get(row, idxLocation))
				if naVal(geography) { geography = "" }
			}
			comments := ""
			if idxComments >= 0 {
				comments = strings.TrimSpace(get(row, idxComments))
				if naVal(comments) { comments = "" }
			}

			if serial == "" {
				errs = append(errs, fmt.Sprintf("BIOS row %d: serial is empty after cleaning, skipping", rowN))
				continue
			}

			laptops = append(laptops, laptop.CreateRequest{
				Serial:              serial,
				Model:               "Unknown", // preserved by keepAvail if serial exists
				BIOSPassword:        biosPass,
				LastBIOSUpdate:      lastBIOS,
				BluetoothDisabledBIOS: btDisabled,
				BIOSDetails:         comments,
				Variant:             variant,
				EmployeeName:        strToPtr(normalize.Name(employeeName)),
				EmployeeEmail:       normalize.Email(employeeEmail),
				EmployeeTalentID:    talentID,
				Geography:           geography,
				Owner:               "IBM",
				Availability:        "ASIGNADA",
				Prep:                "LISTA",
				Comodato:            "N/A",
				PowersOn:            "OK",
				WiFi:                "Desconocido",
				Bluetooth:           "Desconocido",
			})
		}
	}

	if len(laptops) == 0 {
		errs = append(errs, "no BIOS rows found — check that the file has a sheet with 'Serial Number Laptop' and 'Cambio Password Done?' columns")
	}
	return laptops, nil, errs
}
