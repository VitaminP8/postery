package graph

import (
	"github.com/VitaminP8/postery/internal/comment"
	"github.com/VitaminP8/postery/internal/post"
	"github.com/VitaminP8/postery/internal/subscription"
	"github.com/VitaminP8/postery/internal/user"
)

// Resolver служит корневой точкой для всех резолверов.
// Здесь можно внедрять зависимости, например хранилище.
type Resolver struct {
	PostStore           post.PostStorage
	CommentStore        comment.CommentStorage
	UserStore           user.UserStorage
	SubscriptionManager subscription.Manager
}
