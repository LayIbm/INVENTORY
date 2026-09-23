package equipment

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/layssagonzalez/device-inventory/backend/internal/middleware"
	"github.com/layssagonzalez/device-inventory/backend/internal/normalize"
)

// Handler holds the store dependency.
type Handler struct {
	store *Store
}

// NewHandler creates a new Handler.
func NewHandler(store *Store) *Handler {
	return &Handler{store: store}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("writeJSON encode", "err", err)
	}
}

func clientError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// List handles GET /api/equipment
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.store.List(r.Context())
	if err != nil {
		slog.Error("equipment list", "err", err)
		clientError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// Get handles GET /api/equipment/{id}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	e, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			clientError(w, http.StatusNotFound, "equipment not found")
			return
		}
		slog.Error("equipment get", "err", err)
		clientError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, e)
}

// Create handles POST /api/equipment
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		clientError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.Serial = strings.TrimSpace(strings.ToUpper(req.Serial))
	req.Model = strings.TrimSpace(req.Model)
	if req.Serial == "" || req.Model == "" {
		clientError(w, http.StatusBadRequest, "serial and model are required")
		return
	}
	switch req.DeviceType {
	case "Desktop", "Monitor", "Adapter", "Mouse", "Keyboard", "Headset", "Cable",
		"Switch", "Router", "Firewall", "AccessPoint", "License", "OtherNetwork":
	default:
		clientError(w, http.StatusBadRequest, "device_type must be Desktop, Monitor, Adapter, Mouse, Keyboard, Headset, Cable, Switch, Router, Firewall, AccessPoint, License, or OtherNetwork")
		return
	}

	claims := middleware.ClaimsFromContext(r.Context())
	if claims != nil {
		req.Actor = claims.Username
	}
	if req.EmployeeName != nil {
		n := normalize.Name(*req.EmployeeName)
		req.EmployeeName = &n
	}
	req.EmployeeEmail = normalize.Email(req.EmployeeEmail)

	// Defaults
	if req.Brand == "" {
		req.Brand = "Lenovo"
	}
	if req.Availability == "" {
		req.Availability = "DISPONIBLE"
	}
	if req.Assignability == "" {
		req.Assignability = "LISTA"
	}
	if req.Comodato == "" {
		req.Comodato = "N/A"
	}
	if req.PowersOn == "" {
		req.PowersOn = "OK"
	}

	e, err := h.store.Create(r.Context(), req)
	if err != nil {
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			clientError(w, http.StatusConflict, "serial already exists")
			return
		}
		slog.Error("equipment create", "err", err)
		clientError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusCreated, e)
}

// Update handles PUT /api/equipment/{id}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		clientError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	claims := middleware.ClaimsFromContext(r.Context())
	if claims != nil {
		req.Actor = claims.Username
	}
	if req.EmployeeName != nil {
		n := normalize.Name(*req.EmployeeName)
		req.EmployeeName = &n
	}
	req.EmployeeEmail = normalize.Email(req.EmployeeEmail)

	e, err := h.store.Update(r.Context(), id, req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			clientError(w, http.StatusNotFound, "equipment not found")
			return
		}
		slog.Error("equipment update", "err", err)
		clientError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, e)
}

// Delete handles DELETE /api/equipment/{id}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.store.Delete(r.Context(), id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			clientError(w, http.StatusNotFound, "equipment not found")
			return
		}
		slog.Error("equipment delete", "err", err)
		clientError(w, http.StatusInternalServerError, "internal error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
