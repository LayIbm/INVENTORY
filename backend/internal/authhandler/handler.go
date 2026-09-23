package authhandler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/layssagonzalez/device-inventory/backend/internal/authjwt"
	"github.com/layssagonzalez/device-inventory/backend/internal/middleware"
	"github.com/layssagonzalez/device-inventory/backend/internal/user"
	"golang.org/x/crypto/bcrypt"
)

// Handler handles auth-related endpoints.
type Handler struct {
	userStore *user.Store
	jwtSecret string
}

// NewHandler creates a new auth Handler.
func NewHandler(userStore *user.Store, jwtSecret string) *Handler {
	return &Handler{userStore: userStore, jwtSecret: jwtSecret}
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

// Login handles POST /api/auth/login
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		clientError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	u, err := h.userStore.GetByUsername(r.Context(), req.Username)
	if err != nil {
		// Generic message — do not reveal whether username or password was wrong
		clientError(w, http.StatusUnauthorized, "credenciales inválidas")
		return
	}

	if !u.Active {
		clientError(w, http.StatusUnauthorized, "credenciales inválidas")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)); err != nil {
		clientError(w, http.StatusUnauthorized, "credenciales inválidas")
		return
	}

	token, err := authjwt.Sign(h.jwtSecret, u.ID, u.Username, u.Role, u.MustChangePassword)
	if err != nil {
		slog.Error("login sign jwt", "err", err)
		clientError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"token":                token,
		"user_id":              u.ID,
		"username":             u.Username,
		"name":                 u.Name,
		"role":                 u.Role,
		"must_change_password": u.MustChangePassword,
	})
}

// ChangePassword handles POST /api/auth/change-password
func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		clientError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req user.ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		clientError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.CurrentPassword == "" || req.NewPassword == "" {
		clientError(w, http.StatusBadRequest, "current_password and new_password are required")
		return
	}
	if len(req.NewPassword) < 8 {
		clientError(w, http.StatusBadRequest, "new password must be at least 8 characters")
		return
	}

	u, err := h.userStore.GetByID(r.Context(), claims.UserID)
	if err != nil {
		clientError(w, http.StatusNotFound, "user not found")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.CurrentPassword)); err != nil {
		clientError(w, http.StatusUnauthorized, "current password is incorrect")
		return
	}

	if err := h.userStore.UpdatePassword(r.Context(), u.ID, req.NewPassword); err != nil {
		slog.Error("change password", "err", err)
		clientError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "password updated"})
}
