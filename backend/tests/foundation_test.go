package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"live-polling-backend/config"
	"live-polling-backend/handlers"
	"live-polling-backend/routes"
)

func TestHealthCheck(t *testing.T) {
	cfg := &config.Config{
		Port:               "8081",
		GinMode:            "test",
		CORSAllowedOrigins: []string{"*"},
	}

	healthHandler := handlers.NewHealthHandler(nil, nil)
	r := routes.SetupRouter(&routes.RouterDeps{
		Config:        cfg,
		HealthHandler: healthHandler,
	})

	req, _ := http.NewRequest(http.MethodGet, "/api/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status code 200, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestCORSHeaders(t *testing.T) {
	cfg := &config.Config{
		Port:               "8081",
		GinMode:            "test",
		CORSAllowedOrigins: []string{"http://localhost:3000"},
	}

	healthHandler := handlers.NewHealthHandler(nil, nil)
	r := routes.SetupRouter(&routes.RouterDeps{
		Config:        cfg,
		HealthHandler: healthHandler,
	})

	req, _ := http.NewRequest(http.MethodOptions, "/api/health", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("Expected preflight 204 No Content, got %d", w.Code)
	}

	allowOrigin := w.Header().Get("Access-Control-Allow-Origin")
	if allowOrigin != "http://localhost:3000" {
		t.Fatalf("Expected Access-Control-Allow-Origin to be http://localhost:3000, got %s", allowOrigin)
	}
}
