package postgres

import (
	"context"
	"fmt"
	"strconv"

	"github.com/VitaminP8/postery/graph/model"
	"github.com/VitaminP8/postery/internal/auth"
	"github.com/VitaminP8/postery/models"
)

type CommentPostgresStorage struct{}

func NewCommentPostgresStorage() *CommentPostgresStorage {
	return &CommentPostgresStorage{}
}

func (s *CommentPostgresStorage) CreateComment(ctx context.Context, postID, parentID, content string) (*model.Comment, error) {
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
		PostID:  postIDUint,
		UserID:  userID,
		Content: content,
	}

	if parentID != "" {
		parentInt, err := strconv.Atoi(parentID)
		if err != nil {
			return nil, fmt.Errorf("invalid parent ID: %w", err)
		}
		parentUint := uint(parentInt)
		comment.ParentID = &parentUint
	}

	err = DB.Create(comment).Error
	if err != nil {
		return nil, fmt.Errorf("could not create comment: %w", err)
	}

	result := &model.Comment{
		ID:       fmt.Sprint(comment.ID),
		PostID:   fmt.Sprint(comment.PostID),
		Content:  comment.Content,
		AuthorID: fmt.Sprint(comment.UserID),
		Children: []*model.Comment{},
	}
	if comment.ParentID != nil {
		pid := fmt.Sprint(*comment.ParentID)
		result.ParentID = &pid
	}

	return result, nil
}

func (s *CommentPostgresStorage) GetComments(postID string, limit, offset int) ([]*model.Comment, error) {
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
		return []*model.Comment{}, nil
	}

	var rootComments []models.Comment
	err = DB.Where("post_id = ? AND parent_id IS NULL", postIDUint).
		Limit(limit).
		Offset(offset).
		Find(&rootComments).Error
	if err != nil {
		return nil, fmt.Errorf("could not get root comments:  %w", err)
	}

	var results []*model.Comment
	for _, root := range rootComments {
		c := &model.Comment{
			ID:       fmt.Sprint(root.ID),
			PostID:   fmt.Sprint(root.PostID),
			Content:  root.Content,
			AuthorID: fmt.Sprint(root.UserID),
			Children: []*model.Comment{},
		}
		fetchChildren(&root, c)
		results = append(results, c)
	}

	return results, nil
}

func fetchChildren(dbParent *models.Comment, gqlParent *model.Comment) {
	var children []models.Comment
	DB.Where("parent_id = ?", dbParent.ID).Find(&children)

	for _, child := range children {
		mc := &model.Comment{
			ID:       fmt.Sprint(child.ID),
			PostID:   fmt.Sprint(child.PostID),
			Content:  child.Content,
			AuthorID: fmt.Sprint(child.UserID),
			Children: []*model.Comment{},
		}
		pid := fmt.Sprint(child.ParentID)
		mc.ParentID = &pid

		gqlParent.Children = append(gqlParent.Children, mc)
		fetchChildren(&child, mc)
	}
}
