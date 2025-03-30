package auth

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithUserIDAndGetUserIDFromContext(t *testing.T) {
	t.Run("Store and retrieve user ID from context", func(t *testing.T) {
		ctx := context.Background()

		userID := uint(123)
		ctx = WithUserID(ctx, userID)

		retrievedID, err := GetUserIDFromContext(ctx)
		assert.NoError(t, err)
		assert.Equal(t, userID, retrievedID)
	})

	t.Run("Error when user ID not in context", func(t *testing.T) {
		ctx := context.Background()

		_, err := GetUserIDFromContext(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found in context")
	})

	t.Run("Error when context value is not uint", func(t *testing.T) {
		// Создаем контекст с неправильным типом значения
		ctx := context.WithValue(context.Background(), userIDKey, "not-a-uint")

		_, err := GetUserIDFromContext(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found in context")
	})
}

func TestExtractTokenFromHeader(t *testing.T) {
	t.Run("Valid Bearer token", func(t *testing.T) {
		header := "Bearer token123"
		token := extractTokenFromHeader(header)
		assert.Equal(t, "token123", token)
	})

	t.Run("Invalid format - no Bearer prefix", func(t *testing.T) {
		header := "NotBearer token123"
		token := extractTokenFromHeader(header)
		assert.Equal(t, "", token)
	})

	t.Run("Invalid format - no space", func(t *testing.T) {
		header := "Bearertoken123"
		token := extractTokenFromHeader(header)
		assert.Equal(t, "", token)
	})

	t.Run("Empty header", func(t *testing.T) {
		header := ""
		token := extractTokenFromHeader(header)
		assert.Equal(t, "", token)
	})
}

func TestAuthMiddleware(t *testing.T) {
	// Создаем тестовый обработчик, который будет проверять наличие userID в контексте
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, err := GetUserIDFromContext(r.Context())
		if err == nil {
			fmt.Fprintf(w, "User ID: %d", userID)
		} else {
			fmt.Fprint(w, "No user ID in context")
		}
	})

	// Создаем middleware с нашим тестовым обработчиком
	handler := AuthMiddleware(testHandler)

	// Сохраняем текущее значение JWT_SECRET
	originalSecret := os.Getenv("JWT_SECRET")

	// Устанавливаем тестовый секрет для JWT
	testSecret := "test_jwt_secret"
	os.Setenv("JWT_SECRET", testSecret)
	defer os.Setenv("JWT_SECRET", originalSecret)

	t.Run("Valid token", func(t *testing.T) {
		// Создаем валидный JWT токен
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"user_id":  float64(123),
			"username": "testuser",
			"exp":      time.Now().Add(time.Hour).Unix(),
		})

		tokenString, err := token.SignedString([]byte(testSecret))
		require.NoError(t, err)

		// Создаем тестовый запрос с токеном
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Authorization", "Bearer "+tokenString)

		// Создаем recorder для получения ответа
		w := httptest.NewRecorder()

		// Вызываем обработчик
		handler.ServeHTTP(w, req)

		// Проверяем, что userID был установлен в контексте
		assert.Equal(t, "User ID: 123", w.Body.String())
	})

	t.Run("Invalid token signature", func(t *testing.T) {
		// Создаем токен, подписанный другим секретом
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"user_id":  float64(123),
			"username": "testuser",
			"exp":      time.Now().Add(time.Hour).Unix(),
		})

		tokenString, err := token.SignedString([]byte("wrong_secret"))
		require.NoError(t, err)

		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Authorization", "Bearer "+tokenString)

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		assert.Equal(t, "No user ID in context", w.Body.String())
	})

	t.Run("Expired token", func(t *testing.T) {
		// Создаем просроченный токен
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"user_id":  float64(123),
			"username": "testuser",
			"exp":      time.Now().Add(-time.Hour).Unix(),
		})

		tokenString, err := token.SignedString([]byte(testSecret))
		require.NoError(t, err)

		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Authorization", "Bearer "+tokenString)

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		assert.Equal(t, "No user ID in context", w.Body.String())
	})

	t.Run("No token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		assert.Equal(t, "No user ID in context", w.Body.String())
	})

	t.Run("Invalid token format", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Authorization", "InvalidFormat")

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		assert.Equal(t, "No user ID in context", w.Body.String())
	})

	t.Run("No JWT_SECRET", func(t *testing.T) {
		// Временно убираем JWT_SECRET из окружения
		os.Unsetenv("JWT_SECRET")
		defer os.Setenv("JWT_SECRET", testSecret)

		// Создаем валидный JWT токен
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"user_id":  float64(123),
			"username": "testuser",
			"exp":      time.Now().Add(time.Hour).Unix(),
		})

		tokenString, err := token.SignedString([]byte(testSecret))
		require.NoError(t, err)

		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Authorization", "Bearer "+tokenString)

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		// Проверяем статус код 500
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "JWT secret not set")
	})
}
