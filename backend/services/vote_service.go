package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"live-polling-backend/models"
	"live-polling-backend/realtime"
	"live-polling-backend/repositories"
)

var (
	ErrAlreadyVoted   = repositories.ErrAlreadyVoted
	ErrOptionNotFound = errors.New("selected option does not exist on this poll")
)

type VoteService interface {
	CastVote(ctx context.Context, pollID string, req *models.VoteRequest, clientIP string, authUserID string) (*models.Poll, error)
	CheckVoteStatus(ctx context.Context, pollID string, voterIdentifier string, clientIP string, authUserID string) (*models.VoteStatusResponse, error)
}

type voteService struct {
	pollRepo       repositories.PollRepository
	voteRepo       repositories.VoteRepository
	eventPublisher realtime.EventPublisher
}

func NewVoteService(pollRepo repositories.PollRepository, voteRepo repositories.VoteRepository, publisher ...realtime.EventPublisher) VoteService {
	var pub realtime.EventPublisher
	if len(publisher) > 0 {
		pub = publisher[0]
	}
	return &voteService{
		pollRepo:       pollRepo,
		voteRepo:       voteRepo,
		eventPublisher: pub,
	}
}

func hashIP(ip string) string {
	cleaned := strings.TrimSpace(ip)
	if cleaned == "" {
		cleaned = "unknown_ip"
	}
	hash := sha256.Sum256([]byte(cleaned))
	return hex.EncodeToString(hash[:])
}

func determineVoterIdentifier(reqIdentifier string, clientIP string, authUserID string) string {
	if authUserID != "" {
		return "user:" + authUserID
	}
	trimmed := strings.TrimSpace(reqIdentifier)
	if trimmed != "" {
		return "fp:" + trimmed
	}
	return "ip:" + hashIP(clientIP)
}

func (s *voteService) CastVote(ctx context.Context, pollID string, req *models.VoteRequest, clientIP string, authUserID string) (*models.Poll, error) {
	// 1. Fetch poll to verify existence and active status
	poll, err := s.pollRepo.FindByID(ctx, pollID)
	if err != nil {
		return nil, err
	}

	// 2. Enforce closed poll constraint
	if poll.Status != "active" {
		return nil, ErrPollClosed
	}

	// 3. Verify option existence within this poll
	optionFound := false
	for _, opt := range poll.Options {
		if opt.ID == req.OptionID {
			optionFound = true
			break
		}
	}
	if !optionFound {
		return nil, ErrOptionNotFound
	}

	// 4. Determine voter identity & IP hash for deduplication
	voterID := determineVoterIdentifier(req.VoterIdentifier, clientIP, authUserID)
	ipHash := hashIP(clientIP)

	// Pre-check deduplication
	hasVoted, err := s.voteRepo.HasVoted(ctx, pollID, voterID)
	if err != nil {
		return nil, fmt.Errorf("failed to check voter history: %w", err)
	}
	if hasVoted {
		return nil, ErrAlreadyVoted
	}

	// 5. Create vote record (database-enforced compound unique index prevents race conditions)
	vote := &models.Vote{
		PollID:          pollID,
		OptionID:        req.OptionID,
		VoterIdentifier: voterID,
		IPHash:          ipHash,
	}

	if err := s.voteRepo.CastVote(ctx, vote); err != nil {
		return nil, err
	}

	// 6. Atomically increment the option vote count and the total poll votes
	if err := s.pollRepo.IncrementOptionVote(ctx, pollID, req.OptionID); err != nil {
		// Compensating rollback: delete inserted vote record to prevent orphaned voter lock-out
		_ = s.voteRepo.DeleteVote(ctx, vote.ID)
		return nil, fmt.Errorf("failed to increment vote count: %w", err)
	}

	// 7. Return latest updated poll with verified counts (MongoDB is persistent source of truth)
	updatedPoll, err := s.pollRepo.FindByID(ctx, pollID)
	if err != nil {
		return nil, err
	}

	// 8. Publish Realtime VOTE_CAST event to Redis Pub/Sub layer
	if s.eventPublisher != nil {
		counts := make(map[string]int64, len(updatedPoll.Options))
		for _, opt := range updatedPoll.Options {
			counts[opt.ID] = opt.VoteCount
		}
		payload := &models.RealtimeVotePayload{
			Event:        models.EventVoteCast,
			PollID:       pollID,
			OptionID:     req.OptionID,
			OptionCounts: counts,
			TotalVotes:   updatedPoll.TotalVotes,
			Status:       updatedPoll.Status,
			Timestamp:    time.Now().UTC(),
		}
		if pubErr := s.eventPublisher.PublishVoteCast(ctx, payload); pubErr != nil {
			log.Printf("[Realtime] Warning: failed to publish VOTE_CAST event to Redis: %v", pubErr)
			// Non-blocking: MongoDB write succeeded, so vote is persistently recorded.
		}
	}

	return updatedPoll, nil
}

func (s *voteService) CheckVoteStatus(ctx context.Context, pollID string, voterIdentifier string, clientIP string, authUserID string) (*models.VoteStatusResponse, error) {
	voterID := determineVoterIdentifier(voterIdentifier, clientIP, authUserID)

	vote, err := s.voteRepo.FindUserVote(ctx, pollID, voterID)
	if err != nil {
		return nil, err
	}

	if vote == nil {
		return &models.VoteStatusResponse{
			HasVoted: false,
		}, nil
	}

	return &models.VoteStatusResponse{
		HasVoted:      true,
		VotedOptionID: vote.OptionID,
	}, nil
}
