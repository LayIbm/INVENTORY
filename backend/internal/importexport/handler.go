package importexport

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/layssagonzalez/device-inventory/backend/internal/enlace"
	"github.com/layssagonzalez/device-inventory/backend/internal/equipment"
	"github.com/layssagonzalez/device-inventory/backend/internal/laptop"
	"github.com/layssagonzalez/device-inventory/backend/internal/middleware"
	"github.com/xuri/excelize/v2"
)

// Handler holds store dependencies for import/export operations.
type Handler struct {
	laptopStore    *laptop.Store
	equipmentStore *equipment.Store
	linkStore      *enlace.Store
}

// NewHandler creates a new Handler.
func NewHandler(ls *laptop.Store, es *equipment.Store, links *enlace.Store) *Handler {
	return &Handler{laptopStore: ls, equipmentStore: es, linkStore: links}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("importexport writeJSON encode", "err", err)
	}
}

func clientError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// Export handles GET /api/export/excel — returns an Excel file with current data.
// Optional query param: ?view=computer|peripherals|network|epd|bios
// Defaults to full two-sheet export when view is omitted or unrecognised.
func (h *Handler) Export(w http.ResponseWriter, r *http.Request) {
	view := r.URL.Query().Get("view")

	laptops, err := h.laptopStore.List(r.Context())
	if err != nil {
		slog.Error("export list laptops", "err", err)
		clientError(w, http.StatusInternalServerError, "internal error")
		return
	}
	equip, err := h.equipmentStore.List(r.Context())
	if err != nil {
		slog.Error("export list equipment", "err", err)
		clientError(w, http.StatusInternalServerError, "internal error")
		return
	}

	links, err := h.linkStore.List(r.Context())
	if err != nil {
		slog.Error("export list links", "err", err)
		clientError(w, http.StatusInternalServerError, "internal error")
		return
	}

	// Filter rows by view when needed
	laptopRows, equipRows, linkRows := filterByView(view, laptops, equip, links)

	xf, buildErr := BuildViewExportFile(view, laptopRows, equipRows, linkRows)
	if buildErr != nil {
		slog.Error("export build file", "err", buildErr)
		clientError(w, http.StatusInternalServerError, "internal error")
		return
	}

	filename := viewFilename(view, "export")
	writeXLSX(w, xf, filename)
}

// ExportTemplate handles GET /api/export/template — returns a blank Excel template.
// Optional query param: ?view=computer|peripherals|network|epd|bios
func (h *Handler) ExportTemplate(w http.ResponseWriter, r *http.Request) {
	view := r.URL.Query().Get("view")

	f, err := BuildViewTemplateFile(view)
	if err != nil {
		slog.Error("export template build", "err", err)
		clientError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeXLSX(w, f, viewFilename(view, "template"))
}

// importSummary is the JSON response body for a successful import.
type importSummary struct {
	LaptopsInserted  int      `json:"laptops_inserted"`
	LaptopsUpdated   int      `json:"laptops_updated"`
	EquipInserted    int      `json:"equipment_inserted"`
	EquipUpdated     int      `json:"equipment_updated"`
	LinksInserted    int      `json:"links_inserted"`
	LinksUpdated     int      `json:"links_updated"`
	Errors           []string `json:"errors"`
}

// Import handles POST /api/import/excel — upserts rows from a multipart .xlsx upload.
func (h *Handler) Import(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20) // 10 MB

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		clientError(w, http.StatusBadRequest, "request too large or not multipart")
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		clientError(w, http.StatusBadRequest, "missing file field")
		return
	}
	defer file.Close() //nolint:errcheck

	var buf bytes.Buffer
	if _, err = buf.ReadFrom(file); err != nil {
		clientError(w, http.StatusBadRequest, "cannot read uploaded file")
		return
	}

	f, err := excelize.OpenReader(&buf)
	if err != nil {
		clientError(w, http.StatusBadRequest, "invalid xlsx file")
		return
	}
	defer f.Close() //nolint:errcheck

	claims := middleware.ClaimsFromContext(r.Context())
	actor := ""
	if claims != nil {
		actor = claims.Username
	}

	// Detect format: legacy USAA base-file (Equipo Prestado / Bodega / TinyPCs)
	// or the standard export format (Laptops / Equipment sheets).
	var laptopRows []laptop.CreateRequest
	var equipRows  []equipment.CreateRequest
	var linkRows   []enlace.CreateRequest
	var parseErrs  []string

	view := r.URL.Query().Get("view")
	if view == "links" || IsLinksFormat(f) {
		linkRows, parseErrs = ParseLinksFile(f)
	} else if IsLegacyFormat(f) {
		laptopRows, equipRows, parseErrs = ParseLegacyFile(f)
	} else if IsEPDFormat(f) {
		laptopRows, equipRows, parseErrs = ParseEPDFile(f)
	} else if IsBIOSFormat(f) {
		laptopRows, equipRows, parseErrs = ParseBIOSFile(f)
	} else {
		laptopRows, equipRows, parseErrs = ParseImportFile(f)
	}

	// Nothing was parsed at all.
	if len(laptopRows) == 0 && len(equipRows) == 0 && len(linkRows) == 0 && len(parseErrs) == 0 {
		clientError(w, http.StatusUnprocessableEntity,
			"No data was found. For the USAA base file, the workbook must have a sheet named "+
				"\"Equipo Prestado\". For the standard format, sheets must be named "+
				"\"Laptops\" and/or \"Equipment\". Download the template to see the standard format.")
		return
	}

	summary := importSummary{Errors: parseErrs}
	if summary.Errors == nil {
		summary.Errors = []string{}
	}

	ctx := r.Context()

	// Upsert laptops
	for _, req := range laptopRows {
		req.Actor = actor
		inserted, upsertErr := upsertLaptop(ctx, h.laptopStore, req)
		if upsertErr != nil {
			summary.Errors = append(summary.Errors, "laptop "+req.Serial+": "+upsertErr.Error())
			continue
		}
		if inserted {
			summary.LaptopsInserted++
		} else {
			summary.LaptopsUpdated++
		}
	}

	// Upsert equipment
	for _, req := range equipRows {
		req.Actor = actor
		inserted, upsertErr := upsertEquipment(ctx, h.equipmentStore, req)
		if upsertErr != nil {
			summary.Errors = append(summary.Errors, "equipment "+req.Serial+": "+upsertErr.Error())
			continue
		}
		if inserted {
			summary.EquipInserted++
		} else {
			summary.EquipUpdated++
		}
	}

	for _, req := range linkRows {
		req.CreatedBy = actor
		if _, createErr := h.linkStore.Create(ctx, req); createErr != nil {
			summary.Errors = append(summary.Errors, "link "+req.Company+": "+createErr.Error())
		}
	}

	writeJSON(w, http.StatusOK, summary)
}

// ImportLinks handles POST /api/import/links — imports only the Links template.
func (h *Handler) ImportLinks(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		clientError(w, http.StatusBadRequest, "request too large or not multipart")
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		clientError(w, http.StatusBadRequest, "missing file field")
		return
	}
	defer file.Close() //nolint:errcheck

	var buf bytes.Buffer
	if _, err = buf.ReadFrom(file); err != nil {
		clientError(w, http.StatusBadRequest, "cannot read uploaded file")
		return
	}

	f, err := excelize.OpenReader(&buf)
	if err != nil {
		clientError(w, http.StatusBadRequest, "invalid xlsx file")
		return
	}
	defer f.Close() //nolint:errcheck

	claims := middleware.ClaimsFromContext(r.Context())
	actor := ""
	if claims != nil {
		actor = claims.Username
	}

	linkRows, parseErrs := ParseLinksFile(f)
	if len(linkRows) == 0 && len(parseErrs) == 0 {
		clientError(w, http.StatusUnprocessableEntity, "No link rows were found. Use the Links template and keep the Links columns intact.")
		return
	}

	summary := importSummary{Errors: parseErrs}
	if summary.Errors == nil {
		summary.Errors = []string{}
	}

	ctx := r.Context()
	for _, req := range linkRows {
		existing, findErr := h.linkStore.GetByIdentifierOrCompanyIP(ctx, req.IdentifierLink, req.Company, req.PublicIP)
		if findErr != nil && !strings.Contains(findErr.Error(), "no rows") {
			summary.Errors = append(summary.Errors, "link "+req.Company+": "+findErr.Error())
			continue
		}
		if existing != nil {
			upd := enlace.UpdateRequest{
				Type:              req.Type,
				Company:           req.Company,
				PublicIP:          req.PublicIP,
				IP:                req.IP,
				Velocity:          req.Velocity,
				EndDateContract:   req.EndDateContract,
				MonthsOfContract:  req.MonthsOfContract,
				AdditionalService: req.AdditionalService,
				ContractNumber:    req.ContractNumber,
				ClientNumber:      req.ClientNumber,
				IdentifierLink:    req.IdentifierLink,
				NameContact:       req.NameContact,
				PhoneContact:      req.PhoneContact,
				EmailContact:      req.EmailContact,
				SupportPhone:      req.SupportPhone,
				SupportClave:      req.SupportClave,
				CurrentPO:         req.CurrentPO,
				Comments:          req.Comments,
			}
			if _, updateErr := h.linkStore.Update(ctx, existing.ID, upd); updateErr != nil {
				summary.Errors = append(summary.Errors, "link "+req.Company+": "+updateErr.Error())
				continue
			}
			summary.LinksUpdated++
			continue
		}

		req.CreatedBy = actor
		if _, createErr := h.linkStore.Create(ctx, req); createErr != nil {
			summary.Errors = append(summary.Errors, "link "+req.Company+": "+createErr.Error())
			continue
		}
		summary.LinksInserted++
	}

	writeJSON(w, http.StatusOK, summary)
}

// upsertLaptop inserts or updates a laptop; returns true if inserted.
func upsertLaptop(ctx context.Context, s *laptop.Store, req laptop.CreateRequest) (bool, error) {
	existing, err := s.GetBySerial(ctx, req.Serial)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			// INSERT
			if _, createErr := s.Create(ctx, req); createErr != nil {
				return false, createErr
			}
			return true, nil
		}
		return false, err
	}
	// UPDATE — map CreateRequest fields to UpdateRequest (no serial).
	// For fields that come empty from the source file (e.g. EPD format has no
	// model, prep, condition, etc.), preserve whatever is already in the DB so
	// we never blank-out enum columns or overwrite good data with nothing.
	keep := func(incoming, existing string) string {
		if incoming == "" || incoming == "Unknown" {
			return existing
		}
		return incoming
	}
	// A device with an assigned employee is always ASIGNADA.
	// Never let a re-import downgrade it to DISPONIBLE regardless of what the
	// source file says (USAA base file marks bodega laptops as DISPONIBLE even
	// when they are in use by an employee).
	keepAvail := func(incoming string) string {
		// Resolve effective employee: incoming takes priority over existing
		var effEmp *string
		if req.EmployeeName != nil {
			effEmp = req.EmployeeName
		} else {
			effEmp = existing.EmployeeName
		}
		if effEmp != nil && *effEmp != "" {
			return "ASIGNADA"
		}
		return keep(incoming, existing.Availability)
	}
	keepPtr := func(incoming *string, existing *string) *string {
		if incoming == nil {
			return existing
		}
		return incoming
	}
	upd := laptop.UpdateRequest{
		Model:     keep(req.Model, existing.Model),
		Variant:   keep(req.Variant, existing.Variant),
		Brand:     keep(req.Brand, existing.Brand),
		Condition: keep(req.Condition, existing.Condition),
		Availability: keepAvail(req.Availability),
		Prep:      keep(req.Prep, existing.Prep),
		Comodato:  keep(req.Comodato, existing.Comodato),
		PowersOn:  keep(req.PowersOn, existing.PowersOn),
		OS:        keep(req.OS, existing.OS),
		Win11Ready:      req.Win11Ready,
		BIOSPassword:    keep(req.BIOSPassword, existing.BIOSPassword),
		WiFi:            keep(req.WiFi, existing.WiFi),
		Bluetooth:       keep(req.Bluetooth, existing.Bluetooth),
		ChargerIncluded: req.ChargerIncluded,
		LastFormatDate:  keep(req.LastFormatDate, existing.LastFormatDate),
		EmployeeName:    keepPtr(req.EmployeeName, existing.EmployeeName),
		EmployeeEmail:   keep(req.EmployeeEmail, existing.EmployeeEmail),
		EmployeeTalentID: keep(req.EmployeeTalentID, existing.EmployeeTalentID),
		LcdOk:            keep(req.LcdOk, existing.LcdOk),
		ExpectedReturnDate: keep(req.ExpectedReturnDate, existing.ExpectedReturnDate),
		Hostname:  keep(req.Hostname, existing.Hostname),
		Geography: keep(req.Geography, existing.Geography),
		Owner:     keep(req.Owner, existing.Owner),
		LastBIOSUpdate: keep(req.LastBIOSUpdate, existing.LastBIOSUpdate),
		BIOSDetails:    keep(req.BIOSDetails, existing.BIOSDetails),
		BluetoothDisabledBIOS: req.BluetoothDisabledBIOS,
		EpdStatus: keep(req.EpdStatus, existing.EpdStatus),
		IPv6:      keep(req.IPv6, existing.IPv6),
		Notes:     keep(req.Notes, existing.Notes),
		Actor:     req.Actor,
	}
	if _, updateErr := s.Update(ctx, existing.ID, upd); updateErr != nil {
		return false, updateErr
	}
	return false, nil
}

// upsertEquipment inserts or updates equipment; returns true if inserted.
func upsertEquipment(ctx context.Context, s *equipment.Store, req equipment.CreateRequest) (bool, error) {
	existing, err := s.GetBySerial(ctx, req.Serial)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			// INSERT
			if _, createErr := s.Create(ctx, req); createErr != nil {
				return false, createErr
			}
			return true, nil
		}
		return false, err
	}
	// UPDATE — preserve existing values when incoming is empty.
	keepE := func(incoming, existingVal string) string {
		if incoming == "" || incoming == "Unknown" {
			return existingVal
		}
		return incoming
	}
	keepEPtr := func(incoming *string, existingVal *string) *string {
		if incoming == nil {
			return existingVal
		}
		return incoming
	}
	// A device with an assigned employee is always ASIGNADA.
	var effEmp *string
	if req.EmployeeName != nil {
		effEmp = req.EmployeeName
	} else {
		effEmp = existing.EmployeeName
	}
	finalAvail := keepE(req.Availability, existing.Availability)
	if effEmp != nil && *effEmp != "" {
		finalAvail = "ASIGNADA"
	}

	upd := equipment.UpdateRequest{
		DeviceType:  keepE(req.DeviceType, existing.DeviceType),
		Brand:       keepE(req.Brand, existing.Brand),
		Model:       keepE(req.Model, existing.Model),
		Variant:     keepE(req.Variant, existing.Variant),
		Condition:   keepE(req.Condition, existing.Condition),
		Availability:  finalAvail,
		Assignability: keepE(req.Assignability, existing.Assignability),
		Comodato:    keepE(req.Comodato, existing.Comodato),
		PowersOn:    keepE(req.PowersOn, existing.PowersOn),
		OS:          keepE(req.OS, existing.OS),
		BIOSPassword: keepE(req.BIOSPassword, existing.BIOSPassword),
		WiFi:        keepE(req.WiFi, existing.WiFi),
		Bluetooth:   keepE(req.Bluetooth, existing.Bluetooth),
		ChargerIncluded: req.ChargerIncluded,
		LastFormatDate:  keepE(req.LastFormatDate, existing.LastFormatDate),
		EmployeeName:    keepEPtr(req.EmployeeName, existing.EmployeeName),
		EmployeeEmail:   keepE(req.EmployeeEmail, existing.EmployeeEmail),
		EmployeeTalentID: keepE(req.EmployeeTalentID, existing.EmployeeTalentID),
		TransactionDate:    keepE(req.TransactionDate, existing.TransactionDate),
		ExpectedReturnDate: keepE(req.ExpectedReturnDate, existing.ExpectedReturnDate),
		MonitorIncluded: req.MonitorIncluded,
		MonitorSerial:   keepE(req.MonitorSerial, existing.MonitorSerial),
		Owner:       keepE(req.Owner, existing.Owner),
		Geography:   keepE(req.Geography, existing.Geography),
		Notes:  keepE(req.Notes, existing.Notes),
		Actor:  req.Actor,
		// Network fields
		ProductID:              keepE(req.ProductID, existing.ProductID),
		NetType:                keepE(req.NetType, existing.NetType),
		Room:                   keepE(req.Room, existing.Room),
		EOLStatus:              keepE(req.EOLStatus, existing.EOLStatus),
		ContractStatus:         keepE(req.ContractStatus, existing.ContractStatus),
		DeviceCompany:          keepE(req.DeviceCompany, existing.DeviceCompany),
		ContactName:            keepE(req.ContactName, existing.ContactName),
		ContactPhone:           keepE(req.ContactPhone, existing.ContactPhone),
		ContactEmail:           keepE(req.ContactEmail, existing.ContactEmail),
		IBMNetworkEmailSupport: keepE(req.IBMNetworkEmailSupport, existing.IBMNetworkEmailSupport),
		IBMLocalEmailSupport:   keepE(req.IBMLocalEmailSupport, existing.IBMLocalEmailSupport),
	}
	if _, updateErr := s.Update(ctx, existing.ID, upd); updateErr != nil {
		return false, updateErr
	}
	return false, nil
}

func writeXLSX(w http.ResponseWriter, f *excelize.File, filename string) {
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		slog.Error("writeXLSX write", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(buf.Bytes())
}

// viewFilename returns a descriptive filename for the exported file.
func viewFilename(view, kind string) string {
	switch view {
	case "computer":
		return "computer_inventory_" + kind + ".xlsx"
	case "peripherals":
		return "peripherals_" + kind + ".xlsx"
	case "network":
		return "network_infrastructure_" + kind + ".xlsx"
	case "links":
		return "links_" + kind + ".xlsx"
	case "epd":
		return "epd_" + kind + ".xlsx"
	case "bios":
		return "bios_control_" + kind + ".xlsx"
	}
	return "inventory_" + kind + ".xlsx"
}

// filterByView filters laptop and equipment slices to only include rows
// relevant to the requested view. This ensures exports show only the data
// that belongs to each view (e.g. peripherals export only peripheral devices).
func filterByView(view string, laptopRows []laptop.Laptop, equipRows []equipment.Equipment, linkRows []enlace.Link) ([]laptop.Laptop, []equipment.Equipment, []enlace.Link) {
	switch view {
	case "computer":
		var filteredEquip []equipment.Equipment
		for _, e := range equipRows {
			if e.DeviceType == "Desktop" || e.DeviceType == "Monitor" {
				filteredEquip = append(filteredEquip, e)
			}
		}
		return laptopRows, filteredEquip, nil
	case "peripherals":
		var filteredEquip []equipment.Equipment
		for _, e := range equipRows {
			switch e.DeviceType {
			case "Mouse", "Keyboard", "Headset", "Cable", "Adapter":
				filteredEquip = append(filteredEquip, e)
			}
		}
		return nil, filteredEquip, nil
	case "network":
		var filteredEquip []equipment.Equipment
		for _, e := range equipRows {
			switch e.DeviceType {
			case "Switch", "Router", "Firewall", "AccessPoint", "License", "OtherNetwork":
				filteredEquip = append(filteredEquip, e)
			}
		}
		return nil, filteredEquip, nil
	case "links":
		return nil, nil, linkRows
	case "epd":
		var filtered []laptop.Laptop
		for _, l := range laptopRows {
			if l.EmployeeName != nil && *l.EmployeeName != "" {
				filtered = append(filtered, l)
			}
		}
		return filtered, nil, nil
	case "bios":
		return laptopRows, nil, nil
	default:
		return laptopRows, equipRows, linkRows
	}
}

