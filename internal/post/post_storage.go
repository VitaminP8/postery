package post

import "github.com/VitaminP8/postery/graph/model"

type PostStorage interface {
	CreatePost(title, content string) (*model.Post, error)
	GetPostById(id string) (*model.Post, error)
	GetAllPosts() ([]*model.Post, error)
	DisableComment(id string) error
	EnableComment(id string) error
	DeletePostById(id string) error
}
