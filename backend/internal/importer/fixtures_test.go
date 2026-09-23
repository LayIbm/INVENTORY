package importer_test

import "github.com/xuri/excelize/v2"

// createMinimalXLSX builds an in-memory xlsx file with known test data.
// It is used only in tests and does not touch the real Excel files in docs/.
func createMinimalXLSX(t interface{ Fatalf(string, ...any) }) *excelize.File {
	f := excelize.NewFile()
	sheet := "Inventory"
	f.SetSheetName("Sheet1", sheet)

	headers := []string{
		"Serial Number", "Device Type", "Model", "Brand", "Condition",
		"Availability", "Notes",
	}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
	}

	// One laptop row
	laptopRow := []string{
		"TEST-SER-001", "Laptop", "T490", "Lenovo", "Good",
		"AVAILABLE", "Test note",
	}
	for i, v := range laptopRow {
		cell, _ := excelize.CoordinatesToCellName(i+1, 2)
		f.SetCellValue(sheet, cell, v)
	}

	// One switch row
	switchRow := []string{
		"FOC-TEST-002", "Switch", "Catalyst 2960", "Cisco", "Good",
		"AVAILABLE", "",
	}
	for i, v := range switchRow {
		cell, _ := excelize.CoordinatesToCellName(i+1, 3)
		f.SetCellValue(sheet, cell, v)
	}

	// One empty row
	// (excelize ignores it automatically)

	return f
}
