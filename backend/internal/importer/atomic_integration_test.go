//go:build integration

// Integration test for atomic transaction rollback.
//
// This test requires a live PostgreSQL instance reachable via the
// DB_* environment variables (same set as the CLI uses). It is NOT run
// by the default `go test ./...` build; pass -tags=integration to include it.
//
//	DB_USER=inventory_user DB_PASSWORD=... DB_NAME=device_inventory DB_HOST=db \
//	  go test -tags=integration -run TestAtomic_Rollback ./internal/importer/
package importer_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/layssagonzalez/device-inventory/backend/internal/db"
	"github.com/layssagonzalez/device-inventory/backend/internal/importer"
)

// TestAtomic_Rollback verifies that --atomic causes a full rollback when any
// write within the single transaction fails.
//
// Strategy:
//  1. Build a batch of two rows for writeBatch:
//     - Row A: completely valid — would INSERT fine.
//     - Row B: passes app-level validation but injects an invalid enum value
//       (__INVALID_ENUM__) as the availability field, bypassing toPrepStatus /
//       toLaptopAvailability so the DB itself rejects the INSERT.
//  2. Call WriteBatchForTest with opts.Atomic = true.
//  3. Expect writeBatch to return an error and roll back the entire transaction.
//  4. Verify the laptop count is unchanged (Row A was NOT committed).
func TestAtomic_Rollback(t *testing.T) {
	pool := connectTestDB(t)
	defer pool.Close()

	ctx := context.Background()

	// Ensure our test serials are absent before starting.
	cleanupSerial(t, ctx, pool, "ATOMIC-TEST-NEW-001")
	cleanupSerial(t, ctx, pool, "ATOMIC-TEST-BAD-001")
	t.Cleanup(func() {
		cleanupSerial(t, nil, pool, "ATOMIC-TEST-NEW-001")
		cleanupSerial(t, nil, pool, "ATOMIC-TEST-BAD-001")
	})

	// Count laptops before.
	var before int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM laptops`).Scan(&before); err != nil {
		t.Fatalf("count before: %v", err)
	}

	// Row A: a valid laptop — would succeed on its own.
	rowA := importer.ImportRow{
		Entity:         importer.EntityLaptop,
		ProposedAction: importer.ActionInsert,
		Fields: map[string]any{
			"serial":    "ATOMIC-TEST-NEW-001",
			"model":     "T490",
			"brand":     "Lenovo",
			"condition": "Good",
			// availability left blank → toLaptopAvailability("") → "DISPONIBLE"
		},
	}

	// Row B: an equipment row with an invalid device_type value.
	// insertRow for equipment passes str(f,"device_type") directly to the SQL
	// without any mapping, so "__INVALID_TYPE__" will violate the
	// equipment_device_type_check constraint at the DB level.
	rowB := importer.ImportRow{
		Entity:         importer.EntityEquipment,
		ProposedAction: importer.ActionInsert,
		Fields: map[string]any{
			"serial":      "ATOMIC-TEST-BAD-001",
			"device_type": "__INVALID_TYPE__", // violates DB CHECK constraint
			"model":       "TestModel",
			"brand":       "TestBrand",
			"condition":   "Good",
		},
	}

	svc := importer.NewService(pool)
	opts := importer.Options{
		Atomic:    true,
		BatchSize: 100,
		Actor:     "test",
	}

	ins, upd, skp, er, err := importer.WriteBatchForTest(ctx, svc, []importer.ImportRow{rowA, rowB}, opts)
	if err == nil {
		t.Fatalf("expected atomic write to fail, but got no error (ins=%d upd=%d skp=%d er=%d)",
			ins, upd, skp, er)
	}
	t.Logf("Atomic write failed as expected: %v", err)

	// Verify rollback: the laptop count must equal `before` — Row A must NOT have been committed.
	var after int
	if err2 := pool.QueryRow(ctx, `SELECT COUNT(*) FROM laptops`).Scan(&after); err2 != nil {
		t.Fatalf("count after: %v", err2)
	}
	if after != before {
		t.Errorf("atomic rollback failed: laptops count changed from %d to %d (expected no change)", before, after)
	}
	t.Logf("Rollback confirmed: laptops count unchanged at %d", after)
}

// connectTestDB creates a pgxpool from DB_* env vars, skipping the test if unavailable.
func connectTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	user := os.Getenv("DB_USER")
	pass := os.Getenv("DB_PASSWORD")
	name := os.Getenv("DB_NAME")
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	if port == "" {
		port = "5432"
	}
	if user == "" || pass == "" || name == "" || host == "" {
		t.Skip("DB_* env vars not set — skipping integration test")
	}
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", user, pass, host, port, name)
	pool, err := db.Connect(dsn)
	if err != nil {
		t.Fatalf("connect DB: %v", err)
	}
	return pool
}

// cleanupSerial removes a test serial from both tables, ignoring errors.
func cleanupSerial(t *testing.T, ctx context.Context, pool *pgxpool.Pool, serial string) {
	if t != nil {
		t.Helper()
	}
	if ctx == nil {
		ctx = context.Background()
	}
	pool.Exec(ctx, `DELETE FROM laptops WHERE serial = $1`, serial)
	pool.Exec(ctx, `DELETE FROM equipment WHERE serial = $1`, serial)
}
