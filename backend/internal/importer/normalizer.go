package importer

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ─── Header normalisation ────────────────────────────────────────────────────

// NormalizeHeader returns a lower-case, trimmed, single-space version of a
// column header. Used as the canonical map key throughout the importer.
func NormalizeHeader(h string) string {
	h = strings.TrimSpace(h)
	h = strings.ToLower(h)
	// collapse multiple spaces
	spaceRe := regexp.MustCompile(`\s+`)
	h = spaceRe.ReplaceAllString(h, " ")
	return h
}

// ─── String helpers ───────────────────────────────────────────────────────────

var spaceRe = regexp.MustCompile(`\s+`)

// trimAndUpper strips surrounding whitespace and uppercases the string.
func trimAndUpper(s string) string {
	return strings.ToUpper(strings.TrimSpace(s))
}

// trimAndCollapse collapses internal whitespace and trims.
func trimAndCollapse(s string) string {
	return strings.TrimSpace(spaceRe.ReplaceAllString(s, " "))
}

// NormalizeSerial uppercases, trims, and removes internal spaces.
func NormalizeSerial(s string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "")
	return s
}

// ─── Usage normalisation ─────────────────────────────────────────────────────

// NormalizeUsage maps raw usage strings to the canonical values:
// "Exclusive IBM" | "IBM Client" | "Exclusive Client" | "" (unknown/empty → store as-is)
func NormalizeUsage(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "exclusive ibm", "exclusivo ibm", "ibm exclusive", "ibm only":
		return "Exclusive IBM"
	case "ibm client", "ibm-client", "ibm/client", "cliente ibm":
		return "IBM Client"
	case "exclusive client", "exclusivo cliente", "client exclusive", "client only", "cliente":
		return "Exclusive Client"
	}
	return raw // preserve unrecognised values as-is so nothing is silently lost
}

// ─── Device Type normalisation ────────────────────────────────────────────────

// deviceTypeAliases maps raw strings to canonical EntityKind / device_type.
var deviceTypeAliases = map[string]string{
	"laptop":       "Laptop",
	"laptop pc":    "Laptop",
	"notebook":     "Laptop",
	"desktop":      "Desktop",
	"desktop pc":   "Desktop",
	"tinypc":       "Desktop",
	"tiny pc":      "Desktop",
	"tiny desktop": "Desktop", // base-file real value
	"mini pc":      "Desktop",
	"m920q":        "Desktop",
	"monitor":      "Monitor",
	"display":      "Monitor",
	"screen":       "Monitor",
	"adapter":      "Adapter",
	"adaptador":    "Adapter",
	"adaptor":      "Adapter",
	"usb-c":        "Adapter",
	"usb c":        "Adapter",
	// full Spanish text from the real base-file
	"adaptador usb c a hdmi": "Adapter",
	"adaptador usb-c a hdmi": "Adapter",
	"adaptador usb c":        "Adapter",
	"switch":                 "Switch",
	"router":                 "Router",
	"firewall":               "Firewall",
	"access point":           "AccessPoint",
	"accesspoint":            "AccessPoint",
	"ap":                     "AccessPoint",
	"license":                "License",
	"licence":                "License",
	"licencia":               "License",
	"othernetwork":           "OtherNetwork",
	"other network":          "OtherNetwork",
}

// NormalizeDeviceType maps raw device type strings to the canonical values
// expected by the equipment handler. Returns empty string if unrecognised.
func NormalizeDeviceType(raw string) string {
	key := strings.ToLower(strings.TrimSpace(raw))
	if v, ok := deviceTypeAliases[key]; ok {
		return v
	}
	return ""
}

// IsLaptopType returns true for values that map to the laptops table.
func IsLaptopType(deviceType string) bool {
	return deviceType == "Laptop"
}

// IsEquipmentType returns true for values that belong to the equipment table.
func IsEquipmentType(deviceType string) bool {
	switch deviceType {
	case "Desktop", "Monitor", "Adapter", "Switch", "Router", "Firewall",
		"AccessPoint", "License", "OtherNetwork":
		return true
	}
	return false
}

// ─── Availability / Assignability / Condition normalisation ──────────────────

var availAliases = map[string]string{
	"available":     "AVAILABLE",
	"avail":         "AVAILABLE",
	"avaible":       "AVAILABLE", // common typo
	"avaialble":     "AVAILABLE", // common typo
	"disponible":    "AVAILABLE",
	"not_available": "NOT_AVAILABLE",
	"not available": "NOT_AVAILABLE",
	"no disponible": "NOT_AVAILABLE",
	"assigned":      "NOT_AVAILABLE", // assigned means not available in storage
	"asigned":       "NOT_AVAILABLE", // common typo
	"asignada":      "NOT_AVAILABLE",
	"asignado":      "NOT_AVAILABLE",
	"in storage":    "AVAILABLE",
	"en bodega":     "AVAILABLE",
	"bodega":        "AVAILABLE",
	"bodega odc":    "AVAILABLE",
}

// NormalizeAvailability maps raw availability to the system enum values.
// For laptops: AVAILABLE, NOT_AVAILABLE. For equipment: DISPONIBLE, ASIGNADA, NO_DISPONIBLE.
func NormalizeAvailability(raw string) string {
	key := strings.ToLower(strings.TrimSpace(raw))
	if v, ok := availAliases[key]; ok {
		return v
	}
	u := strings.ToUpper(strings.TrimSpace(raw))
	switch u {
	case "AVAILABLE", "NOT_AVAILABLE", "DISPONIBLE", "ASIGNADA", "NO_DISPONIBLE":
		return u
	}
	return ""
}

var conditionAliases = map[string]string{
	"good":     "Good",
	"fair":     "Fair",
	"damaged":  "Damaged",
	"new":      "New",
	"unknown":  "Unknown",
	"poor":     "Fair",
	"regular":  "Fair",
	"bad":      "Damaged",
	"broken":   "Damaged",
	"dañado":   "Damaged",
	"nuevo":    "New",
	"bueno":    "Good",
	"regular ": "Fair",
}

// NormalizeCondition maps raw condition values to canonical ones.
func NormalizeCondition(raw string) string {
	key := strings.ToLower(strings.TrimSpace(raw))
	if v, ok := conditionAliases[key]; ok {
		return v
	}
	// Manual title-case pass for the known set of valid conditions.
	switch key {
	case "good":
		return "Good"
	case "fair":
		return "Fair"
	case "damaged":
		return "Damaged"
	case "new":
		return "New"
	case "unknown":
		return "Unknown"
	}
	return ""
}

// ─── Boolean normalisation ────────────────────────────────────────────────────

// NormalizeBool interprets common truthy / falsy strings.
// Returns (value, ok). ok is false when the value is unrecognisable.
func NormalizeBool(raw string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "true", "yes", "si", "sí", "1", "ok", "done", "signed", "firmado",
		"enabled", "on", "habilitado":
		return true, true
	case "false", "no", "0", "not_applicable", "n/a", "na",
		"disabled", "off", "deshabilitado", "desactivado":
		return false, true
	}
	return false, false
}

// isNoDateSentinel returns true for values that are intentional "no date"
// entries (e.g. "--", "NEOL", "N/A") and should be silently stored as empty
// without raising a warning.
func isNoDateSentinel(raw string) bool {
	s := strings.ToLower(strings.TrimSpace(raw))
	return s == "" || s == "--" || s == "n/a" || s == "na" ||
		s == "neol" || s == "not applicable" || s == "not_applicable"
}

// ─── Date normalisation ───────────────────────────────────────────────────────

// dateLayout pairs a Go time layout with a flag indicating whether the parsed
// result requires an ambiguity check (true only for slash-separated formats
// where DD/MM vs MM/DD cannot be inferred from the value alone).
type dateLayout struct {
	layout    string
	checkAmbi bool // if true, fire isDateAmbiguous before accepting
}

// orderedLayouts lists every format NormalizeDate will attempt, in priority
// order. More specific / unambiguous formats come first.
//
// Real formats confirmed from the USAA Infraestructure.xlsx file via
// excelize.GetRows (Go):
//
//	MM-DD-YY      → "11-06-15", "10-31-22", "03-16-26"  (Excel number format)
//	MM-DD-YYYY    → variant with 4-digit year
//	DD/MM/YYYY    → "23/08/2026" (ISP sheet)
//	YYYY-MM-DD    → already supported
//	ISO 8601 with time → already supported
var orderedLayouts = []dateLayout{
	// ISO first — never ambiguous
	{layout: "2006-01-02", checkAmbi: false},
	{layout: "2006/01/02", checkAmbi: false},
	{layout: "2006-01-02T15:04:05Z", checkAmbi: false},
	{layout: "2006-01-02T15:04:05.000Z", checkAmbi: false},
	{layout: time.RFC3339, checkAmbi: false},

	// MM-DD-YY and MM-DD-YYYY — the format excelize produces for this Excel file.
	// Confirmed unambiguous because the business confirmed MM-DD order from the source.
	// Two-digit year: Go interprets 00–68 as 20xx, 69–99 as 19xx (standard).
	{layout: "01-02-06", checkAmbi: false},
	{layout: "01-02-2006", checkAmbi: false},

	// DD-MM-YYYY and DD-MM-YY — kept for other Excel files that may use European order.
	{layout: "02-01-2006", checkAmbi: false},
	{layout: "02-01-06", checkAmbi: false},

	// Slash variants — require ambiguity check when both parts ≤ 12.
	{layout: "02/01/2006", checkAmbi: true},
	{layout: "01/02/2006", checkAmbi: true},
	{layout: "02/01/06", checkAmbi: true},
	{layout: "01/02/06", checkAmbi: true},
}

// NormalizeDate attempts to parse a raw date string and returns it in
// YYYY-MM-DD format. Returns ("", false, ambiguous) where ambiguous is true
// when the input could be MM/DD or DD/MM and cannot be determined safely.
func NormalizeDate(raw string) (normalized string, ok bool, ambiguous bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "--" || strings.EqualFold(raw, "n/a") ||
		strings.EqualFold(raw, "neol") || strings.EqualFold(raw, "not applicable") {
		return "", false, false
	}

	// Excel serial date (numeric string)
	if n, err := strconv.ParseFloat(raw, 64); err == nil && n > 1000 && n < 100000 {
		// Excel epoch: 1900-01-01 = 1; treat 1899-12-30 as base
		baseDate := time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC)
		d := baseDate.Add(time.Duration(n*24) * time.Hour)
		return d.Format("2006-01-02"), true, false
	}

	for _, dl := range orderedLayouts {
		t, err := time.Parse(dl.layout, raw)
		if err != nil {
			continue
		}
		if dl.checkAmbi && isDateAmbiguous(raw) {
			return "", false, true
		}
		return t.Format("2006-01-02"), true, false
	}
	return "", false, false
}

// isDateAmbiguous returns true when a slash-separated date string has both
// the first and second component ≤ 12, making it impossible to tell DD/MM
// from MM/DD without external context.
func isDateAmbiguous(raw string) bool {
	parts := strings.SplitN(raw, "/", 3)
	if len(parts) < 2 {
		return false
	}
	a, err1 := strconv.Atoi(parts[0])
	b, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil {
		return false
	}
	return a <= 12 && b <= 12
}

// ─── Cost normalisation ───────────────────────────────────────────────────────

var costCleanRe = regexp.MustCompile(`[^\d.,]`)

// NormalizeCost strips currency symbols and separators, returning a plain
// decimal string suitable for a NUMERIC(15,2) column.
// Returns ("", false) when no numeric content is found.
func NormalizeCost(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.EqualFold(raw, "n/a") || raw == "--" || raw == "0" {
		return "0", true
	}
	cleaned := costCleanRe.ReplaceAllString(raw, "")
	// Replace comma decimal separator with dot when it looks like European
	// format (e.g. "1.944,92") - presence of both . and ,
	if strings.Contains(cleaned, ".") && strings.Contains(cleaned, ",") {
		cleaned = strings.ReplaceAll(cleaned, ".", "")
		cleaned = strings.ReplaceAll(cleaned, ",", ".")
	} else {
		cleaned = strings.ReplaceAll(cleaned, ",", ".")
	}
	if cleaned == "" {
		return "", false
	}
	if _, err := strconv.ParseFloat(cleaned, 64); err != nil {
		return "", false
	}
	return cleaned, true
}

// ─── Miscellaneous normalisation ─────────────────────────────────────────────

// NormalizeNetType maps raw Type column values to Primary | Secondary | N/A.
func NormalizeNetType(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "primary", "principal":
		return "Primary"
	case "secondary", "secundary", "secundario", "secundaria":
		return "Secondary"
	case "n/a", "na", "":
		return "N/A"
	}
	return strings.TrimSpace(raw)
}

// NormalizeEmail returns a trimmed lowercase email or empty string.
func NormalizeEmail(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}

var emailRe = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

// IsValidEmail returns true for a plausible email address.
func IsValidEmail(email string) bool {
	return emailRe.MatchString(email)
}

// blankOrNA returns true when the value is effectively empty.
func blankOrNA(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	return s == "" || s == "n/a" || s == "na" || s == "not_applicable" ||
		s == "not applicable" || s == "unknown"
}
