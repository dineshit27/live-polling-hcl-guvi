package routes

import (
	"github.com/gin-gonic/gin"

	"live-polling-backend/config"
	"live-polling-backend/handlers"
	"live-polling-backend/middleware"
	"live-polling-backend/services"
)

type RouterDeps struct {
	Config        *config.Config
	HealthHandler *handlers.HealthHandler
	AuthHandler   *handlers.AuthHandler
	AuthService   services.AuthService
	PollHandler     *handlers.PollHandler
	VoteHandler     *handlers.VoteHandler
	RealtimeHandler *handlers.RealtimeHandler
}

// SetupRouter constructs the Gin Engine, attaches global middleware, and configures route groups.
func SetupRouter(deps *RouterDeps) *gin.Engine {
	if deps.Config.GinMode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

	// Global pipeline middleware
	r.Use(middleware.LoggerMiddleware())
	r.Use(middleware.RecoveryMiddleware())
	r.Use(middleware.CORSMiddleware(deps.Config.CORSAllowedOrigins))

	// Base API Group
	api := r.Group("/api")
	{
		// Health & Observability
		if deps.HealthHandler != nil {
			api.GET("/health", deps.HealthHandler.HealthCheck)
		}

		// Auth Group
		if deps.AuthHandler != nil {
			authGroup := api.Group("/auth")
			{
				authGroup.POST("/signup", deps.AuthHandler.Signup)
				authGroup.POST("/login", deps.AuthHandler.Login)

				// Protected route
				if deps.AuthService != nil {
					authGroup.GET("/me", middleware.AuthMiddleware(deps.AuthService), deps.AuthHandler.GetMe)
				}
			}
		}

		// Polls Group
		if deps.PollHandler != nil {
			pollGroup := api.Group("/polls")
			{
				// Public poll browsing
				pollGroup.GET("", deps.PollHandler.List)
				pollGroup.GET("/:id", deps.PollHandler.GetByID)

				// Realtime SSE streaming for live poll results
				if deps.RealtimeHandler != nil {
					pollGroup.GET("/:id/events", deps.RealtimeHandler.StreamPollEvents)
					pollGroup.GET("/:id/stream", deps.RealtimeHandler.StreamPollEvents)
				}

				// Voting endpoints (accessible by guest or authenticated user)
				if deps.VoteHandler != nil {
					optAuth := middleware.OptionalAuthMiddleware(deps.AuthService)
					pollGroup.POST("/:id/vote", optAuth, deps.VoteHandler.CastVote)
					pollGroup.GET("/:id/vote-status", optAuth, deps.VoteHandler.GetVoteStatus)
				}

				// Authenticated poll management
				if deps.AuthService != nil {
					authProtected := pollGroup.Group("")
					authProtected.Use(middleware.AuthMiddleware(deps.AuthService))
					{
						authProtected.POST("", deps.PollHandler.Create)
						authProtected.GET("/my", deps.PollHandler.ListMyPolls)
						authProtected.PUT("/:id", deps.PollHandler.Update)
						authProtected.DELETE("/:id", deps.PollHandler.Delete)
						authProtected.POST("/:id/close", deps.PollHandler.Close)
					}
				}
			}
		}
	}

	return r
}
