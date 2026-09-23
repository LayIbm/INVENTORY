package enlace

import "time"

// Link mirrors the enlaces table.
type Link struct {
	ID                string    `json:"id"`
	Type              string    `json:"type"`
	Company           string    `json:"company"`
	PublicIP          string    `json:"public_ip"`
	IP                string    `json:"ip"`
	Velocity          string    `json:"velocity"`
	EndDateContract   string    `json:"end_date_contract"`
	MonthsOfContract  string    `json:"months_of_contract"`
	AdditionalService string    `json:"additional_service"`
	ContractNumber    string    `json:"contract_number"`
	ClientNumber      string    `json:"client_number"`
	IdentifierLink    string    `json:"identifier_link"`
	NameContact       string    `json:"name_contact"`
	PhoneContact      string    `json:"phone_contact"`
	EmailContact      string    `json:"email_contact"`
	SupportPhone      string    `json:"support_phone"`
	SupportClave      string    `json:"support_clave"`
	CurrentPO         string    `json:"current_po"`
	Comments          string    `json:"comments"`
	CreatedBy         string    `json:"created_by"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// CreateRequest is the payload for POST /api/enlaces.
type CreateRequest struct {
	Type              string `json:"type"`
	Company           string `json:"company"`
	PublicIP          string `json:"public_ip"`
	IP                string `json:"ip"`
	Velocity          string `json:"velocity"`
	EndDateContract   string `json:"end_date_contract"`
	MonthsOfContract  string `json:"months_of_contract"`
	AdditionalService string `json:"additional_service"`
	ContractNumber    string `json:"contract_number"`
	ClientNumber      string `json:"client_number"`
	IdentifierLink    string `json:"identifier_link"`
	NameContact       string `json:"name_contact"`
	PhoneContact      string `json:"phone_contact"`
	EmailContact      string `json:"email_contact"`
	SupportPhone      string `json:"support_phone"`
	SupportClave      string `json:"support_clave"`
	CurrentPO         string `json:"current_po"`
	Comments          string `json:"comments"`
	CreatedBy         string `json:"-"`
}

// UpdateRequest is the payload for PUT /api/enlaces/:id.
type UpdateRequest struct {
	Type              string `json:"type"`
	Company           string `json:"company"`
	PublicIP          string `json:"public_ip"`
	IP                string `json:"ip"`
	Velocity          string `json:"velocity"`
	EndDateContract   string `json:"end_date_contract"`
	MonthsOfContract  string `json:"months_of_contract"`
	AdditionalService string `json:"additional_service"`
	ContractNumber    string `json:"contract_number"`
	ClientNumber      string `json:"client_number"`
	IdentifierLink    string `json:"identifier_link"`
	NameContact       string `json:"name_contact"`
	PhoneContact      string `json:"phone_contact"`
	EmailContact      string `json:"email_contact"`
	SupportPhone      string `json:"support_phone"`
	SupportClave      string `json:"support_clave"`
	CurrentPO         string `json:"current_po"`
	Comments          string `json:"comments"`
}
