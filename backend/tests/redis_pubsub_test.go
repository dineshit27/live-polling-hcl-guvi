package tests

import (
	"context"
	"testing"
	"time"

	"live-polling-backend/models"
	"live-polling-backend/realtime"
	"live-polling-backend/repositories"
	"live-polling-backend/services"
)

func setupTestRedis(t *testing.T) *repositories.RedisClient {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	redisClient, err := repositories.ConnectRedis(ctx, "localhost:6379", "", 0)
	if err != nil {
		t.Skipf("Skipping Redis tests: Redis server not available: %v", err)
		return nil
	}
	return redisClient
}

func TestRedisConnectionAndPing(t *testing.T) {
	client := setupTestRedis(t)
	if client == nil {
		return
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := client.Ping(ctx); err != nil {
		t.Fatalf("Expected Redis Ping to succeed, got: %v", err)
	}
}

func TestRedisPubSub_VoteCastDelivery(t *testing.T) {
	client := setupTestRedis(t)
	if client == nil {
		return
	}
	defer client.Close()

	publisher := realtime.NewRedisEventPublisher(client.Client)
	subscriber := realtime.NewRedisEventSubscriber(client.Client)

	pollID := "poll_test_delivery_1"
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	msgChan, cleanup, err := subscriber.SubscribePoll(ctx, pollID)
	if err != nil {
		t.Fatalf("Failed to subscribe to poll channel: %v", err)
	}
	defer cleanup()

	// Give subscription brief moment to register
	time.Sleep(100 * time.Millisecond)

	expectedPayload := &models.RealtimeVotePayload{
		Event:    models.EventVoteCast,
		PollID:   pollID,
		OptionID: "opt_alpha",
		OptionCounts: map[string]int64{
			"opt_alpha": 1,
			"opt_beta":  0,
		},
		TotalVotes: 1,
		Status:     "active",
		Timestamp:  time.Now().UTC().Truncate(time.Millisecond),
	}

	if err := publisher.PublishVoteCast(ctx, expectedPayload); err != nil {
		t.Fatalf("Failed to publish VOTE_CAST event: %v", err)
	}

	select {
	case received := <-msgChan:
		if received == nil {
			t.Fatal("Received nil payload from subscription channel")
		}
		if received.Event != models.EventVoteCast {
			t.Errorf("Expected event %s, got %s", models.EventVoteCast, received.Event)
		}
		if received.PollID != pollID {
			t.Errorf("Expected poll ID %s, got %s", pollID, received.PollID)
		}
		if received.OptionID != "opt_alpha" {
			t.Errorf("Expected option ID opt_alpha, got %s", received.OptionID)
		}
		if received.TotalVotes != 1 {
			t.Errorf("Expected total votes 1, got %d", received.TotalVotes)
		}
		if received.OptionCounts["opt_alpha"] != 1 {
			t.Errorf("Expected option count 1, got %d", received.OptionCounts["opt_alpha"])
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Timed out waiting for VOTE_CAST event on Redis Pub/Sub channel")
	}
}

func TestRedisPubSub_BroadcastMultiSubscriber(t *testing.T) {
	client := setupTestRedis(t)
	if client == nil {
		return
	}
	defer client.Close()

	publisher := realtime.NewRedisEventPublisher(client.Client)
	subscriber1 := realtime.NewRedisEventSubscriber(client.Client)
	subscriber2 := realtime.NewRedisEventSubscriber(client.Client)

	pollID := "poll_test_broadcast_1"
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	chan1, cleanup1, err := subscriber1.SubscribePoll(ctx, pollID)
	if err != nil {
		t.Fatalf("Subscriber 1 failed to subscribe: %v", err)
	}
	defer cleanup1()

	chan2, cleanup2, err := subscriber2.SubscribePoll(ctx, pollID)
	if err != nil {
		t.Fatalf("Subscriber 2 failed to subscribe: %v", err)
	}
	defer cleanup2()

	time.Sleep(100 * time.Millisecond)

	payload := &models.RealtimeVotePayload{
		Event:    models.EventVoteCast,
		PollID:   pollID,
		OptionID: "opt_x",
		OptionCounts: map[string]int64{
			"opt_x": 5,
		},
		TotalVotes: 5,
		Status:     "active",
		Timestamp:  time.Now().UTC(),
	}

	if err := publisher.PublishVoteCast(ctx, payload); err != nil {
		t.Fatalf("Failed to publish broadcast event: %v", err)
	}

	// Verify both subscribers receive the event independently
	var rec1, rec2 *models.RealtimeVotePayload

	for i := 0; i < 2; i++ {
		select {
		case rec1 = <-chan1:
		case rec2 = <-chan2:
		case <-time.After(3 * time.Second):
			t.Fatalf("Timed out waiting for multi-subscriber broadcast delivery. Got rec1: %v, rec2: %v", rec1 != nil, rec2 != nil)
		}
	}

	if rec1 == nil || rec2 == nil {
		// Wait remaining in case of arrival order
		select {
		case rec1 = <-chan1:
		case rec2 = <-chan2:
		case <-time.After(1 * time.Second):
		}
	}

	if rec1 == nil || rec2 == nil {
		t.Fatalf("Expected both subscribers to receive event, got sub1=%v, sub2=%v", rec1 != nil, rec2 != nil)
	}

	if rec1.TotalVotes != 5 || rec2.TotalVotes != 5 {
		t.Errorf("Subscribers received inaccurate counts: sub1=%d, sub2=%d", rec1.TotalVotes, rec2.TotalVotes)
	}
}

func TestRedisPubSub_ChannelIsolation(t *testing.T) {
	client := setupTestRedis(t)
	if client == nil {
		return
	}
	defer client.Close()

	publisher := realtime.NewRedisEventPublisher(client.Client)
	subscriber := realtime.NewRedisEventSubscriber(client.Client)

	pollA := "poll_channel_A"
	pollB := "poll_channel_B"

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	chanA, cleanupA, err := subscriber.SubscribePoll(ctx, pollA)
	if err != nil {
		t.Fatalf("Failed to subscribe to poll A: %v", err)
	}
	defer cleanupA()

	chanB, cleanupB, err := subscriber.SubscribePoll(ctx, pollB)
	if err != nil {
		t.Fatalf("Failed to subscribe to poll B: %v", err)
	}
	defer cleanupB()

	time.Sleep(100 * time.Millisecond)

	// Publish strictly to Poll A
	payloadA := &models.RealtimeVotePayload{
		Event:      models.EventVoteCast,
		PollID:     pollA,
		OptionID:   "opt_a1",
		TotalVotes: 1,
	}

	if err := publisher.PublishVoteCast(ctx, payloadA); err != nil {
		t.Fatalf("Failed to publish to poll A: %v", err)
	}

	// chanA must receive it
	select {
	case msg := <-chanA:
		if msg.PollID != pollA {
			t.Errorf("Expected poll A, got %s", msg.PollID)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Timed out waiting for channel A event")
	}

	// chanB must NOT receive anything
	select {
	case msg := <-chanB:
		t.Fatalf("Channel B unexpectedly received message destined for Channel A: %v", msg)
	case <-time.After(200 * time.Millisecond):
		// Expected: no cross-channel leakage
	}
}

func TestRedisPubSub_PollStatusCloseDelivery(t *testing.T) {
	client := setupTestRedis(t)
	if client == nil {
		return
	}
	defer client.Close()

	publisher := realtime.NewRedisEventPublisher(client.Client)
	subscriber := realtime.NewRedisEventSubscriber(client.Client)

	pollID := "poll_status_test_1"
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	msgChan, cleanup, err := subscriber.SubscribePoll(ctx, pollID)
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}
	defer cleanup()

	time.Sleep(100 * time.Millisecond)

	statusPayload := &models.RealtimeVotePayload{
		Event:      models.EventPollStatus,
		PollID:     pollID,
		Status:     "closed",
		TotalVotes: 42,
		Timestamp:  time.Now().UTC(),
	}

	if err := publisher.PublishPollStatus(ctx, statusPayload); err != nil {
		t.Fatalf("Failed to publish status event: %v", err)
	}

	select {
	case received := <-msgChan:
		if received.Event != models.EventPollStatus {
			t.Errorf("Expected event POLL_STATUS, got %s", received.Event)
		}
		if received.Status != "closed" {
			t.Errorf("Expected status closed, got %s", received.Status)
		}
		if received.TotalVotes != 42 {
			t.Errorf("Expected total votes 42, got %d", received.TotalVotes)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Timed out waiting for POLL_STATUS event")
	}
}

func TestVoteService_WithRealRedisPublisher(t *testing.T) {
	mongoClient := setupTestMongoDB(t)
	if mongoClient == nil || mongoClient.Database == nil {
		return
	}
	defer mongoClient.Close(context.Background())
	database := mongoClient.Database

	redisClient := setupTestRedis(t)
	if redisClient == nil {
		return
	}
	defer redisClient.Close()

	pollRepo := repositories.NewPollRepository(database)
	voteRepo := repositories.NewVoteRepository(database)
	publisher := realtime.NewRedisEventPublisher(redisClient.Client)
	subscriber := realtime.NewRedisEventSubscriber(redisClient.Client)

	pollService := services.NewPollService(pollRepo, publisher)
	voteService := services.NewVoteService(pollRepo, voteRepo, publisher)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create real poll in MongoDB (source of truth)
	poll, err := pollService.CreatePoll(ctx, "owner_realtime_1", &models.CreatePollRequest{
		Title:   "Realtime Redis Integration Poll",
		Options: []string{"Redis PubSub", "Local Memory"},
	})
	if err != nil {
		t.Fatalf("Failed to create poll: %v", err)
	}

	// Subscribe to poll channel before casting vote
	msgChan, cleanup, err := subscriber.SubscribePoll(ctx, poll.ID)
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}
	defer cleanup()

	time.Sleep(100 * time.Millisecond)

	// Cast vote via voteService
	votedPoll, err := voteService.CastVote(ctx, poll.ID, &models.VoteRequest{
		OptionID: poll.Options[0].ID,
	}, "192.168.1.100", "")
	if err != nil {
		t.Fatalf("Failed to cast vote: %v", err)
	}

	// 1. Verify MongoDB persistent state
	if votedPoll.TotalVotes != 1 {
		t.Errorf("Expected total votes 1 in MongoDB, got %d", votedPoll.TotalVotes)
	}

	// 2. Verify Redis realtime event emission
	select {
	case event := <-msgChan:
		if event.Event != models.EventVoteCast {
			t.Errorf("Expected event VOTE_CAST, got %s", event.Event)
		}
		if event.PollID != poll.ID {
			t.Errorf("Expected poll ID %s, got %s", poll.ID, event.PollID)
		}
		if event.OptionID != poll.Options[0].ID {
			t.Errorf("Expected option ID %s, got %s", poll.Options[0].ID, event.OptionID)
		}
		if event.TotalVotes != 1 {
			t.Errorf("Expected total votes 1 in event, got %d", event.TotalVotes)
		}
		if event.OptionCounts[poll.Options[0].ID] != 1 {
			t.Errorf("Expected option count 1 in event, got %d", event.OptionCounts[poll.Options[0].ID])
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Timed out waiting for realtime Redis event after CastVote")
	}

	// 3. Test ClosePoll publishes POLL_STATUS
	closedPoll, err := pollService.ClosePoll(ctx, poll.ID, "owner_realtime_1")
	if err != nil {
		t.Fatalf("Failed to close poll: %v", err)
	}
	if closedPoll.Status != "closed" {
		t.Errorf("Expected poll to be closed in MongoDB, got %s", closedPoll.Status)
	}

	select {
	case event := <-msgChan:
		if event.Event != models.EventPollStatus {
			t.Errorf("Expected event POLL_STATUS, got %s", event.Event)
		}
		if event.Status != "closed" {
			t.Errorf("Expected status closed in event, got %s", event.Status)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Timed out waiting for realtime Redis POLL_STATUS event after ClosePoll")
	}
}

func TestVoteService_RejectedAndDuplicateVotesNeverPublish(t *testing.T) {
	mongoClient := setupTestMongoDB(t)
	if mongoClient == nil || mongoClient.Database == nil {
		return
	}
	defer mongoClient.Close(context.Background())
	database := mongoClient.Database

	redisClient := setupTestRedis(t)
	if redisClient == nil {
		return
	}
	defer redisClient.Close()

	pollRepo := repositories.NewPollRepository(database)
	voteRepo := repositories.NewVoteRepository(database)
	publisher := realtime.NewRedisEventPublisher(redisClient.Client)
	subscriber := realtime.NewRedisEventSubscriber(redisClient.Client)

	pollService := services.NewPollService(pollRepo, publisher)
	voteService := services.NewVoteService(pollRepo, voteRepo, publisher)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	poll, err := pollService.CreatePoll(ctx, "owner_rejection_test", &models.CreatePollRequest{
		Title:   "Rejection Test Poll",
		Options: []string{"Opt 1", "Opt 2"},
	})
	if err != nil {
		t.Fatalf("Failed to create poll: %v", err)
	}

	msgChan, cleanup, err := subscriber.SubscribePoll(ctx, poll.ID)
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}
	defer cleanup()

	time.Sleep(100 * time.Millisecond)

	// 1. Initial valid vote: MUST emit 1 event
	_, err = voteService.CastVote(ctx, poll.ID, &models.VoteRequest{
		OptionID:        poll.Options[0].ID,
		VoterIdentifier: "voter_exclusive_1",
	}, "10.0.0.1", "")
	if err != nil {
		t.Fatalf("First vote failed: %v", err)
	}

	select {
	case ev := <-msgChan:
		if ev.TotalVotes != 1 {
			t.Errorf("Expected total votes 1, got %d", ev.TotalVotes)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Expected initial valid vote to publish event, timed out")
	}

	// 2. Duplicate vote from same voter: MUST be rejected and MUST NOT emit any event
	_, err = voteService.CastVote(ctx, poll.ID, &models.VoteRequest{
		OptionID:        poll.Options[1].ID,
		VoterIdentifier: "voter_exclusive_1",
	}, "10.0.0.1", "")
	if err == nil {
		t.Fatal("Expected duplicate vote to be rejected, but got nil error")
	}

	select {
	case ev := <-msgChan:
		t.Fatalf("Duplicate vote unlawfully emitted realtime event to Redis: %+v", ev)
	case <-time.After(300 * time.Millisecond):
		// Expected: zero events emitted for duplicate vote
	}

	// 3. Invalid option vote: MUST be rejected and MUST NOT emit any event
	_, err = voteService.CastVote(ctx, poll.ID, &models.VoteRequest{
		OptionID:        "nonexistent_option_id",
		VoterIdentifier: "voter_exclusive_2",
	}, "10.0.0.2", "")
	if err == nil {
		t.Fatal("Expected invalid option vote to be rejected, but got nil error")
	}

	select {
	case ev := <-msgChan:
		t.Fatalf("Invalid option vote unlawfully emitted realtime event to Redis: %+v", ev)
	case <-time.After(300 * time.Millisecond):
		// Expected: zero events emitted for invalid option
	}

	// 4. Close the poll
	_, err = pollService.ClosePoll(ctx, poll.ID, "owner_rejection_test")
	if err != nil {
		t.Fatalf("Failed to close poll: %v", err)
	}
	// Drain the POLL_STATUS event from the channel
	select {
	case ev := <-msgChan:
		if ev.Event != models.EventPollStatus {
			t.Errorf("Expected POLL_STATUS, got %s", ev.Event)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Expected POLL_STATUS event on poll close")
	}

	// 5. Vote on closed poll: MUST be rejected and MUST NOT emit any event
	_, err = voteService.CastVote(ctx, poll.ID, &models.VoteRequest{
		OptionID:        poll.Options[0].ID,
		VoterIdentifier: "voter_exclusive_3",
	}, "10.0.0.3", "")
	if err == nil {
		t.Fatal("Expected closed poll vote to be rejected, but got nil error")
	}

	select {
	case ev := <-msgChan:
		t.Fatalf("Closed poll vote unlawfully emitted realtime event to Redis: %+v", ev)
	case <-time.After(300 * time.Millisecond):
		// Expected: zero events emitted for closed poll vote
	}
}

func TestRedisSubscriber_CleanupLifecycle(t *testing.T) {
	client := setupTestRedis(t)
	if client == nil {
		return
	}
	defer client.Close()

	subscriber := realtime.NewRedisEventSubscriber(client.Client)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	msgChan, cleanup, err := subscriber.SubscribePoll(ctx, "poll_cleanup_check")
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	// Call cleanup to unsubscribe and close subscription
	cleanup()

	// Channel must close and not hang
	select {
	case _, ok := <-msgChan:
		if ok {
			t.Fatal("Expected subscription channel to be closed after cleanup()")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Subscription channel failed to close within timeout after cleanup()")
	}

	// Calling cleanup a second time must be safe (idempotent)
	cleanup()
}

