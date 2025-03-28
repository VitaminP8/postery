package post

import (
	"context"

	"github.com/VitaminP8/postery/graph/model"
)

type PostStorage interface {
	CreatePost(ctx context.Context, title, content string) (*model.Post, error)
	GetPostById(id string) (*model.Post, error)
	GetAllPosts() ([]*model.Post, error)
	DisableComment(ctx context.Context, id string) error
	EnableComment(ctx context.Context, id string) error
	DeletePostById(ctx context.Context, id string) error
}
