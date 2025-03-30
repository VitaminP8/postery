package mocks

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/VitaminP8/postery/graph/model"
	"github.com/VitaminP8/postery/internal/auth"
	"github.com/VitaminP8/postery/internal/subscription"
)

type MockCommentStorage struct {
	mu        sync.Mutex
	comments  map[string]*model.Comment
	postIDs   map[string][]string // postID -> список ID комментариев
	parentIDs map[string][]string // parentID -> список ID дочерних комментариев
	nextID    int
	manager   subscription.Manager // Для уведомлений о новых комментариях
}

func NewMockCommentStorage(manager subscription.Manager) *MockCommentStorage {
	return &MockCommentStorage{
		comments:  make(map[string]*model.Comment),
		postIDs:   make(map[string][]string),
		parentIDs: make(map[string][]string),
		nextID:    1,
		manager:   manager,
	}
}

func (m *MockCommentStorage) CreateComment(ctx context.Context, postID, parentID, content string) (*model.Comment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	userID, err := auth.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unautorized: %w", err)
	}

	if len(content) > 2000 || len(content) == 0 {
		return nil, errors.New("content is too long or empty")
	}

	commentID := strconv.Itoa(m.nextID)
	m.nextID++

	var parentIDPtr *string
	if parentID != "" {
		if _, exists := m.comments[parentID]; !exists {
			return nil, errors.New("parent comment not found")
		}
		parentIDPtr = &parentID

		m.comments[parentID].HasReplies = true

		m.parentIDs[parentID] = append(m.parentIDs[parentID], commentID)
	}

	comment := &model.Comment{
		ID:         commentID,
		PostID:     postID,
		ParentID:   parentIDPtr,
		Content:    content,
		AuthorID:   fmt.Sprint(userID),
		CreatedAt:  time.Now().Format(time.RFC3339),
		HasReplies: false,
		Children:   []*model.Comment{},
	}

	m.comments[commentID] = comment
	m.postIDs[postID] = append(m.postIDs[postID], commentID)

	if m.manager != nil {
		m.manager.Publish(postID, comment)
	}

	return comment, nil
}

func (m *MockCommentStorage) GetComments(postID string, limit, offset int) (*model.CommentConnection, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	commentIDs, exists := m.postIDs[postID]
	if !exists {
		return &model.CommentConnection{
			Items:      []*model.Comment{},
			HasMore:    false,
			NextOffset: offset,
		}, nil
	}

	var rootComments []*model.Comment
	for _, id := range commentIDs {
		comment := m.comments[id]
		if comment.ParentID == nil {
			rootComments = append(rootComments, comment)
		}
	}

	sort.Slice(rootComments, func(i, j int) bool {
		return rootComments[i].CreatedAt < rootComments[j].CreatedAt
	})

	totalCount := len(rootComments)
	hasMore := offset+limit < totalCount

	var items []*model.Comment
	if offset < totalCount {
		end := offset + limit
		if end > totalCount {
			end = totalCount
		}
		items = rootComments[offset:end]
	}

	return &model.CommentConnection{
		Items:      items,
		HasMore:    hasMore,
		NextOffset: offset + limit,
	}, nil
}

func (m *MockCommentStorage) GetReplies(parentID string, limit, offset int) (*model.CommentConnection, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.comments[parentID]; !exists {
		return nil, errors.New("parent comment not found")
	}

	childIDs, exists := m.parentIDs[parentID]
	if !exists {
		return &model.CommentConnection{
			Items:      []*model.Comment{},
			HasMore:    false,
			NextOffset: offset,
		}, nil
	}

	var childComments []*model.Comment
	for _, id := range childIDs {
		childComments = append(childComments, m.comments[id])
	}

	sort.Slice(childComments, func(i, j int) bool {
		return childComments[i].CreatedAt < childComments[j].CreatedAt
	})

	totalCount := len(childComments)
	hasMore := offset+limit < totalCount

	var items []*model.Comment
	if offset < totalCount {
		end := offset + limit
		if end > totalCount {
			end = totalCount
		}
		items = childComments[offset:end]
	}

	return &model.CommentConnection{
		Items:      items,
		HasMore:    hasMore,
		NextOffset: offset + limit,
	}, nil
}
