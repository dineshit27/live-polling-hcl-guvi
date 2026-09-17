package models

import (
	"time"
)

type Vote struct {
	ID              string    `json:"id" bson:"_id,omitempty"`
	PollID          string    `json:"poll_id" bson:"poll_id"`
	OptionID        string    `json:"option_id" bson:"option_id"`
	VoterIdentifier string    `json:"voter_identifier" bson:"voter_identifier"`
	IPHash          string    `json:"-" bson:"ip_hash"`
	CreatedAt       time.Time `json:"created_at" bson:"created_at"`
}

type VoteRequest struct {
	OptionID        string `json:"option_id" binding:"required"`
	VoterIdentifier string `json:"voter_identifier" binding:"omitempty,max=128"`
}

type VoteResponse struct {
	Success bool `json:"success"`
	Poll    Poll `json:"poll"`
}

type VoteStatusResponse struct {
	HasVoted      bool   `json:"has_voted"`
	VotedOptionID string `json:"voted_option_id,omitempty"`
}
