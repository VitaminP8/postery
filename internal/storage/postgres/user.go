package postgres

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/VitaminP8/postery/graph/model"
	"github.com/VitaminP8/postery/models"
	"github.com/golang-jwt/jwt"

	"golang.org/x/crypto/bcrypt"
)

type UserPostgresStorage struct{}

func NewUserPostgresStorage() *UserPostgresStorage {
	return &UserPostgresStorage{}
}

func (s *UserPostgresStorage) RegisterUser(username, email, password string) (*model.User, error) {
	// проверка - существует ли такой пользователь
	var existUser models.User
	err := DB.Where("username = ?", username).First(&existUser).Error
	if err == nil {
		return nil, fmt.Errorf("user with username %s already exists", username)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &models.User{
		Username: username,
		Email:    email,
		Password: string(hashedPassword),
	}

	err = DB.Create(user).Error
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return &model.User{
		ID:       fmt.Sprint(user.ID),
		Username: user.Username,
		Email:    user.Email,
	}, nil
}

func (s *UserPostgresStorage) LoginUser(username, password string) (string, error) {
	// проверка - существует ли такой пользователь
	var user models.User
	err := DB.Where("username = ?", username).First(&user).Error
	if err != nil {
		return "", fmt.Errorf("user with username %s not found", username)
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return "", fmt.Errorf("invalid password or username: %w", err)
	}

	// достаем из .env jwtSecret
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return "", errors.New("JWT_SECRET is not set in environment")
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"exp":      time.Now().Add(72 * time.Hour).Unix(),
	})

	tokenString, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}
