package post

import (
	"context"

	"github.com/VitaminP8/postery/graph/model"
)

type PostStorage interface {
	CreatePost(ctx context.Context, title, content string) (*model.Post, error)
	GetPostById(id string) (*model.Post, error)
	GetAllPosts() ([]*model.Post, error)
	DisableComment(id string) error
	EnableComment(id string) error
	DeletePostById(id string) error
}
