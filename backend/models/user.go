package models

import (
	"time"
)

type User struct {
	ID           string    `json:"id" bson:"_id,omitempty"`
	Name         string    `json:"name" bson:"name" binding:"required"`
	Email        string    `json:"email" bson:"email" binding:"required,email"`
	PasswordHash string    `json:"-" bson:"password_hash"`
	CreatedAt    time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" bson:"updated_at"`
}

type UserSignupRequest struct {
	Name     string `json:"name" binding:"required,min=2,max=100"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8,max=100"`
}

type UserLoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}
