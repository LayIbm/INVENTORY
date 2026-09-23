package billing

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/layssagonzalez/device-inventory/backend/internal/middleware"
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

// List handles GET /api/invoices
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	invoices, err := h.store.List(r.Context())
	if err != nil {
		slog.Error("invoice list", "err", err)
		clientError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, invoices)
}

// Get handles GET /api/invoices/{id}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	inv, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			clientError(w, http.StatusNotFound, "invoice not found")
			return
		}
		slog.Error("invoice get", "err", err)
		clientError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, inv)
}

// Create handles POST /api/invoices
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		clientError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.Number = strings.TrimSpace(strings.ToUpper(req.Number))
	req.Client = strings.TrimSpace(req.Client)
	if req.Number == "" || req.Client == "" {
		clientError(w, http.StatusBadRequest, "number and client are required")
		return
	}
	if req.Status != "" && !validStatus(req.Status) {
		clientError(w, http.StatusBadRequest, "invalid status value")
		return
	}

	claims := middleware.ClaimsFromContext(r.Context())
	if claims != nil {
		req.CreatedBy = claims.Username
	}

	inv, err := h.store.Create(r.Context(), req)
	if err != nil {
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			clientError(w, http.StatusConflict, "invoice number already exists")
			return
		}
		slog.Error("invoice create", "err", err)
		clientError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusCreated, inv)
}

// Update handles PUT /api/invoices/{id}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		clientError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Status != "" && !validStatus(req.Status) {
		clientError(w, http.StatusBadRequest, "invalid status value")
		return
	}

	inv, err := h.store.Update(r.Context(), id, req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			clientError(w, http.StatusNotFound, "invoice not found")
			return
		}
		slog.Error("invoice update", "err", err)
		clientError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, inv)
}

// Delete handles DELETE /api/invoices/{id}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.store.Delete(r.Context(), id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			clientError(w, http.StatusNotFound, "invoice not found")
			return
		}
		slog.Error("invoice delete", "err", err)
		clientError(w, http.StatusInternalServerError, "internal error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
