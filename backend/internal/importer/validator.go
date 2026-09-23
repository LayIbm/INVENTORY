package importer

import (
	"fmt"
	"strings"
)

// validateRow classifies the entity, normalises all recognised fields, and
// appends issues for any validation failures. The row's ProposedAction is set
// to ERROR if any ERROR-severity issue is found.
//
// This function is deterministic and has no DB side effects; it is called
// identically in both dry-run and apply.
func validateRow(row *ImportRow, seenSerials map[string]int) {
	f := row.Fields
	rawSerial := str(f, "serial")

	// ── 1. Serial ─────────────────────────────────────────────────────────────
	normalSerial := NormalizeSerial(rawSerial)
	if normalSerial == "" {
		addError(row, "serial", rawSerial, "serial is required")
		row.ProposedAction = ActionError
		return
	}
	f["serial"] = normalSerial

	// ── 2. Device type classification ─────────────────────────────────────────
	rawType := str(f, "device_type")
	deviceType := NormalizeDeviceType(rawType)
	// NormalizeDeviceType already covers all known types; no further fallback needed.

	if deviceType == "" {
		addError(row, "device_type", rawType, fmt.Sprintf("unrecognised device type %q", rawType))
		row.ProposedAction = ActionError
		return
	}
	f["device_type"] = deviceType

	if IsLaptopType(deviceType) {
		row.Entity = EntityLaptop
	} else if IsEquipmentType(deviceType) {
		row.Entity = EntityEquipment
	} else {
		addError(row, "device_type", rawType, "device type not supported")
		row.ProposedAction = ActionError
		return
	}

	// ── 3. Model ──────────────────────────────────────────────────────────────
	model := strings.TrimSpace(str(f, "model"))
	if model == "" {
		// EPD-style files do not include model information. Use "Unknown" as a
		// placeholder so the row can still be inserted / matched by serial.
		model = "Unknown"
		addWarning(row, "model", "", model, "model not provided; defaulting to 'Unknown'")
	}
	f["model"] = model

	// ── 4. Duplicate within file ──────────────────────────────────────────────
	prevRow, alreadySeen := seenSerials[normalSerial]
	if alreadySeen {
		addWarning(row, "serial", normalSerial, normalSerial,
			fmt.Sprintf("duplicate serial in file (first seen at row %d), skipping", prevRow))
		row.ProposedAction = ActionSkip
		return
	}
	seenSerials[normalSerial] = row.Row

	// ── 5. Optional field normalisation ──────────────────────────────────────

	// Condition
	rawCond := str(f, "condition")
	if rawCond != "" {
		norm := NormalizeCondition(rawCond)
		if norm == "" {
			addWarning(row, "condition", rawCond, "", "unrecognised condition value; stored as-is")
		} else {
			f["condition"] = norm
		}
	}

	// Availability
	rawAvail := str(f, "availability")
	if rawAvail != "" {
		norm := NormalizeAvailability(rawAvail)
		if norm == "" {
			addWarning(row, "availability", rawAvail, "", "unrecognised availability; stored as-is")
		} else {
			f["availability"] = norm
		}
	}

	// Email
	rawEmail := str(f, "employee_email")
	if rawEmail != "" {
		norm := NormalizeEmail(rawEmail)
		if !blankOrNA(norm) && !IsValidEmail(norm) {
			addWarning(row, "employee_email", rawEmail, norm, "invalid email format; stored anyway")
		}
		f["employee_email"] = norm
	}

	// Manager email
	rawMgr := str(f, "employee_manager_email")
	if rawMgr != "" && !blankOrNA(rawMgr) {
		norm := NormalizeEmail(rawMgr)
		if !IsValidEmail(norm) {
			addWarning(row, "employee_manager_email", rawMgr, norm, "invalid manager email; stored anyway")
		}
		f["employee_manager_email"] = norm
	}

	// Dates
	for _, dateField := range []string{"last_format_date", "transaction_date", "expected_return_date",
		"end_of_sale", "end_of_life", "end_contract_support"} {
		rawDate := str(f, dateField)
		if rawDate == "" {
			continue
		}
		// Sentinel values ("--", "NEOL", "N/A", etc.) are intentional "no date"
		// entries — silently store as empty without a warning.
		if isNoDateSentinel(rawDate) {
			f[dateField] = ""
			continue
		}
		norm, ok, ambiguous := NormalizeDate(rawDate)
		if ambiguous {
			addWarning(row, dateField, rawDate, "", "ambiguous date format (could be MM/DD or DD/MM); stored as empty")
			f[dateField] = ""
		} else if ok {
			f[dateField] = norm
		} else {
			addWarning(row, dateField, rawDate, "", "unparseable date; stored as empty")
			f[dateField] = ""
		}
	}

	// Costs
	for _, costField := range []string{"device_cost", "contract_cost"} {
		rawCost := str(f, costField)
		if rawCost == "" {
			continue
		}
		norm, ok := NormalizeCost(rawCost)
		if !ok {
			addWarning(row, costField, rawCost, "", "unparseable cost; stored as NULL")
			f[costField] = nil
		} else {
			f[costField] = norm
		}
	}

	// Net type
	rawNet := str(f, "net_type")
	if rawNet != "" {
		f["net_type"] = NormalizeNetType(rawNet)
	}

	// Usage (laptop-only)
	rawUsage := str(f, "usage")
	if rawUsage != "" {
		f["usage"] = NormalizeUsage(rawUsage)
	}

	// Column1 warning
	if col1, exists := f["IGNORE"]; exists && col1 != "" {
		addWarning(row, "Column1", fmt.Sprintf("%v", col1), "", "Column1 has content but is ignored (meaning undetermined)")
	}

	// Default proposed action if not yet set
	if row.ProposedAction == "" || row.ProposedAction == ActionUnknown {
		row.ProposedAction = ActionInsert
	}
}

// str is a helper that returns the string value of a field, returning "" for
// nil or non-string values.
func str(f map[string]any, key string) string {
	v, ok := f[key]
	if !ok {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}
