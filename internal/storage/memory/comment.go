package memory

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/VitaminP8/postery/graph/model"
	"github.com/VitaminP8/postery/internal/auth"
	"github.com/VitaminP8/postery/internal/post"
	"github.com/VitaminP8/postery/internal/subscription"
)

type CommentMemoryStorage struct {
	mu          sync.Mutex
	comments    map[string]*model.Comment
	nextID      int              // Для хранения актуального ID (можно было использовать UUID)
	postStorage post.PostStorage // Хранилище постов (внедрение зависимости (DI))
	manager     subscription.Manager
}

func NewCommentMemoryStorage(postStore post.PostStorage, manager subscription.Manager) *CommentMemoryStorage {
	return &CommentMemoryStorage{
		comments:    make(map[string]*model.Comment),
		nextID:      1,
		postStorage: postStore,
		manager:     manager,
	}
}

func (s *CommentMemoryStorage) CreateComment(ctx context.Context, postID, parentID, content string) (*model.Comment, error) {
	if len(content) > 2000 || len(content) == 0 {
		return nil, fmt.Errorf("content is too long or empty")
	}

	userID, err := auth.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unautorized: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	curPost, err := s.postStorage.GetPostById(postID)
	if err != nil {
		return nil, fmt.Errorf("post with ID %s not found", postID)
	}

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
		parentComment.HasReplies = true
	}

	comment := &model.Comment{
		ID:         id,
		PostID:     postID,
		ParentID:   parentPtr,
		Content:    content,
		AuthorID:   fmt.Sprint(userID),
		CreatedAt:  time.Now().Format(time.RFC3339),
		HasReplies: false,
		Children:   []*model.Comment{},
	}

	// в случае, если комментарий вложенный - добавляем его в Children родительского комментария
	if parentPtr != nil {
		parent, ok := s.comments[*parentPtr]
		if ok {
			parent.Children = append(parent.Children, comment)
		}
	}

	s.comments[id] = comment

	if s.manager != nil {
		s.manager.Publish(postID, comment)
	}

	return comment, nil
}

func (s *CommentMemoryStorage) GetComments(postID string, limit, offset int) (*model.CommentConnection, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	curPost, err := s.postStorage.GetPostById(postID)
	if err != nil {
		return nil, fmt.Errorf("post with ID %s not found", postID)
	}
	if curPost.CommentsDisabled {
		return &model.CommentConnection{
			Items:      []*model.Comment{},
			HasMore:    false,
			NextOffset: offset,
		}, nil
	}

	// Получаем только корневые комментарии
	var roots []*model.Comment
	for _, comment := range s.comments {
		if comment.PostID == postID && comment.ParentID == nil {
			roots = append(roots, comment)
		}
	}

	// Сортируем по CreatedAt (по возрастанию) (и по ID в случае одинаково времени создания)
	sort.Slice(roots, func(i, j int) bool {
		if roots[i].CreatedAt == roots[j].CreatedAt {
			return roots[i].ID < roots[j].ID // Дополнительная сортировка по ID
		}
		return roots[i].CreatedAt < roots[j].CreatedAt
	})

	// Пагинация по корневым
	if offset >= len(roots) {
		return &model.CommentConnection{
			Items:      []*model.Comment{},
			HasMore:    false,
			NextOffset: offset,
		}, nil
	}

	end := offset + limit
	if end > len(roots) {
		end = len(roots)
	}
	items := roots[offset:end]

	// узнаем, останутся ли комментарии после limit
	hasMore := end < len(roots)

	return &model.CommentConnection{
		Items:      items,
		HasMore:    hasMore,
		NextOffset: offset + limit,
	}, nil
}

func (s *CommentMemoryStorage) GetReplies(parentID string, limit, offset int) (*model.CommentConnection, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	parent, ok := s.comments[parentID]
	if !ok {
		return nil, fmt.Errorf("parent comment with ID %s not found", parentID)
	}

	children := parent.Children

	// Сортируем по CreatedAt (по возрастанию) (и по ID в случае одинаково времени создания)
	sort.Slice(children, func(i, j int) bool {
		if children[i].CreatedAt == children[j].CreatedAt {
			return children[i].ID < children[j].ID // Дополнительная сортировка по ID
		}
		return children[i].CreatedAt < children[j].CreatedAt
	})

	if offset >= len(children) {
		return &model.CommentConnection{
			Items:      []*model.Comment{},
			HasMore:    false,
			NextOffset: offset,
		}, nil
	}

	end := offset + limit
	if end > len(children) {
		end = len(children)
	}
	items := children[offset:end]

	// узнаем, останутся ли комментарии после limit
	hasMore := end < len(children)

	return &model.CommentConnection{
		Items:      items,
		HasMore:    hasMore,
		NextOffset: offset + limit,
	}, nil
}
