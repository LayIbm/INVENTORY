// Command import-inventory imports inventory from Excel or CSV files into PostgreSQL.
//
// Usage:
//
//	go run ./cmd/import-inventory --file <path> --dry-run
//	go run ./cmd/import-inventory --file <path> --apply [--upsert] [--batch 100] [--atomic]
//
// Required environment variables (DB only — no JWT or seed passwords needed):
//
//	DB_USER, DB_PASSWORD, DB_NAME, DB_HOST, DB_PORT (optional, default 5432)
//
// When running --dry-run, the DB connection is attempted but failures are
// non-fatal: the importer will skip the duplicate check and continue.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/layssagonzalez/device-inventory/backend/internal/db"
	"github.com/layssagonzalez/device-inventory/backend/internal/importer"
)

func main() {
	// ── Flags ────────────────────────────────────────────────────────────────
	fileFlag := flag.String("file", "", "Path to the Excel (.xlsx) or CSV (.csv) file to import (required)")
	dryRun := flag.Bool("dry-run", false, "Analyse and validate without writing to the database")
	apply := flag.Bool("apply", false, "Validate and write to the database")
	upsert := flag.Bool("upsert", false, "Update existing records (requires --apply)")
	batch := flag.Int("batch", importer.DefaultBatchSize, "Records per transaction batch (requires --apply)")
	atomic := flag.Bool("atomic", false, "Use a single atomic transaction (requires --apply)")
	errOutput := flag.String("errors-output", "", "Path for the CSV error report (default: ./import-errors-TIMESTAMP.csv)")

	flag.Parse()

	// ── Validate flags ───────────────────────────────────────────────────────
	if *fileFlag == "" {
		fmt.Fprintln(os.Stderr, "ERROR: --file is required")
		flag.Usage()
		os.Exit(1)
	}
	if !*dryRun && !*apply {
		fmt.Fprintln(os.Stderr, "ERROR: exactly one of --dry-run or --apply is required")
		flag.Usage()
		os.Exit(1)
	}
	if *dryRun && *apply {
		fmt.Fprintln(os.Stderr, "ERROR: --dry-run and --apply are mutually exclusive")
		flag.Usage()
		os.Exit(1)
	}
	if *batch <= 0 {
		fmt.Fprintln(os.Stderr, "ERROR: --batch must be greater than zero")
		flag.Usage()
		os.Exit(1)
	}

	// ── Validate file ────────────────────────────────────────────────────────
	absPath, err := filepath.Abs(*fileFlag)
	if err != nil {
		slog.Error("cannot resolve file path", "err", err)
		os.Exit(1)
	}
	ext := strings.ToLower(filepath.Ext(absPath))
	switch ext {
	case ".xlsx", ".csv":
		// supported
	case ".xls":
		fmt.Fprintln(os.Stderr, "ERROR: .xls (legacy Excel 97-2003) is not supported. Please save as .xlsx first.")
		os.Exit(1)
	default:
		fmt.Fprintf(os.Stderr, "ERROR: unsupported file extension %q (allowed: .xlsx, .csv)\n", ext)
		os.Exit(1)
	}
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "ERROR: file not found: %s\n", absPath)
		os.Exit(1)
	}

	// ── Connect to PostgreSQL ─────────────────────────────────────────────────
	// Read only the DB credentials from environment. JWT_SECRET and seed
	// passwords are NOT required by this tool.
	pool := connectDB(*dryRun)
	if pool != nil {
		defer pool.Close()
	}
	ctx := context.Background()

	opts := importer.Options{
		DryRun:    *dryRun,
		Upsert:    *upsert,
		BatchSize: *batch,
		Atomic:    *atomic,
		Actor:     "import-cli",
	}

	svc := importer.NewService(pool)

	// ── Run ──────────────────────────────────────────────────────────────────
	if *dryRun {
		runDryRun(ctx, svc, absPath, opts, *errOutput)
	} else {
		runApply(ctx, svc, absPath, opts, *errOutput)
	}
}

func runDryRun(ctx context.Context, svc *importer.Service, path string, opts importer.Options, errOut string) {
	slog.Info("starting dry-run", "file", path)
	result, err := svc.Preview(ctx, path, opts)
	if err != nil {
		slog.Error("dry-run failed", "err", err)
		os.Exit(1)
	}

	importer.PrintSummary(os.Stdout, result.Summary)

	if len(result.Errors) > 0 || len(result.Warnings) > 0 {
		writeErrorReport(result.Errors, result.Warnings, errOut)
	}

	slog.Info("dry-run complete", "insertable", result.Summary.Insertable,
		"errors", result.Summary.WithErrors, "warnings", result.Summary.WithWarnings)
}

func runApply(ctx context.Context, svc *importer.Service, path string, opts importer.Options, errOut string) {
	slog.Info("starting apply", "file", path, "upsert", opts.Upsert, "batch", opts.BatchSize, "atomic", opts.Atomic)
	result, err := svc.Apply(ctx, path, opts)
	if err != nil {
		slog.Error("apply failed", "err", err)
		os.Exit(1)
	}

	importer.PrintSummary(os.Stdout, result.Summary)

	slog.Info("apply complete",
		"inserted", result.Summary.Inserted,
		"updated", result.Summary.Updated,
		"skipped", result.Summary.Skipped,
		"errored", result.Summary.Errored,
		"duration_ms", result.Summary.DurationMs,
	)
}

func writeErrorReport(errors, warnings []importer.ImportRow, outPath string) {
	if outPath == "" {
		outPath = importer.ErrorFileName()
	}
	f, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0600)
	if err != nil {
		slog.Error("cannot create error report", "path", outPath, "err", err)
		return
	}
	defer f.Close()

	all := append(errors, warnings...)
	if err := importer.WriteErrorCSV(f, all); err != nil {
		slog.Error("error writing CSV report", "err", err)
		return
	}
	slog.Info("error report written", "path", outPath, "time", time.Now().Format(time.RFC3339))
}

// connectDB reads DB credentials from environment variables and attempts to
// connect. When dryRun is true, a connection failure is non-fatal (returns nil
// so the importer skips the duplicate check). When dryRun is false, any
// connection failure is fatal.
func connectDB(dryRun bool) *pgxpool.Pool {
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	name := os.Getenv("DB_NAME")
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")

	if user == "" || password == "" || name == "" || host == "" {
		if dryRun {
			slog.Warn("DB credentials incomplete — skipping duplicate check in dry-run mode",
				"missing", missingDBVars(user, password, name, host))
			return nil
		}
		fmt.Fprintln(os.Stderr, "ERROR: DB_USER, DB_PASSWORD, DB_NAME and DB_HOST are required for --apply")
		os.Exit(1)
	}

	if port == "" {
		port = "5432"
	}
	// Credentials come only from env; no interpolation of user-supplied values.
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		user, password, host, port, name)

	pool, err := db.Connect(dsn)
	if err != nil {
		if dryRun {
			slog.Warn("DB connection failed — skipping duplicate check in dry-run mode", "err", err)
			return nil
		}
		slog.Error("database connection failed", "err", err)
		os.Exit(1)
	}
	slog.Info("database connected", "host", host, "db", name)
	return pool
}

// missingDBVars returns a comma-separated list of unset variable names.
func missingDBVars(user, password, name, host string) string {
	var missing []string
	if user == "" {
		missing = append(missing, "DB_USER")
	}
	if password == "" {
		missing = append(missing, "DB_PASSWORD")
	}
	if name == "" {
		missing = append(missing, "DB_NAME")
	}
	if host == "" {
		missing = append(missing, "DB_HOST")
	}
	return strings.Join(missing, ",")
}
