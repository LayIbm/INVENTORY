package user

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

// Store handles all database operations for users.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore creates a new Store.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// List returns all users ordered by created_at.
func (s *Store) List(ctx context.Context) ([]User, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, username, password_hash, role, active, must_change_password, created_at, updated_at
		FROM users ORDER BY created_at ASC`)
	if err != nil {
		return nil, fmt.Errorf("user list: %w", err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Name, &u.Username, &u.PasswordHash,
			&u.Role, &u.Active, &u.MustChangePassword, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, fmt.Errorf("user list scan: %w", err)
		}
		users = append(users, u)
	}
	if users == nil {
		users = []User{}
	}
	return users, nil
}

// GetByID returns a single user by UUID.
func (s *Store) GetByID(ctx context.Context, id string) (*User, error) {
	var u User
	err := s.pool.QueryRow(ctx, `
		SELECT id, name, username, password_hash, role, active, must_change_password, created_at, updated_at
		FROM users WHERE id = $1`, id).
		Scan(&u.ID, &u.Name, &u.Username, &u.PasswordHash,
			&u.Role, &u.Active, &u.MustChangePassword, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("user get: %w", err)
	}
	return &u, nil
}

// GetByUsername returns a user by username (used for login).
func (s *Store) GetByUsername(ctx context.Context, username string) (*User, error) {
	var u User
	err := s.pool.QueryRow(ctx, `
		SELECT id, name, username, password_hash, role, active, must_change_password, created_at, updated_at
		FROM users WHERE username = $1`, username).
		Scan(&u.ID, &u.Name, &u.Username, &u.PasswordHash,
			&u.Role, &u.Active, &u.MustChangePassword, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("user get by username: %w", err)
	}
	return &u, nil
}

// Create inserts a new user with a bcrypt-hashed password.
func (s *Store) Create(ctx context.Context, req CreateRequest) (*User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
	if err != nil {
		return nil, fmt.Errorf("user create hash: %w", err)
	}
	var id string
	err = s.pool.QueryRow(ctx, `
		INSERT INTO users (name, username, password_hash, role, must_change_password)
		VALUES ($1, $2, $3, $4, TRUE) RETURNING id`,
		req.Name, req.Username, string(hash), req.Role,
	).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("user create: %w", err)
	}
	return s.GetByID(ctx, id)
}

// Update modifies a user's name, username, role and active flag.
func (s *Store) Update(ctx context.Context, id string, req UpdateRequest) (*User, error) {
	_, err := s.pool.Exec(ctx, `
		UPDATE users SET name=$1, username=$2, role=$3, active=$4 WHERE id=$5`,
		req.Name, req.Username, req.Role, req.Active, id,
	)
	if err != nil {
		return nil, fmt.Errorf("user update: %w", err)
	}
	return s.GetByID(ctx, id)
}

// ToggleActive flips the active flag.
func (s *Store) ToggleActive(ctx context.Context, id string) (*User, error) {
	_, err := s.pool.Exec(ctx, `
		UPDATE users SET active = NOT active WHERE id = $1`, id)
	if err != nil {
		return nil, fmt.Errorf("user toggle active: %w", err)
	}
	return s.GetByID(ctx, id)
}

// Delete removes a user by ID.
func (s *Store) Delete(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx, "DELETE FROM users WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("user delete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

// UpdatePassword sets a new bcrypt-hashed password and clears must_change_password.
func (s *Store) UpdatePassword(ctx context.Context, id, newPassword string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), 12)
	if err != nil {
		return fmt.Errorf("update password hash: %w", err)
	}
	_, err = s.pool.Exec(ctx, `
		UPDATE users SET password_hash=$1, must_change_password=FALSE WHERE id=$2`,
		string(hash), id,
	)
	return err
}
