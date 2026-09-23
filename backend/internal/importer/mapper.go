package importer

import "strings"

// ─── Alias catalogue ─────────────────────────────────────────────────────────
// Maps every known alias (lower-case) to a canonical field name.

var headerAliases = map[string]string{
	// Serial
	"serial":                "serial",
	"serial number":         "serial",
	"serial no.":            "serial",
	"serial no":             "serial",
	"sn":                    "serial",
	"s/n":                   "serial",
	"numero de serie":       "serial",
	"machine serial number": "serial", // EPD list format
	"machine serial":        "serial", // EPD list format (short)

	// Usage (laptop-only field)
	"usage":          "usage",
	"device usage":   "usage",
	"uso":            "usage",

	// Device type
	"device_type":    "device_type",
	"device type":    "device_type",
	"type of device": "device_type",
	"equipment type": "device_type",
	"description":    "device_type", // first column in base-file sheets (Laptop/Monitor/etc.)
	"tipo":           "device_type",
	// NOTE: bare "type" is intentionally NOT mapped to device_type — the base-file
	// "Type" column holds model variant text (e.g. "20N3-S5DV14"), not device type.
	// "Type " (trailing space) is mapped below to net_type for the Infrastructure sheet.

	// Brand
	"brand":        "brand",
	"marca":        "brand",
	"manufacturer": "brand",

	// Model / variant
	"model":             "model",
	"modelo":            "model",
	"model description": "model",
	"model variant":     "model_variant",
	"type":              "model_variant", // base-file "Type" column = Lenovo variant string

	// Condition
	"condition": "condition",
	"condicion": "condition",
	"condición": "condition",

	// Location / room
	"location":  "location",
	"room":      "room",
	"ubicacion": "location",
	"ubicación": "location",

	// Availability
	"availability":   "availability",
	"disponibilidad": "availability",

	// Assignability / prep
	"assignability":     "assignability",
	"prep":              "assignability",
	"preparacion":       "assignability",
	"preparación":       "assignability",
	"se puede asignar?": "assignability",
	"se puede asignar":  "assignability",

	// Employee
	"employee_name":   "employee_name",
	"employee name":   "employee_name",
	"name":            "employee_name",
	"nombre":          "employee_name",
	"nombre empleado": "employee_name",
	"employee":        "employee_name",
	"full name":       "employee_name", // EPD list format

	"employee_talent_id": "employee_talent_id",
	"employee number":    "employee_talent_id",
	"employee talent id": "employee_talent_id",
	"employee id":        "employee_talent_id",
	"talent id":          "employee_talent_id",
	"numero empleado":    "employee_talent_id",
	"resource: resource": "employee_talent_id", // EPD list format: "Resource: Resource" column
	"resource":           "employee_talent_id", // EPD list format (fallback)

	"employee_email":  "employee_email",
	"employee email":  "employee_email",
	"email":           "contact_email", // Infrastructure xlsx: provider contact email
	"correo":          "employee_email",
	"correo empleado": "employee_email",

	"employee manager email": "employee_manager_email",
	"manager email":          "employee_manager_email",
	"manager":                "employee_manager_email",

	// OS
	"operating system":  "os",
	"os":                "os",
	"sistema operativo": "os",

	// Technical fields
	"powers on":                "powers_on",
	"powers_on":                "powers_on",
	"carga electrica/enciende": "powers_on",
	"enciende":                 "powers_on",
	"lcd ok":                   "lcd_ok",
	"lcd_ok":                   "lcd_ok",
	"monitor lcd":              "lcd_ok",
	"wifi status":              "wifi",
	"wifi":                     "wifi",
	"bluetooth status":         "bluetooth",
	"bluetooth":                "bluetooth",
	"bios password":            "bios_password",
	"bios_password":            "bios_password",
	"charger included":         "charger_included",
	"charger_included":         "charger_included",
	"os configured":            "os_configured",
	"windows 11 ready":         "win11_ready",
	"win11_ready":               "win11_ready",
	"last formatted date":      "last_format_date",
	"last_format_date":         "last_format_date",
	"last_bios_update":         "last_bios_update",
	"bios_details":             "bios_details",
	"bluetooth_disabled_bios":  "bluetooth_disabled_bios",
	"epd_status":               "epd_status",
	"ipv6":                     "ipv6",

	// Comodato / commodatum
	"commodatum created": "comodato",
	"comodato":           "comodato",
	"firmó comodato":     "comodato",
	"firmo comodato":     "comodato",
	"commodatum doc id":  "comodato_doc_id",
	"commodatum signed":  "comodato_signed",

	// Transaction
	"transaction type":      "transaction_type",
	"transaction date":      "transaction_date",
	"transaction_date":      "transaction_date",
	"dia de entrega/recibo": "transaction_date",
	"expected return date":  "expected_return_date",
	"expected_return_date":  "expected_return_date",

	// Monitor pairing
	"monitor included":      "monitor_included",
	"monitor_included":      "monitor_included",
	"monitor":               "monitor_included",
	"monitor serial number": "monitor_serial",
	"monitor serial":        "monitor_serial",
	"monitor_serial":        "monitor_serial",

	// Owner / hostname / geography
	"owner":               "owner",
	"device owner":        "owner",
	"hostname":            "hostname",
	"host name":           "hostname",
	"machine host name":   "hostname", // EPD list format
	"geography":           "geography",
	"geographic location": "geography",
	"country":             "geography", // EPD list format: "Country" = geography

	// Notes
	"notes":         "notes",
	"observaciones": "notes",
	"comments":      "notes",

	// Traceability
	"created at": "created_at",
	"updated at": "updated_at",

	// Network-specific
	"product id":     "product_id",
	"product number": "product_id",
	"pid":            "product_id",

	"type ":    "net_type", // NOTE trailing space from Excel
	"net type": "net_type",

	"end of sale": "end_of_sale",
	"end of life": "end_of_life",
	"eol":         "end_of_life",
	"eol date":    "end_of_life",

	"end contract support": "end_contract_support",
	"end of contract":      "end_contract_support",

	"devices cost":   "device_cost",
	"device cost":    "device_cost",
	"device price":   "device_cost",
	"contract cost":  "contract_cost",
	"contract price": "contract_cost",

	"eol status":      "eol_status",
	"contract status": "contract_status",

	"device company": "device_company",
	"company":        "device_company",

	"name conctact provider": "contact_name",
	"name contact provider":  "contact_name",
	"contact name":           "contact_name",
	"nombre proveedor":       "contact_name",

	"phone contact  provider": "contact_phone",
	"phone contact provider":  "contact_phone",
	"contact phone":           "contact_phone",

	"ibm -network - email contact support": "ibm_network_email_support",
	"ibm - local - email contact support":  "ibm_local_email_support",

	"covered": "covered",

	// Column1 — always treated as ignored
	"column1":  "IGNORE",
	"column 1": "IGNORE",

	// Proceso column in base-file (workflow status — not stored)
	"proceso": "proceso_status",

	// Internal base-file columns — not stored
	"fixed?": "IGNORE",
}

// ─── Column resolution ────────────────────────────────────────────────────────

// ResolveColumn maps a raw header string to a canonical field name.
// Returns "IGNORE" for known-ignored columns, "" when not recognized.
func ResolveColumn(raw string) string {
	key := NormalizeHeader(raw)
	if v, ok := headerAliases[key]; ok {
		return v
	}
	// Fallback: try stripping trailing spaces (common Excel artefact)
	key2 := strings.TrimRight(key, " ")
	if v, ok := headerAliases[key2]; ok {
		return v
	}
	return ""
}

// ─── Sheet classification ─────────────────────────────────────────────────────

// ClassifySheet returns a hint about what kind of data a sheet likely contains
// based on its name.
func ClassifySheet(name string) EntityKind {
	lower := strings.ToLower(strings.TrimSpace(name))
	switch {
	case strings.Contains(lower, "laptop"), strings.Contains(lower, "equipo prestado"),
		strings.Contains(lower, "bodega"), strings.Contains(lower, "prestado"):
		return EntityLaptop
	case strings.Contains(lower, "tinypc"), strings.Contains(lower, "tiny pc"),
		strings.Contains(lower, "asignadas"), strings.Contains(lower, "desktop"):
		return EntityEquipment
	case strings.Contains(lower, "inventario"), strings.Contains(lower, "inventory"):
		// Could be either — row-level classification will decide
		return EntityUnknown
	case strings.Contains(lower, "isp"), strings.Contains(lower, "infraestructur"),
		strings.Contains(lower, "network"), strings.Contains(lower, "red"):
		return EntityEquipment
	case strings.Contains(lower, "legend"), strings.Contains(lower, "leyenda"):
		return EntityUnknown // metadata sheet, skip
	}
	return EntityUnknown
}

// isMetadataSheet returns true for sheets that should not be imported.
func isMetadataSheet(name string) bool {
	lower := strings.ToLower(strings.TrimSpace(name))
	return lower == "legend" || lower == "leyenda" || lower == "isp"
}
