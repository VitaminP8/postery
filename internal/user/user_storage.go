package user

import (
	"github.com/VitaminP8/postery/models"
)

type UserStorage interface {
	RegisterUser(username, email, password string) (*models.User, error)
	LoginUser(email, password string) (string, error) // JWT
}
