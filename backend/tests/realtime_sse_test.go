package tests

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"live-polling-backend/config"
	"live-polling-backend/handlers"
	"live-polling-backend/models"
	"live-polling-backend/realtime"
	"live-polling-backend/repositories"
	"live-polling-backend/routes"
	"live-polling-backend/services"
)

// Helper to set up the router with real MongoDB and Redis for SSE tests
func setupSSETestRouter(t *testing.T) (*httptest.Server, *repositories.MongoClient, *repositories.RedisClient, services.PollService, services.VoteService) {
	mongoClient := setupTestMongoDB(t)
	if mongoClient == nil || mongoClient.Database == nil {
		t.Skip("MongoDB not available")
		return nil, nil, nil, nil, nil
	}

	redisClient := setupTestRedis(t)
	if redisClient == nil || redisClient.Client == nil {
		t.Skip("Redis not available")
		return nil, nil, nil, nil, nil
	}

	pollRepo := repositories.NewPollRepository(mongoClient.Database)
	voteRepo := repositories.NewVoteRepository(mongoClient.Database)
	publisher := realtime.NewRedisEventPublisher(redisClient.Client)
	subscriber := realtime.NewRedisEventSubscriber(redisClient.Client)

	pollService := services.NewPollService(pollRepo, publisher)
	voteService := services.NewVoteService(pollRepo, voteRepo, publisher)

	pollHandler := handlers.NewPollHandler(pollService)
	voteHandler := handlers.NewVoteHandler(voteService)
	realtimeHandler := handlers.NewRealtimeHandler(pollService, subscriber)

	r := routes.SetupRouter(&routes.RouterDeps{
		Config: &config.Config{
			GinMode:            "release",
			CORSAllowedOrigins: []string{"*"},
		},
		PollHandler:     pollHandler,
		VoteHandler:     voteHandler,
		RealtimeHandler: realtimeHandler,
	})

	ts := httptest.NewServer(r)
	return ts, mongoClient, redisClient, pollService, voteService
}

// readSSEEvent reads one SSE event block from a bufio.Reader
func readSSEEvent(reader *bufio.Reader, timeout time.Duration) (eventType string, data string, err error) {
	linesChan := make(chan struct {
		evt  string
		data string
		err  error
	}, 1)

	go func() {
		var curEvent, curData string
		for {
			line, rErr := reader.ReadString('\n')
			if rErr != nil {
				linesChan <- struct {
					evt  string
					data string
					err  error
				}{"", "", rErr}
				return
			}

			line = strings.TrimRight(line, "\r\n")
			if line == "" {
				// End of event block
				if curEvent != "" || curData != "" {
					linesChan <- struct {
						evt  string
						data string
						err  error
					}{curEvent, curData, nil}
					return
				}
				continue
			}

			if strings.HasPrefix(line, "event: ") {
				curEvent = strings.TrimPrefix(line, "event: ")
			} else if strings.HasPrefix(line, "data: ") {
				curData = strings.TrimPrefix(line, "data: ")
			}
		}
	}()

	select {
	case res := <-linesChan:
		return res.evt, res.data, res.err
	case <-time.After(timeout):
		return "", "", fmt.Errorf("timeout waiting for SSE event (%v)", timeout)
	}
}

func TestRealtimeSSE_ConnectionAndInitialSnapshot(t *testing.T) {
	ts, mongoClient, redisClient, pollService, _ := setupSSETestRouter(t)
	if ts == nil {
		return
	}
	defer ts.Close()
	defer mongoClient.Close(context.Background())
	defer redisClient.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 1. Create a poll with initial votes
	poll, err := pollService.CreatePoll(ctx, "user_sse_1", &models.CreatePollRequest{
		Title:   "SSE Test Poll Initial Snapshot",
		Options: []string{"React", "Vue", "Svelte"},
	})
	if err != nil {
		t.Fatalf("Failed to create poll: %v", err)
	}

	// 2. Connect to SSE endpoint: GET /api/polls/:id/events
	streamURL := fmt.Sprintf("%s/api/polls/%s/events", ts.URL, poll.ID)
	req, err := http.NewRequestWithContext(ctx, "GET", streamURL, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to connect to SSE stream: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/event-stream") {
		t.Fatalf("Expected Content-Type text/event-stream, got %s", contentType)
	}

	// 3. First event must be POLL_SNAPSHOT with options
	reader := bufio.NewReader(resp.Body)
	eventType, data, err := readSSEEvent(reader, 3*time.Second)
	if err != nil {
		t.Fatalf("Failed to read initial SSE event: %v", err)
	}

	if eventType != string(models.EventPollSnapshot) {
		t.Errorf("Expected event type '%s', got '%s'", models.EventPollSnapshot, eventType)
	}

	var snapshot models.RealtimeVotePayload
	if err := json.Unmarshal([]byte(data), &snapshot); err != nil {
		t.Fatalf("Failed to parse snapshot JSON: %v", err)
	}

	if snapshot.PollID != poll.ID {
		t.Errorf("Expected poll ID %s, got %s", poll.ID, snapshot.PollID)
	}
	if len(snapshot.OptionCounts) != 3 {
		t.Errorf("Expected 3 option counts, got %d", len(snapshot.OptionCounts))
	}
}

func TestRealtimeSSE_VoteCastDelivery(t *testing.T) {
	ts, mongoClient, redisClient, pollService, voteService := setupSSETestRouter(t)
	if ts == nil {
		return
	}
	defer ts.Close()
	defer mongoClient.Close(context.Background())
	defer redisClient.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	poll, err := pollService.CreatePoll(ctx, "user_sse_2", &models.CreatePollRequest{
		Title:   "Live Voting SSE Delivery Test",
		Options: []string{"Go", "Rust"},
	})
	if err != nil {
		t.Fatalf("Failed to create poll: %v", err)
	}

	// Connect SSE stream
	streamURL := fmt.Sprintf("%s/api/polls/%s/events", ts.URL, poll.ID)
	req, err := http.NewRequestWithContext(ctx, "GET", streamURL, nil)
	if err != nil {
		t.Fatalf("Failed to create SSE request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to connect to SSE stream: %v", err)
	}
	defer resp.Body.Close()

	reader := bufio.NewReader(resp.Body)

	// Consume initial snapshot
	evt1, _, err := readSSEEvent(reader, 3*time.Second)
	if err != nil || evt1 != string(models.EventPollSnapshot) {
		t.Fatalf("Failed to read initial snapshot: %v", err)
	}

	// Give Redis subscription a moment to attach
	time.Sleep(100 * time.Millisecond)

	// Cast a vote via VoteService (simulating voter)
	targetOptionID := poll.Options[0].ID
	_, err = voteService.CastVote(ctx, poll.ID, &models.VoteRequest{
		OptionID:        targetOptionID,
		VoterIdentifier: "voter_live_test_1",
	}, "192.168.1.1", "")
	if err != nil {
		t.Fatalf("Failed to cast vote: %v", err)
	}

	// Next event on SSE stream MUST be VOTE_CAST with updated tally
	evt2, data2, err := readSSEEvent(reader, 3*time.Second)
	if err != nil {
		t.Fatalf("Failed to read VOTE_CAST event: %v", err)
	}

	if evt2 != string(models.EventVoteCast) {
		t.Errorf("Expected event type '%s', got '%s'", models.EventVoteCast, evt2)
	}

	var votePayload models.RealtimeVotePayload
	if err := json.Unmarshal([]byte(data2), &votePayload); err != nil {
		t.Fatalf("Failed to unmarshal vote payload: %v", err)
	}

	if votePayload.TotalVotes != 1 {
		t.Errorf("Expected total votes 1, got %d", votePayload.TotalVotes)
	}
	if votePayload.OptionCounts[targetOptionID] != 1 {
		t.Errorf("Expected option count 1 for %s, got %d", targetOptionID, votePayload.OptionCounts[targetOptionID])
	}
}

func TestRealtimeSSE_ChannelIsolation(t *testing.T) {
	ts, mongoClient, redisClient, pollService, voteService := setupSSETestRouter(t)
	if ts == nil {
		return
	}
	defer ts.Close()
	defer mongoClient.Close(context.Background())
	defer redisClient.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	// Create Poll A and Poll B
	pollA, err := pollService.CreatePoll(ctx, "user_a", &models.CreatePollRequest{
		Title:   "Poll A - Private",
		Options: []string{"A1", "A2"},
	})
	if err != nil {
		t.Fatalf("Failed to create Poll A: %v", err)
	}

	pollB, err := pollService.CreatePoll(ctx, "user_b", &models.CreatePollRequest{
		Title:   "Poll B - Private",
		Options: []string{"B1", "B2"},
	})
	if err != nil {
		t.Fatalf("Failed to create Poll B: %v", err)
	}

	// Connect SSE stream to Poll B ONLY
	streamBURL := fmt.Sprintf("%s/api/polls/%s/events", ts.URL, pollB.ID)
	reqB, err := http.NewRequestWithContext(ctx, "GET", streamBURL, nil)
	if err != nil {
		t.Fatalf("Request error: %v", err)
	}
	respB, err := http.DefaultClient.Do(reqB)
	if err != nil {
		t.Fatalf("SSE B connect error: %v", err)
	}
	defer respB.Body.Close()

	readerB := bufio.NewReader(respB.Body)

	// Consume Poll B initial snapshot
	evt, data, err := readSSEEvent(readerB, 3*time.Second)
	if err != nil || evt != string(models.EventPollSnapshot) {
		t.Fatalf("Failed to get initial snapshot on Poll B: %v", err)
	}
	var snapB models.RealtimeVotePayload
	_ = json.Unmarshal([]byte(data), &snapB)
	if snapB.PollID != pollB.ID {
		t.Fatalf("Snapshot is not for Poll B")
	}

	time.Sleep(100 * time.Millisecond)

	// Now cast vote on Poll A
	_, err = voteService.CastVote(ctx, pollA.ID, &models.VoteRequest{
		OptionID:        pollA.Options[0].ID,
		VoterIdentifier: "voter_poll_a",
	}, "10.0.0.1", "")
	if err != nil {
		t.Fatalf("Vote on Poll A failed: %v", err)
	}

	// Poll B stream MUST NOT receive any event for Poll A
	_, _, err = readSSEEvent(readerB, 500*time.Millisecond)
	if err == nil {
		t.Fatal("Poll B received an unlawful event from Poll A! Channel isolation failed.")
	}
}

func TestRealtimeSSE_MultipleSubscribers(t *testing.T) {
	ts, mongoClient, redisClient, pollService, voteService := setupSSETestRouter(t)
	if ts == nil {
		return
	}
	defer ts.Close()
	defer mongoClient.Close(context.Background())
	defer redisClient.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	poll, err := pollService.CreatePoll(ctx, "multi_sub_owner", &models.CreatePollRequest{
		Title:   "Multi Viewer Poll",
		Options: []string{"Alpha", "Beta"},
	})
	if err != nil {
		t.Fatalf("Failed to create poll: %v", err)
	}

	// Connect 3 independent client viewers to the same poll
	numClients := 3
	var responses []*http.Response
	var readers []*bufio.Reader

	for i := 0; i < numClients; i++ {
		streamURL := fmt.Sprintf("%s/api/polls/%s/events", ts.URL, poll.ID)
		req, _ := http.NewRequestWithContext(ctx, "GET", streamURL, nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Client %d failed to connect: %v", i, err)
		}
		responses = append(responses, resp)
		reader := bufio.NewReader(resp.Body)
		readers = append(readers, reader)

		// Consume initial snapshot for each
		_, _, err = readSSEEvent(reader, 3*time.Second)
		if err != nil {
			t.Fatalf("Client %d failed to receive initial snapshot: %v", i, err)
		}
	}

	defer func() {
		for _, r := range responses {
			r.Body.Close()
		}
	}()

	time.Sleep(100 * time.Millisecond)

	// Cast a vote
	_, err = voteService.CastVote(ctx, poll.ID, &models.VoteRequest{
		OptionID:        poll.Options[0].ID,
		VoterIdentifier: "broadcast_voter_1",
	}, "10.0.0.1", "")
	if err != nil {
		t.Fatalf("Failed to cast vote: %v", err)
	}

	// All 3 clients must receive the VOTE_CAST event
	for i, reader := range readers {
		evt, data, err := readSSEEvent(reader, 3*time.Second)
		if err != nil {
			t.Fatalf("Client %d timed out waiting for VOTE_CAST: %v", i, err)
		}
		if evt != string(models.EventVoteCast) {
			t.Errorf("Client %d expected VOTE_CAST, got %s", i, evt)
		}

		var p models.RealtimeVotePayload
		if err := json.Unmarshal([]byte(data), &p); err != nil {
			t.Fatalf("Client %d failed to parse payload: %v", i, err)
		}
		if p.TotalVotes != 1 {
			t.Errorf("Client %d expected total votes 1, got %d", i, p.TotalVotes)
		}
	}
}

func TestRealtimeSSE_NonExistentPoll(t *testing.T) {
	ts, mongoClient, redisClient, _, _ := setupSSETestRouter(t)
	if ts == nil {
		return
	}
	defer ts.Close()
	defer mongoClient.Close(context.Background())
	defer redisClient.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	streamURL := fmt.Sprintf("%s/api/polls/6aaaad215fa6199670736699/events", ts.URL)
	req, _ := http.NewRequestWithContext(ctx, "GET", streamURL, nil)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Request error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected 404 Not Found for non-existent poll, got %d", resp.StatusCode)
	}
}
