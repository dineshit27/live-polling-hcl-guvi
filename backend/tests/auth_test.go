package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"live-polling-backend/config"
	"live-polling-backend/handlers"
	"live-polling-backend/models"
	"live-polling-backend/repositories"
	"live-polling-backend/routes"
	"live-polling-backend/services"
)

func setupAuthTestEnv(t *testing.T) (*repositories.MongoClient, http.Handler) {
	client := setupTestMongoDB(t)
	if client == nil {
		t.Skip("MongoDB not available")
		return nil, nil
	}

	cfg := &config.Config{
		Port:               "8081",
		GinMode:            "test",
		JWTSecret:          "test_jwt_secret_at_least_32_characters_long",
		CORSAllowedOrigins: []string{"*"},
	}

	userRepo := repositories.NewUserRepository(client.Database)
	authService := services.NewAuthService(userRepo, cfg.JWTSecret)
	authHandler := handlers.NewAuthHandler(authService)
	healthHandler := handlers.NewHealthHandler(client.Client, nil)

	router := routes.SetupRouter(&routes.RouterDeps{
		Config:        cfg,
		HealthHandler: healthHandler,
		AuthHandler:   authHandler,
		AuthService:   authService,
	})

	return client, router
}

func TestAuth_SignupSuccessAndValidation(t *testing.T) {
	client, router := setupAuthTestEnv(t)
	if client == nil {
		return
	}
	defer client.Close(context.Background())

	// 1. Validation error: password too short (< 8 chars)
	invalidPayload := map[string]string{
		"name":     "Bob Smith",
		"email":    "bob@example.com",
		"password": "short",
	}
	body, _ := json.Marshal(invalidPayload)
	req, _ := http.NewRequest(http.MethodPost, "/api/auth/signup", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected status 400 on short password validation failure, got %d: %s", w.Code, w.Body.String())
	}

	// 2. Successful signup
	validPayload := map[string]string{
		"name":     "Bob Valid",
		"email":    "bob.valid@example.com",
		"password": "strongPassword123!",
	}
	body, _ = json.Marshal(validPayload)
	req, _ = http.NewRequest(http.MethodPost, "/api/auth/signup", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Expected status 201 on valid signup, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			Token string      `json:"token"`
			User  models.User `json:"user"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse signup response: %v", err)
	}

	if resp.Data.Token == "" {
		t.Fatalf("Expected non-empty JWT token on signup")
	}
	if resp.Data.User.Email != "bob.valid@example.com" {
		t.Fatalf("Expected email bob.valid@example.com, got %s", resp.Data.User.Email)
	}

	// 3. Duplicate signup returns 409 Conflict
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPost, "/api/auth/signup", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("Expected status 409 Conflict on duplicate email, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuth_LoginAndProtectedGetMe(t *testing.T) {
	client, router := setupAuthTestEnv(t)
	if client == nil {
		return
	}
	defer client.Close(context.Background())

	// Pre-signup user
	signupPayload := map[string]string{
		"name":     "Charlie Auth",
		"email":    "charlie@example.com",
		"password": "charliePassword99!",
	}
	body, _ := json.Marshal(signupPayload)
	req, _ := http.NewRequest(http.MethodPost, "/api/auth/signup", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("Failed to prepare user for login test: %s", w.Body.String())
	}

	// 1. Invalid Password Login
	invalidLogin := map[string]string{
		"email":    "charlie@example.com",
		"password": "wrongPassword!",
	}
	body, _ = json.Marshal(invalidLogin)
	req, _ = http.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("Expected 401 on wrong password, got %d", w.Code)
	}

	// 2. Valid Login
	validLogin := map[string]string{
		"email":    "charlie@example.com",
		"password": "charliePassword99!",
	}
	body, _ = json.Marshal(validLogin)
	req, _ = http.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 on successful login, got %d", w.Code)
	}

	var loginResp struct {
		Data struct {
			Token string      `json:"token"`
			User  models.User `json:"user"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &loginResp)
	token := loginResp.Data.Token
	if token == "" {
		t.Fatalf("Expected valid JWT token from login")
	}

	// 3. Access Protected /api/auth/me WITHOUT token -> 401 Unauthorized
	req, _ = http.NewRequest(http.MethodGet, "/api/auth/me", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("Expected 401 when calling /api/auth/me without token, got %d", w.Code)
	}

	// 4. Access Protected /api/auth/me WITH Bearer Token -> 200 OK
	req, _ = http.NewRequest(http.MethodGet, "/api/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 with valid Bearer token, got %d: %s", w.Code, w.Body.String())
	}

	var meResp struct {
		Data models.User `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &meResp)
	if meResp.Data.Email != "charlie@example.com" {
		t.Fatalf("Expected user email charlie@example.com, got %s", meResp.Data.Email)
	}
}
