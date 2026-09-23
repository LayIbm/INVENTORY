package importer

import "context"

// WriteBatchForTest exposes writeBatch for integration testing.
// It is compiled only into test binaries; normal builds do not include it.
// This lets integration tests inject rows that bypass the application-level
// validator and hit the DB directly — the only way to exercise the atomic
// rollback path from outside the package.
func WriteBatchForTest(ctx context.Context, s *Service, rows []ImportRow, opts Options) (inserted, updated, skipped, errored int, err error) {
	return s.writeBatch(ctx, rows, opts)
}

// ToLaptopAvailabilityForTest exposes toLaptopAvailability for unit tests.
func ToLaptopAvailabilityForTest(s string) string { return toLaptopAvailability(s) }

// ToEquipmentAvailabilityForTest exposes toEquipmentAvailability for unit tests.
func ToEquipmentAvailabilityForTest(s string) string { return toEquipmentAvailability(s) }
