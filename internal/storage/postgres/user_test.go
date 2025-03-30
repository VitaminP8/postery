package postgres

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserPostgresStorage_RegisterUser(t *testing.T) {
	storage := NewUserPostgresStorage()

	t.Run("Successful user registration", func(t *testing.T) {
		// Настраиваем тестовую БД
		oldDB := setupTestDB(t)
		defer teardownTestDB(oldDB)

		username := "testuser"
		email := "test@example.com"
		password := "password123"

		user, err := storage.RegisterUser(username, email, password)
		require.NoError(t, err)
		assert.NotEmpty(t, user.ID)
		assert.Equal(t, username, user.Username)
		assert.Equal(t, email, user.Email)
	})

	t.Run("Register user with duplicate username", func(t *testing.T) {
		// Настраиваем тестовую БД
		oldDB := setupTestDB(t)
		defer teardownTestDB(oldDB)

		username := "duplicateuser"
		email := "duplicate@example.com"
		password := "password123"

		// Первая регистрация должна быть успешной
		_, err := storage.RegisterUser(username, email, password)
		require.NoError(t, err)

		// Вторая регистрация с тем же именем пользователя должна вернуть ошибку
		_, err = storage.RegisterUser(username, "another@example.com", "anotherpassword")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})
}

func TestUserPostgresStorage_LoginUser(t *testing.T) {
	storage := NewUserPostgresStorage()

	// Устанавливаем переменную окружения JWT_SECRET перед тестами
	originalJWTSecret := os.Getenv("JWT_SECRET")
	err := os.Setenv("JWT_SECRET", "test_secret_key_for_jwt")
	require.NoError(t, err)

	// Восстанавливаем оригинальное значение после тестов
	defer os.Setenv("JWT_SECRET", originalJWTSecret)

	t.Run("Successful login", func(t *testing.T) {
		// Настраиваем тестовую БД
		oldDB := setupTestDB(t)
		defer teardownTestDB(oldDB)

		username := "loginuser"
		email := "login@example.com"
		password := "loginpassword123"

		_, err = storage.RegisterUser(username, email, password)
		require.NoError(t, err)

		token, err := storage.LoginUser(username, password)
		require.NoError(t, err)
		assert.NotEmpty(t, token)

		// Простая проверка, что это похоже на JWT токен
		// JWT токен должен содержать две точки, разделяющие три части
		assert.Contains(t, token, ".")
		parts := 0
		for _, char := range token {
			if char == '.' {
				parts++
			}
		}
		assert.Equal(t, 2, parts, "JWT token должен состоять из трех частей, разделенных двумя точками")
	})

	t.Run("Login with incorrect password", func(t *testing.T) {
		// Настраиваем тестовую БД
		oldDB := setupTestDB(t)
		defer teardownTestDB(oldDB)

		username := "wrongpassuser"
		email := "wrongpass@example.com"
		password := "correctpassword123"

		_, err = storage.RegisterUser(username, email, password)
		require.NoError(t, err)

		_, err := storage.LoginUser(username, "wrongpassword")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid password")
	})

	t.Run("Login with non-existent user", func(t *testing.T) {
		// Настраиваем тестовую БД
		oldDB := setupTestDB(t)
		defer teardownTestDB(oldDB)

		_, err := storage.LoginUser("nonexistentuser", "anypassword")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestUserPostgresStorage_ErrorCases(t *testing.T) {
	storage := NewUserPostgresStorage()

	t.Run("Login without JWT_SECRET set", func(t *testing.T) {
		// Настраиваем тестовую БД
		oldDB := setupTestDB(t)
		defer teardownTestDB(oldDB)

		// Сохраняем текущее значение JWT_SECRET и сбрасываем его
		originalJWTSecret := os.Getenv("JWT_SECRET")
		os.Unsetenv("JWT_SECRET")
		defer os.Setenv("JWT_SECRET", originalJWTSecret)

		username := "jwt_secret_test"
		email := "jwt_secret@example.com"
		password := "password123"

		_, err := storage.RegisterUser(username, email, password)
		require.NoError(t, err)

		// Пытаемся войти без установленного JWT_SECRET
		_, err = storage.LoginUser(username, password)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "JWT_SECRET is not set")
	})
}
