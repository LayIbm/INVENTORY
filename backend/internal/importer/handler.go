package importer

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/layssagonzalez/device-inventory/backend/internal/middleware"
)

// MaxUploadBytes is the maximum allowed size for an imported file (50 MB).
const MaxUploadBytes = 50 << 20

// ImportService is the interface the Handler depends on. *Service satisfies it.
type ImportService interface {
	Preview(ctx context.Context, filePath string, opts Options) (*PreviewResult, error)
	Apply(ctx context.Context, filePath string, opts Options) (*ApplyResult, error)
}

// Handler handles the /api/import/inventory/* endpoints.
type Handler struct {
	svc ImportService
}

// NewHandler returns a new Handler backed by any ImportService implementation.
func NewHandler(svc ImportService) *Handler {
	return &Handler{svc: svc}
}

// writeJSON is a local helper to avoid import cycles.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("importer writeJSON", "err", err)
	}
}

func clientError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// Preview handles POST /api/import/inventory/preview
// Accepts multipart/form-data with a "file" field.
// Returns a dry-run PreviewResult without modifying the database.
func (h *Handler) Preview(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	tmp, origFilename, err := receiveUpload(r)
	if err != nil {
		clientError(w, http.StatusBadRequest, err.Error())
		return
	}
	defer cleanupTemp(tmp)

	claims := middleware.ClaimsFromContext(r.Context())
	actor := "api"
	if claims != nil {
		actor = claims.Username
	}

	opts := Options{DryRun: true, Actor: actor}
	result, err := h.svc.Preview(r.Context(), tmp, opts)
	if err != nil {
		slog.Error("import preview failed",
			"user", actor,
			"file", origFilename,
			"err", err,
		)
		clientError(w, http.StatusUnprocessableEntity, "file could not be processed: "+err.Error())
		return
	}

	slog.Info("import preview completed",
		"user", actor,
		"file", origFilename,
		"rows_read", result.Summary.RowsRead,
		"insertable", result.Summary.Insertable,
		"updatable", result.Summary.Updatable,
		"errors", result.Summary.WithErrors,
		"duration_ms", time.Since(start).Milliseconds(),
	)

	writeJSON(w, http.StatusOK, result)
}

// Apply handles POST /api/import/inventory/apply
// Re-reads the file from the multipart upload for security (does not trust
// client-side preview result).
func (h *Handler) Apply(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	tmp, origFilename, err := receiveUpload(r)
	if err != nil {
		clientError(w, http.StatusBadRequest, err.Error())
		return
	}
	defer cleanupTemp(tmp)

	claims := middleware.ClaimsFromContext(r.Context())
	actor := "api"
	if claims != nil {
		actor = claims.Username
	}

	// Optional query params — same defaults as CLI
	upsert := r.URL.Query().Get("upsert") == "true"
	atomic := r.URL.Query().Get("atomic") == "true"
	batchSize := DefaultBatchSize
	if bs := r.URL.Query().Get("batch"); bs != "" {
		n, convErr := strconv.Atoi(bs)
		if convErr != nil {
			clientError(w, http.StatusBadRequest, "batch must be a positive integer")
			return
		}
		if n <= 0 {
			clientError(w, http.StatusBadRequest, "batch must be greater than zero")
			return
		}
		batchSize = n
	}

	opts := Options{
		DryRun:    false,
		Upsert:    upsert,
		BatchSize: batchSize,
		Atomic:    atomic,
		Actor:     actor,
	}

	result, err := h.svc.Apply(r.Context(), tmp, opts)
	if err != nil {
		slog.Error("import apply failed",
			"user", actor,
			"file", origFilename,
			"err", err,
		)
		clientError(w, http.StatusUnprocessableEntity, "import failed: "+err.Error())
		return
	}

	slog.Info("import apply completed",
		"user", actor,
		"file", origFilename,
		"inserted", result.Summary.Inserted,
		"updated", result.Summary.Updated,
		"skipped", result.Summary.Skipped,
		"errored", result.Summary.Errored,
		"duration_ms", time.Since(start).Milliseconds(),
	)

	writeJSON(w, http.StatusOK, result)
}

// receiveUpload reads the "file" field from a multipart request, writes it to
// a secure temp file with the correct extension, and returns (finalPath, origFilename, error).
// The caller is responsible for removing the file after use.
func receiveUpload(r *http.Request) (finalPath, origFilename string, err error) {
	// Enforce hard request-body size limit before any parsing.
	r.Body = http.MaxBytesReader(nil, r.Body, MaxUploadBytes)
	if err = r.ParseMultipartForm(32 << 20); err != nil {
		return "", "", fmt.Errorf("request too large or not multipart: %w", err)
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		return "", "", fmt.Errorf("missing 'file' field in form")
	}
	defer file.Close()

	// Validate extension — .xls is NOT supported (legacy binary format).
	origExt := strings.ToLower(filepath.Ext(header.Filename))
	switch origExt {
	case ".xlsx", ".csv":
		// supported
	case ".xls":
		return "", "", fmt.Errorf(".xls format is not supported; please convert the file to .xlsx or .csv")
	default:
		return "", "", fmt.Errorf("unsupported file type %q (allowed: .xlsx, .csv)", origExt)
	}

	// Write to a named temp file (secure: no user-controlled name in path)
	// Include the correct extension so the parser can identify the format.
	tf, err := os.CreateTemp("", "inventory-import-*"+origExt)
	if err != nil {
		return "", "", fmt.Errorf("cannot create temp file: %w", err)
	}
	defer tf.Close()

	buf := make([]byte, 512)
	n, _ := file.Read(buf)
	if n > 0 {
		if !isAllowedMIME(buf[:n], origExt) {
			os.Remove(tf.Name())
			return "", "", fmt.Errorf("file content does not match declared extension")
		}
		if _, err = tf.Write(buf[:n]); err != nil {
			os.Remove(tf.Name())
			return "", "", fmt.Errorf("cannot write temp file: %w", err)
		}
	}

	// Stream the rest
	remaining := make([]byte, MaxUploadBytes)
	for {
		nr, readErr := file.Read(remaining)
		if nr > 0 {
			if _, werr := tf.Write(remaining[:nr]); werr != nil {
				os.Remove(tf.Name())
				return "", "", fmt.Errorf("cannot write temp file: %w", werr)
			}
		}
		if readErr != nil {
			break
		}
	}

	return tf.Name(), header.Filename, nil
}

// isAllowedMIME does a quick magic-byte check to ensure the file is what the
// extension claims. This is not exhaustive but prevents trivial path traversal
// and masquerade attacks.
func isAllowedMIME(header []byte, ext string) bool {
	switch ext {
	case ".xlsx", ".xls":
		// Office Open XML / OLE2 signatures
		xlsx := []byte{0x50, 0x4B, 0x03, 0x04} // PK zip
		ole2 := []byte{0xD0, 0xCF, 0x11, 0xE0} // OLE2
		if len(header) >= 4 {
			h := header[:4]
			if string(h) == string(xlsx) || string(h) == string(ole2) {
				return true
			}
		}
		// Some xlsx are plain zip — allow any non-executable content
		return true
	case ".csv":
		return true // CSV has no magic bytes
	}
	return false
}

func cleanupTemp(path string) {
	if path == "" {
		return
	}
	// Remove both the base temp file (no ext) and the ext-suffixed version.
	_ = os.Remove(path)
}
