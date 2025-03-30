package mocks

import (
	"errors"
	"strconv"
	"sync"

	"github.com/VitaminP8/postery/graph/model"
)

type MockUserStorage struct {
	mu        sync.Mutex
	users     map[string]*model.User // username -> user
	emails    map[string]string      // email -> username
	passwords map[string]string      // username -> password
	nextID    int
}

func NewMockUserStorage() *MockUserStorage {
	return &MockUserStorage{
		users:     make(map[string]*model.User),
		emails:    make(map[string]string),
		passwords: make(map[string]string),
		nextID:    1,
	}
}

func (m *MockUserStorage) RegisterUser(username, email, password string) (*model.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.users[username]; exists {
		return nil, errors.New("user with username " + username + " already exists")
	}

	if existingUsername, exists := m.emails[email]; exists {
		return nil, errors.New("email " + email + " already registered to user " + existingUsername)
	}

	id := m.nextID
	m.nextID++

	user := &model.User{
		ID:       strconv.Itoa(id),
		Username: username,
		Email:    email,
	}

	m.users[username] = user
	m.emails[email] = username
	m.passwords[username] = password

	return user, nil
}

func (m *MockUserStorage) LoginUser(username, password string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	user, exists := m.users[username]
	if !exists {
		return "", errors.New("user with username " + username + " not found")
	}

	storedPassword, exists := m.passwords[username]
	if !exists || storedPassword != password {
		return "", errors.New("invalid password or username")
	}

	token := "jwt-token-for-user-" + user.ID

	return token, nil
}

func (m *MockUserStorage) GetUserByUsername(username string) (*model.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	user, exists := m.users[username]
	if !exists {
		return nil, errors.New("user not found")
	}

	return user, nil
}

func (m *MockUserStorage) GetUserByID(id string) (*model.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, user := range m.users {
		if user.ID == id {
			return user, nil
		}
	}

	return nil, errors.New("user not found")
}
