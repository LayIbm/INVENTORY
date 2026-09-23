package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/layssagonzalez/device-inventory/backend/internal/config"
	"github.com/layssagonzalez/device-inventory/backend/internal/db"
	"github.com/layssagonzalez/device-inventory/backend/internal/routes"
	"github.com/layssagonzalez/device-inventory/backend/internal/seed"
)

func main() {
	// Structured JSON logging
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config load failed", "err", err)
		os.Exit(1)
	}

	pool, err := db.Connect(cfg.DSN())
	if err != nil {
		slog.Error("database connection failed", "err", err)
		os.Exit(1)
	}
	defer pool.Close()
	slog.Info("database connected")

	if err := seed.Run(context.Background(), pool,
		cfg.SeedAdminPassword, cfg.SeedManagerPassword, cfg.SeedUserPassword,
	); err != nil {
		slog.Error("seed failed", "err", err)
		os.Exit(1)
	}

	handler := routes.Register(pool, cfg.JWTSecret)

	addr := ":8080"
	slog.Info("api server starting", "addr", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		slog.Error("server error", "err", err)
		os.Exit(1)
	}
}
