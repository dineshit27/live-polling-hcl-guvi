package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"github.com/redis/go-redis/v9"

	"live-polling-backend/config"
	"live-polling-backend/handlers"
	"live-polling-backend/realtime"
	"live-polling-backend/repositories"
	"live-polling-backend/routes"
	"live-polling-backend/services"
)

func main() {
	cfg := config.LoadConfig()

	log.Printf("Starting Live Polling Service on port %s (mode: %s)", cfg.Port, cfg.GinMode)

	// Connect to MongoDB with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	mongoClient, err := repositories.ConnectMongoDB(ctx, cfg.MongoURI, cfg.MongoDBName)
	if err != nil {
		log.Printf("[MongoDB] Warning: Initial MongoDB connection failed: %v", err)
	}

	var rawClient *mongo.Client
	if mongoClient != nil {
		rawClient = mongoClient.Client
	}

	// Connect to Redis for Phase 8 Pub/Sub Event Layer
	redisCtx, redisCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer redisCancel()

	var rawRedisClient *redis.Client
	var eventPublisher realtime.EventPublisher

	redisClient, err := repositories.ConnectRedis(redisCtx, cfg.RedisAddr, cfg.RedisPassword, 0)
	if err != nil {
		log.Printf("[Redis] Warning: Initial Redis connection failed: %v", err)
	} else if redisClient != nil {
		rawRedisClient = redisClient.Client
		eventPublisher = realtime.NewRedisEventPublisher(rawRedisClient)
		log.Printf("[Realtime] Successfully initialized Redis Pub/Sub event layer at %s", cfg.RedisAddr)
	}

	// Initialize Repositories if MongoDB is active
	var userRepo repositories.UserRepository
	var pollRepo repositories.PollRepository
	var voteRepo repositories.VoteRepository

	if mongoClient != nil && mongoClient.Database != nil {
		userRepo = repositories.NewUserRepository(mongoClient.Database)
		pollRepo = repositories.NewPollRepository(mongoClient.Database)
		voteRepo = repositories.NewVoteRepository(mongoClient.Database)
		log.Printf("[Repositories] Successfully initialized User, Poll, and Vote repositories on database '%s'", cfg.MongoDBName)
	}

	// Initialize Services & Handlers
	var authService services.AuthService
	var authHandler *handlers.AuthHandler
	var pollService services.PollService
	var pollHandler *handlers.PollHandler
	var voteService services.VoteService
	var voteHandler *handlers.VoteHandler
	var realtimeHandler *handlers.RealtimeHandler

	if userRepo != nil {
		authService = services.NewAuthService(userRepo, cfg.JWTSecret)
		authHandler = handlers.NewAuthHandler(authService)
		log.Printf("[Services] Initialized AuthService and AuthHandler")
	}

	if pollRepo != nil {
		pollService = services.NewPollService(pollRepo, eventPublisher)
		pollHandler = handlers.NewPollHandler(pollService)
		log.Printf("[Services] Initialized PollService and PollHandler (PubSub: %v)", eventPublisher != nil)
	}

	if pollRepo != nil && voteRepo != nil {
		voteService = services.NewVoteService(pollRepo, voteRepo, eventPublisher)
		voteHandler = handlers.NewVoteHandler(voteService)
		log.Printf("[Services] Initialized VoteService and VoteHandler (PubSub: %v)", eventPublisher != nil)
	}

	if rawRedisClient != nil && pollService != nil {
		eventSubscriber := realtime.NewRedisEventSubscriber(rawRedisClient)
		realtimeHandler = handlers.NewRealtimeHandler(pollService, eventSubscriber)
		log.Printf("[Services] Initialized RealtimeHandler (SSE endpoint backed by Redis Pub/Sub)")
	}

	// Initialize Health Handler with real MongoDB client and real Redis client
	healthHandler := handlers.NewHealthHandler(rawClient, rawRedisClient)

	// Wire Router with Middleware and Handlers
	r := routes.SetupRouter(&routes.RouterDeps{
		Config:          cfg,
		HealthHandler:   healthHandler,
		AuthHandler:     authHandler,
		AuthService:     authService,
		PollHandler:     pollHandler,
		VoteHandler:     voteHandler,
		RealtimeHandler: realtimeHandler,
	})

	addr := fmt.Sprintf("0.0.0.0:%s", cfg.Port)
	log.Printf("Server listening on http://%s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Fatal: server terminated with error: %v", err)
	}
}
