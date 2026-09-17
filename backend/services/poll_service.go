package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"live-polling-backend/models"
	"live-polling-backend/realtime"
	"live-polling-backend/repositories"
)

var (
	ErrPollNotFound     = repositories.ErrPollNotFound
	ErrUnauthorizedPoll = errors.New("you do not have permission to modify this poll")
	ErrInvalidOptions   = errors.New("poll must have between 2 and 10 unique options")
	ErrPollClosed       = errors.New("this poll is closed for voting")
)

type PollService interface {
	CreatePoll(ctx context.Context, ownerID string, req *models.CreatePollRequest) (*models.Poll, error)
	GetPollByID(ctx context.Context, id string) (*models.Poll, error)
	ListActivePolls(ctx context.Context, limit, offset int64) ([]models.Poll, error)
	ListMyPolls(ctx context.Context, ownerID string) ([]models.Poll, error)
	UpdatePoll(ctx context.Context, pollID string, ownerID string, req *models.UpdatePollRequest) (*models.Poll, error)
	DeletePoll(ctx context.Context, pollID string, ownerID string) error
	ClosePoll(ctx context.Context, pollID string, ownerID string) (*models.Poll, error)
}

type pollService struct {
	pollRepo       repositories.PollRepository
	eventPublisher realtime.EventPublisher
}

func NewPollService(pollRepo repositories.PollRepository, publisher ...realtime.EventPublisher) PollService {
	var pub realtime.EventPublisher
	if len(publisher) > 0 {
		pub = publisher[0]
	}
	return &pollService{
		pollRepo:       pollRepo,
		eventPublisher: pub,
	}
}

func (s *pollService) CreatePoll(ctx context.Context, ownerID string, req *models.CreatePollRequest) (*models.Poll, error) {
	if strings.TrimSpace(ownerID) == "" {
		return nil, errors.New("owner_id is required")
	}

	// Validate options length and uniqueness
	cleanedOptions := make([]models.PollOption, 0, len(req.Options))
	seen := make(map[string]bool)

	for _, opt := range req.Options {
		trimmed := strings.TrimSpace(opt)
		if trimmed == "" {
			continue
		}
		lower := strings.ToLower(trimmed)
		if seen[lower] {
			return nil, fmt.Errorf("%w: duplicate option '%s'", ErrInvalidOptions, trimmed)
		}
		seen[lower] = true
		cleanedOptions = append(cleanedOptions, models.PollOption{
			ID:        bson.NewObjectID().Hex(),
			Text:      trimmed,
			VoteCount: 0,
		})
	}

	if len(cleanedOptions) < 2 || len(cleanedOptions) > 10 {
		return nil, ErrInvalidOptions
	}

	poll := &models.Poll{
		OwnerID:     ownerID,
		Title:       strings.TrimSpace(req.Title),
		Description: strings.TrimSpace(req.Description),
		Status:      "active",
		Options:     cleanedOptions,
		TotalVotes:  0,
	}

	if err := s.pollRepo.Create(ctx, poll); err != nil {
		return nil, fmt.Errorf("failed to create poll: %w", err)
	}

	return poll, nil
}

func (s *pollService) GetPollByID(ctx context.Context, id string) (*models.Poll, error) {
	return s.pollRepo.FindByID(ctx, id)
}

func (s *pollService) ListActivePolls(ctx context.Context, limit, offset int64) ([]models.Poll, error) {
	return s.pollRepo.FindAllActive(ctx, limit, offset)
}

func (s *pollService) ListMyPolls(ctx context.Context, ownerID string) ([]models.Poll, error) {
	return s.pollRepo.FindByOwnerID(ctx, ownerID)
}

func (s *pollService) UpdatePoll(ctx context.Context, pollID string, ownerID string, req *models.UpdatePollRequest) (*models.Poll, error) {
	existing, err := s.pollRepo.FindByID(ctx, pollID)
	if err != nil {
		return nil, err
	}

	// Ownership Authorization Check: Only creator may modify
	if existing.OwnerID != ownerID {
		return nil, ErrUnauthorizedPoll
	}

	updateDoc := bson.M{}
	if req.Title != nil && strings.TrimSpace(*req.Title) != "" {
		updateDoc["title"] = strings.TrimSpace(*req.Title)
	}
	if req.Description != nil {
		updateDoc["description"] = strings.TrimSpace(*req.Description)
	}
	if req.Status != nil && (*req.Status == "active" || *req.Status == "closed") {
		updateDoc["status"] = *req.Status
	}

	if len(updateDoc) == 0 {
		return existing, nil
	}

	if err := s.pollRepo.Update(ctx, pollID, updateDoc); err != nil {
		return nil, fmt.Errorf("failed to update poll: %w", err)
	}

	return s.pollRepo.FindByID(ctx, pollID)
}

func (s *pollService) DeletePoll(ctx context.Context, pollID string, ownerID string) error {
	existing, err := s.pollRepo.FindByID(ctx, pollID)
	if err != nil {
		return err
	}

	// Ownership Authorization Check: Only creator may delete
	if existing.OwnerID != ownerID {
		return ErrUnauthorizedPoll
	}

	return s.pollRepo.Delete(ctx, pollID)
}

func (s *pollService) ClosePoll(ctx context.Context, pollID string, ownerID string) (*models.Poll, error) {
	statusClosed := "closed"
	updatedPoll, err := s.UpdatePoll(ctx, pollID, ownerID, &models.UpdatePollRequest{
		Status: &statusClosed,
	})
	if err != nil {
		return nil, err
	}

	if s.eventPublisher != nil {
		counts := make(map[string]int64, len(updatedPoll.Options))
		for _, opt := range updatedPoll.Options {
			counts[opt.ID] = opt.VoteCount
		}
		payload := &models.RealtimeVotePayload{
			Event:        models.EventPollStatus,
			PollID:       pollID,
			OptionCounts: counts,
			TotalVotes:   updatedPoll.TotalVotes,
			Status:       "closed",
			Timestamp:    time.Now().UTC(),
		}
		if pubErr := s.eventPublisher.PublishPollStatus(ctx, payload); pubErr != nil {
			log.Printf("[Realtime] Warning: failed to publish POLL_STATUS event to Redis: %v", pubErr)
		}
	}

	return updatedPoll, nil
}
