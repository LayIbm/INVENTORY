package importer

import (
	"encoding/csv"
	"fmt"
	"io"
	"time"
)

// WriteErrorCSV writes a CSV error report to w from all rows that have issues.
// The file name format is: import-errors-YYYYMMDD-HHMMSS.csv
func WriteErrorCSV(w io.Writer, rows []ImportRow) error {
	cw := csv.NewWriter(w)
	defer cw.Flush()

	header := []string{
		"file", "sheet", "row", "entity", "serial",
		"field", "original_value", "normalized_value",
		"severity", "message", "action",
	}
	if err := cw.Write(header); err != nil {
		return fmt.Errorf("write CSV header: %w", err)
	}

	for _, row := range rows {
		if len(row.Issues) == 0 {
			continue
		}
		serial := str(row.Fields, "serial")
		// Mask serial partially to avoid leaking PII if it happens to be a person identifier
		maskedSerial := maskSerial(serial)
		action := string(row.ExecutedAction)
		if action == "" {
			action = string(row.ProposedAction)
		}
		for _, issue := range row.Issues {
			record := []string{
				row.File,
				row.Sheet,
				fmt.Sprintf("%d", row.Row),
				string(row.Entity),
				maskedSerial,
				issue.Field,
				issue.OriginalValue,
				issue.NormalizedValue,
				string(issue.Severity),
				issue.Message,
				action,
			}
			if err := cw.Write(record); err != nil {
				return fmt.Errorf("write CSV row: %w", err)
			}
		}
	}
	return cw.Error()
}

// ErrorFileName generates a timestamped error file name.
func ErrorFileName() string {
	return fmt.Sprintf("import-errors-%s.csv", time.Now().Format("20060102-150405"))
}

// maskSerial replaces all but the last 4 chars of a serial with asterisks.
func maskSerial(s string) string {
	if len(s) <= 4 {
		return s
	}
	return "****" + s[len(s)-4:]
}

// PrintSummary writes a human-readable summary to w.
func PrintSummary(w io.Writer, s Summary) {
	fmt.Fprintf(w, "\n──────────────────────────────────\n")
	fmt.Fprintf(w, " IMPORT SUMMARY\n")
	fmt.Fprintf(w, "──────────────────────────────────\n")
	fmt.Fprintf(w, " Files          : %d\n", s.Files)
	fmt.Fprintf(w, " Sheets         : %d\n", s.Sheets)
	fmt.Fprintf(w, " Rows read      : %d\n", s.RowsRead)
	fmt.Fprintf(w, " Rows ignored   : %d\n", s.RowsIgnored)
	fmt.Fprintf(w, " Valid          : %d\n", s.Valid)
	fmt.Fprintf(w, "   Insertable   : %d\n", s.Insertable)
	fmt.Fprintf(w, "   Updatable    : %d\n", s.Updatable)
	fmt.Fprintf(w, " Duplicates     : %d\n", s.Duplicates)
	fmt.Fprintf(w, " With warnings  : %d\n", s.WithWarnings)
	fmt.Fprintf(w, " With errors    : %d\n", s.WithErrors)
	if s.Inserted > 0 || s.Updated > 0 {
		fmt.Fprintf(w, " Inserted       : %d\n", s.Inserted)
		fmt.Fprintf(w, " Updated        : %d\n", s.Updated)
		fmt.Fprintf(w, " Skipped        : %d\n", s.Skipped)
		fmt.Fprintf(w, " Errored        : %d\n", s.Errored)
		fmt.Fprintf(w, " Duration       : %dms\n", s.DurationMs)
	}
	fmt.Fprintf(w, " Entities:\n")
	fmt.Fprintf(w, "   Laptops      : %d\n", s.LaptopRows)
	fmt.Fprintf(w, "   Equipment    : %d\n", s.EquipmentRows)
	fmt.Fprintf(w, "   Unknown      : %d\n", s.UnknownRows)
	fmt.Fprintf(w, "──────────────────────────────────\n\n")
}
