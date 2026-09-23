package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/layssagonzalez/device-inventory/backend/internal/authjwt"
)

type contextKey string

const ClaimsKey contextKey = "claims"

// Auth validates the Bearer token and injects claims into the request context.
func Auth(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			tokenStr := strings.TrimPrefix(header, "Bearer ")
			claims, err := authjwt.Verify(jwtSecret, tokenStr)
			if err != nil {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), ClaimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ClaimsFromContext retrieves claims from context, returns nil if absent.
func ClaimsFromContext(ctx context.Context) *authjwt.Claims {
	c, _ := ctx.Value(ClaimsKey).(*authjwt.Claims)
	return c
}
