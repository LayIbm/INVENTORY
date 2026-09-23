package billing

import "time"

// Invoice mirrors the invoices table.
type Invoice struct {
	ID        string    `json:"id"`
	Number    string    `json:"number"`
	Client    string    `json:"client"`
	Concept   string    `json:"concept"`
	Amount    float64   `json:"amount"`
	Status    string    `json:"status"`
	IssueDate string    `json:"issue_date"`
	DueDate   string    `json:"due_date"`
	Notes     string    `json:"notes"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateRequest is the payload for POST /api/invoices.
type CreateRequest struct {
	Number    string  `json:"number"`
	Client    string  `json:"client"`
	Concept   string  `json:"concept"`
	Amount    float64 `json:"amount"`
	Status    string  `json:"status"`
	IssueDate string  `json:"issue_date"`
	DueDate   string  `json:"due_date"`
	Notes     string  `json:"notes"`
	// CreatedBy is set by the handler from JWT claims, not from request body.
	CreatedBy string `json:"-"`
}

// UpdateRequest is the payload for PUT /api/invoices/:id.
type UpdateRequest struct {
	Client    string  `json:"client"`
	Concept   string  `json:"concept"`
	Amount    float64 `json:"amount"`
	Status    string  `json:"status"`
	IssueDate string  `json:"issue_date"`
	DueDate   string  `json:"due_date"`
	Notes     string  `json:"notes"`
}
