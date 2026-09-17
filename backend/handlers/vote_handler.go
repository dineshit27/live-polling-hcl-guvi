package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"live-polling-backend/middleware"
	"live-polling-backend/models"
	"live-polling-backend/services"
)

type VoteHandler struct {
	voteService services.VoteService
}

func NewVoteHandler(voteService services.VoteService) *VoteHandler {
	return &VoteHandler{
		voteService: voteService,
	}
}

// CastVote handles POST /api/polls/:id/vote
func (h *VoteHandler) CastVote(c *gin.Context) {
	pollID := c.Param("id")
	if pollID == "" {
		RespondError(c, http.StatusBadRequest, "invalid_id", "Poll ID is required")
		return
	}

	var req models.VoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, err)
		return
	}

	authUserID, _ := middleware.GetAuthUserID(c)
	clientIP := c.ClientIP()

	updatedPoll, err := h.voteService.CastVote(c.Request.Context(), pollID, &req, clientIP, authUserID)
	if err != nil {
		if errors.Is(err, services.ErrPollNotFound) {
			RespondError(c, http.StatusNotFound, "not_found", "Poll not found")
			return
		}
		if errors.Is(err, services.ErrPollClosed) {
			RespondError(c, http.StatusBadRequest, "poll_closed", "This poll is closed for voting")
			return
		}
		if errors.Is(err, services.ErrAlreadyVoted) {
			RespondError(c, http.StatusConflict, "already_voted", "You have already voted on this poll")
			return
		}
		if errors.Is(err, services.ErrOptionNotFound) {
			RespondError(c, http.StatusBadRequest, "option_not_found", "Selected option does not exist on this poll")
			return
		}

		RespondError(c, http.StatusInternalServerError, "internal_error", "Failed to register vote")
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: "Vote recorded successfully",
		Data:    updatedPoll,
	})
}

// GetVoteStatus handles GET /api/polls/:id/vote-status
func (h *VoteHandler) GetVoteStatus(c *gin.Context) {
	pollID := c.Param("id")
	if pollID == "" {
		RespondError(c, http.StatusBadRequest, "invalid_id", "Poll ID is required")
		return
	}

	voterIdentifier := c.Query("voter_identifier")
	authUserID, _ := middleware.GetAuthUserID(c)
	clientIP := c.ClientIP()

	status, err := h.voteService.CheckVoteStatus(c.Request.Context(), pollID, voterIdentifier, clientIP, authUserID)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "internal_error", "Failed to check voter status")
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    status,
	})
}
