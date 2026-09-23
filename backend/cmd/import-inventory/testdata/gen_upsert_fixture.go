//go:build ignore

// gen_upsert_fixture.go creates a minimal xlsx file for upsert testing.
// Run with: go run ./cmd/import-inventory/testdata/gen_upsert_fixture.go
package main

import (
	"fmt"
	"os"

	"github.com/xuri/excelize/v2"
)

func main() {
	f := excelize.NewFile()
	sheet := "Inventory"
	f.SetSheetName("Sheet1", sheet)

	headers := []string{
		"Serial Number", "Device Type", "Model", "Brand", "Condition",
		"Availability", "Assignability", "Notes",
	}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
	}

	// Row 2: existing serial PF9FZMX1 with modified notes and a PRESENT model.
	// A separate availability field is EMPTY to verify COALESCE keeps existing.
	// Specifically: notes is updated, availability is omitted (empty string → COALESCE keeps DB value).
	row := []string{
		"PF9FZMX1", "Laptop", "T490", "Lenovo", "Good",
		"", "NEEDS_PREP", "NOTES ACTUALIZADAS POR UPSERT",
	}
	for i, v := range row {
		cell, _ := excelize.CoordinatesToCellName(i+1, 2)
		f.SetCellValue(sheet, cell, v)
	}

	out := "cmd/import-inventory/testdata/upsert-fixture.xlsx"
	if err := f.SaveAs(out); err != nil {
		fmt.Fprintln(os.Stderr, "ERROR:", err)
		os.Exit(1)
	}
	fmt.Println("Written:", out)
}
