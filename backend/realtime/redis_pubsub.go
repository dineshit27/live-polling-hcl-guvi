package realtime

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"

	"live-polling-backend/models"
)

// Channel name formatting helpers
func PollChannel(pollID string) string {
	return fmt.Sprintf("poll:%s:events", pollID)
}

func GlobalPollsChannel() string {
	return "polls:events"
}

// EventPublisher defines the contract for emitting real-time polling events.
type EventPublisher interface {
	PublishVoteCast(ctx context.Context, payload *models.RealtimeVotePayload) error
	PublishPollStatus(ctx context.Context, payload *models.RealtimeVotePayload) error
	PublishPollSnapshot(ctx context.Context, payload *models.RealtimeVotePayload) error
	PublishEvent(ctx context.Context, channel string, payload *models.RealtimeVotePayload) error
}

// EventSubscriber defines the contract for listening to real-time polling events.
type EventSubscriber interface {
	SubscribePoll(ctx context.Context, pollID string) (<-chan *models.RealtimeVotePayload, func(), error)
	SubscribeAll(ctx context.Context) (<-chan *models.RealtimeVotePayload, func(), error)
}

// RedisEventPublisher publishes events via Redis Pub/Sub.
type RedisEventPublisher struct {
	client *redis.Client
}

// NewRedisEventPublisher instantiates an EventPublisher backed by Redis.
func NewRedisEventPublisher(client *redis.Client) EventPublisher {
	return &RedisEventPublisher{
		client: client,
	}
}

// PublishEvent serializes payload to JSON and publishes to a specific channel.
func (p *RedisEventPublisher) PublishEvent(ctx context.Context, channel string, payload *models.RealtimeVotePayload) error {
	if p == nil || p.client == nil {
		return fmt.Errorf("redis publisher client is uninitialized")
	}

	if payload.Timestamp.IsZero() {
		payload.Timestamp = time.Now().UTC()
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal realtime payload: %w", err)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	cmd := p.client.Publish(timeoutCtx, channel, data)
	if err := cmd.Err(); err != nil {
		return fmt.Errorf("failed to publish to redis channel '%s': %w", channel, err)
	}

	return nil
}

// PublishVoteCast emits a VOTE_CAST event to both the poll-specific and global channels.
func (p *RedisEventPublisher) PublishVoteCast(ctx context.Context, payload *models.RealtimeVotePayload) error {
	payload.Event = models.EventVoteCast
	pollChan := PollChannel(payload.PollID)

	// Publish to specific poll channel
	if err := p.PublishEvent(ctx, pollChan, payload); err != nil {
		return err
	}

	// Also fan out to global channel for system-wide dashboard listeners
	_ = p.PublishEvent(ctx, GlobalPollsChannel(), payload)
	return nil
}

// PublishPollStatus emits a POLL_STATUS event (e.g. poll closed or reactivated).
func (p *RedisEventPublisher) PublishPollStatus(ctx context.Context, payload *models.RealtimeVotePayload) error {
	payload.Event = models.EventPollStatus
	pollChan := PollChannel(payload.PollID)

	if err := p.PublishEvent(ctx, pollChan, payload); err != nil {
		return err
	}

	_ = p.PublishEvent(ctx, GlobalPollsChannel(), payload)
	return nil
}

// PublishPollSnapshot emits a snapshot state of a poll.
func (p *RedisEventPublisher) PublishPollSnapshot(ctx context.Context, payload *models.RealtimeVotePayload) error {
	payload.Event = models.EventPollSnapshot
	pollChan := PollChannel(payload.PollID)
	return p.PublishEvent(ctx, pollChan, payload)
}

// RedisEventSubscriber manages subscriptions to Redis Pub/Sub channels.
type RedisEventSubscriber struct {
	client *redis.Client
}

// NewRedisEventSubscriber instantiates an EventSubscriber backed by Redis.
func NewRedisEventSubscriber(client *redis.Client) EventSubscriber {
	return &RedisEventSubscriber{
		client: client,
	}
}

// SubscribePoll subscribes to updates for a specific poll.
// Returns a payload channel, an unsubscribe cleanup function, or an error.
func (s *RedisEventSubscriber) SubscribePoll(ctx context.Context, pollID string) (<-chan *models.RealtimeVotePayload, func(), error) {
	channelName := PollChannel(pollID)
	return s.subscribeChannel(ctx, channelName)
}

// SubscribeAll subscribes to global polling updates.
func (s *RedisEventSubscriber) SubscribeAll(ctx context.Context) (<-chan *models.RealtimeVotePayload, func(), error) {
	channelName := GlobalPollsChannel()
	return s.subscribeChannel(ctx, channelName)
}

func (s *RedisEventSubscriber) subscribeChannel(ctx context.Context, channelName string) (<-chan *models.RealtimeVotePayload, func(), error) {
	if s == nil || s.client == nil {
		return nil, nil, fmt.Errorf("redis subscriber client is uninitialized")
	}

	pubsub := s.client.Subscribe(ctx, channelName)

	// Verify subscription was registered with Redis
	subCtx, cancelSub := context.WithTimeout(ctx, 3*time.Second)
	_, err := pubsub.Receive(subCtx)
	cancelSub()
	if err != nil {
		_ = pubsub.Close()
		return nil, nil, fmt.Errorf("failed to subscribe to redis channel '%s': %w", channelName, err)
	}

	outChan := make(chan *models.RealtimeVotePayload, 64)
	subCloseChan := make(chan struct{})

	cleanup := func() {
		select {
		case <-subCloseChan:
			// Already closed
			return
		default:
			close(subCloseChan)
			_ = pubsub.Close()
		}
	}

	go func() {
		defer close(outChan)
		ch := pubsub.Channel()

		for {
			select {
			case <-ctx.Done():
				cleanup()
				return
			case <-subCloseChan:
				return
			case msg, ok := <-ch:
				if !ok {
					return
				}
				var payload models.RealtimeVotePayload
				if err := json.Unmarshal([]byte(msg.Payload), &payload); err != nil {
					log.Printf("[Redis PubSub] Warning: failed to parse payload on channel '%s': %v", channelName, err)
					continue
				}

				select {
				case outChan <- &payload:
				case <-ctx.Done():
					return
				case <-subCloseChan:
					return
				default:
					log.Printf("[Redis PubSub] Warning: buffer full, dropping message on '%s'", channelName)
				}
			}
		}
	}()

	return outChan, cleanup, nil
}
