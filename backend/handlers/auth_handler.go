package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"live-polling-backend/models"
	"live-polling-backend/repositories"
	"live-polling-backend/services"
)

type AuthHandler struct {
	authService services.AuthService
}

func NewAuthHandler(authService services.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// Signup handles POST /api/auth/signup
func (h *AuthHandler) Signup(c *gin.Context) {
	var req models.UserSignupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, err)
		return
	}

	res, err := h.authService.Signup(c.Request.Context(), &req)
	if err != nil {
		if errors.Is(err, repositories.ErrUserAlreadyExists) {
			RespondError(c, http.StatusConflict, "conflict", "A user with this email already exists")
			return
		}
		RespondError(c, http.StatusInternalServerError, "internal_error", "Failed to register user account")
		return
	}

	c.JSON(http.StatusCreated, SuccessResponse{
		Success: true,
		Message: "User registered successfully",
		Data:    res,
	})
}

// Login handles POST /api/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req models.UserLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, err)
		return
	}

	res, err := h.authService.Login(c.Request.Context(), &req)
	if err != nil {
		if errors.Is(err, services.ErrInvalidCredentials) {
			RespondError(c, http.StatusUnauthorized, "unauthorized", "Invalid email or password")
			return
		}
		RespondError(c, http.StatusInternalServerError, "internal_error", "Failed to process login")
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: "Logged in successfully",
		Data:    res,
	})
}

// GetMe handles GET /api/auth/me (Protected by AuthMiddleware)
func (h *AuthHandler) GetMe(c *gin.Context) {
	val, exists := c.Get("auth_user_id")
	if !exists {
		RespondError(c, http.StatusUnauthorized, "unauthorized", "Authentication context missing")
		return
	}

	userID, ok := val.(string)
	if !ok || userID == "" {
		RespondError(c, http.StatusUnauthorized, "unauthorized", "Invalid user ID in context")
		return
	}

	user, err := h.authService.GetProfile(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, repositories.ErrUserNotFound) {
			RespondError(c, http.StatusNotFound, "not_found", "User account not found")
			return
		}
		RespondError(c, http.StatusInternalServerError, "internal_error", "Failed to retrieve user profile")
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    user,
	})
}
