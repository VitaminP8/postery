package memory

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/VitaminP8/postery/graph/model"
	"github.com/VitaminP8/postery/internal/post"
)

type CommentMemoryStorage struct {
	mu          sync.Mutex
	comments    map[string]*model.Comment
	nextID      int              // Для хранения актуального ID (можно было использовать UUID)
	postStorage post.PostStorage // Хранилище постов (внедрение зависимости (DI))
}

func NewCommentMemoryStorage(postStore post.PostStorage) *CommentMemoryStorage {
	return &CommentMemoryStorage{
		comments:    make(map[string]*model.Comment),
		nextID:      1,
		postStorage: postStore,
	}
}

func (s *CommentMemoryStorage) CreateComment(postID, parentID, content string) (*model.Comment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Проверка - существует ли пост
	curPost, err := s.postStorage.GetPostById(postID)
	if err != nil {
		return nil, fmt.Errorf("post with ID %s not found", postID)
	}

	// Проверка - включены ли комментарии
	if curPost.CommentsDisabled {
		return nil, fmt.Errorf("comments are disabled for post %s", postID)
	}

	id := strconv.Itoa(s.nextID)
	s.nextID++

	var parentPtr *string
	if parentID != "" {
		parentPtr = &parentID

		// проверяем что родительский комментарий существует и принадлежит тому же посту
		parentComment, ok := s.comments[parentID]
		if !ok {
			return nil, fmt.Errorf("parent comment with ID %s not found", parentID)
		}
		if parentComment.PostID != postID {
			return nil, fmt.Errorf("parent comment belongs to a different post")
		}
	}

	comment := &model.Comment{
		ID:       id,
		PostID:   postID,
		ParentID: parentPtr,
		Content:  content,
		Children: []*model.Comment{},
	}

	// в случае, если комментарий вложенный - добавляем его в Children родительского комментария
	if parentPtr != nil {
		parent, ok := s.comments[*parentPtr]
		if ok {
			parent.Children = append(parent.Children, comment)
		}
	}

	s.comments[id] = comment
	return comment, nil
}

func (s *CommentMemoryStorage) GetComments(postID string, limit, offset int) ([]*model.Comment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Проверяем, существует ли пост
	post, err := s.postStorage.GetPostById(postID)
	if err != nil {
		return nil, fmt.Errorf("post with ID %s not found", postID)
	}

	// Проверяем, разрешены ли комментарии
	if post.CommentsDisabled {
		return []*model.Comment{}, nil // просто возвращаем пустой список
	}

	var filtered []*model.Comment
	for _, comment := range s.comments {
		if comment.PostID == postID {
			filtered = append(filtered, comment)
		}
	}

	if offset >= len(filtered) {
		return []*model.Comment{}, nil
	}

	finish := offset + limit
	if finish > len(filtered) {
		finish = len(filtered)
	}

	result := filtered[offset:finish]
	return result, nil
}
