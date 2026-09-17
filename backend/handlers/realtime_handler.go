package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"live-polling-backend/models"
	"live-polling-backend/realtime"
	"live-polling-backend/services"
)

// RealtimeHandler manages SSE streaming for live poll updates.
type RealtimeHandler struct {
	pollService services.PollService
	subscriber  realtime.EventSubscriber
}

// NewRealtimeHandler constructs a new RealtimeHandler instance.
func NewRealtimeHandler(pollService services.PollService, subscriber realtime.EventSubscriber) *RealtimeHandler {
	return &RealtimeHandler{
		pollService: pollService,
		subscriber:  subscriber,
	}
}

// StreamPollEvents handles GET /api/polls/:id/events (and /api/polls/:id/stream)
// Establishes a Server-Sent Events (SSE) stream for real-time poll updates.
func (h *RealtimeHandler) StreamPollEvents(c *gin.Context) {
	pollID := c.Param("id")
	if pollID == "" {
		RespondError(c, http.StatusBadRequest, "invalid_poll_id", "Poll ID is required")
		return
	}

	if h.pollService == nil || h.subscriber == nil {
		RespondError(c, http.StatusServiceUnavailable, "realtime_unavailable", "Realtime service is currently unavailable")
		return
	}

	// 1. Validate poll exists in MongoDB (source of truth) before opening stream
	poll, err := h.pollService.GetPollByID(c.Request.Context(), pollID)
	if err != nil {
		if err == services.ErrPollNotFound {
			RespondError(c, http.StatusNotFound, "poll_not_found", "Poll not found")
			return
		}
		RespondError(c, http.StatusInternalServerError, "internal_error", "Failed to retrieve poll")
		return
	}

	// 2. Set SSE response headers
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache, no-transform")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no") // Disable proxy buffering for nginx / vite
	c.Writer.Flush()

	// 3. Send initial state snapshot immediately so client is synchronized on connect
	counts := make(map[string]int64)
	for _, opt := range poll.Options {
		counts[opt.ID] = opt.VoteCount
	}

	initialSnapshot := models.RealtimeVotePayload{
		Event:        models.EventPollSnapshot,
		PollID:       poll.ID,
		OptionCounts: counts,
		TotalVotes:   poll.TotalVotes,
		Status:       poll.Status,
		Timestamp:    time.Now().UTC(),
	}

	if err := sendSSE(c, string(initialSnapshot.Event), initialSnapshot); err != nil {
		return
	}

	// 4. Subscribe to Redis Pub/Sub for this specific poll channel
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	payloadChan, cleanup, err := h.subscriber.SubscribePoll(ctx, pollID)
	if err != nil {
		_ = sendSSEComment(c, fmt.Sprintf("subscription error: %v", err))
		return
	}
	defer cleanup()

	// 5. Heartbeat ticker to keep connection alive through proxies
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	// 6. Event loop: stream Redis Pub/Sub messages until client disconnects
	for {
		select {
		case <-c.Request.Context().Done():
			// Client disconnected (tab closed, navigation, network drop)
			return

		case payload, ok := <-payloadChan:
			if !ok {
				// Subscription closed
				return
			}
			if err := sendSSE(c, string(payload.Event), payload); err != nil {
				return
			}

		case <-ticker.C:
			if err := sendSSEComment(c, "ping"); err != nil {
				return
			}
		}
	}
}

// sendSSE serializes data to JSON and formats as a standard SSE message:
// event: <event>\ndata: <json>\n\n
func sendSSE(c *gin.Context, event string, data any) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event, string(jsonData))
	if err != nil {
		return err
	}
	c.Writer.Flush()
	return nil
}

// sendSSEComment sends a comment line (: <comment>\n\n) as a keep-alive heartbeat.
func sendSSEComment(c *gin.Context, comment string) error {
	_, err := fmt.Fprintf(c.Writer, ": %s\n\n", comment)
	if err != nil {
		return err
	}
	c.Writer.Flush()
	return nil
}
