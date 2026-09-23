package user

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

// List handles GET /api/users  (admin only)
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	users, err := h.store.List(r.Context())
	if err != nil {
		slog.Error("user list", "err", err)
		clientError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, users)
}

// Create handles POST /api/users  (admin only)
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		clientError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Username = strings.TrimSpace(req.Username)
	if req.Name == "" || req.Username == "" || req.Password == "" {
		clientError(w, http.StatusBadRequest, "name, username and password are required")
		return
	}
	validRoles := map[string]bool{"viewer": true, "manager": true}
	if !validRoles[req.Role] {
		req.Role = "viewer"
	}

	u, err := h.store.Create(r.Context(), req)
	if err != nil {
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			clientError(w, http.StatusConflict, "ese nombre de usuario ya existe")
			return
		}
		slog.Error("user create", "err", err)
		clientError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusCreated, u)
}

// Update handles PUT /api/users/:id  (admin only)
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Prevent admin from editing themselves via this endpoint
	claims := middleware.ClaimsFromContext(r.Context())
	if claims != nil && claims.UserID == id {
		clientError(w, http.StatusForbidden, "cannot edit your own account via this endpoint")
		return
	}

	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		clientError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	u, err := h.store.Update(r.Context(), id, req)
	if err != nil {
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			clientError(w, http.StatusConflict, "ese nombre de usuario ya existe")
			return
		}
		if strings.Contains(err.Error(), "no rows") {
			clientError(w, http.StatusNotFound, "user not found")
			return
		}
		slog.Error("user update", "err", err)
		clientError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, u)
}

// ToggleActive handles PATCH /api/users/:id/toggle-active  (admin only)
func (h *Handler) ToggleActive(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Prevent admin from deactivating themselves
	claims := middleware.ClaimsFromContext(r.Context())
	if claims != nil && claims.UserID == id {
		clientError(w, http.StatusForbidden, "cannot deactivate your own account")
		return
	}

	u, err := h.store.ToggleActive(r.Context(), id)
	if err != nil {
		slog.Error("user toggle", "err", err)
		clientError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, u)
}

// Delete handles DELETE /api/users/:id  (admin only)
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Prevent admin from deleting themselves
	claims := middleware.ClaimsFromContext(r.Context())
	if claims != nil && claims.UserID == id {
		clientError(w, http.StatusForbidden, "cannot delete your own account")
		return
	}

	if err := h.store.Delete(r.Context(), id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			clientError(w, http.StatusNotFound, "user not found")
			return
		}
		slog.Error("user delete", "err", err)
		clientError(w, http.StatusInternalServerError, "internal error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
