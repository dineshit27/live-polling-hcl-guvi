package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"live-polling-backend/config"
	"live-polling-backend/handlers"
	"live-polling-backend/models"
	"live-polling-backend/repositories"
	"live-polling-backend/routes"
	"live-polling-backend/services"
)

type voteTestEnv struct {
	client      *repositories.MongoClient
	router      http.Handler
	ownerToken  string
	ownerID     string
	voterToken  string
	voterID     string
	authService services.AuthService
	pollService services.PollService
	voteService services.VoteService
}

func setupVoteTestEnv(t *testing.T) *voteTestEnv {
	client := setupTestMongoDB(t)
	if client == nil {
		t.Skip("MongoDB not available")
		return nil
	}

	cfg := &config.Config{
		Port:               "8081",
		GinMode:            "test",
		JWTSecret:          "test_jwt_secret_at_least_32_characters_long",
		CORSAllowedOrigins: []string{"*"},
	}

	userRepo := repositories.NewUserRepository(client.Database)
	pollRepo := repositories.NewPollRepository(client.Database)
	voteRepo := repositories.NewVoteRepository(client.Database)

	authService := services.NewAuthService(userRepo, cfg.JWTSecret)
	authHandler := handlers.NewAuthHandler(authService)

	pollService := services.NewPollService(pollRepo)
	pollHandler := handlers.NewPollHandler(pollService)

	voteService := services.NewVoteService(pollRepo, voteRepo)
	voteHandler := handlers.NewVoteHandler(voteService)

	healthHandler := handlers.NewHealthHandler(client.Client, nil)

	router := routes.SetupRouter(&routes.RouterDeps{
		Config:        cfg,
		HealthHandler: healthHandler,
		AuthHandler:   authHandler,
		AuthService:   authService,
		PollHandler:   pollHandler,
		VoteHandler:   voteHandler,
	})

	ctx := context.Background()

	// Register Owner
	ownerSignup, err := authService.Signup(ctx, &models.UserSignupRequest{
		Name:     "Poll Creator",
		Email:    "vote_creator@example.com",
		Password: "creatorPassword123!",
	})
	if err != nil {
		t.Fatalf("Failed to create owner user: %v", err)
	}

	// Register Authenticated Voter
	voterSignup, err := authService.Signup(ctx, &models.UserSignupRequest{
		Name:     "Registered Voter",
		Email:    "voter_registered@example.com",
		Password: "voterPassword123!",
	})
	if err != nil {
		t.Fatalf("Failed to create voter user: %v", err)
	}

	return &voteTestEnv{
		client:      client,
		router:      router,
		ownerToken:  ownerSignup.Token,
		ownerID:     ownerSignup.User.ID,
		voterToken:  voterSignup.Token,
		voterID:     voterSignup.User.ID,
		authService: authService,
		pollService: pollService,
		voteService: voteService,
	}
}

func createTestPoll(t *testing.T, env *voteTestEnv) *models.Poll {
	ctx := context.Background()
	poll, err := env.pollService.CreatePoll(ctx, env.ownerID, &models.CreatePollRequest{
		Title:       "What is the best distributed architecture?",
		Description: "Comparing paradigms",
		Options:     []string{"Event Driven", "Actor Model", "Service Mesh"},
	})
	if err != nil {
		t.Fatalf("Failed to create test poll: %v", err)
	}
	return poll
}

func TestVote_SuccessfulAtomicVote(t *testing.T) {
	env := setupVoteTestEnv(t)
	if env == nil {
		return
	}
	defer env.client.Close(context.Background())

	poll := createTestPoll(t, env)
	targetOptionID := poll.Options[0].ID

	// 1. Initial vote-status check -> has_voted: false
	statusReq, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/polls/%s/vote-status?voter_identifier=fp_browser_1", poll.ID), nil)
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, statusReq)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 on initial vote-status, got %d: %s", w.Code, w.Body.String())
	}
	var initStatusResp struct {
		Data models.VoteStatusResponse `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &initStatusResp)
	if initStatusResp.Data.HasVoted {
		t.Fatalf("Expected has_voted to be false initially")
	}

	// 2. Anonymous client casts vote with fingerprint
	votePayload := map[string]interface{}{
		"option_id":        targetOptionID,
		"voter_identifier": "fp_browser_1",
	}
	body, _ := json.Marshal(votePayload)
	voteReq, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/polls/%s/vote", poll.ID), bytes.NewReader(body))
	voteReq.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	env.router.ServeHTTP(w, voteReq)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK on successful vote, got %d: %s", w.Code, w.Body.String())
	}

	var voteResp struct {
		Success bool        `json:"success"`
		Data    models.Poll `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &voteResp)

	if voteResp.Data.TotalVotes != 1 {
		t.Fatalf("Expected total_votes to be 1, got %d", voteResp.Data.TotalVotes)
	}

	foundOption := false
	for _, opt := range voteResp.Data.Options {
		if opt.ID == targetOptionID {
			foundOption = true
			if opt.VoteCount != 1 {
				t.Fatalf("Expected option vote_count 1, got %d", opt.VoteCount)
			}
		} else {
			if opt.VoteCount != 0 {
				t.Fatalf("Expected other options vote_count 0, got %d", opt.VoteCount)
			}
		}
	}
	if !foundOption {
		t.Fatalf("Target option was not found in response")
	}

	// 3. Re-check vote status -> has_voted: true, voted_option_id matches
	statusReq, _ = http.NewRequest(http.MethodGet, fmt.Sprintf("/api/polls/%s/vote-status?voter_identifier=fp_browser_1", poll.ID), nil)
	w = httptest.NewRecorder()
	env.router.ServeHTTP(w, statusReq)

	var afterStatusResp struct {
		Data models.VoteStatusResponse `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &afterStatusResp)
	if !afterStatusResp.Data.HasVoted {
		t.Fatalf("Expected has_voted to be true after voting")
	}
	if afterStatusResp.Data.VotedOptionID != targetOptionID {
		t.Fatalf("Expected voted_option_id '%s', got '%s'", targetOptionID, afterStatusResp.Data.VotedOptionID)
	}
}

func TestVote_DeduplicationConflict(t *testing.T) {
	env := setupVoteTestEnv(t)
	if env == nil {
		return
	}
	defer env.client.Close(context.Background())

	poll := createTestPoll(t, env)
	targetOptionID := poll.Options[0].ID

	// 1. Cast first vote
	votePayload := map[string]interface{}{
		"option_id":        targetOptionID,
		"voter_identifier": "fp_duplicate_check_99",
	}
	body, _ := json.Marshal(votePayload)
	voteReq, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/polls/%s/vote", poll.ID), bytes.NewReader(body))
	voteReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, voteReq)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 on first vote, got %d", w.Code)
	}

	// 2. Cast second vote with SAME fingerprint -> 409 Conflict
	secondPayload := map[string]interface{}{
		"option_id":        poll.Options[1].ID,
		"voter_identifier": "fp_duplicate_check_99",
	}
	secondBody, _ := json.Marshal(secondPayload)
	secondReq, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/polls/%s/vote", poll.ID), bytes.NewReader(secondBody))
	secondReq.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	env.router.ServeHTTP(w, secondReq)

	if w.Code != http.StatusConflict {
		t.Fatalf("Expected 409 Conflict on duplicate vote, got %d: %s", w.Code, w.Body.String())
	}

	// 3. Authenticated user cast first vote -> 200 OK
	authUserPayload := map[string]interface{}{
		"option_id": targetOptionID,
	}
	authBody, _ := json.Marshal(authUserPayload)
	authReq, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/polls/%s/vote", poll.ID), bytes.NewReader(authBody))
	authReq.Header.Set("Content-Type", "application/json")
	authReq.Header.Set("Authorization", "Bearer "+env.voterToken)
	w = httptest.NewRecorder()
	env.router.ServeHTTP(w, authReq)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 on authenticated user first vote, got %d: %s", w.Code, w.Body.String())
	}

	// 4. Authenticated user casts second vote -> 409 Conflict
	authReq2, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/polls/%s/vote", poll.ID), bytes.NewReader(authBody))
	authReq2.Header.Set("Content-Type", "application/json")
	authReq2.Header.Set("Authorization", "Bearer "+env.voterToken)
	w = httptest.NewRecorder()
	env.router.ServeHTTP(w, authReq2)

	if w.Code != http.StatusConflict {
		t.Fatalf("Expected 409 Conflict on authenticated duplicate vote, got %d: %s", w.Code, w.Body.String())
	}

	// Verify total votes is exactly 2 (1 from fp, 1 from auth user)
	updatedPoll, err := env.pollService.GetPollByID(context.Background(), poll.ID)
	if err != nil {
		t.Fatalf("Failed to fetch poll: %v", err)
	}
	if updatedPoll.TotalVotes != 2 {
		t.Fatalf("Expected exactly 2 total votes after deduplication, got %d", updatedPoll.TotalVotes)
	}
}

func TestVote_ClosedPollCheck(t *testing.T) {
	env := setupVoteTestEnv(t)
	if env == nil {
		return
	}
	defer env.client.Close(context.Background())

	poll := createTestPoll(t, env)
	targetOptionID := poll.Options[0].ID

	// Owner closes the poll
	_, err := env.pollService.ClosePoll(context.Background(), poll.ID, env.ownerID)
	if err != nil {
		t.Fatalf("Failed to close poll: %v", err)
	}

	// Attempt to vote on closed poll -> 400 Bad Request (poll_closed)
	votePayload := map[string]interface{}{
		"option_id":        targetOptionID,
		"voter_identifier": "fp_late_voter_1",
	}
	body, _ := json.Marshal(votePayload)
	voteReq, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/polls/%s/vote", poll.ID), bytes.NewReader(body))
	voteReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, voteReq)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected 400 Bad Request on closed poll vote, got %d: %s", w.Code, w.Body.String())
	}

	var errResp handlers.ErrorResponse
	_ = json.Unmarshal(w.Body.Bytes(), &errResp)
	if errResp.Error != "poll_closed" {
		t.Fatalf("Expected error code 'poll_closed', got '%s'", errResp.Error)
	}
}

func TestVote_InvalidOptionAndPollNotFound(t *testing.T) {
	env := setupVoteTestEnv(t)
	if env == nil {
		return
	}
	defer env.client.Close(context.Background())

	poll := createTestPoll(t, env)

	// 1. Invalid option ID -> 400 Bad Request
	votePayload := map[string]interface{}{
		"option_id":        "non_existent_option_id_12345",
		"voter_identifier": "fp_voter_invalid_opt",
	}
	body, _ := json.Marshal(votePayload)
	voteReq, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/polls/%s/vote", poll.ID), bytes.NewReader(body))
	voteReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, voteReq)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected 400 Bad Request on non-existent option, got %d: %s", w.Code, w.Body.String())
	}

	// 2. Non-existent Poll ID -> 404 Not Found
	nonExistentPollReq, _ := http.NewRequest(http.MethodPost, "/api/polls/6aaaad215fa6199670736699/vote", bytes.NewReader(body))
	nonExistentPollReq.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	env.router.ServeHTTP(w, nonExistentPollReq)

	if w.Code != http.StatusNotFound {
		t.Fatalf("Expected 404 Not Found on non-existent poll ID, got %d: %s", w.Code, w.Body.String())
	}
}
