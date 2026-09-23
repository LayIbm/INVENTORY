package importer

import "fmt"

// ImportError represents a fatal error that prevents the entire file from being
// processed (e.g. file not found, unsupported format, connection failure).
type ImportError struct {
	Code    string
	Message string
	Cause   error
}

func (e *ImportError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("import %s: %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("import %s: %s", e.Code, e.Message)
}

func (e *ImportError) Unwrap() error { return e.Cause }

func errFile(msg string, cause error) *ImportError {
	return &ImportError{Code: "FILE_ERROR", Message: msg, Cause: cause}
}

func errDB(msg string, cause error) *ImportError {
	return &ImportError{Code: "DB_ERROR", Message: msg, Cause: cause}
}

// addError appends an ERROR-level issue to a row.
func addError(row *ImportRow, field, original, message string) {
	row.Issues = append(row.Issues, RowIssue{
		Field:         field,
		OriginalValue: original,
		Severity:      SeverityError,
		Message:       message,
	})
}

// addWarning appends a WARNING-level issue to a row.
func addWarning(row *ImportRow, field, original, normalized, message string) {
	row.Issues = append(row.Issues, RowIssue{
		Field:           field,
		OriginalValue:   original,
		NormalizedValue: normalized,
		Severity:        SeverityWarning,
		Message:         message,
	})
}

// addInfo appends an INFO-level note to a row.
func addInfo(row *ImportRow, field, original, message string) {
	row.Issues = append(row.Issues, RowIssue{
		Field:         field,
		OriginalValue: original,
		Severity:      SeverityInfo,
		Message:       message,
	})
}
