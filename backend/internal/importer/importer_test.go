package importer_test

import (
	"context"
	"encoding/csv"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/layssagonzalez/device-inventory/backend/internal/importer"
)

// ─── Normalizer tests ─────────────────────────────────────────────────────────

func TestNormalizeHeader(t *testing.T) {
	cases := []struct{ input, want string }{
		{"Serial Number", "serial number"},
		{"  BRAND  ", "brand"},
		{"Model  Variant", "model variant"},
		{"End of Life", "end of life"},
	}
	for _, tc := range cases {
		got := importer.NormalizeHeader(tc.input)
		if got != tc.want {
			t.Errorf("NormalizeHeader(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestNormalizeSerial(t *testing.T) {
	cases := []struct{ input, want string }{
		{"pf9fzmx1", "PF9FZMX1"},
		{"  PF 9FZM X1  ", "PF9FZMX1"},
		{"foc1624w0c5", "FOC1624W0C5"},
	}
	for _, tc := range cases {
		got := importer.NormalizeSerial(tc.input)
		if got != tc.want {
			t.Errorf("NormalizeSerial(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestNormalizeDeviceType(t *testing.T) {
	cases := []struct{ input, want string }{
		{"laptop", "Laptop"},
		{"LAPTOP", "Laptop"},
		{"Laptop PC", "Laptop"},
		{"monitor", "Monitor"},
		{"switch", "Switch"},
		{"router", "Router"},
		{"access point", "AccessPoint"},
		{"Desktop", "Desktop"},
		{"desktop pc", "Desktop"},
		{"adapter", "Adapter"},
		{"adaptador", "Adapter"},
		{"firewall", "Firewall"},
		{"license", "License"},
		{"unknown-device", ""},
		{"", ""},
	}
	for _, tc := range cases {
		got := importer.NormalizeDeviceType(tc.input)
		if got != tc.want {
			t.Errorf("NormalizeDeviceType(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestNormalizeDate(t *testing.T) {
	cases := []struct {
		input     string
		wantNorm  string
		wantOK    bool
		wantAmbig bool
	}{
		{"2025-06-16", "2025-06-16", true, false},
		{"2026-07-13T00:00:00.000Z", "2026-07-13", true, false},
		{"", "", false, false},
		{"--", "", false, false},
		{"n/a", "", false, false},
		{"NEOL", "", false, false},
		{"32/13/2025", "", false, false}, // invalid
		{"2020-11-30T00:00:00.000Z", "2020-11-30", true, false},
	}
	for _, tc := range cases {
		norm, ok, ambig := importer.NormalizeDate(tc.input)
		if ok != tc.wantOK {
			t.Errorf("NormalizeDate(%q) ok=%v, want %v", tc.input, ok, tc.wantOK)
		}
		if ambig != tc.wantAmbig {
			t.Errorf("NormalizeDate(%q) ambig=%v, want %v", tc.input, ambig, tc.wantAmbig)
		}
		if ok && norm != tc.wantNorm {
			t.Errorf("NormalizeDate(%q) norm=%q, want %q", tc.input, norm, tc.wantNorm)
		}
	}
}

func TestNormalizeCost(t *testing.T) {
	cases := []struct {
		input    string
		wantNorm string
		wantOK   bool
	}{
		{"1944.92", "1944.92", true},
		{"$1944.92", "1944.92", true},
		{"0", "0", true},
		{"n/a", "0", true},
		{"", "0", true},
		{"251.58", "251.58", true},
		{"not-a-number", "", false},
	}
	for _, tc := range cases {
		norm, ok := importer.NormalizeCost(tc.input)
		if ok != tc.wantOK {
			t.Errorf("NormalizeCost(%q) ok=%v, want %v", tc.input, ok, tc.wantOK)
		}
		if ok && norm != tc.wantNorm {
			t.Errorf("NormalizeCost(%q) norm=%q, want %q", tc.input, norm, tc.wantNorm)
		}
	}
}

func TestNormalizeBool(t *testing.T) {
	trueVals := []string{"true", "yes", "si", "1", "ok", "YES", "Si", "Sí"}
	falseVals := []string{"false", "no", "0", "n/a", "NA", "NOT_APPLICABLE", "disabled"}
	for _, v := range trueVals {
		b, ok := importer.NormalizeBool(v)
		if !ok || !b {
			t.Errorf("NormalizeBool(%q) should be (true, true), got (%v, %v)", v, b, ok)
		}
	}
	for _, v := range falseVals {
		b, ok := importer.NormalizeBool(v)
		if !ok || b {
			t.Errorf("NormalizeBool(%q) should be (false, true), got (%v, %v)", v, b, ok)
		}
	}
}

// ─── Alias / column resolution tests ─────────────────────────────────────────

func TestResolveColumn(t *testing.T) {
	cases := []struct{ input, want string }{
		{"Serial Number", "serial"},
		{"serial no.", "serial"},
		{"SN", "serial"},
		{"S/N", "serial"},
		{"Device Type", "device_type"},
		{"Type of device", "device_type"},
		{"End of Life", "end_of_life"},
		{"EOL", "end_of_life"},
		{"Product Id", "product_id"},
		{"PID", "product_id"},
		{"Column1", "IGNORE"},
		{"column1", "IGNORE"},
		{"Description", "device_type"},
		{"Model Variant", "model_variant"},
	}
	for _, tc := range cases {
		got := importer.ResolveColumn(tc.input)
		if got != tc.want {
			t.Errorf("ResolveColumn(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// ─── CSV parsing tests ────────────────────────────────────────────────────────

func writeTempCSV(t *testing.T, rows [][]string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "test-*.csv")
	if err != nil {
		t.Fatalf("create temp CSV: %v", err)
	}
	defer f.Close()
	w := csv.NewWriter(f)
	if err := w.WriteAll(rows); err != nil {
		t.Fatalf("write CSV: %v", err)
	}
	w.Flush()
	return f.Name()
}

func TestParseCSV_Basic(t *testing.T) {
	rows := [][]string{
		{"Serial Number", "Device Type", "Model", "Brand"},
		{"PF9FZMX1", "Laptop", "T490", "Lenovo"},
		{"FOC1624W0C5", "Switch", "Catalyst 2960S", "Cisco"},
		{"", "", "", ""}, // empty row — should be skipped
	}
	path := writeTempCSV(t, rows)

	sheets, err := importer.ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if len(sheets) != 1 {
		t.Fatalf("expected 1 sheet, got %d", len(sheets))
	}
	ps := sheets[0]
	if len(ps.Rows) != 2 {
		t.Errorf("expected 2 data rows, got %d", len(ps.Rows))
	}
	if ps.EmptyRows != 1 {
		t.Errorf("expected 1 empty row, got %d", ps.EmptyRows)
	}
	// Check field resolution
	if ps.Rows[0]["serial"] != "PF9FZMX1" {
		t.Errorf("serial = %q, want PF9FZMX1", ps.Rows[0]["serial"])
	}
}

func TestParseCSV_EmptySheet(t *testing.T) {
	rows := [][]string{}
	path := writeTempCSV(t, rows)
	sheets, err := importer.ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if len(sheets) != 1 {
		t.Fatalf("expected 1 sheet, got %d", len(sheets))
	}
	if len(sheets[0].Rows) != 0 {
		t.Errorf("expected 0 rows for empty sheet")
	}
}

func TestParseCSV_MissingColumns(t *testing.T) {
	rows := [][]string{
		{"Brand", "Notes"}, // no serial or device_type
		{"Lenovo", "some notes"},
	}
	path := writeTempCSV(t, rows)
	sheets, err := importer.ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if len(sheets[0].Rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(sheets[0].Rows))
	}
}

func TestParseFile_UnsupportedExt(t *testing.T) {
	_, err := importer.ParseFile("/tmp/test.pdf")
	if err == nil {
		t.Fatal("expected error for unsupported extension")
	}
}

// ─── Validation / policy tests ────────────────────────────────────────────────

// buildService returns a Service with a nil pool (suitable for dry-run / validation tests
// that don't need DB access).
func buildServiceNoPool() *importer.Service {
	return importer.NewService(nil)
}

func TestPreview_DryRunNoWrites(t *testing.T) {
	rows := [][]string{
		{"Serial Number", "Device Type", "Model", "Brand", "Condition"},
		{"TEST-SERIAL-01", "Laptop", "T490", "Lenovo", "Good"},
	}
	path := writeTempCSV(t, rows)

	svc := buildServiceNoPool()
	result, err := svc.Preview(context.Background(), path, importer.Options{DryRun: true})
	if err != nil {
		t.Fatalf("Preview: %v", err)
	}
	if result.Summary.RowsRead != 1 {
		t.Errorf("rows_read = %d, want 1", result.Summary.RowsRead)
	}
	// No DB available, so it cannot check duplicates — should still return insertable
	if result.Summary.Insertable != 1 {
		t.Errorf("insertable = %d, want 1", result.Summary.Insertable)
	}
}

func TestPreview_SerialRequired(t *testing.T) {
	rows := [][]string{
		{"Serial Number", "Device Type", "Model"},
		{"", "Laptop", "T490"}, // empty serial → ERROR
	}
	path := writeTempCSV(t, rows)

	svc := buildServiceNoPool()
	result, err := svc.Preview(context.Background(), path, importer.Options{DryRun: true})
	if err != nil {
		t.Fatalf("Preview: %v", err)
	}
	if result.Summary.WithErrors != 1 {
		t.Errorf("with_errors = %d, want 1", result.Summary.WithErrors)
	}
	if len(result.Errors) != 1 {
		t.Fatalf("errors = %d, want 1", len(result.Errors))
	}
}

func TestPreview_DuplicateSerial(t *testing.T) {
	rows := [][]string{
		{"Serial Number", "Device Type", "Model"},
		{"DUPE-001", "Laptop", "T490"},
		{"DUPE-001", "Laptop", "T490"}, // duplicate
	}
	path := writeTempCSV(t, rows)

	svc := buildServiceNoPool()
	result, err := svc.Preview(context.Background(), path, importer.Options{DryRun: true})
	if err != nil {
		t.Fatalf("Preview: %v", err)
	}
	if result.Summary.Duplicates != 1 {
		t.Errorf("duplicates = %d, want 1", result.Summary.Duplicates)
	}
}

func TestPreview_UnknownDeviceType(t *testing.T) {
	rows := [][]string{
		{"Serial Number", "Device Type", "Model"},
		{"SER-ABC", "Toaster", "Pro 3000"}, // unsupported type
	}
	path := writeTempCSV(t, rows)

	svc := buildServiceNoPool()
	result, err := svc.Preview(context.Background(), path, importer.Options{DryRun: true})
	if err != nil {
		t.Fatalf("Preview: %v", err)
	}
	if result.Summary.WithErrors != 1 {
		t.Errorf("with_errors = %d, want 1", result.Summary.WithErrors)
	}
}

// ─── Error CSV tests ──────────────────────────────────────────────────────────

func TestWriteErrorCSV(t *testing.T) {
	rows := []importer.ImportRow{
		{
			File: "test.csv", Sheet: "Sheet1", Row: 2,
			Entity: importer.EntityLaptop,
			Fields: map[string]any{"serial": "TESTSERIAL"},
			Issues: []importer.RowIssue{
				{Field: "condition", OriginalValue: "bad", Severity: importer.SeverityWarning, Message: "unknown condition"},
			},
			ProposedAction: importer.ActionInsert,
		},
		{
			File: "test.csv", Sheet: "Sheet1", Row: 3,
			Entity: importer.EntityUnknown,
			Fields: map[string]any{},
			Issues: []importer.RowIssue{
				{Field: "serial", OriginalValue: "", Severity: importer.SeverityError, Message: "serial is required"},
			},
			ProposedAction: importer.ActionError,
		},
	}

	var sb strings.Builder
	if err := importer.WriteErrorCSV(&sb, rows); err != nil {
		t.Fatalf("WriteErrorCSV: %v", err)
	}

	out := sb.String()
	if !strings.Contains(out, "WARNING") {
		t.Error("CSV should contain WARNING severity")
	}
	if !strings.Contains(out, "ERROR") {
		t.Error("CSV should contain ERROR severity")
	}
	if !strings.Contains(out, "serial is required") {
		t.Error("CSV should contain 'serial is required' message")
	}
}

func TestErrorFileName(t *testing.T) {
	name := importer.ErrorFileName()
	if !strings.HasPrefix(name, "import-errors-") {
		t.Errorf("error file name should start with 'import-errors-', got %q", name)
	}
	if !strings.HasSuffix(name, ".csv") {
		t.Errorf("error file name should end with '.csv', got %q", name)
	}
}

// ─── Excel fixture test ───────────────────────────────────────────────────────

// writeMinimalXLSX creates a minimal xlsx fixture for tests without needing
// the real Excel files (which must not be committed).
func writeMinimalXLSX(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "fixture.xlsx")

	// Use excelize to create a minimal xlsx in memory.
	// We import it locally to keep the test self-contained.
	f := createMinimalXLSX(t)
	if err := f.SaveAs(path); err != nil {
		t.Fatalf("save fixture xlsx: %v", err)
	}
	return path
}

func TestParseXLSX_Fixture(t *testing.T) {
	path := writeMinimalXLSX(t)

	sheets, err := importer.ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if len(sheets) == 0 {
		t.Fatal("expected at least 1 sheet")
	}
	// First sheet should have at least 1 data row
	if len(sheets[0].Rows) == 0 {
		t.Error("expected at least 1 data row in fixture")
	}
}

// ─── Entity classification tests ─────────────────────────────────────────────

func TestIsLaptopType(t *testing.T) {
	laptopTypes := []string{"Laptop"}
	nonLaptop := []string{"Desktop", "Monitor", "Switch", "Router", "Firewall", "AccessPoint", "Adapter", "License", "OtherNetwork"}
	for _, v := range laptopTypes {
		if !importer.IsLaptopType(v) {
			t.Errorf("IsLaptopType(%q) should be true", v)
		}
	}
	for _, v := range nonLaptop {
		if importer.IsLaptopType(v) {
			t.Errorf("IsLaptopType(%q) should be false", v)
		}
	}
}

func TestIsEquipmentType(t *testing.T) {
	eqTypes := []string{"Desktop", "Monitor", "Switch", "Router", "Firewall", "AccessPoint", "Adapter", "License", "OtherNetwork"}
	nonEq := []string{"Laptop", "", "Unknown"}
	for _, v := range eqTypes {
		if !importer.IsEquipmentType(v) {
			t.Errorf("IsEquipmentType(%q) should be true", v)
		}
	}
	for _, v := range nonEq {
		if importer.IsEquipmentType(v) {
			t.Errorf("IsEquipmentType(%q) should be false", v)
		}
	}
}

// ─── Email validation tests ───────────────────────────────────────────────────

func TestNormalizeEmail(t *testing.T) {
	cases := []struct{ input, want string }{
		{"  User@Example.COM  ", "user@example.com"},
		{"plain@domain.org", "plain@domain.org"},
		{"", ""},
	}
	for _, tc := range cases {
		got := importer.NormalizeEmail(tc.input)
		if got != tc.want {
			t.Errorf("NormalizeEmail(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestIsValidEmail(t *testing.T) {
	valid := []string{"user@example.com", "foo.bar+tag@domain.co.mx"}
	invalid := []string{"", "not-an-email", "missing@", "@nodomain", "no at sign"}
	for _, v := range valid {
		if !importer.IsValidEmail(v) {
			t.Errorf("IsValidEmail(%q) should be true", v)
		}
	}
	for _, v := range invalid {
		if importer.IsValidEmail(v) {
			t.Errorf("IsValidEmail(%q) should be false", v)
		}
	}
}

// ─── Ambiguous date tests ─────────────────────────────────────────────────────

func TestNormalizeDate_Ambiguous(t *testing.T) {
	// 05/06/2024 — day=5, month=6 → both ≤ 12 → ambiguous
	_, _, ambig := importer.NormalizeDate("05/06/2024")
	if !ambig {
		t.Error("05/06/2024 should be ambiguous (could be DD/MM or MM/DD)")
	}
	// 15/06/2024 — day=15 > 12 → not ambiguous (must be DD/MM)
	norm, ok, ambig2 := importer.NormalizeDate("15/06/2024")
	if ambig2 {
		t.Error("15/06/2024 should NOT be ambiguous")
	}
	if !ok {
		t.Error("15/06/2024 should parse successfully as DD/MM/YYYY")
	}
	if norm != "2024-06-15" {
		t.Errorf("15/06/2024 normalized = %q, want 2024-06-15", norm)
	}
}

func TestNormalizeDate_ExcelSerial(t *testing.T) {
	// Excel serial for 2025-06-16 (days since 1899-12-30)
	norm, ok, ambig := importer.NormalizeDate("45824")
	if !ok || ambig {
		t.Errorf("Excel serial 45824 failed: ok=%v ambig=%v", ok, ambig)
	}
	if norm != "2025-06-16" {
		t.Errorf("Excel serial 45824 = %q, want 2025-06-16", norm)
	}
}

// ─── NormalizeNetType tests ───────────────────────────────────────────────────

func TestNormalizeNetType(t *testing.T) {
	cases := []struct{ input, want string }{
		{"primary", "Primary"},
		{"PRIMARY", "Primary"},
		{"secondary", "Secondary"},
		{"Secundary", "Secondary"}, // typo fix
		{"secundario", "Secondary"},
		{"N/A", "N/A"},
		{"na", "N/A"},
		{"", "N/A"},
	}
	for _, tc := range cases {
		got := importer.NormalizeNetType(tc.input)
		if got != tc.want {
			t.Errorf("NormalizeNetType(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// ─── NormalizeAvailability tests ─────────────────────────────────────────────

func TestNormalizeAvailability(t *testing.T) {
	cases := []struct{ input, want string }{
		{"available", "AVAILABLE"},
		{"AVAILABLE", "AVAILABLE"},
		{"disponible", "AVAILABLE"}, // Spanish alias → AVAILABLE
		{"DISPONIBLE", "AVAILABLE"}, // uppercase Spanish → alias hit → AVAILABLE
		{"in storage", "AVAILABLE"},
		{"en bodega", "AVAILABLE"},
		{"assigned", "NOT_AVAILABLE"},
		{"asignada", "NOT_AVAILABLE"},
		{"ASIGNADA", "NOT_AVAILABLE"}, // alias via lowercase
		{"NOT_AVAILABLE", "NOT_AVAILABLE"},
		{"NO_DISPONIBLE", "NO_DISPONIBLE"}, // not in alias map, passes switch
		{"xyz", ""},
	}
	for _, tc := range cases {
		got := importer.NormalizeAvailability(tc.input)
		if got != tc.want {
			t.Errorf("NormalizeAvailability(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// ─── NormalizeCondition tests ─────────────────────────────────────────────────

func TestNormalizeCondition(t *testing.T) {
	cases := []struct{ input, want string }{
		{"good", "Good"},
		{"GOOD", "Good"},
		{"Good", "Good"},
		{"fair", "Fair"},
		{"Fair", "Fair"},
		{"damaged", "Damaged"},
		{"Damaged", "Damaged"},
		{"new", "New"},
		{"unknown", "Unknown"},
		{"bad", "Damaged"},
		{"broken", "Damaged"},
		{"regular", "Fair"},
		{"nuevo", "New"},
		{"bueno", "Good"},
		{"dañado", "Damaged"},
		{"xyz", ""},
	}
	for _, tc := range cases {
		got := importer.NormalizeCondition(tc.input)
		if got != tc.want {
			t.Errorf("NormalizeCondition(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// ─── Model required validation test ──────────────────────────────────────────

func TestPreview_ModelRequired(t *testing.T) {
	rows := [][]string{
		{"Serial Number", "Device Type", "Model"},
		{"SER-001", "Laptop", ""}, // missing model → ERROR
	}
	path := writeTempCSV(t, rows)

	svc := buildServiceNoPool()
	result, err := svc.Preview(context.Background(), path, importer.Options{DryRun: true})
	if err != nil {
		t.Fatalf("Preview: %v", err)
	}
	if result.Summary.WithErrors != 1 {
		t.Errorf("with_errors = %d, want 1 (model is required)", result.Summary.WithErrors)
	}
}

// ─── XLSX fixture with entity detection ──────────────────────────────────────

func TestPreview_XLSXFixtureEntities(t *testing.T) {
	path := writeMinimalXLSX(t)

	svc := buildServiceNoPool()
	result, err := svc.Preview(context.Background(), path, importer.Options{DryRun: true})
	if err != nil {
		t.Fatalf("Preview: %v", err)
	}
	if result.Summary.LaptopRows < 1 {
		t.Errorf("expected at least 1 laptop row, got %d", result.Summary.LaptopRows)
	}
	if result.Summary.EquipmentRows < 1 {
		t.Errorf("expected at least 1 equipment row, got %d", result.Summary.EquipmentRows)
	}
}

// ─── Column1 warning test ─────────────────────────────────────────────────────

func TestPreview_Column1Warning(t *testing.T) {
	rows := [][]string{
		{"Serial Number", "Device Type", "Model", "Column1"},
		{"SER-001", "Laptop", "T490", "some continuation text"},
	}
	path := writeTempCSV(t, rows)

	svc := buildServiceNoPool()
	result, err := svc.Preview(context.Background(), path, importer.Options{DryRun: true})
	if err != nil {
		t.Fatalf("Preview: %v", err)
	}
	// Column1 with content generates a WARNING, not an error
	if result.Summary.WithWarnings == 0 {
		t.Error("expected WARNING for Column1 with content")
	}
	if result.Summary.WithErrors != 0 {
		t.Errorf("expected 0 errors, got %d", result.Summary.WithErrors)
	}
}

// ─── SKIP policy (INSERT → SKIP for in-file duplicate) ───────────────────────

func TestPreview_SkipPolicyInsertable(t *testing.T) {
	rows := [][]string{
		{"Serial Number", "Device Type", "Model"},
		{"UNIQUE-001", "Laptop", "T490"},
		{"UNIQUE-002", "Switch", "Catalyst"},
	}
	path := writeTempCSV(t, rows)

	svc := buildServiceNoPool()
	result, err := svc.Preview(context.Background(), path, importer.Options{DryRun: true})
	if err != nil {
		t.Fatalf("Preview: %v", err)
	}
	if result.Summary.WithErrors != 0 {
		t.Errorf("expected 0 errors, got %d", result.Summary.WithErrors)
	}
	// Both rows should be insertable when no DB available
	if result.Summary.Insertable != 2 {
		t.Errorf("insertable = %d, want 2", result.Summary.Insertable)
	}
}

// ─── RowsIgnored populated from empty rows ────────────────────────────────────

func TestPreview_RowsIgnored(t *testing.T) {
	rows := [][]string{
		{"Serial Number", "Device Type", "Model"},
		{"SER-001", "Laptop", "T490"},
		{"", "", ""}, // empty row
		{"", "", ""}, // empty row
		{"SER-002", "Switch", "Catalyst"},
	}
	path := writeTempCSV(t, rows)

	svc := buildServiceNoPool()
	result, err := svc.Preview(context.Background(), path, importer.Options{DryRun: true})
	if err != nil {
		t.Fatalf("Preview: %v", err)
	}
	if result.Summary.RowsIgnored != 2 {
		t.Errorf("rows_ignored = %d, want 2", result.Summary.RowsIgnored)
	}
	if result.Summary.RowsRead != 2 {
		t.Errorf("rows_read = %d, want 2", result.Summary.RowsRead)
	}
}

// ─── toLaptopAvailability SCRAP tests ─────────────────────────────────────────

func TestToLaptopAvailability_Scrap(t *testing.T) {
	cases := []struct{ input, want string }{
		// SCRAP must be preserved as-is
		{"SCRAP", "SCRAP"},
		{"scrap", "SCRAP"}, // case-insensitive
		// Existing values still work
		{"DISPONIBLE", "DISPONIBLE"},
		{"AVAILABLE", "DISPONIBLE"},
		{"ASIGNADA", "ASIGNADA"},
		{"NOT_AVAILABLE", "ASIGNADA"},
		{"NO_DISPONIBLE", "NO_DISPONIBLE"},
		// Unknown falls back to DISPONIBLE
		{"xyz", "DISPONIBLE"},
		{"", "DISPONIBLE"},
	}
	for _, tc := range cases {
		got := importer.ToLaptopAvailabilityForTest(tc.input)
		if got != tc.want {
			t.Errorf("toLaptopAvailability(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// ─── toEquipmentAvailability SCRAP tests ──────────────────────────────────────

func TestToEquipmentAvailability_Scrap(t *testing.T) {
	cases := []struct{ input, want string }{
		// SCRAP must be preserved as-is
		{"SCRAP", "SCRAP"},
		{"scrap", "SCRAP"}, // case-insensitive
		// Existing values still work
		{"DISPONIBLE", "DISPONIBLE"},
		{"AVAILABLE", "DISPONIBLE"},
		{"ASIGNADA", "ASIGNADA"},
		{"NOT_AVAILABLE", "ASIGNADA"},
		{"NO_DISPONIBLE", "NO_DISPONIBLE"},
		// Unknown falls back to DISPONIBLE
		{"xyz", "DISPONIBLE"},
		{"", "DISPONIBLE"},
	}
	for _, tc := range cases {
		got := importer.ToEquipmentAvailabilityForTest(tc.input)
		if got != tc.want {
			t.Errorf("toEquipmentAvailability(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// ─── ResolveColumn owner/hostname/geography aliases ───────────────────────────

func TestResolveColumn_OwnerHostnameGeography(t *testing.T) {
	cases := []struct{ input, want string }{
		{"owner", "owner"},
		{"Owner", "owner"},
		{"device owner", "owner"},
		{"Device Owner", "owner"},
		{"hostname", "hostname"},
		{"Hostname", "hostname"},
		{"host name", "hostname"},
		{"Host Name", "hostname"},
		{"geography", "geography"},
		{"Geography", "geography"},
		{"geographic location", "geography"},
		{"Geographic Location", "geography"},
	}
	for _, tc := range cases {
		got := importer.ResolveColumn(tc.input)
		if got != tc.want {
			t.Errorf("ResolveColumn(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// ─── Preview passes owner/hostname/geography from CSV ─────────────────────────

func TestPreview_OwnerHostnameGeographyColumns(t *testing.T) {
	rows := [][]string{
		{"Serial Number", "Device Type", "Model", "Owner", "Hostname", "Geography"},
		{"SER-OHG-01", "Laptop", "T490", "IBM", "MXIBM-HOST01", "CIC1 Guadalajara"},
	}
	path := writeTempCSV(t, rows)

	svc := buildServiceNoPool()
	result, err := svc.Preview(context.Background(), path, importer.Options{DryRun: true})
	if err != nil {
		t.Fatalf("Preview: %v", err)
	}
	if result.Summary.WithErrors != 0 {
		t.Errorf("expected 0 errors, got %d", result.Summary.WithErrors)
	}
	if result.Summary.Insertable != 1 {
		t.Errorf("insertable = %d, want 1", result.Summary.Insertable)
	}
}

// ─── Preview passes SCRAP availability from CSV ───────────────────────────────

func TestPreview_ScrapAvailabilityLaptop(t *testing.T) {
	rows := [][]string{
		{"Serial Number", "Device Type", "Model", "Availability"},
		{"SER-SCRAP-01", "Laptop", "T490", "SCRAP"},
	}
	path := writeTempCSV(t, rows)

	svc := buildServiceNoPool()
	result, err := svc.Preview(context.Background(), path, importer.Options{DryRun: true})
	if err != nil {
		t.Fatalf("Preview: %v", err)
	}
	if result.Summary.WithErrors != 0 {
		t.Errorf("expected 0 errors for SCRAP laptop, got %d", result.Summary.WithErrors)
	}
}

func TestPreview_ScrapAvailabilityEquipment(t *testing.T) {
	rows := [][]string{
		{"Serial Number", "Device Type", "Model", "Availability"},
		{"SER-SCRAP-02", "Switch", "Catalyst 2960", "SCRAP"},
	}
	path := writeTempCSV(t, rows)

	svc := buildServiceNoPool()
	result, err := svc.Preview(context.Background(), path, importer.Options{DryRun: true})
	if err != nil {
		t.Fatalf("Preview: %v", err)
	}
	if result.Summary.WithErrors != 0 {
		t.Errorf("expected 0 errors for SCRAP equipment, got %d", result.Summary.WithErrors)
	}
}

// ─── TestNormalizeDate_RealExcelFormats ───────────────────────────────────────
//
// Confirms that the exact strings produced by excelize.GetRows on the USAA
// Infraestructure.xlsx file are parsed correctly by NormalizeDate.
//
// Formats confirmed via Go probe (cmd/probe_dates):
//
//   "11-06-15"   → 2015-11-06   (MM-DD-YY — dominant format, Inventario sheet)
//   "10-31-22"   → 2022-10-31
//   "03-16-26"   → 2026-03-16
//   "08-12-26"   → 2026-08-12   (End Contract Support)
//   "12-31-32"   → 2032-12-31   (End of Life far future)
//   "23/08/2026" → 2026-08-23   (DD/MM/YYYY, ISP sheet)
//
//   Values treated as "no date" (ok=false, ambiguous=false):
//   "--", "NEOL", "N/A", ""
func TestNormalizeDate_RealExcelFormats(t *testing.T) {
	cases := []struct {
		input     string
		wantNorm  string
		wantOK    bool
		wantAmbig bool
		desc      string
	}{
		// ── MM-DD-YY (Excel number format produced by excelize) ───────────────
		{"11-06-15", "2015-11-06", true, false, "MM-DD-YY from Inventario End of Sale"},
		{"10-31-22", "2022-10-31", true, false, "MM-DD-YY End of Sale"},
		{"11-30-20", "2020-11-30", true, false, "MM-DD-YY End of Life"},
		{"03-16-26", "2026-03-16", true, false, "MM-DD-YY End of Sale 2026"},
		{"08-12-26", "2026-08-12", true, false, "MM-DD-YY End Contract Support"},
		{"12-31-32", "2032-12-31", true, false, "MM-DD-YY far-future End of Life"},
		{"10-31-27", "2027-10-31", true, false, "MM-DD-YY End of Life 2027"},
		{"07-31-27", "2027-07-31", true, false, "MM-DD-YY End of Life Jul 2027"},
		{"11-30-28", "2028-11-30", true, false, "MM-DD-YY End of Life 2028"},
		{"03-16-25", "2025-03-16", true, false, "MM-DD-YY End of Sale 2025"},
		{"10-31-25", "2025-10-31", true, false, "MM-DD-YY End of Life 2025"},
		{"03-16-27", "2027-03-16", true, false, "MM-DD-YY End of Life Mar 2027"},

		// ── MM-DD-YYYY (4-digit year variant) ────────────────────────────────
		{"03-16-2026", "2026-03-16", true, false, "MM-DD-YYYY 4-digit year"},
		{"12-31-2032", "2032-12-31", true, false, "MM-DD-YYYY far future"},

		// ── DD/MM/YYYY (ISP sheet — day unambiguous because > 12) ────────────
		{"23/08/2026", "2026-08-23", true, false, "DD/MM/YYYY from ISP sheet"},
		{"15/03/2026", "2026-03-15", true, false, "DD/MM/YYYY day > 12"},

		// ── YYYY-MM-DD (ISO — always accepted) ───────────────────────────────
		{"2025-06-16", "2025-06-16", true, false, "YYYY-MM-DD ISO"},
		{"2020-11-30", "2020-11-30", true, false, "YYYY-MM-DD past"},

		// ── ISO with time component ───────────────────────────────────────────
		{"2026-07-13T00:00:00.000Z", "2026-07-13", true, false, "ISO with ms"},
		{"2020-11-30T00:00:00.000Z", "2020-11-30", true, false, "ISO with ms past"},

		// ── Excel serial date ─────────────────────────────────────────────────
		{"45824", "2025-06-16", true, false, "Excel serial date"},

		// ── No-date sentinel values (ok=false, not errors) ────────────────────
		{"", "", false, false, "empty string"},
		{"--", "", false, false, "double dash"},
		{"NEOL", "", false, false, "NEOL sentinel"},
		{"N/A", "", false, false, "N/A"},
		{"n/a", "", false, false, "n/a lowercase"},
		{"not applicable", "", false, false, "not applicable"},

		// ── Truly invalid ─────────────────────────────────────────────────────
		{"32/13/2025", "", false, false, "invalid date (day 32, month 13)"},
		{"not-a-date", "", false, false, "garbage string"},

		// ── Ambiguous slash-separated (day ≤ 12 AND month ≤ 12) ──────────────
		{"05/06/2024", "", false, true, "ambiguous DD/MM vs MM/DD slash"},
		{"01/02/2024", "", false, true, "ambiguous slash 01/02"},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			norm, ok, ambig := importer.NormalizeDate(tc.input)
			if ok != tc.wantOK {
				t.Errorf("NormalizeDate(%q) ok=%v, want %v", tc.input, ok, tc.wantOK)
			}
			if ambig != tc.wantAmbig {
				t.Errorf("NormalizeDate(%q) ambig=%v, want %v", tc.input, ambig, tc.wantAmbig)
			}
			if tc.wantOK && norm != tc.wantNorm {
				t.Errorf("NormalizeDate(%q) = %q, want %q", tc.input, norm, tc.wantNorm)
			}
		})
	}
}

// ─── TestPreview_DateSentinelsNoWarning ───────────────────────────────────────
//
// Verifies that sentinel "no date" values ("--", "NEOL", "N/A") in date
// fields do NOT produce warnings. They should be silently stored as empty.
func TestPreview_DateSentinelsNoWarning(t *testing.T) {
	rows := [][]string{
		{"Serial Number", "Device Type", "Model", "End of Life", "End of Sale", "End Contract Support"},
		{"SER-SENT-01", "Switch", "Catalyst 2960", "--", "NEOL", "N/A"},
		{"SER-SENT-02", "Router", "ISR 4321", "NEOL", "--", ""},
	}
	path := writeTempCSV(t, rows)

	svc := buildServiceNoPool()
	result, err := svc.Preview(context.Background(), path, importer.Options{DryRun: true})
	if err != nil {
		t.Fatalf("Preview: %v", err)
	}

	if result.Summary.WithWarnings != 0 {
		t.Errorf("expected 0 warnings for sentinel date rows, got %d", result.Summary.WithWarnings)
		for _, r := range result.Warnings {
			for _, issue := range r.Issues {
				t.Logf("  Row %d issue: field=%q message=%q", r.Row, issue.Field, issue.Message)
			}
		}
	}
	if result.Summary.WithErrors != 0 {
		t.Errorf("expected 0 errors for sentinel date rows, got %d", result.Summary.WithErrors)
	}
}

// ─── TestPreview_MMDDYYDates ──────────────────────────────────────────────────
//
// Verifies that the MM-DD-YY format (exact output of excelize on the real
// USAA Infraestructure.xlsx file) is correctly normalised via the Preview
// pipeline: no warnings, correct YYYY-MM-DD in the result fields.
func TestPreview_MMDDYYDates(t *testing.T) {
	rows := [][]string{
		{"Serial Number", "Device Type", "Model", "End of Sale", "End of Life", "End Contract Support"},
		// "11-06-15" → 2015-11-06, "11-30-20" → 2020-11-30, "08-12-26" → 2026-08-12
		{"SER-DATE-01", "Switch", "Catalyst 2960", "11-06-15", "11-30-20", "08-12-26"},
		// "03-16-26" → 2026-03-16, "03-16-27" → 2027-03-16
		{"SER-DATE-02", "Router", "ISR 4321", "03-16-26", "03-16-27", ""},
	}
	path := writeTempCSV(t, rows)

	svc := buildServiceNoPool()
	result, err := svc.Preview(context.Background(), path, importer.Options{DryRun: true})
	if err != nil {
		t.Fatalf("Preview: %v", err)
	}

	if result.Summary.WithWarnings != 0 {
		t.Errorf("expected 0 warnings for MM-DD-YY date rows, got %d", result.Summary.WithWarnings)
		for _, r := range result.Warnings {
			for _, issue := range r.Issues {
				t.Logf("  Row %d warning: field=%q val=%q msg=%q", r.Row, issue.Field, issue.OriginalValue, issue.Message)
			}
		}
	}
	if result.Summary.WithErrors != 0 {
		t.Errorf("expected 0 errors for MM-DD-YY date rows, got %d", result.Summary.WithErrors)
	}
	if result.Summary.Insertable != 2 {
		t.Errorf("expected 2 insertable rows, got %d", result.Summary.Insertable)
	}

	// Verify normalized date values in the sample rows
	type wantDates struct {
		eos string
		eol string
		ecs string
	}
	wants := []wantDates{
		{"2015-11-06", "2020-11-30", "2026-08-12"},
		{"2026-03-16", "2027-03-16", ""},
	}
	for i, want := range wants {
		if i >= len(result.Sample) {
			t.Errorf("sample row %d missing", i)
			continue
		}
		row := result.Sample[i]
		eos, _ := row.Fields["end_of_sale"].(string)
		eol, _ := row.Fields["end_of_life"].(string)
		ecs, _ := row.Fields["end_contract_support"].(string)
		if eos != want.eos {
			t.Errorf("row %d end_of_sale = %q, want %q", i+1, eos, want.eos)
		}
		if eol != want.eol {
			t.Errorf("row %d end_of_life = %q, want %q", i+1, eol, want.eol)
		}
		if ecs != want.ecs {
			t.Errorf("row %d end_contract_support = %q, want %q", i+1, ecs, want.ecs)
		}
	}
}
