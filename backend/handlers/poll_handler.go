package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"live-polling-backend/middleware"
	"live-polling-backend/models"
	"live-polling-backend/services"
)

type PollHandler struct {
	pollService services.PollService
}

func NewPollHandler(pollService services.PollService) *PollHandler {
	return &PollHandler{
		pollService: pollService,
	}
}

// Create handles POST /api/polls (Auth required)
func (h *PollHandler) Create(c *gin.Context) {
	userID, exists := middleware.GetAuthUserID(c)
	if !exists || userID == "" {
		RespondError(c, http.StatusUnauthorized, "unauthorized", "Authentication required to create a poll")
		return
	}

	var req models.CreatePollRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, err)
		return
	}

	poll, err := h.pollService.CreatePoll(c.Request.Context(), userID, &req)
	if err != nil {
		if errors.Is(err, services.ErrInvalidOptions) {
			RespondError(c, http.StatusBadRequest, "invalid_options", err.Error())
			return
		}
		RespondError(c, http.StatusInternalServerError, "internal_error", "Failed to create poll")
		return
	}

	c.JSON(http.StatusCreated, SuccessResponse{
		Success: true,
		Message: "Poll created successfully",
		Data:    poll,
	})
}

// List handles GET /api/polls (Public)
func (h *PollHandler) List(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "20")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, _ := strconv.ParseInt(limitStr, 10, 64)
	offset, _ := strconv.ParseInt(offsetStr, 10, 64)

	polls, err := h.pollService.ListActivePolls(c.Request.Context(), limit, offset)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "internal_error", "Failed to retrieve polls")
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    polls,
	})
}

// ListMyPolls handles GET /api/polls/my (Auth required)
func (h *PollHandler) ListMyPolls(c *gin.Context) {
	userID, exists := middleware.GetAuthUserID(c)
	if !exists || userID == "" {
		RespondError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	polls, err := h.pollService.ListMyPolls(c.Request.Context(), userID)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "internal_error", "Failed to retrieve user polls")
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    polls,
	})
}

// GetByID handles GET /api/polls/:id (Public)
func (h *PollHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		RespondError(c, http.StatusBadRequest, "invalid_id", "Poll ID is required")
		return
	}

	poll, err := h.pollService.GetPollByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, services.ErrPollNotFound) {
			RespondError(c, http.StatusNotFound, "not_found", "Poll not found")
			return
		}
		RespondError(c, http.StatusInternalServerError, "internal_error", "Failed to retrieve poll")
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    poll,
	})
}

// Update handles PUT /api/polls/:id (Auth + Creator ownership required)
func (h *PollHandler) Update(c *gin.Context) {
	userID, exists := middleware.GetAuthUserID(c)
	if !exists || userID == "" {
		RespondError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	id := c.Param("id")
	var req models.UpdatePollRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, err)
		return
	}

	updatedPoll, err := h.pollService.UpdatePoll(c.Request.Context(), id, userID, &req)
	if err != nil {
		if errors.Is(err, services.ErrPollNotFound) {
			RespondError(c, http.StatusNotFound, "not_found", "Poll not found")
			return
		}
		if errors.Is(err, services.ErrUnauthorizedPoll) {
			RespondError(c, http.StatusForbidden, "forbidden", "Only the poll creator can update this poll")
			return
		}
		RespondError(c, http.StatusInternalServerError, "internal_error", "Failed to update poll")
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: "Poll updated successfully",
		Data:    updatedPoll,
	})
}

// Delete handles DELETE /api/polls/:id (Auth + Creator ownership required)
func (h *PollHandler) Delete(c *gin.Context) {
	userID, exists := middleware.GetAuthUserID(c)
	if !exists || userID == "" {
		RespondError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	id := c.Param("id")
	err := h.pollService.DeletePoll(c.Request.Context(), id, userID)
	if err != nil {
		if errors.Is(err, services.ErrPollNotFound) {
			RespondError(c, http.StatusNotFound, "not_found", "Poll not found")
			return
		}
		if errors.Is(err, services.ErrUnauthorizedPoll) {
			RespondError(c, http.StatusForbidden, "forbidden", "Only the poll creator can delete this poll")
			return
		}
		RespondError(c, http.StatusInternalServerError, "internal_error", "Failed to delete poll")
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: "Poll deleted successfully",
	})
}

// Close handles POST /api/polls/:id/close (Auth + Creator ownership required)
func (h *PollHandler) Close(c *gin.Context) {
	userID, exists := middleware.GetAuthUserID(c)
	if !exists || userID == "" {
		RespondError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	id := c.Param("id")
	closedPoll, err := h.pollService.ClosePoll(c.Request.Context(), id, userID)
	if err != nil {
		if errors.Is(err, services.ErrPollNotFound) {
			RespondError(c, http.StatusNotFound, "not_found", "Poll not found")
			return
		}
		if errors.Is(err, services.ErrUnauthorizedPoll) {
			RespondError(c, http.StatusForbidden, "forbidden", "Only the poll creator can close this poll")
			return
		}
		RespondError(c, http.StatusInternalServerError, "internal_error", "Failed to close poll")
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: "Poll closed successfully",
		Data:    closedPoll,
	})
}
