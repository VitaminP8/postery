package memory

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"

	"github.com/VitaminP8/postery/graph/model"
	"github.com/VitaminP8/postery/internal/auth"
)

type PostMemoryStorage struct {
	mu     sync.Mutex
	posts  map[string]*model.Post
	nextId int // Для хранения актуального ID (можно было использовать UUID)
}

func NewPostMemoryStorage() *PostMemoryStorage {
	return &PostMemoryStorage{
		posts:  make(map[string]*model.Post),
		nextId: 1,
	}
}

func (s *PostMemoryStorage) CreatePost(ctx context.Context, title, content string) (*model.Post, error) {
	// Контекст — это read-only структура (при каждом запросе он не обновляется, а создается заново)(поэтому над мьютексом)
	userID, err := auth.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unautorized: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	id := strconv.Itoa(s.nextId)
	s.nextId++

	post := &model.Post{
		ID:               id,
		Title:            title,
		Content:          content,
		AuthorID:         fmt.Sprint(userID),
		CommentsDisabled: false,
	}

	s.posts[id] = post
	return post, nil
}

func (s *PostMemoryStorage) GetPostById(id string) (*model.Post, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	post, exists := s.posts[id]
	if !exists {
		return nil, errors.New("post not found")
	}

	return post, nil
}

func (s *PostMemoryStorage) GetAllPosts() ([]*model.Post, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var posts []*model.Post
	for _, post := range s.posts {
		posts = append(posts, post)
	}

	return posts, nil
}

func (s *PostMemoryStorage) DisableComment(ctx context.Context, id string) error {
	userID, err := auth.GetUserIDFromContext(ctx)
	if err != nil {
		return fmt.Errorf("unautorized: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	post, exists := s.posts[id]
	if !exists {
		return errors.New("post not found")
	}

	if post.AuthorID != fmt.Sprint(userID) {
		return errors.New("forbidden: not author")
	}

	post.CommentsDisabled = true
	return nil
}

func (s *PostMemoryStorage) EnableComment(ctx context.Context, id string) error {
	userID, err := auth.GetUserIDFromContext(ctx)
	if err != nil {
		return fmt.Errorf("unautorized: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	post, exists := s.posts[id]
	if !exists {
		return errors.New("post not found")
	}

	if post.AuthorID != fmt.Sprint(userID) {
		return errors.New("forbidden: not author")
	}

	post.CommentsDisabled = false
	return nil
}

func (s *PostMemoryStorage) DeletePostById(ctx context.Context, id string) error {
	userID, err := auth.GetUserIDFromContext(ctx)
	if err != nil {
		return fmt.Errorf("unautorized: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	post, exists := s.posts[id]
	if !exists {
		return errors.New("post not found")
	}

	if post.AuthorID != fmt.Sprint(userID) {
		return errors.New("forbidden: not author")
	}

	delete(s.posts, id)
	return nil
}
