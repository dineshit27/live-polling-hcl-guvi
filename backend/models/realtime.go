package models

import (
	"time"
)

type RealtimeEventType string

const (
	EventVoteCast      RealtimeEventType = "VOTE_CAST"
	EventPollStatus    RealtimeEventType = "POLL_STATUS"
	EventPollSnapshot  RealtimeEventType = "POLL_SNAPSHOT"
)

type RealtimeVotePayload struct {
	Event        RealtimeEventType `json:"event"`
	PollID       string            `json:"poll_id"`
	OptionID     string            `json:"option_id,omitempty"`
	OptionCounts map[string]int64  `json:"option_counts"`
	TotalVotes   int64             `json:"total_votes"`
	Status       string            `json:"status"`
	Timestamp    time.Time         `json:"timestamp"`
}
