package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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

type pollTestEnv struct {
	client     *repositories.MongoClient
	router     http.Handler
	ownerToken string
	ownerID    string
	otherToken string
	otherID    string
}

func setupPollTestEnv(t *testing.T) *pollTestEnv {
	client := setupTestMongoDB(t)
	if client == nil {
		t.Skip("MongoDB not available")
		return nil
	}

	cfg := &config.Config{
		Port:               "8081",
		GinMode:            "test",
		JWTSecret:          "test_jwt_secret_at_least_32_characters_long",
		CORSAllowedOrigins: []string{"*"},
	}

	userRepo := repositories.NewUserRepository(client.Database)
	pollRepo := repositories.NewPollRepository(client.Database)

	authService := services.NewAuthService(userRepo, cfg.JWTSecret)
	authHandler := handlers.NewAuthHandler(authService)

	pollService := services.NewPollService(pollRepo)
	pollHandler := handlers.NewPollHandler(pollService)

	healthHandler := handlers.NewHealthHandler(client.Client, nil)

	router := routes.SetupRouter(&routes.RouterDeps{
		Config:        cfg,
		HealthHandler: healthHandler,
		AuthHandler:   authHandler,
		AuthService:   authService,
		PollHandler:   pollHandler,
	})

	ctx := context.Background()

	// Register Owner
	ownerSignup, err := authService.Signup(ctx, &models.UserSignupRequest{
		Name:     "Poll Creator",
		Email:    "creator@example.com",
		Password: "creatorPassword123!",
	})
	if err != nil {
		t.Fatalf("Failed to create owner user: %v", err)
	}

	// Register Other User
	otherSignup, err := authService.Signup(ctx, &models.UserSignupRequest{
		Name:     "Other User",
		Email:    "other@example.com",
		Password: "otherPassword123!",
	})
	if err != nil {
		t.Fatalf("Failed to create other user: %v", err)
	}

	return &pollTestEnv{
		client:     client,
		router:     router,
		ownerToken: ownerSignup.Token,
		ownerID:    ownerSignup.User.ID,
		otherToken: otherSignup.Token,
		otherID:    otherSignup.User.ID,
	}
}

func TestPoll_CreateAndValidation(t *testing.T) {
	env := setupPollTestEnv(t)
	if env == nil {
		return
	}
	defer env.client.Close(context.Background())

	// 1. Unauthorized create without token -> 401
	payload := map[string]interface{}{
		"title":   "What is your stack?",
		"options": []string{"Go", "Node"},
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest(http.MethodPost, "/api/polls", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("Expected 401 when creating poll without token, got %d", w.Code)
	}

	// 2. Validation error: less than 2 options -> 400
	invalidPayload := map[string]interface{}{
		"title":   "Only one option poll",
		"options": []string{"SingleOption"},
	}
	body, _ = json.Marshal(invalidPayload)
	req, _ = http.NewRequest(http.MethodPost, "/api/polls", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+env.ownerToken)
	w = httptest.NewRecorder()
	env.router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected 400 when options < 2, got %d: %s", w.Code, w.Body.String())
	}

	// 3. Validation error: duplicate options -> 400
	dupPayload := map[string]interface{}{
		"title":   "Poll with duplicates",
		"options": []string{"Option A", "Option a"},
	}
	body, _ = json.Marshal(dupPayload)
	req, _ = http.NewRequest(http.MethodPost, "/api/polls", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+env.ownerToken)
	w = httptest.NewRecorder()
	env.router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected 400 on duplicate options, got %d: %s", w.Code, w.Body.String())
	}

	// 4. Valid creation by owner -> 201 Created
	validPayload := map[string]interface{}{
		"title":       "Which cloud runtime do you prefer?",
		"description": "Evaluating microservices platforms",
		"options":     []string{"Cloud Run", "Kubernetes", "AWS ECS"},
	}
	body, _ = json.Marshal(validPayload)
	req, _ = http.NewRequest(http.MethodPost, "/api/polls", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+env.ownerToken)
	w = httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Expected 201 on valid poll creation, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Success bool        `json:"success"`
		Data    models.Poll `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Data.ID == "" {
		t.Fatalf("Expected poll ID to be generated")
	}
	if resp.Data.OwnerID != env.ownerID {
		t.Fatalf("Expected owner ID '%s', got '%s'", env.ownerID, resp.Data.OwnerID)
	}
	if len(resp.Data.Options) != 3 {
		t.Fatalf("Expected 3 options, got %d", len(resp.Data.Options))
	}
}

func TestPoll_OwnershipAuthorization_UpdateDeleteClose(t *testing.T) {
	env := setupPollTestEnv(t)
	if env == nil {
		return
	}
	defer env.client.Close(context.Background())

	// 1. Create a poll as Owner
	validPayload := map[string]interface{}{
		"title":   "Which cache store do you use?",
		"options": []string{"Redis", "Memcached", "Dragonfly"},
	}
	body, _ := json.Marshal(validPayload)
	req, _ := http.NewRequest(http.MethodPost, "/api/polls", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+env.ownerToken)
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	var createResp struct {
		Data models.Poll `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &createResp)
	pollID := createResp.Data.ID

	// 2. Non-owner attempts to update poll -> 403 Forbidden
	updatePayload := map[string]interface{}{
		"title": "Hacked Title Attempt",
	}
	body, _ = json.Marshal(updatePayload)
	req, _ = http.NewRequest(http.MethodPut, "/api/polls/"+pollID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+env.otherToken)
	w = httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("Expected 403 Forbidden when non-owner updates poll, got %d: %s", w.Code, w.Body.String())
	}

	// 3. Non-owner attempts to delete poll -> 403 Forbidden
	req, _ = http.NewRequest(http.MethodDelete, "/api/polls/"+pollID, nil)
	req.Header.Set("Authorization", "Bearer "+env.otherToken)
	w = httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("Expected 403 Forbidden when non-owner deletes poll, got %d: %s", w.Code, w.Body.String())
	}

	// 4. Non-owner attempts to close poll -> 403 Forbidden
	req, _ = http.NewRequest(http.MethodPost, "/api/polls/"+pollID+"/close", nil)
	req.Header.Set("Authorization", "Bearer "+env.otherToken)
	w = httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("Expected 403 Forbidden when non-owner closes poll, got %d: %s", w.Code, w.Body.String())
	}

	// 5. Owner successfully updates poll -> 200 OK
	ownerUpdatePayload := map[string]interface{}{
		"title": "Which caching engine is best?",
	}
	body, _ = json.Marshal(ownerUpdatePayload)
	req, _ = http.NewRequest(http.MethodPut, "/api/polls/"+pollID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+env.ownerToken)
	w = httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK when owner updates poll, got %d: %s", w.Code, w.Body.String())
	}

	// 6. Owner closes poll -> 200 OK
	req, _ = http.NewRequest(http.MethodPost, "/api/polls/"+pollID+"/close", nil)
	req.Header.Set("Authorization", "Bearer "+env.ownerToken)
	w = httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK when owner closes poll, got %d: %s", w.Code, w.Body.String())
	}

	var closeResp struct {
		Data models.Poll `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &closeResp)
	if closeResp.Data.Status != "closed" {
		t.Fatalf("Expected poll status 'closed', got '%s'", closeResp.Data.Status)
	}

	// 7. Owner deletes poll -> 200 OK
	req, _ = http.NewRequest(http.MethodDelete, "/api/polls/"+pollID, nil)
	req.Header.Set("Authorization", "Bearer "+env.ownerToken)
	w = httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK when owner deletes poll, got %d: %s", w.Code, w.Body.String())
	}

	// 8. Poll is no longer found -> 404
	req, _ = http.NewRequest(http.MethodGet, fmt.Sprintf("/api/polls/%s", pollID), nil)
	w = httptest.NewRecorder()
	env.router.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("Expected 404 after deletion, got %d", w.Code)
	}
}
