package enlace

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Store handles all database operations for enlaces.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore creates a new Store.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// List returns all enlaces ordered by created_at desc.
func (s *Store) List(ctx context.Context) ([]Link, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, type, company, public_ip, ip, velocity,
		       end_date_contract, months_of_contract, additional_service,
		       contract_number, client_number, identifier_link,
		       name_contact, phone_contact, email_contact,
		       support_phone, support_clave, current_po, comments,
		       created_by, created_at, updated_at
		FROM enlaces ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("enlace list: %w", err)
	}
	defer rows.Close()

	var links []Link
	for rows.Next() {
		var link Link
		if err := rows.Scan(
			&link.ID, &link.Type, &link.Company, &link.PublicIP, &link.IP, &link.Velocity,
			&link.EndDateContract, &link.MonthsOfContract, &link.AdditionalService,
			&link.ContractNumber, &link.ClientNumber, &link.IdentifierLink,
			&link.NameContact, &link.PhoneContact, &link.EmailContact,
			&link.SupportPhone, &link.SupportClave, &link.CurrentPO, &link.Comments,
			&link.CreatedBy, &link.CreatedAt, &link.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("enlace list scan: %w", err)
		}
		links = append(links, link)
	}
	if links == nil {
		links = []Link{}
	}
	return links, nil
}

// GetByIdentifierOrCompanyIP returns an existing enlace matched by identifier_link,
// or by company + public_ip when no identifier is provided.
func (s *Store) GetByIdentifierOrCompanyIP(ctx context.Context, identifierLink, company, publicIP string) (*Link, error) {
	var link Link
	if identifierLink != "" {
		err := s.pool.QueryRow(ctx, `
			SELECT id, type, company, public_ip, ip, velocity,
			       end_date_contract, months_of_contract, additional_service,
			       contract_number, client_number, identifier_link,
			       name_contact, phone_contact, email_contact,
			       support_phone, support_clave, current_po, comments,
			       created_by, created_at, updated_at
			FROM enlaces WHERE identifier_link = $1
			ORDER BY created_at DESC LIMIT 1`, identifierLink).Scan(
			&link.ID, &link.Type, &link.Company, &link.PublicIP, &link.IP, &link.Velocity,
			&link.EndDateContract, &link.MonthsOfContract, &link.AdditionalService,
			&link.ContractNumber, &link.ClientNumber, &link.IdentifierLink,
			&link.NameContact, &link.PhoneContact, &link.EmailContact,
			&link.SupportPhone, &link.SupportClave, &link.CurrentPO, &link.Comments,
			&link.CreatedBy, &link.CreatedAt, &link.UpdatedAt,
		)
		if err == nil {
			return &link, nil
		}
	}

	err := s.pool.QueryRow(ctx, `
		SELECT id, type, company, public_ip, ip, velocity,
		       end_date_contract, months_of_contract, additional_service,
		       contract_number, client_number, identifier_link,
		       name_contact, phone_contact, email_contact,
		       support_phone, support_clave, current_po, comments,
		       created_by, created_at, updated_at
		FROM enlaces WHERE company = $1 AND public_ip = $2
		ORDER BY created_at DESC LIMIT 1`, company, publicIP).Scan(
		&link.ID, &link.Type, &link.Company, &link.PublicIP, &link.IP, &link.Velocity,
		&link.EndDateContract, &link.MonthsOfContract, &link.AdditionalService,
		&link.ContractNumber, &link.ClientNumber, &link.IdentifierLink,
		&link.NameContact, &link.PhoneContact, &link.EmailContact,
		&link.SupportPhone, &link.SupportClave, &link.CurrentPO, &link.Comments,
		&link.CreatedBy, &link.CreatedAt, &link.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("enlace get: %w", err)
	}
	return &link, nil
}

// GetByID returns a single enlace by its UUID.
func (s *Store) GetByID(ctx context.Context, id string) (*Link, error) {
	var link Link
	err := s.pool.QueryRow(ctx, `
		SELECT id, type, company, public_ip, ip, velocity,
		       end_date_contract, months_of_contract, additional_service,
		       contract_number, client_number, identifier_link,
		       name_contact, phone_contact, email_contact,
		       support_phone, support_clave, current_po, comments,
		       created_by, created_at, updated_at
		FROM enlaces WHERE id = $1`, id).Scan(
		&link.ID, &link.Type, &link.Company, &link.PublicIP, &link.IP, &link.Velocity,
		&link.EndDateContract, &link.MonthsOfContract, &link.AdditionalService,
		&link.ContractNumber, &link.ClientNumber, &link.IdentifierLink,
		&link.NameContact, &link.PhoneContact, &link.EmailContact,
		&link.SupportPhone, &link.SupportClave, &link.CurrentPO, &link.Comments,
		&link.CreatedBy, &link.CreatedAt, &link.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("enlace get: %w", err)
	}
	return &link, nil
}

// Create inserts a new enlace and returns it.
func (s *Store) Create(ctx context.Context, req CreateRequest) (*Link, error) {
	var id string
	err := s.pool.QueryRow(ctx, `
		INSERT INTO enlaces (
			type, company, public_ip, ip, velocity,
			end_date_contract, months_of_contract, additional_service,
			contract_number, client_number, identifier_link,
			name_contact, phone_contact, email_contact,
			support_phone, support_clave, current_po, comments, created_by
		) VALUES (
			$1,$2,$3,$4,$5,
			$6,$7,$8,
			$9,$10,$11,
			$12,$13,$14,
			$15,$16,$17,$18,$19
		) RETURNING id`,
		req.Type, req.Company, req.PublicIP, req.IP, req.Velocity,
		req.EndDateContract, req.MonthsOfContract, req.AdditionalService,
		req.ContractNumber, req.ClientNumber, req.IdentifierLink,
		req.NameContact, req.PhoneContact, req.EmailContact,
		req.SupportPhone, req.SupportClave, req.CurrentPO, req.Comments, req.CreatedBy,
	).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("enlace create: %w", err)
	}
	return s.GetByID(ctx, id)
}

// Update modifies an existing enlace and returns it.
func (s *Store) Update(ctx context.Context, id string, req UpdateRequest) (*Link, error) {
	tag, err := s.pool.Exec(ctx, `
		UPDATE enlaces SET
			type=$1, company=$2, public_ip=$3, ip=$4, velocity=$5,
			end_date_contract=$6, months_of_contract=$7, additional_service=$8,
			contract_number=$9, client_number=$10, identifier_link=$11,
			name_contact=$12, phone_contact=$13, email_contact=$14,
			support_phone=$15, support_clave=$16, current_po=$17, comments=$18
		WHERE id=$19`,
		req.Type, req.Company, req.PublicIP, req.IP, req.Velocity,
		req.EndDateContract, req.MonthsOfContract, req.AdditionalService,
		req.ContractNumber, req.ClientNumber, req.IdentifierLink,
		req.NameContact, req.PhoneContact, req.EmailContact,
		req.SupportPhone, req.SupportClave, req.CurrentPO, req.Comments,
		id,
	)
	if err != nil {
		return nil, fmt.Errorf("enlace update: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, fmt.Errorf("enlace not found")
	}
	return s.GetByID(ctx, id)
}

// Delete removes an enlace by ID.
func (s *Store) Delete(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx, "DELETE FROM enlaces WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("enlace delete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("enlace not found")
	}
	return nil
}
