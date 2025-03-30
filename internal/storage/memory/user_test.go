package memory

import (
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserMemoryStorage_RegisterUser(t *testing.T) {
	storage := NewUserMemoryStorage()

	t.Run("Successful user registration", func(t *testing.T) {
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

func TestUserMemoryStorage_LoginUser(t *testing.T) {
	storage := NewUserMemoryStorage()

	// Устанавливаем переменную окружения JWT_SECRET перед тестами
	originalJWTSecret := os.Getenv("JWT_SECRET")
	err := os.Setenv("JWT_SECRET", "test_secret_key_for_jwt")
	require.NoError(t, err)

	// Восстанавливаем оригинальное значение после тестов
	defer os.Setenv("JWT_SECRET", originalJWTSecret)

	// Регистрируем пользователя для тестирования входа
	username := "loginuser"
	email := "login@example.com"
	password := "loginpassword123"

	_, err = storage.RegisterUser(username, email, password)
	require.NoError(t, err)

	t.Run("Successful login", func(t *testing.T) {
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
		_, err := storage.LoginUser(username, "wrongpassword")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "incorrect")
	})

	t.Run("Login with non-existent user", func(t *testing.T) {
		_, err := storage.LoginUser("nonexistentuser", "anypassword")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestUserMemoryStorage_ConcurrentOperations(t *testing.T) {
	storage := NewUserMemoryStorage()

	originalJWTSecret := os.Getenv("JWT_SECRET")
	err := os.Setenv("JWT_SECRET", "test_secret_key_for_jwt")
	require.NoError(t, err)

	defer os.Setenv("JWT_SECRET", originalJWTSecret)

	t.Run("Concurrent user registration", func(t *testing.T) {
		var wg sync.WaitGroup
		numGoroutines := 10

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()

				username := "concurrent_user_" + strconv.Itoa(idx)
				email := "concurrent" + strconv.Itoa(idx) + "@example.com"
				password := "pass" + strconv.Itoa(idx)

				user, err := storage.RegisterUser(username, email, password)

				assert.NoError(t, err)
				if err == nil {
					assert.Equal(t, username, user.Username)
					assert.Equal(t, email, user.Email)
				}
			}(i)
		}

		wg.Wait()
	})

	t.Run("Concurrent login attempts", func(t *testing.T) {
		// Регистрируем одного пользователя для множественных попыток входа
		username := "login_test_user"
		email := "login_test@example.com"
		password := "login_test_password"

		_, err := storage.RegisterUser(username, email, password)
		require.NoError(t, err)

		var wg sync.WaitGroup
		numGoroutines := 5

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				token, err := storage.LoginUser(username, password)
				assert.NoError(t, err)
				assert.NotEmpty(t, token)
			}()
		}

		wg.Wait()
	})
}

func TestUserMemoryStorage_MixedConcurrentOperations(t *testing.T) {
	storage := NewUserMemoryStorage()

	originalJWTSecret := os.Getenv("JWT_SECRET")
	err := os.Setenv("JWT_SECRET", "test_secret_key_for_jwt")
	require.NoError(t, err)

	defer os.Setenv("JWT_SECRET", originalJWTSecret)

	// Регистрируем некоторых пользователей заранее
	preregisteredUsers := 5
	for i := 0; i < preregisteredUsers; i++ {
		username := "preregistered_user_" + strconv.Itoa(i)
		email := "preregistered" + strconv.Itoa(i) + "@example.com"
		password := "preregistered_pass" + strconv.Itoa(i)

		_, err := storage.RegisterUser(username, email, password)
		require.NoError(t, err)
	}

	t.Run("Concurrent registrations and logins", func(t *testing.T) {
		var wg sync.WaitGroup

		// Количество пользователей, которые будут регистрироваться и входить параллельно
		numNewUsers := 10
		// Создаем канал для отслеживания завершенных регистраций
		registeredCh := make(chan string, numNewUsers)

		// Запускаем горутины для регистрации новых пользователей
		for i := 0; i < numNewUsers; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()

				username := "mixed_user_" + strconv.Itoa(idx)
				email := "mixed" + strconv.Itoa(idx) + "@example.com"
				password := "mixed_pass" + strconv.Itoa(idx)

				user, err := storage.RegisterUser(username, email, password)
				if err == nil {
					// Сообщаем об успешной регистрации
					registeredCh <- username
					assert.Equal(t, username, user.Username)
					assert.Equal(t, email, user.Email)
				} else {
					t.Logf("Registration failed for %s: %v", username, err)
				}
			}(i)
		}

		// Запускаем горутины для попыток входа предварительно зарегистрированных пользователей
		for i := 0; i < preregisteredUsers; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()

				username := "preregistered_user_" + strconv.Itoa(idx)
				password := "preregistered_pass" + strconv.Itoa(idx)

				token, err := storage.LoginUser(username, password)
				assert.NoError(t, err)
				assert.NotEmpty(t, token)
			}(i)
		}

		// Запускаем горутины, которые будут пытаться войти как только что зарегистрированные пользователи
		for i := 0; i < numNewUsers/2; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				// Ждем завершения регистрации какого-либо пользователя
				select {
				case username := <-registeredCh:
					password := "mixed_pass" + username[len("mixed_user_"):]
					// Пытаемся войти как этот пользователь
					token, err := storage.LoginUser(username, password)
					assert.NoError(t, err)
					assert.NotEmpty(t, token)
				case <-time.After(time.Second):
					// Тайм-аут, если ни одна регистрация не завершилась вовремя
					t.Log("Timeout waiting for registration")
				}
			}()
		}

		// Ожидаем завершения всех операций
		wg.Wait()
		close(registeredCh)
	})

	t.Run("Registration and login for the same username concurrently", func(t *testing.T) {
		var wg sync.WaitGroup
		username := "contested_user"
		password := "contested_password"

		// Запускаем регистрацию
		wg.Add(1)
		var registrationErr error
		var registrationDone bool
		go func() {
			defer wg.Done()
			_, registrationErr = storage.RegisterUser(username, "contested@example.com", password)
			registrationDone = true
		}()

		// Запускаем попытку входа (она должна либо дождаться регистрации, либо вернуть ошибку)
		wg.Add(1)
		var loginErr error
		var loginSuccess bool
		go func() {
			defer wg.Done()
			// Небольшая задержка, чтобы увеличить шансы на доступ
			time.Sleep(10 * time.Millisecond)
			token, err := storage.LoginUser(username, password)
			loginErr = err
			loginSuccess = (err == nil && token != "")
		}()

		wg.Wait()

		if registrationDone && registrationErr == nil {
			if loginSuccess {
				// Все ок, регистрация и вход прошли успешно
			} else {
				// Возможно, вход был слишком рано
				assert.Contains(t, loginErr.Error(), "not found")
			}
		}
	})
}
