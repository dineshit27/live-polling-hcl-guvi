package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

type HealthHandler struct {
	mongoClient *mongo.Client
	redisClient *redis.Client
}

func NewHealthHandler(mongoClient *mongo.Client, redisClient *redis.Client) *HealthHandler {
	return &HealthHandler{
		mongoClient: mongoClient,
		redisClient: redisClient,
	}
}

type HealthStatus struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Services  map[string]string `json:"services"`
}

func (h *HealthHandler) HealthCheck(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	services := make(map[string]string)
	overallStatus := "healthy"

	if h.mongoClient != nil {
		if err := h.mongoClient.Ping(ctx, readpref.Primary()); err != nil {
			services["mongodb"] = "unreachable: " + err.Error()
			overallStatus = "degraded"
		} else {
			services["mongodb"] = "connected"
		}
	} else {
		services["mongodb"] = "not_initialized"
	}

	if h.redisClient != nil {
		if err := h.redisClient.Ping(ctx).Err(); err != nil {
			services["redis"] = "unreachable: " + err.Error()
			overallStatus = "degraded"
		} else {
			services["redis"] = "connected"
		}
	} else {
		services["redis"] = "not_initialized"
	}

	statusCode := http.StatusOK
	if overallStatus == "degraded" {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, HealthStatus{
		Status:    overallStatus,
		Timestamp: time.Now().UTC(),
		Services:  services,
	})
}
