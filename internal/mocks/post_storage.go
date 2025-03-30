package mocks

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"github.com/VitaminP8/postery/graph/model"
	"github.com/VitaminP8/postery/internal/auth"
)

type MockPostStorage struct {
	posts map[string]*model.Post
	mu    sync.Mutex
}

func NewMockPostStorage() *MockPostStorage {
	return &MockPostStorage{
		posts: make(map[string]*model.Post),
	}
}

func (m *MockPostStorage) CreatePost(ctx context.Context, title, content string) (*model.Post, error) {
	userID, err := auth.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	id := strconv.Itoa(len(m.posts) + 1)
	post := &model.Post{
		ID:               id,
		Title:            title,
		Content:          content,
		AuthorID:         strconv.Itoa(int(userID)),
		CommentsDisabled: false,
	}
	m.posts[id] = post
	return post, nil
}

func (m *MockPostStorage) GetPostById(id string) (*model.Post, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	post, ok := m.posts[id]
	if !ok {
		return nil, fmt.Errorf("post not found")
	}
	return post, nil
}

func (m *MockPostStorage) GetAllPosts() ([]*model.Post, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	posts := make([]*model.Post, 0, len(m.posts))
	for _, post := range m.posts {
		posts = append(posts, post)
	}
	return posts, nil
}

func (m *MockPostStorage) DisableComment(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	post, ok := m.posts[id]
	if !ok {
		return fmt.Errorf("post not found")
	}
	post.CommentsDisabled = true
	return nil
}

func (m *MockPostStorage) EnableComment(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	post, ok := m.posts[id]
	if !ok {
		return fmt.Errorf("post not found")
	}
	post.CommentsDisabled = false
	return nil
}

func (m *MockPostStorage) DeletePostById(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.posts[id]; !ok {
		return fmt.Errorf("post not found")
	}
	delete(m.posts, id)
	return nil
}
