package importer

import "os"

// readFileBytes reads all bytes from a local file path.
// Kept as a thin wrapper so tests can observe the only place file I/O happens.
func readFileBytes(path string) ([]byte, error) {
	return os.ReadFile(path) //nolint:wrapcheck
}
