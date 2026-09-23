package billing

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Store handles all database operations for invoices.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore creates a new Store.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// List returns all invoices ordered by created_at desc.
func (s *Store) List(ctx context.Context) ([]Invoice, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, number, client, concept, amount, status,
		       issue_date, due_date, notes, created_by,
		       created_at, updated_at
		FROM invoices ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("invoice list: %w", err)
	}
	defer rows.Close()

	var invoices []Invoice
	for rows.Next() {
		var inv Invoice
		if err := rows.Scan(
			&inv.ID, &inv.Number, &inv.Client, &inv.Concept, &inv.Amount, &inv.Status,
			&inv.IssueDate, &inv.DueDate, &inv.Notes, &inv.CreatedBy,
			&inv.CreatedAt, &inv.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("invoice list scan: %w", err)
		}
		invoices = append(invoices, inv)
	}
	if invoices == nil {
		invoices = []Invoice{}
	}
	return invoices, nil
}

// GetByID returns a single invoice by its UUID.
func (s *Store) GetByID(ctx context.Context, id string) (*Invoice, error) {
	var inv Invoice
	err := s.pool.QueryRow(ctx, `
		SELECT id, number, client, concept, amount, status,
		       issue_date, due_date, notes, created_by,
		       created_at, updated_at
		FROM invoices WHERE id = $1`, id).Scan(
		&inv.ID, &inv.Number, &inv.Client, &inv.Concept, &inv.Amount, &inv.Status,
		&inv.IssueDate, &inv.DueDate, &inv.Notes, &inv.CreatedBy,
		&inv.CreatedAt, &inv.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("invoice get: %w", err)
	}
	return &inv, nil
}

// Create inserts a new invoice and returns it.
func (s *Store) Create(ctx context.Context, req CreateRequest) (*Invoice, error) {
	if req.Status == "" {
		req.Status = "PENDIENTE"
	}
	var id string
	err := s.pool.QueryRow(ctx, `
		INSERT INTO invoices
		  (number, client, concept, amount, status, issue_date, due_date, notes, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING id`,
		req.Number, req.Client, req.Concept, req.Amount, req.Status,
		req.IssueDate, req.DueDate, req.Notes, req.CreatedBy,
	).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("invoice create: %w", err)
	}
	return s.GetByID(ctx, id)
}

// Update modifies an existing invoice and returns it.
func (s *Store) Update(ctx context.Context, id string, req UpdateRequest) (*Invoice, error) {
	tag, err := s.pool.Exec(ctx, `
		UPDATE invoices SET
		  client=$1, concept=$2, amount=$3, status=$4,
		  issue_date=$5, due_date=$6, notes=$7
		WHERE id=$8`,
		req.Client, req.Concept, req.Amount, req.Status,
		req.IssueDate, req.DueDate, req.Notes, id,
	)
	if err != nil {
		return nil, fmt.Errorf("invoice update: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, fmt.Errorf("invoice not found")
	}
	return s.GetByID(ctx, id)
}

// Delete removes an invoice by ID.
func (s *Store) Delete(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx, "DELETE FROM invoices WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("invoice delete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("invoice not found")
	}
	return nil
}

// validStatus checks that a status string is a valid invoice_status enum value.
func validStatus(s string) bool {
	switch strings.ToUpper(s) {
	case "PENDIENTE", "PAGADA", "CANCELADA":
		return true
	}
	return false
}
