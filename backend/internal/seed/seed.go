package seed

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type seedUser struct {
	name     string
	username string
	password string
	role     string
}

// Run checks if any users exist; if not, inserts the three seed users.
func Run(ctx context.Context, pool *pgxpool.Pool, adminPwd, managerPwd, userPwd string) error {
	var count int
	if err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&count); err != nil {
		return fmt.Errorf("seed: count users: %w", err)
	}
	if count > 0 {
		slog.Info("seed: users already exist, skipping")
		return nil
	}

	seeds := []seedUser{
		{name: "Administrator", username: "admin", password: adminPwd, role: "admin"},
		{name: "Manager", username: "manager", password: managerPwd, role: "manager"},
		{name: "Read-only User", username: "user", password: userPwd, role: "viewer"},
	}

	for _, s := range seeds {
		hash, err := bcrypt.GenerateFromPassword([]byte(s.password), 12)
		if err != nil {
			return fmt.Errorf("seed: bcrypt for %s: %w", s.username, err)
		}
		_, err = pool.Exec(ctx,
			`INSERT INTO users (name, username, password_hash, role, must_change_password)
			 VALUES ($1, $2, $3, $4, TRUE)`,
			s.name, s.username, string(hash), s.role,
		)
		if err != nil {
			return fmt.Errorf("seed: insert user %s: %w", s.username, err)
		}
		slog.Info("seed: created user", "username", s.username, "role", s.role)
	}
	return nil
}
