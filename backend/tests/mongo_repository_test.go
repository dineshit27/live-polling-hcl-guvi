package tests

import (
	"context"
	"testing"
	"time"

	"live-polling-backend/models"
	"live-polling-backend/repositories"
)

const (
	testMongoURI = "mongodb://localhost:27017"
	testDBName   = "polling_test_db"
)

func setupTestMongoDB(t *testing.T) *repositories.MongoClient {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, err := repositories.ConnectMongoDB(ctx, testMongoURI, testDBName)
	if err != nil {
		t.Skipf("Skipping MongoDB integration tests: MongoDB not available: %v", err)
		return nil
	}

	// Drop test database to start fresh
	_ = client.Database.Drop(ctx)
	// Recreate indexes
	_ = client.EnsureIndexes(ctx)

	return client
}

func TestUserRepository_CreateAndFind(t *testing.T) {
	client := setupTestMongoDB(t)
	if client == nil {
		return
	}
	defer client.Close(context.Background())

	userRepo := repositories.NewUserRepository(client.Database)
	ctx := context.Background()

	user := &models.User{
		Name:         "Alice Voter",
		Email:        "alice@example.com",
		PasswordHash: "hashed_secret_test",
	}

	err := userRepo.Create(ctx, user)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	if user.ID == "" {
		t.Fatalf("Expected user.ID to be populated, got empty string")
	}

	// Duplicate email test (unique index verification)
	dupUser := &models.User{
		Name:         "Alice Duplicate",
		Email:        "alice@example.com",
		PasswordHash: "another_hash",
	}
	dupErr := userRepo.Create(ctx, dupUser)
	if dupErr != repositories.ErrUserAlreadyExists {
		t.Fatalf("Expected ErrUserAlreadyExists on duplicate email, got %v", dupErr)
	}

	// Find by email
	foundByEmail, err := userRepo.FindByEmail(ctx, "alice@example.com")
	if err != nil {
		t.Fatalf("Expected to find user by email, got error: %v", err)
	}
	if foundByEmail.Name != "Alice Voter" {
		t.Fatalf("Expected name 'Alice Voter', got '%s'", foundByEmail.Name)
	}

	// Find by ID
	foundByID, err := userRepo.FindByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("Expected to find user by ID, got error: %v", err)
	}
	if foundByID.Email != "alice@example.com" {
		t.Fatalf("Expected email 'alice@example.com', got '%s'", foundByID.Email)
	}
}

func TestPollRepository_CRUDAndIncrement(t *testing.T) {
	client := setupTestMongoDB(t)
	if client == nil {
		return
	}
	defer client.Close(context.Background())

	pollRepo := repositories.NewPollRepository(client.Database)
	ctx := context.Background()

	poll := &models.Poll{
		OwnerID:     "owner_123",
		Title:       "Favorite Backend Language?",
		Description: "Community survey for developer ergonomics",
		Status:      "active",
		Options: []models.PollOption{
			{Text: "Go", VoteCount: 0},
			{Text: "Rust", VoteCount: 0},
			{Text: "TypeScript", VoteCount: 0},
		},
	}

	err := pollRepo.Create(ctx, poll)
	if err != nil {
		t.Fatalf("Failed to create poll: %v", err)
	}

	if poll.ID == "" {
		t.Fatalf("Expected poll.ID to be assigned")
	}
	if len(poll.Options[0].ID) == 0 {
		t.Fatalf("Expected option.ID to be assigned")
	}

	// Find by ID
	fetched, err := pollRepo.FindByID(ctx, poll.ID)
	if err != nil {
		t.Fatalf("Failed to find poll by ID: %v", err)
	}
	if fetched.Title != poll.Title {
		t.Fatalf("Expected title '%s', got '%s'", poll.Title, fetched.Title)
	}

	// Atomic vote increment
	targetOptionID := fetched.Options[0].ID
	incErr := pollRepo.IncrementOptionVote(ctx, poll.ID, targetOptionID)
	if incErr != nil {
		t.Fatalf("Failed to increment option vote: %v", incErr)
	}

	updated, err := pollRepo.FindByID(ctx, poll.ID)
	if err != nil {
		t.Fatalf("Failed to find updated poll: %v", err)
	}
	if updated.TotalVotes != 1 {
		t.Fatalf("Expected TotalVotes 1, got %d", updated.TotalVotes)
	}
	if updated.Options[0].VoteCount != 1 {
		t.Fatalf("Expected option vote count 1, got %d", updated.Options[0].VoteCount)
	}
}

func TestVoteRepository_CompoundUniqueConstraint(t *testing.T) {
	client := setupTestMongoDB(t)
	if client == nil {
		return
	}
	defer client.Close(context.Background())

	voteRepo := repositories.NewVoteRepository(client.Database)
	ctx := context.Background()

	vote1 := &models.Vote{
		PollID:          "poll_abc",
		OptionID:        "opt_1",
		VoterIdentifier: "voter_fingerprint_unique_xyz",
		IPHash:          "hash_123",
	}

	err := voteRepo.CastVote(ctx, vote1)
	if err != nil {
		t.Fatalf("Failed to cast initial vote: %v", err)
	}

	// Verify HasVoted
	hasVoted, err := voteRepo.HasVoted(ctx, "poll_abc", "voter_fingerprint_unique_xyz")
	if err != nil || !hasVoted {
		t.Fatalf("Expected HasVoted to be true, got %v, err=%v", hasVoted, err)
	}

	// Second vote on same poll by same voter MUST fail due to compound unique index
	vote2 := &models.Vote{
		PollID:          "poll_abc",
		OptionID:        "opt_2",
		VoterIdentifier: "voter_fingerprint_unique_xyz",
		IPHash:          "hash_123",
	}

	dupErr := voteRepo.CastVote(ctx, vote2)
	if dupErr != repositories.ErrAlreadyVoted {
		t.Fatalf("Expected ErrAlreadyVoted on duplicate vote from same voter, got %v", dupErr)
	}
}
