package models

import (
	"time"
)

type PollOption struct {
	ID        string `json:"id" bson:"id"`
	Text      string `json:"text" bson:"text" binding:"required,min=1,max=200"`
	VoteCount int64  `json:"vote_count" bson:"vote_count"`
}

type Poll struct {
	ID          string       `json:"id" bson:"_id,omitempty"`
	OwnerID     string       `json:"owner_id" bson:"owner_id"`
	Title       string       `json:"title" bson:"title" binding:"required,min=5,max=200"`
	Description string       `json:"description" bson:"description"`
	Status      string       `json:"status" bson:"status"` // "active" | "closed"
	Options     []PollOption `json:"options" bson:"options"`
	TotalVotes  int64        `json:"total_votes" bson:"total_votes"`
	CreatedAt   time.Time    `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at" bson:"updated_at"`
}

type CreatePollRequest struct {
	Title       string   `json:"title" binding:"required,min=5,max=200"`
	Description string   `json:"description"`
	Options     []string `json:"options" binding:"required,min=2,max=10,dive,min=1,max=200"`
}

type UpdatePollRequest struct {
	Title       *string `json:"title,omitempty" binding:"omitempty,min=5,max=200"`
	Description *string `json:"description,omitempty"`
	Status      *string `json:"status,omitempty" binding:"omitempty,oneof=active closed"`
}
