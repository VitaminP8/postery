package memory

import (
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/VitaminP8/postery/graph/model"
	"github.com/VitaminP8/postery/internal/config"

	"github.com/golang-jwt/jwt"
	"golang.org/x/crypto/bcrypt"
)

type UserMemoryStorage struct {
	mu        sync.Mutex
	users     map[string]*model.User
	passwords map[string]string
	nextId    int
}

func NewUserMemoryStorage() *UserMemoryStorage {
	return &UserMemoryStorage{
		users:     make(map[string]*model.User),
		passwords: make(map[string]string),
		nextId:    1,
	}
}

func (s *UserMemoryStorage) RegisterUser(username, email, password string) (*model.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, exists := s.users[username]
	if exists {
		return nil, fmt.Errorf("user %s already exists", username)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	id := strconv.Itoa(s.nextId)
	s.nextId++

	user := &model.User{
		ID:       id,
		Username: username,
		Email:    email,
	}

	s.users[username] = user
	s.passwords[username] = string(hashedPassword)

	return user, nil
}

func (s *UserMemoryStorage) LoginUser(username, password string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, exists := s.users[username]
	if !exists {
		return "", fmt.Errorf("user %s not found", username)
	}

	hashedPassword, ok := s.passwords[username]
	if !ok {
		return "", fmt.Errorf("password for user %s not found", username)
	}

	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil {
		return "", fmt.Errorf("password for user %s is incorrect", username)
	}

	// достаем из .env jwtSecret
	jwtSecret := config.GetEnv("JWT_SECRET")
	if jwtSecret == "" {
		return "", errors.New("JWT_SECRET is not set in environment")
	}

	userIDInt, err := strconv.Atoi(user.ID)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  userIDInt,
		"username": user.Username,
		"exp":      time.Now().Add(72 * time.Hour).Unix(),
	})

	tokenString, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}
