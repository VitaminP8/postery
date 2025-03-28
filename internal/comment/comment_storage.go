package comment

import (
	"context"

	"github.com/VitaminP8/postery/graph/model"
)

type CommentStorage interface {
	CreateComment(ctx context.Context, postID, parentID, content string) (*model.Comment, error)
	GetComments(postID string, limit, offset int) ([]*model.Comment, error)
}
