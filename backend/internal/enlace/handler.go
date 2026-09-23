package enlace

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

// List handles GET /api/enlaces
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	links, err := h.store.List(r.Context())
	if err != nil {
		slog.Error("enlace list", "err", err)
		clientError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, links)
}

// Get handles GET /api/enlaces/{id}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	link, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			clientError(w, http.StatusNotFound, "enlace not found")
			return
		}
		slog.Error("enlace get", "err", err)
		clientError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, link)
}

// Create handles POST /api/enlaces
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		clientError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Type = strings.TrimSpace(req.Type)
	req.Company = strings.TrimSpace(req.Company)
	if req.Type == "" || req.Company == "" {
		clientError(w, http.StatusBadRequest, "type and company are required")
		return
	}
	if req.Type != "Primary" && req.Type != "Secondary" {
		clientError(w, http.StatusBadRequest, "type must be Primary or Secondary")
		return
	}

	req.NameContact = normalize.Name(req.NameContact)
	req.EmailContact = normalize.Email(req.EmailContact)

	claims := middleware.ClaimsFromContext(r.Context())
	if claims != nil {
		req.CreatedBy = claims.Username
	}

	link, err := h.store.Create(r.Context(), req)
	if err != nil {
		slog.Error("enlace create", "err", err)
		clientError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusCreated, link)
}

// Update handles PUT /api/enlaces/{id}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		clientError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Type = strings.TrimSpace(req.Type)
	req.Company = strings.TrimSpace(req.Company)
	if req.Type == "" || req.Company == "" {
		clientError(w, http.StatusBadRequest, "type and company are required")
		return
	}
	if req.Type != "Primary" && req.Type != "Secondary" {
		clientError(w, http.StatusBadRequest, "type must be Primary or Secondary")
		return
	}

	req.NameContact = normalize.Name(req.NameContact)
	req.EmailContact = normalize.Email(req.EmailContact)

	link, err := h.store.Update(r.Context(), id, req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			clientError(w, http.StatusNotFound, "enlace not found")
			return
		}
		slog.Error("enlace update", "err", err)
		clientError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, link)
}

// Delete handles DELETE /api/enlaces/{id}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.store.Delete(r.Context(), id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			clientError(w, http.StatusNotFound, "enlace not found")
			return
		}
		slog.Error("enlace delete", "err", err)
		clientError(w, http.StatusInternalServerError, "internal error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
