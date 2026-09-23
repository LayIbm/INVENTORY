package user

import (
	"time"
)

// User mirrors the users table. PasswordHash is excluded from JSON responses.
type User struct {
	ID                 string    `json:"id"`
	Name               string    `json:"name"`
	Username           string    `json:"username"`
	PasswordHash       string    `json:"-"` // never serialized
	Role               string    `json:"role"`
	Active             bool      `json:"active"`
	MustChangePassword bool      `json:"must_change_password"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// CreateRequest is the payload for POST /api/users.
type CreateRequest struct {
	Name     string `json:"name"`
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

// UpdateRequest is the payload for PUT /api/users/:id.
type UpdateRequest struct {
	Name     string `json:"name"`
	Username string `json:"username"`
	Role     string `json:"role"`
	Active   bool   `json:"active"`
}

// ChangePasswordRequest is the payload for POST /api/auth/change-password.
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}
