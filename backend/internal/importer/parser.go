package importer

import (
	"encoding/csv"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/xuri/excelize/v2"
)

// ─── ParsedSheet holds the header and rows for one sheet. ─────────────────────

// ParsedSheet is the raw output of parsing a single sheet.
type ParsedSheet struct {
	File         string
	Sheet        string
	Headers      []string // original header strings (row 0 of data)
	CanonHeaders []string // canonicalized (NormalizeHeader) corresponding to each Header
	FieldKeys    []string // ResolveColumn result for each header
	Rows         []RawRow // one entry per data row (skips header row)
	EmptyRows    int      // fully empty rows skipped
	MergedTitle  string   // non-empty if a merged title row was detected above headers
}

// ─── File parsing ─────────────────────────────────────────────────────────────

// ParseFile opens a spreadsheet (xlsx or csv) and returns all non-metadata
// sheets. It does NOT alter the source file.
func ParseFile(path string) ([]*ParsedSheet, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".xlsx", ".xls":
		return parseExcel(path)
	case ".csv":
		return parseCSV(path)
	default:
		return nil, &ImportError{
			Code:    "UNSUPPORTED_FORMAT",
			Message: fmt.Sprintf("unsupported file extension: %s", ext),
		}
	}
}

// ─── Excel ────────────────────────────────────────────────────────────────────

func parseExcel(path string) ([]*ParsedSheet, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, errFile("cannot open Excel file", err)
	}
	defer f.Close()

	var sheets []*ParsedSheet

	for _, name := range f.GetSheetList() {
		if isMetadataSheet(name) {
			continue
		}

		rows, err := f.GetRows(name)
		if err != nil {
			return nil, errFile(fmt.Sprintf("cannot read sheet %q", name), err)
		}

		ps, err := buildParsedSheet(path, name, rows)
		if err != nil {
			// log and skip malformed sheet
			continue
		}
		sheets = append(sheets, ps)
	}

	return sheets, nil
}

// buildParsedSheet turns a [][]string (as returned by excelize) into a
// ParsedSheet. It handles:
//   - a merged title row above the real headers (detected heuristically)
//   - completely empty rows
//   - rows with more cells than headers (extra cells ignored)
//   - rows with fewer cells than headers (missing cells treated as "")
func buildParsedSheet(file, sheetName string, rows [][]string) (*ParsedSheet, error) {
	if len(rows) == 0 {
		return &ParsedSheet{File: file, Sheet: sheetName}, nil
	}

	headerRowIdx := 0
	mergedTitle := ""

	// Heuristic: detect a merged-category / grouped-header row above the real
	// column headers. We treat row 0 as a title row when:
	//   (a) the second row has at least 3× as many non-empty cells as the first, OR
	//   (b) none of the non-empty cells in row 0 resolve to a known field key.
	// Both conditions are evaluated together to avoid misclassifying real headers.
	if len(rows) >= 2 {
		nonEmptyFirst := countNonEmpty(rows[0])
		nonEmptySecond := countNonEmpty(rows[1])

		isTitleRow := false
		if nonEmptyFirst == 0 {
			isTitleRow = true // blank first row
		} else if nonEmptyFirst <= 2 && nonEmptySecond > nonEmptyFirst+2 {
			isTitleRow = true // very few cells in first row
		} else if nonEmptySecond >= nonEmptyFirst*2 {
			// First row has significantly fewer headers — check if none resolve
			resolvedFirst := 0
			for _, h := range rows[0] {
				if strings.TrimSpace(h) != "" && ResolveColumn(h) != "" {
					resolvedFirst++
				}
			}
			if resolvedFirst == 0 {
				isTitleRow = true // no known column names in first row
			}
		}

		if isTitleRow {
			mergedTitle = strings.TrimSpace(joinNonEmpty(rows[0]))
			headerRowIdx = 1
		}
	}

	if headerRowIdx >= len(rows) {
		return &ParsedSheet{File: file, Sheet: sheetName, MergedTitle: mergedTitle}, nil
	}

	rawHeaders := rows[headerRowIdx]
	headers := make([]string, len(rawHeaders))
	canonHeaders := make([]string, len(rawHeaders))
	fieldKeys := make([]string, len(rawHeaders))

	for i, h := range rawHeaders {
		headers[i] = h
		canonHeaders[i] = NormalizeHeader(h)
		fieldKeys[i] = ResolveColumn(h)
	}

	ps := &ParsedSheet{
		File:         file,
		Sheet:        sheetName,
		Headers:      headers,
		CanonHeaders: canonHeaders,
		FieldKeys:    fieldKeys,
		MergedTitle:  mergedTitle,
	}

	// Detect whether this sheet has an explicit device_type column distinct from
	// "description". When true, "description" is remapped to "model" during row
	// parsing so that product descriptions are not discarded.
	hasExplicitDeviceType := false
	for i, k := range fieldKeys {
		if k == "device_type" {
			h := NormalizeHeader(rawHeaders[i])
			if h != "description" {
				hasExplicitDeviceType = true
				break
			}
		}
	}

	for ri := headerRowIdx + 1; ri < len(rows); ri++ {
		rawRow := rows[ri]
		if isEntirelyEmpty(rawRow) {
			ps.EmptyRows++
			continue
		}
		rr := make(RawRow, len(headers))
		for ci, hdr := range fieldKeys {
			val := ""
			if ci < len(rawRow) {
				val = strings.TrimSpace(rawRow[ci])
			}
			if hdr == "" {
				if ci < len(rawHeaders) {
					// store under original header so nothing is lost
					rr["_col_"+rawHeaders[ci]] = val
				}
				continue
			}
			// When there is an explicit device_type column (not "description"),
			// re-map "description" → "model" so the product description is not lost.
			if hdr == "device_type" && hasExplicitDeviceType {
				origHeader := NormalizeHeader(rawHeaders[ci])
				if origHeader == "description" {
					// Only set model if not already provided by a "model" column.
					if _, alreadySet := rr["model"]; !alreadySet {
						rr["model"] = val
					}
					continue // do NOT store under device_type
				}
			}
			rr[hdr] = val
		}
		ps.Rows = append(ps.Rows, rr)
	}

	return ps, nil
}

// ─── CSV ──────────────────────────────────────────────────────────────────────

func parseCSV(path string) ([]*ParsedSheet, error) {
	// We need an io.Reader; use excelize's filesystem tools since we already have that dep.
	// Actually use stdlib since excelize doesn't expose CSV reading.
	// We'll open via os.
	data, err := readFileBytes(path)
	if err != nil {
		return nil, errFile("cannot read CSV file", err)
	}

	r := csv.NewReader(strings.NewReader(string(data)))
	r.LazyQuotes = true
	r.TrimLeadingSpace = true

	var rows [][]string
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, errFile("CSV parse error", err)
		}
		rows = append(rows, rec)
	}

	sheetName := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	ps, err := buildParsedSheet(path, sheetName, rows)
	if err != nil {
		return nil, err
	}
	return []*ParsedSheet{ps}, nil
}

// ─── Utilities ────────────────────────────────────────────────────────────────

func countNonEmpty(row []string) int {
	n := 0
	for _, v := range row {
		if strings.TrimSpace(v) != "" {
			n++
		}
	}
	return n
}

func joinNonEmpty(row []string) string {
	var parts []string
	for _, v := range row {
		if t := strings.TrimSpace(v); t != "" {
			parts = append(parts, t)
		}
	}
	return strings.Join(parts, " ")
}

func isEntirelyEmpty(row []string) bool {
	for _, v := range row {
		if strings.TrimSpace(v) != "" {
			return false
		}
	}
	return true
}
