package config

import (
	"fmt"
	"os"
)

// Config holds all runtime configuration loaded from environment variables.
type Config struct {
	DBUser              string
	DBPassword          string
	DBName              string
	DBHost              string
	DBPort              string
	JWTSecret           string
	SeedAdminPassword   string
	SeedManagerPassword string
	SeedUserPassword    string
}

// Load reads all required environment variables. Returns an error if any are missing.
func Load() (*Config, error) {
	c := &Config{
		DBUser:              os.Getenv("DB_USER"),
		DBPassword:          os.Getenv("DB_PASSWORD"),
		DBName:              os.Getenv("DB_NAME"),
		DBHost:              os.Getenv("DB_HOST"),
		DBPort:              os.Getenv("DB_PORT"),
		JWTSecret:           os.Getenv("JWT_SECRET"),
		SeedAdminPassword:   os.Getenv("SEED_ADMIN_PASSWORD"),
		SeedManagerPassword: os.Getenv("SEED_MANAGER_PASSWORD"),
		SeedUserPassword:    os.Getenv("SEED_USER_PASSWORD"),
	}

	required := map[string]string{
		"DB_USER":              c.DBUser,
		"DB_PASSWORD":          c.DBPassword,
		"DB_NAME":              c.DBName,
		"DB_HOST":              c.DBHost,
		"JWT_SECRET":           c.JWTSecret,
		"SEED_ADMIN_PASSWORD":  c.SeedAdminPassword,
		"SEED_MANAGER_PASSWORD": c.SeedManagerPassword,
		"SEED_USER_PASSWORD":   c.SeedUserPassword,
	}

	for k, v := range required {
		if v == "" {
			return nil, fmt.Errorf("missing required environment variable: %s", k)
		}
	}

	if c.DBPort == "" {
		c.DBPort = "5432"
	}

	return c, nil
}

// DSN returns a PostgreSQL connection string.
func (c *Config) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName,
	)
}
