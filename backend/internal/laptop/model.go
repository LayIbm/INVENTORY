package laptop

import (
	"encoding/json"
	"time"
)

// HistoryEntry represents a single event in a laptop's loan history.
type HistoryEntry struct {
	Date     string `json:"date"`
	Type     string `json:"type"`
	Tone     string `json:"tone"`
	Employee string `json:"employee"`
	Notes    string `json:"notes"`
}

// Laptop mirrors the laptops table.
type Laptop struct {
	ID              string         `json:"id"`
	Serial          string         `json:"serial"`
	Model           string         `json:"model"`
	Variant         string         `json:"variant"`
	Brand           string         `json:"brand"`
	Condition       string         `json:"condition"`
	Availability    string         `json:"availability"`
	Prep            string         `json:"prep"`
	Comodato        string         `json:"comodato"`
	PowersOn        string         `json:"powers_on"`
	OS              string         `json:"os"`
	Win11Ready      bool           `json:"win11_ready"`
	BIOSPassword    string         `json:"bios_password"`
	WiFi            string         `json:"wifi"`
	Bluetooth       string         `json:"bluetooth"`
	ChargerIncluded    bool           `json:"charger_included"`
	LastFormatDate     string         `json:"last_format_date"`
	EmployeeName       *string        `json:"employee_name"`
	EmployeeEmail        string         `json:"employee_email"`
	EmployeeTalentID     string         `json:"employee_talent_id"`
	EmployeeManagerEmail string         `json:"employee_manager_email"`
	LcdOk                string         `json:"lcd_ok"`
	ExpectedReturnDate      string         `json:"expected_return_date"`
	Owner                   string         `json:"owner"`
	Hostname                string         `json:"hostname"`
	Geography               string         `json:"geography"`
	LastBIOSUpdate          string         `json:"last_bios_update"`
	BIOSDetails             string         `json:"bios_details"`
	BluetoothDisabledBIOS   bool           `json:"bluetooth_disabled_bios"`
	EpdStatus               string         `json:"epd_status"`
	IPv6                    string         `json:"ipv6"`
	Usage                   string         `json:"usage"`
	Notes                   string         `json:"notes"`
	History         []HistoryEntry `json:"history"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

// CreateRequest is the payload for POST /api/laptops.
type CreateRequest struct {
	Serial          string  `json:"serial"`
	Model           string  `json:"model"`
	Variant         string  `json:"variant"`
	Brand           string  `json:"brand"`
	Condition       string  `json:"condition"`
	Availability    string  `json:"availability"`
	Prep            string  `json:"prep"`
	Comodato        string  `json:"comodato"`
	PowersOn        string  `json:"powers_on"`
	OS              string  `json:"os"`
	Win11Ready      bool    `json:"win11_ready"`
	BIOSPassword    string  `json:"bios_password"`
	WiFi            string  `json:"wifi"`
	Bluetooth       string  `json:"bluetooth"`
	ChargerIncluded    bool    `json:"charger_included"`
	LastFormatDate     string  `json:"last_format_date"`
	EmployeeName         *string `json:"employee_name"`
	EmployeeEmail        string  `json:"employee_email"`
	EmployeeTalentID     string  `json:"employee_talent_id"`
	EmployeeManagerEmail string  `json:"employee_manager_email"`
	LcdOk                string  `json:"lcd_ok"`
	ExpectedReturnDate      string  `json:"expected_return_date"`
	Owner                   string  `json:"owner"`
	Hostname                string  `json:"hostname"`
	Geography               string  `json:"geography"`
	LastBIOSUpdate          string  `json:"last_bios_update"`
	BIOSDetails             string  `json:"bios_details"`
	BluetoothDisabledBIOS   bool    `json:"bluetooth_disabled_bios"`
	EpdStatus               string  `json:"epd_status"`
	IPv6                    string  `json:"ipv6"`
	Usage                   string  `json:"usage"`
	Notes                   string  `json:"notes"`
	// Actor is set by the handler from JWT claims, not from request body.
	Actor string `json:"-"`
}

// UpdateRequest is the payload for PUT /api/laptops/:id.
type UpdateRequest struct {
	Model           string  `json:"model"`
	Variant         string  `json:"variant"`
	Brand           string  `json:"brand"`
	Condition       string  `json:"condition"`
	Availability    string  `json:"availability"`
	Prep            string  `json:"prep"`
	Comodato        string  `json:"comodato"`
	PowersOn        string  `json:"powers_on"`
	OS              string  `json:"os"`
	Win11Ready      bool    `json:"win11_ready"`
	BIOSPassword    string  `json:"bios_password"`
	WiFi            string  `json:"wifi"`
	Bluetooth       string  `json:"bluetooth"`
	ChargerIncluded      bool    `json:"charger_included"`
	LastFormatDate       string  `json:"last_format_date"`
	EmployeeName         *string `json:"employee_name"`
	EmployeeEmail        string  `json:"employee_email"`
	EmployeeTalentID     string  `json:"employee_talent_id"`
	EmployeeManagerEmail string  `json:"employee_manager_email"`
	LcdOk                string  `json:"lcd_ok"`
	ExpectedReturnDate      string  `json:"expected_return_date"`
	Owner                   string  `json:"owner"`
	Hostname                string  `json:"hostname"`
	Geography               string  `json:"geography"`
	LastBIOSUpdate          string  `json:"last_bios_update"`
	BIOSDetails             string  `json:"bios_details"`
	BluetoothDisabledBIOS   bool    `json:"bluetooth_disabled_bios"`
	EpdStatus               string  `json:"epd_status"`
	IPv6                    string  `json:"ipv6"`
	Usage                   string  `json:"usage"`
	Notes                   string  `json:"notes"`
	Actor                   string  `json:"-"`
}

// historyJSON marshals a slice of HistoryEntry to a JSON []byte.
func historyJSON(entries []HistoryEntry) ([]byte, error) {
	return json.Marshal(entries)
}

// todayStr returns the current date as DD/MM/YYYY.
func todayStr() string {
	return time.Now().Format("02/01/2006")
}
