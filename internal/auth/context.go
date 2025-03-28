// internal/auth/context.go
package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v4"
)

type contextKey string

const userIDKey = contextKey("userID")

// Сохраняет userID в контексте
func WithUserID(ctx context.Context, userID uint) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// Достает userID из контекста
func GetUserIDFromContext(ctx context.Context) (uint, error) {
	val := ctx.Value(userIDKey)
	id, ok := val.(uint)
	if !ok {
		return 0, errors.New("user ID not found in context")
	}
	return id, nil
}

// Для извлечения userID из JWT и помещения в context
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenStr := extractTokenFromHeader(r.Header.Get("Authorization"))
		if tokenStr == "" {
			next.ServeHTTP(w, r) // неавторизованный доступ — пропускаем
			return
		}

		secret := os.Getenv("JWT_SECRET")
		if secret == "" {
			http.Error(w, "JWT secret not set", http.StatusInternalServerError)
			return
		}

		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(secret), nil
		})
		if err != nil || !token.Valid {
			next.ServeHTTP(w, r) // если невалидный токен — пропускаем
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			next.ServeHTTP(w, r)
			return
		}

		idFloat, ok := claims["user_id"].(float64)
		if !ok {
			next.ServeHTTP(w, r)
			return
		}

		userID := uint(idFloat)
		ctx := WithUserID(r.Context(), userID)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

func extractTokenFromHeader(header string) string {
	parts := strings.Split(header, " ")
	if len(parts) == 2 && parts[0] == "Bearer" {
		return parts[1]
	}
	return ""
}
