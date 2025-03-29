package subscription

import "github.com/VitaminP8/postery/graph/model"

type Manager interface {
	Subscribe(postID string) (<-chan *model.Comment, func())
	Publish(postID string, comment *model.Comment)
}
