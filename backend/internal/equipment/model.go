package equipment

import (
	"encoding/json"
	"time"
)

// HistoryEntry records a single event in an equipment's audit trail.
type HistoryEntry struct {
	Date     string `json:"date"`
	Type     string `json:"type"`
	Tone     string `json:"tone"`
	Employee string `json:"employee"`
	Notes    string `json:"notes"`
}

// Equipment mirrors the equipment table.
type Equipment struct {
	ID                 string         `json:"id"`
	Serial             string         `json:"serial"`
	DeviceType         string         `json:"device_type"`
	Brand              string         `json:"brand"`
	Model              string         `json:"model"`
	Variant            string         `json:"variant"`
	Condition          string         `json:"condition"`
	Availability       string         `json:"availability"`
	Assignability      string         `json:"assignability"`
	Comodato           string         `json:"comodato"`
	PowersOn           string         `json:"powers_on"`
	OS                 string         `json:"os"`
	BIOSPassword       string         `json:"bios_password"`
	WiFi               string         `json:"wifi"`
	Bluetooth          string         `json:"bluetooth"`
	ChargerIncluded    bool           `json:"charger_included"`
	LastFormatDate     string         `json:"last_format_date"`
	EmployeeName       *string        `json:"employee_name"`
	EmployeeEmail      string         `json:"employee_email"`
	EmployeeTalentID   string         `json:"employee_talent_id"`
	TransactionDate    string         `json:"transaction_date"`
	ExpectedReturnDate string         `json:"expected_return_date"`
	MonitorIncluded    bool           `json:"monitor_included"`
	MonitorSerial      string         `json:"monitor_serial"`
	Owner              string         `json:"owner"`
	Hostname           string         `json:"hostname"`
	Geography          string         `json:"geography"`
	Notes              string         `json:"notes"`
	History            []HistoryEntry `json:"history"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`

	// ── Network-infrastructure fields (added in migration 006) ────────────
	// product_id: Part number / Product ID as printed on device
	ProductID string `json:"product_id"`
	// net_type: Primary | Secondary | N/A
	NetType string `json:"net_type"`
	// Lifecycle dates stored as nullable strings (YYYY-MM-DD from DATE column)
	EndOfSale          *string `json:"end_of_sale"`
	EndOfLife          *string `json:"end_of_life"`
	EndContractSupport *string `json:"end_contract_support"`
	// Costs: exact decimal, no assumed currency (see migration notes)
	DeviceCost   *string `json:"device_cost"`
	ContractCost *string `json:"contract_cost"`
	// Location sub-field
	Room string `json:"room"`
	// Status fields
	EOLStatus      string `json:"eol_status"`
	ContractStatus string `json:"contract_status"`
	// Provider / support contacts
	DeviceCompany          string `json:"device_company"`
	ContactName            string `json:"contact_name"`
	ContactPhone           string `json:"contact_phone"`
	ContactEmail           string `json:"contact_email"`
	IBMNetworkEmailSupport string `json:"ibm_network_email_support"`
	IBMLocalEmailSupport   string `json:"ibm_local_email_support"`
}

// CreateRequest is the payload for POST /api/equipment.
type CreateRequest struct {
	Serial             string  `json:"serial"`
	DeviceType         string  `json:"device_type"`
	Brand              string  `json:"brand"`
	Model              string  `json:"model"`
	Variant            string  `json:"variant"`
	Condition          string  `json:"condition"`
	Availability       string  `json:"availability"`
	Assignability      string  `json:"assignability"`
	Comodato           string  `json:"comodato"`
	PowersOn           string  `json:"powers_on"`
	OS                 string  `json:"os"`
	BIOSPassword       string  `json:"bios_password"`
	WiFi               string  `json:"wifi"`
	Bluetooth          string  `json:"bluetooth"`
	ChargerIncluded    bool    `json:"charger_included"`
	LastFormatDate     string  `json:"last_format_date"`
	EmployeeName       *string `json:"employee_name"`
	EmployeeEmail      string  `json:"employee_email"`
	EmployeeTalentID   string  `json:"employee_talent_id"`
	TransactionDate    string  `json:"transaction_date"`
	ExpectedReturnDate string  `json:"expected_return_date"`
	MonitorIncluded    bool    `json:"monitor_included"`
	MonitorSerial      string  `json:"monitor_serial"`
	Owner              string  `json:"owner"`
	Hostname           string  `json:"hostname"`
	Geography          string  `json:"geography"`
	Notes              string  `json:"notes"`
	// Actor is set by the handler from JWT claims, not from request body.
	Actor string `json:"-"`

	// Network fields
	ProductID              string  `json:"product_id"`
	NetType                string  `json:"net_type"`
	EndOfSale              *string `json:"end_of_sale"`
	EndOfLife              *string `json:"end_of_life"`
	EndContractSupport     *string `json:"end_contract_support"`
	DeviceCost             *string `json:"device_cost"`
	ContractCost           *string `json:"contract_cost"`
	Room                   string  `json:"room"`
	EOLStatus              string  `json:"eol_status"`
	ContractStatus         string  `json:"contract_status"`
	DeviceCompany          string  `json:"device_company"`
	ContactName            string  `json:"contact_name"`
	ContactPhone           string  `json:"contact_phone"`
	ContactEmail           string  `json:"contact_email"`
	IBMNetworkEmailSupport string  `json:"ibm_network_email_support"`
	IBMLocalEmailSupport   string  `json:"ibm_local_email_support"`
}

// UpdateRequest is the payload for PUT /api/equipment/:id.
type UpdateRequest struct {
	DeviceType         string  `json:"device_type"`
	Brand              string  `json:"brand"`
	Model              string  `json:"model"`
	Variant            string  `json:"variant"`
	Condition          string  `json:"condition"`
	Availability       string  `json:"availability"`
	Assignability      string  `json:"assignability"`
	Comodato           string  `json:"comodato"`
	PowersOn           string  `json:"powers_on"`
	OS                 string  `json:"os"`
	BIOSPassword       string  `json:"bios_password"`
	WiFi               string  `json:"wifi"`
	Bluetooth          string  `json:"bluetooth"`
	ChargerIncluded    bool    `json:"charger_included"`
	LastFormatDate     string  `json:"last_format_date"`
	EmployeeName       *string `json:"employee_name"`
	EmployeeEmail      string  `json:"employee_email"`
	EmployeeTalentID   string  `json:"employee_talent_id"`
	TransactionDate    string  `json:"transaction_date"`
	ExpectedReturnDate string  `json:"expected_return_date"`
	MonitorIncluded    bool    `json:"monitor_included"`
	MonitorSerial      string  `json:"monitor_serial"`
	Owner              string  `json:"owner"`
	Hostname           string  `json:"hostname"`
	Geography          string  `json:"geography"`
	Notes              string  `json:"notes"`
	Actor              string  `json:"-"`

	// Network fields
	ProductID              string  `json:"product_id"`
	NetType                string  `json:"net_type"`
	EndOfSale              *string `json:"end_of_sale"`
	EndOfLife              *string `json:"end_of_life"`
	EndContractSupport     *string `json:"end_contract_support"`
	DeviceCost             *string `json:"device_cost"`
	ContractCost           *string `json:"contract_cost"`
	Room                   string  `json:"room"`
	EOLStatus              string  `json:"eol_status"`
	ContractStatus         string  `json:"contract_status"`
	DeviceCompany          string  `json:"device_company"`
	ContactName            string  `json:"contact_name"`
	ContactPhone           string  `json:"contact_phone"`
	ContactEmail           string  `json:"contact_email"`
	IBMNetworkEmailSupport string  `json:"ibm_network_email_support"`
	IBMLocalEmailSupport   string  `json:"ibm_local_email_support"`
}

// historyJSON marshals a slice of HistoryEntry to JSON bytes.
func historyJSON(entries []HistoryEntry) ([]byte, error) {
	return json.Marshal(entries)
}

// todayStr returns the current date as DD/MM/YYYY.
func todayStr() string {
	return time.Now().Format("02/01/2006")
}
