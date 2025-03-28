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

// InitDB подключается к базе данных PostgreSQL
func InitDB() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
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
	DB, err = gorm.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("failed to connect to the database: %v", err)
	}
	fmt.Println("Successfully connected to the database.")
}

// CloseDB закрывает соединение с базой данных
func CloseDB() {
	err := DB.Close()
	if err != nil {
		log.Fatalf("failed to close the database connection: %v", err)
	}
	fmt.Println("Database connection closed.")
}
