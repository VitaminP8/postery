package user

import (
	"github.com/VitaminP8/postery/graph/model"
)

type UserStorage interface {
	RegisterUser(username, email, password string) (*model.User, error)
	LoginUser(username, password string) (string, error) // JWT
}
