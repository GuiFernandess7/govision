package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	accessTokenExpiry  = 15 * time.Minute
	refreshTokenExpiry = 7 * 24 * time.Hour
	bcryptCost         = 12
	refreshTokenBytes  = 32
)

// JWTClaims defines the custom claims embedded in the access token.
type JWTClaims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

// Service handles authentication business logic.
type Service struct {
	repo      AuthRepository
	jwtSecret []byte
}

// NewService creates a new auth service.
func NewService(repo AuthRepository, jwtSecret string) *Service {
	return &Service{
		repo:      repo,
		jwtSecret: []byte(jwtSecret),
	}
}

// Register creates a new user with a hashed password.
func (s *Service) Register(email, password string) error {
	log.Printf("[AUTH] - Registering user: %s", email)

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	user := &User{
		Email:        email,
		PasswordHash: string(hash),
	}

	if err := s.repo.CreateUser(user); err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	log.Printf("[AUTH] - User %s registered successfully", email)
	return nil
}

// Login validates credentials and returns an access/refresh token pair.
func (s *Service) Login(email, password string) (*TokenResponse, error) {
	log.Printf("[AUTH] - Login attempt for: %s", email)

	user, err := s.repo.FindUserByEmail(email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("invalid credentials")
		}
		return nil, fmt.Errorf("error finding user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := s.createRefreshToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	log.Printf("[AUTH] - User %s logged in successfully", email)
	return &TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(accessTokenExpiry.Seconds()),
	}, nil
}

// Refresh validates a refresh token and issues a new access/refresh token pair.
// The old refresh token is revoked (rotation).
func (s *Service) Refresh(rawRefreshToken string) (*TokenResponse, error) {
	log.Println("[AUTH] - Processing token refresh")

	tokenHash := hashToken(rawRefreshToken)

	stored, err := s.repo.FindRefreshTokenByHash(tokenHash)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("invalid refresh token")
		}
		return nil, fmt.Errorf("error validating refresh token: %w", err)
	}

	if time.Now().After(stored.ExpiresAt) {
		_ = s.repo.RevokeRefreshToken(stored.ID)
		return nil, fmt.Errorf("refresh token expired")
	}

	// Revoke the used refresh token (rotation)
	if err := s.repo.RevokeRefreshToken(stored.ID); err != nil {
		return nil, fmt.Errorf("failed to revoke old refresh token: %w", err)
	}

	user, err := s.repo.FindUserByID(stored.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	newRefreshToken, err := s.createRefreshToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	log.Println("[AUTH] - Token refreshed successfully")
	return &TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    int64(accessTokenExpiry.Seconds()),
	}, nil
}

// ValidateAccessToken parses and validates a JWT access token.
func (s *Service) ValidateAccessToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}

func (s *Service) generateAccessToken(user *User) (string, error) {
	now := time.Now()
	claims := JWTClaims{
		UserID: user.ID.String(),
		Email:  user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(accessTokenExpiry)),
			Issuer:    "govision",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

func (s *Service) createRefreshToken(userID uuid.UUID) (string, error) {
	rawToken, err := generateRandomToken()
	if err != nil {
		return "", err
	}

	tokenHash := hashToken(rawToken)

	rt := &RefreshToken{
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(refreshTokenExpiry),
	}

	if err := s.repo.SaveRefreshToken(rt); err != nil {
		return "", fmt.Errorf("failed to save refresh token: %w", err)
	}

	return rawToken, nil
}

func generateRandomToken() (string, error) {
	b := make([]byte, refreshTokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
