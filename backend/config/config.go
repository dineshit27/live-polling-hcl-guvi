package config

import (
	"os"
	"strings"
)

type Config struct {
	Port               string
	MongoURI           string
	MongoDBName        string
	RedisAddr          string
	RedisPassword      string
	JWTSecret          string
	GinMode            string
	CORSAllowedOrigins []string
}

func LoadConfig() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mongoURI := os.Getenv("MONGODB_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017"
	}

	mongoDBName := os.Getenv("MONGODB_DATABASE")
	if mongoDBName == "" {
		mongoDBName = "polling_db"
	}

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	redisPassword := os.Getenv("REDIS_PASSWORD")

	ginMode := os.Getenv("GIN_MODE")
	if ginMode == "" {
		ginMode = "debug"
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		if ginMode == "release" {
			println("[Config] CRITICAL WARNING: Running in release mode without JWT_SECRET set in environment! Using fallback dev key. Please provide a secure JWT_SECRET via environment/secrets.")
		}
		jwtSecret = "dev_secret_key_change_in_production_12345"
	}

	corsOriginsEnv := os.Getenv("CORS_ALLOWED_ORIGINS")
	var corsOrigins []string
	if corsOriginsEnv != "" {
		for _, o := range strings.Split(corsOriginsEnv, ",") {
			trimmed := strings.TrimSpace(o)
			if trimmed != "" {
				corsOrigins = append(corsOrigins, trimmed)
			}
		}
	} else {
		if ginMode == "release" {
			corsOrigins = []string{"http://localhost:3000", "http://localhost:8080"}
		} else {
			corsOrigins = []string{"http://localhost:3000", "http://localhost:8080", "http://127.0.0.1:3000", "http://127.0.0.1:8080"}
		}
	}

	return &Config{
		Port:               port,
		MongoURI:           mongoURI,
		MongoDBName:        mongoDBName,
		RedisAddr:          redisAddr,
		RedisPassword:      redisPassword,
		JWTSecret:          jwtSecret,
		GinMode:            ginMode,
		CORSAllowedOrigins: corsOrigins,
	}
}
