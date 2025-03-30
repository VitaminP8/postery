package postgres

import (
	"testing"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetDB(t *testing.T) {
	// Сохраняем текущее значение DB
	originalDB := DB

	// Создаем тестовую БД
	testDB, err := gorm.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer testDB.Close()

	// Устанавливаем тестовую БД
	DB = testDB

	// Проверяем, что GetDB возвращает установленную БД
	result := GetDB()
	assert.Equal(t, DB, result)

	// Восстанавливаем исходное значение
	DB = originalDB
}

func TestInitDBWithConnection(t *testing.T) {
	// Сохраняем текущее значение DB
	originalDB := DB

	// Создаем тестовую БД
	testDB, err := gorm.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer testDB.Close()

	// Устанавливаем соединение через функцию
	InitDBWithConnection(testDB)

	// Проверяем, что глобальная DB теперь равна тестовой
	assert.Equal(t, testDB, DB)

	// Восстанавливаем исходное значение
	DB = originalDB
}

// Тест для проверки поведения CloseDB с NULL-базой данных
func TestCloseDBWithNilDB(t *testing.T) {
	// Сохраняем текущее значение DB
	originalDB := DB

	// Устанавливаем DB в nil
	DB = nil

	// Проверяем, что CloseDB не вызывает панику и возвращает nil
	err := CloseDB()
	assert.NoError(t, err)

	// Восстанавливаем исходное значение
	DB = originalDB
}

// Примечание: Тесты InitDB и CloseDB с реальным подключением не включены, так как они требуют настоящую PostgreSQL базу данных.
