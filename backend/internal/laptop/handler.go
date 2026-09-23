package laptop

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

// List handles GET /api/laptops
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	laptops, err := h.store.List(r.Context())
	if err != nil {
		slog.Error("laptop list", "err", err)
		clientError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, laptops)
}

// Get handles GET /api/laptops/{id}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	l, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			clientError(w, http.StatusNotFound, "laptop not found")
			return
		}
		slog.Error("laptop get", "err", err)
		clientError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, l)
}

// Create handles POST /api/laptops
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
	if req.EmployeeName != nil {
		n := normalize.Name(*req.EmployeeName)
		req.EmployeeName = &n
	}
	req.EmployeeEmail = normalize.Email(req.EmployeeEmail)

	claims := middleware.ClaimsFromContext(r.Context())
	if claims != nil {
		req.Actor = claims.Username
	}

	// Default values
	if req.Brand == "" {
		req.Brand = "Lenovo"
	}
	if req.Availability == "" {
		req.Availability = "DISPONIBLE"
	}
	if req.Prep == "" {
		req.Prep = "NECESITA_PREP"
	}
	if req.Comodato == "" {
		req.Comodato = "N/A"
	}
	if req.PowersOn == "" {
		req.PowersOn = "OK"
	}
	if req.LcdOk == "" {
		req.LcdOk = "OK"
	}

	l, err := h.store.Create(r.Context(), req)
	if err != nil {
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			clientError(w, http.StatusConflict, "serial already exists")
			return
		}
		slog.Error("laptop create", "err", err)
		clientError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusCreated, l)
}

// Update handles PUT /api/laptops/{id}
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

	l, err := h.store.Update(r.Context(), id, req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			clientError(w, http.StatusNotFound, "laptop not found")
			return
		}
		slog.Error("laptop update", "err", err)
		clientError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, l)
}

// Delete handles DELETE /api/laptops/{id}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.store.Delete(r.Context(), id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			clientError(w, http.StatusNotFound, "laptop not found")
			return
		}
		slog.Error("laptop delete", "err", err)
		clientError(w, http.StatusInternalServerError, "internal error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
