package postgres

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/VitaminP8/postery/graph/model"
	"github.com/VitaminP8/postery/internal/auth"
	"github.com/VitaminP8/postery/internal/subscription"
	"github.com/VitaminP8/postery/models"
)

type CommentPostgresStorage struct {
	manager subscription.Manager
}

func NewCommentPostgresStorage(manager subscription.Manager) *CommentPostgresStorage {
	return &CommentPostgresStorage{
		manager: manager,
	}
}

func (s *CommentPostgresStorage) CreateComment(ctx context.Context, postID, parentID, content string) (*model.Comment, error) {
	if len(content) > 2000 || len(content) == 0 {
		return nil, fmt.Errorf("content is too long or empty")
	}

	userID, err := auth.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unautorized: %w", err)
	}

	postIDint, err := strconv.Atoi(postID)
	if err != nil {
		return nil, fmt.Errorf("invalid post ID: %w", err)
	}
	postIDUint := uint(postIDint)

	var post models.Post
	err = DB.First(&post, postIDUint).Error
	if err != nil {
		return nil, fmt.Errorf("post not found: %w", err)
	}

	if post.CommentsDisabled {
		return nil, fmt.Errorf("comments are disabled for this post")
	}

	comment := &models.Comment{
		PostID:     postIDUint,
		UserID:     userID,
		Content:    content,
		HasReplies: false,
	}

	if parentID != "" {
		parentInt, err := strconv.Atoi(parentID)
		if err != nil {
			return nil, fmt.Errorf("invalid parent ID: %w", err)
		}
		parentUint := uint(parentInt)
		comment.ParentID = &parentUint

		DB.Model(&models.Comment{}).Where("id = ?", parentUint).Update("has_replies", true)
	}

	err = DB.Create(comment).Error
	if err != nil {
		return nil, fmt.Errorf("could not create comment: %w", err)
	}

	var parentStr *string
	if comment.ParentID != nil {
		pid := fmt.Sprint(*comment.ParentID)
		parentStr = &pid
	}
	result := &model.Comment{
		ID:         fmt.Sprint(comment.ID),
		PostID:     fmt.Sprint(comment.PostID),
		Content:    comment.Content,
		AuthorID:   fmt.Sprint(comment.UserID),
		ParentID:   parentStr,
		CreatedAt:  comment.CreatedAt.Format(time.RFC3339),
		HasReplies: comment.HasReplies,
		Children:   []*model.Comment{},
	}

	if s.manager != nil {
		s.manager.Publish(postID, result)
	}

	return result, nil
}

func (s *CommentPostgresStorage) GetComments(postID string, limit, offset int) (*model.CommentConnection, error) {
	postIDUint, err := strconv.Atoi(postID)
	if err != nil {
		return nil, fmt.Errorf("invalid post ID: %w", err)
	}

	// Проверяем, если комментарии отключены — ничего не возвращаем
	var post models.Post
	err = DB.First(&post, postIDUint).Error
	if err != nil {
		return nil, fmt.Errorf("could not get post: %w", err)
	}
	if post.CommentsDisabled {
		return &model.CommentConnection{
			Items:      []*model.Comment{},
			HasMore:    false,
			NextOffset: offset,
		}, nil
	}

	var rootComments []models.Comment
	err = DB.Where("post_id = ? AND parent_id IS NULL", postIDUint).
		Order("created_at").
		Limit(limit + 1). // Загружаем +1 чтобы проверить hasMore
		Offset(offset).
		Find(&rootComments).Error
	if err != nil {
		return nil, fmt.Errorf("could not get root comments:  %w", err)
	}

	// узнаем, останутся ли комментарии после limit
	hasMore := len(rootComments) > limit
	if hasMore {
		rootComments = rootComments[:limit]
	}

	var results []*model.Comment
	for _, root := range rootComments {
		c := &model.Comment{
			ID:         fmt.Sprint(root.ID),
			PostID:     fmt.Sprint(root.PostID),
			Content:    root.Content,
			AuthorID:   fmt.Sprint(root.UserID),
			HasReplies: root.HasReplies,
			CreatedAt:  root.CreatedAt.Format(time.RFC3339),
			Children:   []*model.Comment{},
		}
		results = append(results, c)
	}

	return &model.CommentConnection{
		Items:      results,
		HasMore:    hasMore,
		NextOffset: offset + limit,
	}, nil
}

func (s *CommentPostgresStorage) GetReplies(parentID string, limit, offset int) (*model.CommentConnection, error) {
	parentUint, err := strconv.Atoi(parentID)
	if err != nil {
		return nil, fmt.Errorf("invalid parent ID: %w", err)
	}

	var replies []models.Comment
	err = DB.Where("parent_id = ?", parentUint).
		Order("created_at").
		Limit(limit + 1).
		Offset(offset).
		Find(&replies).Error
	if err != nil {
		return nil, fmt.Errorf("could not get replies: %w", err)
	}

	// узнаем, останутся ли комментарии после limit
	hasMore := len(replies) > limit
	if hasMore {
		replies = replies[:limit]
	}

	var results []*model.Comment
	for _, r := range replies {
		pid := fmt.Sprint(*r.ParentID)
		curRep := &model.Comment{
			ID:         fmt.Sprint(r.ID),
			PostID:     fmt.Sprint(r.PostID),
			ParentID:   &pid,
			Content:    r.Content,
			AuthorID:   fmt.Sprint(r.UserID),
			HasReplies: r.HasReplies,
			CreatedAt:  r.CreatedAt.Format(time.RFC3339),
			Children:   []*model.Comment{},
		}
		results = append(results, curRep)
	}

	return &model.CommentConnection{
		Items:      results,
		HasMore:    hasMore,
		NextOffset: offset + limit,
	}, nil
}
