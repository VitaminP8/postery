package postgres

import (
	"context"
	"fmt"

	"github.com/VitaminP8/postery/graph/model"
	"github.com/VitaminP8/postery/internal/auth"
	"github.com/VitaminP8/postery/models"
)

type PostPostgresStorage struct{}

func NewPostPostgresStorage() *PostPostgresStorage {
	return &PostPostgresStorage{}
}

func (s *PostPostgresStorage) CreatePost(ctx context.Context, title, content string) (*model.Post, error) {
	userID, err := auth.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not get user id from context:  %w", err)
	}

	post := &models.Post{
		Title:            title,
		Content:          content,
		UserID:           userID,
		CommentsDisabled: false,
	}

	err = DB.Create(post).Error
	if err != nil {
		return nil, fmt.Errorf("could not create post: %w", err)
	}

	return &model.Post{
		ID:               fmt.Sprint(post.ID),
		Title:            post.Title,
		Content:          post.Content,
		AuthorID:         fmt.Sprint(post.UserID),
		CommentsDisabled: post.CommentsDisabled,
	}, nil
}

func (s *PostPostgresStorage) GetPostById(id string) (*model.Post, error) {
	var post models.Post
	err := DB.First(&post, id).Error
	if err != nil {
		return nil, fmt.Errorf("could not get post by id: %w", err)
	}

	return &model.Post{
		ID:               fmt.Sprint(post.ID),
		Title:            post.Title,
		Content:          post.Content,
		AuthorID:         fmt.Sprint(post.UserID),
		CommentsDisabled: post.CommentsDisabled,
	}, nil
}

func (s *PostPostgresStorage) GetAllPosts() ([]*model.Post, error) {
	var posts []models.Post
	err := DB.Find(&posts).Error
	if err != nil {
		return nil, fmt.Errorf("could not get posts: %w", err)
	}

	var results []*model.Post
	for _, post := range posts {
		results = append(results, &model.Post{
			ID:               fmt.Sprint(post.ID),
			Title:            post.Title,
			Content:          post.Content,
			AuthorID:         fmt.Sprint(post.UserID),
			CommentsDisabled: post.CommentsDisabled,
		})
	}

	return results, nil
}

func (s *PostPostgresStorage) DisableComment(ctx context.Context, id string) error {
	userID, err := auth.GetUserIDFromContext(ctx)
	if err != nil {
		return fmt.Errorf("unauthorized: %w", err)
	}

	var post models.Post
	err = DB.First(&post, id).Error
	if err != nil {
		return fmt.Errorf("post not found: %w", err)
	}

	if post.UserID != userID {
		return fmt.Errorf("forbidden: you are not the author of this post")
	}

	err = DB.Model(&models.Post{}).Where("id = ?", id).Update("comments_disabled", true).Error
	if err != nil {
		return fmt.Errorf("could not disable comment: %w", err)
	}

	return nil
}

func (s *PostPostgresStorage) EnableComment(ctx context.Context, id string) error {
	userID, err := auth.GetUserIDFromContext(ctx)
	if err != nil {
		return fmt.Errorf("unauthorized: %w", err)
	}

	var post models.Post
	err = DB.First(&post, id).Error
	if err != nil {
		return fmt.Errorf("post not found: %w", err)
	}

	if post.UserID != userID {
		return fmt.Errorf("forbidden: you are not the author of this post")
	}

	err = DB.Model(&models.Post{}).Where("id = ?", id).Update("comments_disabled", false).Error
	if err != nil {
		return fmt.Errorf("could not enable comment: %w", err)
	}

	return nil
}

func (s *PostPostgresStorage) DeletePostById(ctx context.Context, id string) error {
	userID, err := auth.GetUserIDFromContext(ctx)
	if err != nil {
		return fmt.Errorf("unauthorized: %w", err)
	}

	var post models.Post
	err = DB.First(&post, id).Error
	if err != nil {
		return fmt.Errorf("post not found: %w", err)
	}

	if post.UserID != userID {
		return fmt.Errorf("forbidden: you are not the author of this post")
	}

	err = DB.Delete(&models.Post{}, id).Error
	if err != nil {
		return fmt.Errorf("could not delete post: %w", err)
	}

	return nil
}
