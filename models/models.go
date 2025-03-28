package models

import "github.com/jinzhu/gorm"

type User struct {
	gorm.Model
	Username string `gorm:"unique"`
	Email    string `gorm:"unique"`
	Password string
	Posts    []Post    `gorm:"foreignkey:UserID"`
	Comments []Comment `gorm:"foreignkey:UserID"`
}

type Post struct {
	gorm.Model
	Title            string
	Content          string
	CommentsDisabled bool
	UserID           uint
	Comments         []Comment `gorm:"foreignkey:PostID"`
}

type Comment struct {
	gorm.Model
	Content  string
	PostID   uint
	UserID   uint
	ParentID *uint
	Children []Comment `gorm:"foreignkey:ParentID"`
}
