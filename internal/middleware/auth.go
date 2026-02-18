package middleware

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"contract-service/internal/model"
)

// JWTAuth validates a signed JWT in the Authorization: Bearer header.
// The token must be signed with HS256 using the provided secret.
// No specific claims are enforced beyond signature and expiry — extend
// the Claims struct below if you need sub, iss, aud, etc.
func JWTAuth(secret string) func(http.Handler) http.Handler {
	keyFunc := func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(secret), nil
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw, ok := extractBearer(r)
			if !ok {
				writeError(w, http.StatusUnauthorized, "missing or malformed Authorization header")
				return
			}

			_, err := jwt.Parse(raw, keyFunc,
				jwt.WithValidMethods([]string{"HS256"}),
				jwt.WithExpirationRequired(),
			)
			if err != nil {
				status, msg := jwtErrorResponse(err)
				writeError(w, status, msg)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// extractBearer pulls the token string out of "Authorization: Bearer <token>".
func extractBearer(r *http.Request) (string, bool) {
	h := r.Header.Get("Authorization")
	if h == "" {
		return "", false
	}
	parts := strings.SplitN(h, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return "", false
	}
	return parts[1], true
}

// jwtErrorResponse maps jwt library errors to HTTP status codes and messages.
func jwtErrorResponse(err error) (int, string) {
	switch {
	case errors.Is(err, jwt.ErrTokenExpired):
		return http.StatusUnauthorized, "token expired"
	case errors.Is(err, jwt.ErrTokenNotValidYet):
		return http.StatusUnauthorized, "token not yet valid"
	case errors.Is(err, jwt.ErrSignatureInvalid):
		return http.StatusForbidden, "invalid token signature"
	default:
		return http.StatusUnauthorized, "invalid token: " + err.Error()
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(model.ErrorResponse{Error: msg}) //nolint:errcheck
}
