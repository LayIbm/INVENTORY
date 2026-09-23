package authjwt

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const tokenDuration = 8 * time.Hour

// Claims holds the custom JWT payload.
type Claims struct {
	UserID             string `json:"user_id"`
	Username           string `json:"username"`
	Role               string `json:"role"`
	MustChangePassword bool   `json:"must_change_password"`
	jwt.RegisteredClaims
}

// Sign creates a signed JWT string for the given user data.
func Sign(secret, userID, username, role string, mustChange bool) (string, error) {
	claims := Claims{
		UserID:             userID,
		Username:           username,
		Role:               role,
		MustChangePassword: mustChange,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(tokenDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// Verify parses and validates a JWT string, returning the claims.
func Verify(secret, tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}
