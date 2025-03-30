package postgres

import (
	"fmt"
	"log"

	"github.com/VitaminP8/postery/internal/config"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/joho/godotenv"
)

var DB *gorm.DB

// GetDB возвращает глобальную переменную DB (для тестирования)
func GetDB() *gorm.DB {
	return DB
}

// InitDB подключается к базе данных PostgreSQL и устанавливает глобальную переменную DB
func InitDB() error {
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
	}

	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		config.GetEnv("DB_HOST"),
		config.GetEnv("DB_USER"),
		config.GetEnv("DB_PASSWORD"),
		config.GetEnv("DB_NAME"),
		config.GetEnv("DB_PORT"),
		config.GetEnv("DB_SSLMODE"),
	)

	db, err := gorm.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("failed to connect to the database: %v", err)
	}

	DB = db
	log.Println("Successfully connected to the database.")
	return nil
}

// CloseDB закрывает соединение с базой данных
func CloseDB() error {
	if DB == nil {
		return nil
	}

	err := DB.Close()
	if err != nil {
		return fmt.Errorf("failed to close the database connection: %v", err)
	}

	log.Println("Database connection closed.")
	return nil
}

// InitDBWithConnection для тестирования (позволяет инъекцию соединения БД)
func InitDBWithConnection(db *gorm.DB) {
	DB = db
}
