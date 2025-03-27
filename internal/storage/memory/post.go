package memory

import (
	"errors"
	"strconv"
	"sync"

	"github.com/VitaminP8/postery/graph/model"
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

func (s *PostMemoryStorage) CreatePost(title, content string) (*model.Post, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := strconv.Itoa(s.nextId)
	s.nextId++

	post := &model.Post{
		ID:               id,
		Title:            title,
		Content:          content,
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

	posts := make([]*model.Post, 0, len(s.posts))
	for _, post := range s.posts {
		posts = append(posts, post)
	}

	return posts, nil
}

func (s *PostMemoryStorage) DisableComment(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	post, exists := s.posts[id]
	if !exists {
		return errors.New("post not found")
	}

	post.CommentsDisabled = true
	return nil
}

func (s *PostMemoryStorage) EnableComment(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	post, exists := s.posts[id]
	if !exists {
		return errors.New("post not found")
	}

	post.CommentsDisabled = false
	return nil
}

func (s *PostMemoryStorage) DeletePostById(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, exists := s.posts[id]
	if !exists {
		return errors.New("post not found")
	}

	delete(s.posts, id)
	return nil
}
