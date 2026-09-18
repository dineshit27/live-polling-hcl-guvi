package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"live-polling-backend/models"
	"live-polling-backend/repositories"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrInvalidToken       = errors.New("invalid or expired token")
)

type JWTClaims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Name   string `json:"name"`
	jwt.RegisteredClaims
}

type AuthService interface {
	Signup(ctx context.Context, req *models.UserSignupRequest) (*models.AuthResponse, error)
	Login(ctx context.Context, req *models.UserLoginRequest) (*models.AuthResponse, error)
	ValidateToken(tokenString string) (*JWTClaims, error)
	GetProfile(ctx context.Context, userID string) (*models.User, error)
}

type authService struct {
	userRepo  repositories.UserRepository
	jwtSecret string
}

func NewAuthService(userRepo repositories.UserRepository, jwtSecret string) AuthService {
	if jwtSecret == "" {
		jwtSecret = "dev_secret_fallback_key_32_chars_long"
	}
	return &authService{
		userRepo:  userRepo,
		jwtSecret: jwtSecret,
	}
}

func (s *authService) Signup(ctx context.Context, req *models.UserSignupRequest) (*models.AuthResponse, error) {
	// Normalize email
	email := strings.ToLower(strings.TrimSpace(req.Email))
	name := strings.TrimSpace(req.Name)

	// Hash password using bcrypt
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &models.User{
		Name:         name,
		Email:        email,
		PasswordHash: string(hashedPassword),
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	// Generate JWT
	token, err := s.generateToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate auth token: %w", err)
	}

	return &models.AuthResponse{
		Token: token,
		User:  *user,
	}, nil
}

func (s *authService) Login(ctx context.Context, req *models.UserLoginRequest) (*models.AuthResponse, error) {
	email := strings.ToLower(strings.TrimSpace(req.Email))

	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repositories.ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	// Compare bcrypt password hash
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	token, err := s.generateToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate auth token: %w", err)
	}

	return &models.AuthResponse{
		Token: token,
		User:  *user,
	}, nil
}

func (s *authService) GetProfile(ctx context.Context, userID string) (*models.User, error) {
	return s.userRepo.FindByID(ctx, userID)
}

func (s *authService) generateToken(user *models.User) (string, error) {
	expirationTime := time.Now().Add(72 * time.Hour) // 3 days validity
	claims := &JWTClaims{
		UserID: user.ID,
		Email:  user.Email,
		Name:   user.Name,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "live-polling-api",
			Subject:   user.ID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}

func (s *authService) ValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.jwtSecret), nil
	})

	if err != nil {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
