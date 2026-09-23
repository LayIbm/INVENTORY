// Package importer provides a reusable Excel/CSV inventory importer shared
// between the CLI and the HTTP endpoint.
package importer

import "time"

// RowAction describes what should happen (or happened) to an imported row.
type RowAction string

const (
	ActionInsert  RowAction = "INSERT"
	ActionUpdate  RowAction = "UPDATE"
	ActionSkip    RowAction = "SKIP"
	ActionError   RowAction = "ERROR"
	ActionUnknown RowAction = "UNKNOWN"
)

// Severity levels for errors and warnings.
type Severity string

const (
	SeverityError   Severity = "ERROR"
	SeverityWarning Severity = "WARNING"
	SeverityInfo    Severity = "INFO"
)

// RowIssue records a single validation or mapping problem for a row field.
type RowIssue struct {
	Field           string   `json:"field"`
	OriginalValue   string   `json:"original_value"`
	NormalizedValue string   `json:"normalized_value,omitempty"`
	Severity        Severity `json:"severity"`
	Message         string   `json:"message"`
}

// EntityKind identifies which database table a row belongs to.
type EntityKind string

const (
	EntityLaptop    EntityKind = "laptop"
	EntityEquipment EntityKind = "equipment"
	EntityUnknown   EntityKind = "unknown"
)

// RawRow holds the original cell values from one Excel/CSV row, keyed by
// normalised header name (lower-case, trimmed).
type RawRow map[string]string

// ImportRow is the neutral representation of one parsed and analysed row.
type ImportRow struct {
	// Source location
	File  string `json:"file"`
	Sheet string `json:"sheet"`
	Row   int    `json:"row"` // 1-based

	// Original cell data (header → raw string value)
	Raw RawRow `json:"raw,omitempty"`

	// Classification
	Entity EntityKind `json:"entity"`

	// Normalised fields ready for DB insertion (may be partial).
	Fields map[string]any `json:"fields,omitempty"`

	// Issues found during validation / normalisation.
	Issues []RowIssue `json:"issues,omitempty"`

	// Proposed / executed action.
	ProposedAction RowAction `json:"proposed_action"`
	ExecutedAction RowAction `json:"executed_action,omitempty"`
}

// HasErrors returns true when any issue is of severity ERROR.
func (r *ImportRow) HasErrors() bool {
	for _, i := range r.Issues {
		if i.Severity == SeverityError {
			return true
		}
	}
	return false
}

// HasWarnings returns true when any issue is of severity WARNING.
func (r *ImportRow) HasWarnings() bool {
	for _, i := range r.Issues {
		if i.Severity == SeverityWarning {
			return true
		}
	}
	return false
}

// Summary holds aggregate statistics for a single import run.
type Summary struct {
	Files        int   `json:"files"`
	Sheets       int   `json:"sheets"`
	RowsRead     int   `json:"rows_read"`
	RowsIgnored  int   `json:"rows_ignored"`
	Valid        int   `json:"valid"`
	Insertable   int   `json:"insertable"`
	Updatable    int   `json:"updatable"`
	Duplicates   int   `json:"duplicates"`
	WithWarnings int   `json:"with_warnings"`
	WithErrors   int   `json:"with_errors"`
	Inserted     int   `json:"inserted"`
	Updated      int   `json:"updated"`
	Skipped      int   `json:"skipped"`
	Errored      int   `json:"errored"`
	DurationMs   int64 `json:"duration_ms"`

	// Entity breakdown
	LaptopRows    int `json:"laptop_rows"`
	EquipmentRows int `json:"equipment_rows"`
	UnknownRows   int `json:"unknown_rows"`
}

// SheetInfo describes one detected sheet.
type SheetInfo struct {
	File      string   `json:"file"`
	Sheet     string   `json:"sheet"`
	RowCount  int      `json:"row_count"`
	EmptyRows int      `json:"empty_rows"`
	Columns   []string `json:"columns"`
}

// PreviewResult is the response from a dry-run (used by the web endpoint and CLI).
type PreviewResult struct {
	Summary  Summary     `json:"summary"`
	Sheets   []SheetInfo `json:"sheets"`
	Sample   []ImportRow `json:"sample"`   // limited subset of rows
	Errors   []ImportRow `json:"errors"`   // rows with ERROR issues
	Warnings []ImportRow `json:"warnings"` // rows with WARNING issues (no errors)
	CanApply bool        `json:"can_apply"`
}

// ApplyResult is the result after actually writing to the database.
type ApplyResult struct {
	Summary    Summary   `json:"summary"`
	ErrorsFile string    `json:"errors_file,omitempty"`
	FinishedAt time.Time `json:"finished_at"`
}

// Options configures an import run.
type Options struct {
	DryRun    bool
	Upsert    bool
	BatchSize int
	Atomic    bool
	Actor     string // username for history entries
}

// DefaultBatchSize is used when Options.BatchSize is 0.
const DefaultBatchSize = 100
