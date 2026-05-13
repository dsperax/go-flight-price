package auth

import (
	"net/http"
	"strings"

	"github.com/sperax/flight-price-service/internal/httpx"
)

func Middleware(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				httpx.Unauthorized(w, "authorization token is required")
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				httpx.Unauthorized(w, "authorization header format must be: Bearer <token>")
				return
			}

			if err := ValidateToken(parts[1], jwtSecret); err != nil {
				httpx.Unauthorized(w, "invalid or expired token")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
