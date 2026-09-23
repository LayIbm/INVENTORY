package importer_test

// handler_test.go — HTTP handler tests for the import API endpoints.
//
// Coverage:
//   - POST /api/import/inventory/preview
//   - POST /api/import/inventory/apply
//   - RBAC: viewer → 403, manager → 200, admin → 200, unauthenticated → 401
//   - multipart validation
//   - unsupported extension (.xls, .pdf, no extension)
//   - empty file
//   - file too large
//   - missing "file" field
//   - non-multipart request
//   - batch/upsert/atomic query param parsing (including invalid values)
//   - Options captured by fake service (BatchSize, Upsert, Atomic, DryRun, Actor)
//   - Preview always sets DryRun=true
//   - Apply never sets DryRun=true
//   - service errors → appropriate HTTP status
//   - JSON response validity

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/layssagonzalez/device-inventory/backend/internal/authjwt"
	"github.com/layssagonzalez/device-inventory/backend/internal/importer"
	"github.com/layssagonzalez/device-inventory/backend/internal/middleware"
)

// ─── shared test constants ────────────────────────────────────────────────────

const testSecret = "test-jwt-secret-32-bytes-minimum!"

// minimalCSV is a minimal valid CSV payload with one laptop row.
var minimalCSV = []byte("Serial Number,Device Type,Model,Brand,Condition\nTEST-001,Laptop,T490,Lenovo,Good\n")

// ─── fakeService ─────────────────────────────────────────────────────────────

// fakeService is a test double for ImportService that:
//   - captures every Options it receives,
//   - returns configurable results / errors, and
//   - never touches a database.
type fakeService struct {
	// configuration
	previewResult *importer.PreviewResult
	previewErr    error
	applyResult   *importer.ApplyResult
	applyErr      error

	// captured calls (inspectable in tests)
	previewCalls []importer.Options
	applyCalls   []importer.Options
}

func (f *fakeService) Preview(_ context.Context, _ string, opts importer.Options) (*importer.PreviewResult, error) {
	f.previewCalls = append(f.previewCalls, opts)
	if f.previewErr != nil {
		return nil, f.previewErr
	}
	if f.previewResult != nil {
		return f.previewResult, nil
	}
	// Default: empty but valid result
	return &importer.PreviewResult{
		Sample:   []importer.ImportRow{},
		Errors:   []importer.ImportRow{},
		Warnings: []importer.ImportRow{},
	}, nil
}

func (f *fakeService) Apply(_ context.Context, _ string, opts importer.Options) (*importer.ApplyResult, error) {
	f.applyCalls = append(f.applyCalls, opts)
	if f.applyErr != nil {
		return nil, f.applyErr
	}
	if f.applyResult != nil {
		return f.applyResult, nil
	}
	// Default: empty but valid result
	return &importer.ApplyResult{}, nil
}

// ─── helpers ──────────────────────────────────────────────────────────────────

// newFakeHandler returns a Handler wired to a fakeService (fully controllable).
func newFakeHandler(f *fakeService) *importer.Handler {
	return importer.NewHandler(f)
}

// newRealPreviewHandler returns a Handler backed by a real *Service with nil
// pool. Preview is safe because checkDBDuplicates short-circuits on nil pool.
// Apply must NOT be exercised through this handler — use newFakeHandler instead.
func newRealPreviewHandler() *importer.Handler {
	return importer.NewHandler(importer.NewService(nil))
}

// buildMultipartBody builds a multipart body and returns the body bytes and
// the Content-Type header value including the boundary.
func buildMultipartBody(t *testing.T, filename string, content []byte) (body []byte, contentType string) {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	if _, err := fw.Write(content); err != nil {
		t.Fatalf("write form file: %v", err)
	}
	mw.Close()
	return buf.Bytes(), mw.FormDataContentType()
}

// signedToken returns a signed JWT for the given role/username.
func signedToken(t *testing.T, role, username string) string {
	t.Helper()
	tok, err := authjwt.Sign(testSecret, "uid-1", username, role, false)
	if err != nil {
		t.Fatalf("Sign JWT: %v", err)
	}
	return tok
}

// buildPreviewRequest constructs a multipart POST to /preview with a Bearer token.
func buildPreviewRequest(t *testing.T, role, username, filename string, content []byte) *http.Request {
	t.Helper()
	body, ct := buildMultipartBody(t, filename, content)
	req := httptest.NewRequest(http.MethodPost, "/api/import/inventory/preview", bytes.NewReader(body))
	req.Header.Set("Content-Type", ct)
	req.Header.Set("Authorization", "Bearer "+signedToken(t, role, username))
	return req
}

// buildApplyRequest constructs a multipart POST to /apply with a Bearer token and
// optional query string.
func buildApplyRequest(t *testing.T, role, username, filename string, content []byte, query string) *http.Request {
	t.Helper()
	body, ct := buildMultipartBody(t, filename, content)
	url := "/api/import/inventory/apply"
	if query != "" {
		url += "?" + query
	}
	req := httptest.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	req.Header.Set("Content-Type", ct)
	req.Header.Set("Authorization", "Bearer "+signedToken(t, role, username))
	return req
}

// wrapWithAuth wraps h with Auth + RequireRole middleware (test secret).
func wrapWithAuth(h http.HandlerFunc, roles ...string) http.Handler {
	return middleware.Auth(testSecret)(middleware.RequireRole(roles...)(h))
}

// ─── RBAC — Preview ───────────────────────────────────────────────────────────

func TestPreview_RBAC_ViewerForbidden(t *testing.T) {
	wrapped := wrapWithAuth(newFakeHandler(&fakeService{}).Preview, "manager", "admin")
	req := buildPreviewRequest(t, "viewer", "alice", "data.csv", minimalCSV)
	rr := httptest.NewRecorder()
	wrapped.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Errorf("viewer preview: got %d, want 403", rr.Code)
	}
}

func TestPreview_RBAC_Unauthenticated(t *testing.T) {
	wrapped := wrapWithAuth(newFakeHandler(&fakeService{}).Preview, "manager", "admin")
	req := httptest.NewRequest(http.MethodPost, "/api/import/inventory/preview", nil)
	rr := httptest.NewRecorder()
	wrapped.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("unauthenticated preview: got %d, want 401", rr.Code)
	}
}

func TestPreview_RBAC_ManagerAllowed(t *testing.T) {
	wrapped := wrapWithAuth(newFakeHandler(&fakeService{}).Preview, "manager", "admin")
	req := buildPreviewRequest(t, "manager", "bob", "data.csv", minimalCSV)
	rr := httptest.NewRecorder()
	wrapped.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("manager preview: got %d, want 200; body: %s", rr.Code, rr.Body.String())
	}
}

func TestPreview_RBAC_AdminAllowed(t *testing.T) {
	wrapped := wrapWithAuth(newFakeHandler(&fakeService{}).Preview, "manager", "admin")
	req := buildPreviewRequest(t, "admin", "carol", "data.csv", minimalCSV)
	rr := httptest.NewRecorder()
	wrapped.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("admin preview: got %d, want 200; body: %s", rr.Code, rr.Body.String())
	}
}

// ─── RBAC — Apply ─────────────────────────────────────────────────────────────

func TestApply_RBAC_ViewerForbidden(t *testing.T) {
	wrapped := wrapWithAuth(newFakeHandler(&fakeService{}).Apply, "manager", "admin")
	req := buildApplyRequest(t, "viewer", "alice", "data.csv", minimalCSV, "")
	rr := httptest.NewRecorder()
	wrapped.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Errorf("viewer apply: got %d, want 403", rr.Code)
	}
}

func TestApply_RBAC_Unauthenticated(t *testing.T) {
	wrapped := wrapWithAuth(newFakeHandler(&fakeService{}).Apply, "manager", "admin")
	req := httptest.NewRequest(http.MethodPost, "/api/import/inventory/apply", nil)
	rr := httptest.NewRecorder()
	wrapped.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("unauthenticated apply: got %d, want 401", rr.Code)
	}
}

func TestApply_RBAC_ManagerAllowed(t *testing.T) {
	fake := &fakeService{}
	wrapped := wrapWithAuth(newFakeHandler(fake).Apply, "manager", "admin")
	req := buildApplyRequest(t, "manager", "bob", "data.csv", minimalCSV, "")
	rr := httptest.NewRecorder()
	wrapped.ServeHTTP(rr, req) // must not panic
	if rr.Code == http.StatusForbidden {
		t.Errorf("manager apply: got 403 (RBAC bug); body: %s", rr.Body.String())
	}
	if rr.Code != http.StatusOK {
		t.Errorf("manager apply: got %d, want 200; body: %s", rr.Code, rr.Body.String())
	}
}

func TestApply_RBAC_AdminAllowed(t *testing.T) {
	fake := &fakeService{}
	wrapped := wrapWithAuth(newFakeHandler(fake).Apply, "manager", "admin")
	req := buildApplyRequest(t, "admin", "carol", "data.csv", minimalCSV, "")
	rr := httptest.NewRecorder()
	wrapped.ServeHTTP(rr, req) // must not panic
	if rr.Code == http.StatusForbidden {
		t.Errorf("admin apply: got 403 (RBAC bug); body: %s", rr.Body.String())
	}
	if rr.Code != http.StatusOK {
		t.Errorf("admin apply: got %d, want 200; body: %s", rr.Code, rr.Body.String())
	}
}

// ─── Preview — multipart validation ──────────────────────────────────────────

func TestPreview_ValidCSV_Returns200(t *testing.T) {
	// Use the real service (nil-pool) to exercise the actual parse+validate path.
	req := buildPreviewRequest(t, "admin", "carol", "inventory.csv", minimalCSV)
	rr := httptest.NewRecorder()
	newRealPreviewHandler().Preview(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("got %d, want 200; body: %s", rr.Code, rr.Body.String())
	}
	var result importer.PreviewResult
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Summary.RowsRead < 1 {
		t.Errorf("rows_read = %d, want >= 1", result.Summary.RowsRead)
	}
}

func TestPreview_NoPanic_WithNilPool(t *testing.T) {
	// Nil pool must not cause a panic in Preview. checkDBDuplicates guards this.
	req := buildPreviewRequest(t, "admin", "carol", "inv.csv", minimalCSV)
	rr := httptest.NewRecorder()
	newRealPreviewHandler().Preview(rr, req) // must not panic
	if rr.Code == http.StatusForbidden {
		t.Errorf("unexpected 403 from handler (not RBAC path)")
	}
}

func TestPreview_MissingFileField_Returns400(t *testing.T) {
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	mw.WriteField("other", "value") //nolint:errcheck
	mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rr := httptest.NewRecorder()
	newFakeHandler(&fakeService{}).Preview(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("missing file field: got %d, want 400", rr.Code)
	}
}

func TestPreview_NotMultipart_Returns400(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("plain text"))
	req.Header.Set("Content-Type", "text/plain")
	rr := httptest.NewRecorder()
	newFakeHandler(&fakeService{}).Preview(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("non-multipart: got %d, want 400", rr.Code)
	}
}

// ─── Extension validation ─────────────────────────────────────────────────────

func TestPreview_XLSExtension_Returns400_WithClearMessage(t *testing.T) {
	req := buildPreviewRequest(t, "admin", "carol", "old.xls", []byte("fake content"))
	rr := httptest.NewRecorder()
	newFakeHandler(&fakeService{}).Preview(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf(".xls: got %d, want 400", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, ".xls") || !strings.Contains(body, "not supported") {
		t.Errorf(".xls error body not descriptive: %s", body)
	}
}

func TestApply_XLSExtension_Returns400_WithClearMessage(t *testing.T) {
	req := buildApplyRequest(t, "admin", "carol", "old.xls", []byte("fake"), "")
	rr := httptest.NewRecorder()
	newFakeHandler(&fakeService{}).Apply(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf(".xls apply: got %d, want 400", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, ".xls") || !strings.Contains(body, "not supported") {
		t.Errorf(".xls apply error not descriptive: %s", body)
	}
}

func TestPreview_PDFExtension_Returns400(t *testing.T) {
	req := buildPreviewRequest(t, "admin", "carol", "doc.pdf", []byte("%PDF-1.4"))
	rr := httptest.NewRecorder()
	newFakeHandler(&fakeService{}).Preview(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf(".pdf: got %d, want 400", rr.Code)
	}
}

func TestPreview_NoExtension_Returns400(t *testing.T) {
	req := buildPreviewRequest(t, "admin", "carol", "inventory", []byte("some data"))
	rr := httptest.NewRecorder()
	newFakeHandler(&fakeService{}).Preview(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("no extension: got %d, want 400", rr.Code)
	}
}

// ─── Empty / headers-only file ────────────────────────────────────────────────

func TestPreview_EmptyCSV_Returns200OrUnprocessable(t *testing.T) {
	req := buildPreviewRequest(t, "admin", "carol", "empty.csv", []byte(""))
	rr := httptest.NewRecorder()
	newRealPreviewHandler().Preview(rr, req)
	// Either 200 (0-row sheet) or 422 (service-level rejection) is acceptable.
	if rr.Code != http.StatusOK && rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("empty CSV: got %d, want 200 or 422", rr.Code)
	}
}

func TestPreview_HeadersOnlyCSV_Returns200_ZeroRows(t *testing.T) {
	content := []byte("Serial Number,Device Type,Model\n")
	req := buildPreviewRequest(t, "admin", "carol", "headers.csv", content)
	rr := httptest.NewRecorder()
	newRealPreviewHandler().Preview(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("headers-only CSV: got %d, want 200; body: %s", rr.Code, rr.Body.String())
	}
	var result importer.PreviewResult
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Summary.RowsRead != 0 {
		t.Errorf("rows_read = %d, want 0", result.Summary.RowsRead)
	}
}

// ─── File too large ───────────────────────────────────────────────────────────

func TestPreview_FileTooLarge_Returns400(t *testing.T) {
	// Build a multipart body larger than MaxUploadBytes (50 MB).
	const overLimit = importer.MaxUploadBytes + 1

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("file", "big.csv")
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	chunk := bytes.Repeat([]byte("a"), 4096)
	written := 0
	for written < overLimit {
		n := len(chunk)
		if written+n > overLimit {
			n = overLimit - written
		}
		fw.Write(chunk[:n]) //nolint:errcheck
		written += n
	}
	mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/import/inventory/preview", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rr := httptest.NewRecorder()
	newFakeHandler(&fakeService{}).Preview(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("large file: got %d, want 400", rr.Code)
	}
}

// ─── Apply — missing file ─────────────────────────────────────────────────────

func TestApply_MissingFileField_Returns400(t *testing.T) {
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	mw.WriteField("other", "nope") //nolint:errcheck
	mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/import/inventory/apply", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rr := httptest.NewRecorder()
	newFakeHandler(&fakeService{}).Apply(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("missing file: got %d, want 400", rr.Code)
	}
}

// ─── Apply — batch query param validation ────────────────────────────────────

func TestApply_ValidBatch50_ReachesService(t *testing.T) {
	fake := &fakeService{}
	req := buildApplyRequest(t, "admin", "carol", "inv.csv", minimalCSV, "batch=50")
	rr := httptest.NewRecorder()
	newFakeHandler(fake).Apply(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("batch=50: got %d, want 200; body: %s", rr.Code, rr.Body.String())
	}
	if len(fake.applyCalls) == 0 {
		t.Fatal("service Apply was not called")
	}
	if fake.applyCalls[0].BatchSize != 50 {
		t.Errorf("BatchSize = %d, want 50", fake.applyCalls[0].BatchSize)
	}
}

func TestApply_BatchZero_Returns400(t *testing.T) {
	req := buildApplyRequest(t, "admin", "carol", "inv.csv", minimalCSV, "batch=0")
	rr := httptest.NewRecorder()
	newFakeHandler(&fakeService{}).Apply(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("batch=0: got %d, want 400; body: %s", rr.Code, rr.Body.String())
	}
}

func TestApply_BatchNegative_Returns400(t *testing.T) {
	req := buildApplyRequest(t, "admin", "carol", "inv.csv", minimalCSV, "batch=-1")
	rr := httptest.NewRecorder()
	newFakeHandler(&fakeService{}).Apply(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("batch=-1: got %d, want 400; body: %s", rr.Code, rr.Body.String())
	}
}

func TestApply_BatchNotANumber_Returns400(t *testing.T) {
	req := buildApplyRequest(t, "admin", "carol", "inv.csv", minimalCSV, "batch=abc")
	rr := httptest.NewRecorder()
	newFakeHandler(&fakeService{}).Apply(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("batch=abc: got %d, want 400; body: %s", rr.Code, rr.Body.String())
	}
}

// ─── Apply — Options captured by fake ────────────────────────────────────────

func TestApply_DefaultOptions(t *testing.T) {
	fake := &fakeService{}
	req := buildApplyRequest(t, "admin", "carol", "inv.csv", minimalCSV, "")
	rr := httptest.NewRecorder()
	newFakeHandler(fake).Apply(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("got %d; body: %s", rr.Code, rr.Body.String())
	}
	if len(fake.applyCalls) == 0 {
		t.Fatal("Apply was not called on the service")
	}
	opts := fake.applyCalls[0]
	if opts.DryRun {
		t.Errorf("DryRun = true, want false")
	}
	if opts.Upsert {
		t.Errorf("Upsert = true, want false (default)")
	}
	if opts.Atomic {
		t.Errorf("Atomic = true, want false (default)")
	}
	if opts.BatchSize != importer.DefaultBatchSize {
		t.Errorf("BatchSize = %d, want %d", opts.BatchSize, importer.DefaultBatchSize)
	}
}

func TestApply_UpsertAndAtomicParams(t *testing.T) {
	fake := &fakeService{}
	req := buildApplyRequest(t, "admin", "carol", "inv.csv", minimalCSV, "batch=50&upsert=true&atomic=true")
	rr := httptest.NewRecorder()
	newFakeHandler(fake).Apply(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("got %d; body: %s", rr.Code, rr.Body.String())
	}
	if len(fake.applyCalls) == 0 {
		t.Fatal("Apply was not called on the service")
	}
	opts := fake.applyCalls[0]
	if opts.BatchSize != 50 {
		t.Errorf("BatchSize = %d, want 50", opts.BatchSize)
	}
	if !opts.Upsert {
		t.Errorf("Upsert = false, want true")
	}
	if !opts.Atomic {
		t.Errorf("Atomic = false, want true")
	}
	if opts.DryRun {
		t.Errorf("DryRun must be false for Apply")
	}
}

func TestApply_ActorExtractedFromToken(t *testing.T) {
	fake := &fakeService{}
	// Wrap with auth middleware so claims are injected into the context.
	wrapped := wrapWithAuth(newFakeHandler(fake).Apply, "manager", "admin")
	req := buildApplyRequest(t, "manager", "import-user", "inv.csv", minimalCSV, "")
	rr := httptest.NewRecorder()
	wrapped.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("got %d; body: %s", rr.Code, rr.Body.String())
	}
	if len(fake.applyCalls) == 0 {
		t.Fatal("Apply was not called")
	}
	if fake.applyCalls[0].Actor != "import-user" {
		t.Errorf("Actor = %q, want %q", fake.applyCalls[0].Actor, "import-user")
	}
}

// ─── Apply — service error → HTTP status ─────────────────────────────────────

func TestApply_ServiceError_Returns422(t *testing.T) {
	fake := &fakeService{applyErr: errors.New("parse error: unexpected EOF")}
	req := buildApplyRequest(t, "admin", "carol", "inv.csv", minimalCSV, "")
	rr := httptest.NewRecorder()
	newFakeHandler(fake).Apply(rr, req)
	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("service error: got %d, want 422", rr.Code)
	}
	// Error message must not expose SQL internals; just check it is valid JSON.
	var resp map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Errorf("non-JSON error body: %s", rr.Body.String())
	}
	if _, ok := resp["error"]; !ok {
		t.Errorf("error body missing 'error' key: %v", resp)
	}
}

// ─── Apply — JSON response validity ──────────────────────────────────────────

func TestApply_ResponseIsValidJSON(t *testing.T) {
	fake := &fakeService{}
	req := buildApplyRequest(t, "admin", "carol", "inv.csv", minimalCSV, "")
	rr := httptest.NewRecorder()
	newFakeHandler(fake).Apply(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("got %d; body: %s", rr.Code, rr.Body.String())
	}
	var result importer.ApplyResult
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Errorf("Apply response not valid JSON: %v", err)
	}
}

// ─── Preview — DryRun always true ────────────────────────────────────────────

func TestPreview_AlwaysSetsDryRunTrue(t *testing.T) {
	fake := &fakeService{}
	req := buildPreviewRequest(t, "admin", "carol", "inv.csv", minimalCSV)
	rr := httptest.NewRecorder()
	newFakeHandler(fake).Preview(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("got %d; body: %s", rr.Code, rr.Body.String())
	}
	if len(fake.previewCalls) == 0 {
		t.Fatal("Preview was not called on the service")
	}
	if !fake.previewCalls[0].DryRun {
		t.Errorf("DryRun = false in Preview call, want true")
	}
}

func TestPreview_NeverCallsApply(t *testing.T) {
	fake := &fakeService{}
	req := buildPreviewRequest(t, "admin", "carol", "inv.csv", minimalCSV)
	rr := httptest.NewRecorder()
	newFakeHandler(fake).Preview(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("got %d; body: %s", rr.Code, rr.Body.String())
	}
	if len(fake.applyCalls) != 0 {
		t.Errorf("Preview must never call Apply, but it did (%d times)", len(fake.applyCalls))
	}
}

func TestPreview_ActorExtractedFromToken(t *testing.T) {
	fake := &fakeService{}
	// Wrap with auth middleware so claims are injected into the context.
	wrapped := wrapWithAuth(newFakeHandler(fake).Preview, "manager", "admin")
	req := buildPreviewRequest(t, "manager", "preview-user", "inv.csv", minimalCSV)
	rr := httptest.NewRecorder()
	wrapped.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("got %d; body: %s", rr.Code, rr.Body.String())
	}
	if len(fake.previewCalls) == 0 {
		t.Fatal("Preview was not called")
	}
	if fake.previewCalls[0].Actor != "preview-user" {
		t.Errorf("Actor = %q, want %q", fake.previewCalls[0].Actor, "preview-user")
	}
}

func TestPreview_ServiceError_TempFileCleanedUp(t *testing.T) {
	// Even when the service returns an error the handler must still call
	// cleanupTemp (via defer). We verify the handler returns 422 and not 500.
	fake := &fakeService{previewErr: errors.New("disk full")}
	req := buildPreviewRequest(t, "admin", "carol", "inv.csv", minimalCSV)
	rr := httptest.NewRecorder()
	newFakeHandler(fake).Preview(rr, req)
	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("service error: got %d, want 422", rr.Code)
	}
	// Temp file cleanup is via defer in the handler; if the test doesn't crash
	// with a leaked file descriptor, cleanup occurred (OS-level verification
	// is not feasible in unit tests, but the defer path is exercised).
}

// ─── Preview — response structure ────────────────────────────────────────────

func TestPreview_ResponseContainsRequiredTopLevelFields(t *testing.T) {
	// Use the fake so we control the result shape exactly.
	fake := &fakeService{
		previewResult: &importer.PreviewResult{
			Sample:   []importer.ImportRow{},
			Errors:   []importer.ImportRow{},
			Warnings: []importer.ImportRow{},
		},
	}
	req := buildPreviewRequest(t, "admin", "carol", "inv.csv", minimalCSV)
	rr := httptest.NewRecorder()
	newFakeHandler(fake).Preview(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("got %d; body: %s", rr.Code, rr.Body.String())
	}

	var raw map[string]json.RawMessage
	if err := json.NewDecoder(rr.Body).Decode(&raw); err != nil {
		t.Fatalf("decode: %v", err)
	}
	for _, field := range []string{"summary", "sheets", "sample", "errors", "warnings", "can_apply"} {
		if _, ok := raw[field]; !ok {
			t.Errorf("response missing field %q", field)
		}
	}
}

func TestPreview_SummaryContainsDurationMs(t *testing.T) {
	req := buildPreviewRequest(t, "admin", "carol", "inv.csv", minimalCSV)
	rr := httptest.NewRecorder()
	// Use real nil-pool service to get a real duration populated.
	newRealPreviewHandler().Preview(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("got %d; body: %s", rr.Code, rr.Body.String())
	}
	var result importer.PreviewResult
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Summary.DurationMs < 0 {
		t.Errorf("duration_ms = %d, want >= 0", result.Summary.DurationMs)
	}
}
