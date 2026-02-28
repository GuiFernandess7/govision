package auth

import (
	"log"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

// Handler exposes HTTP endpoints for authentication.
type Handler struct {
	service *Service
}

// NewHandler creates a new auth handler with its dependencies.
func NewHandler(db *gorm.DB, jwtSecret string) *Handler {
	repo := NewAuthRepository(db)
	return &Handler{service: NewService(repo, jwtSecret)}
}

// Register handles POST /auth/register.
func (h *Handler) Register(c echo.Context) error {
	log.Println("[STARTING] - calling route /auth/register...")

	var req RegisterRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"message": "Invalid payload",
		})
	}

	if err := ValidateRegisterRequest(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"message": err.Error(),
		})
	}

	if err := h.service.Register(req.Email, req.Password); err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			return c.JSON(http.StatusConflict, map[string]string{
				"message": "Email already registered",
			})
		}
		log.Printf("[ERROR] - %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"message": "Error creating user",
		})
	}

	return c.JSON(http.StatusCreated, map[string]string{
		"message": "User registered successfully",
	})
}

// Login handles POST /auth/login.
func (h *Handler) Login(c echo.Context) error {
	log.Println("[STARTING] - calling route /auth/login...")

	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"message": "Invalid payload",
		})
	}

	if err := ValidateLoginRequest(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"message": err.Error(),
		})
	}

	tokens, err := h.service.Login(req.Email, req.Password)
	if err != nil {
		if strings.Contains(err.Error(), "invalid credentials") {
			return c.JSON(http.StatusUnauthorized, map[string]string{
				"message": "Invalid credentials",
			})
		}
		log.Printf("[ERROR] - %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"message": "Error during login",
		})
	}

	return c.JSON(http.StatusOK, tokens)
}

// Refresh handles POST /auth/refresh.
func (h *Handler) Refresh(c echo.Context) error {
	log.Println("[STARTING] - calling route /auth/refresh...")

	var req RefreshRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"message": "Invalid payload",
		})
	}

	if req.RefreshToken == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"message": "Refresh token is required",
		})
	}

	tokens, err := h.service.Refresh(req.RefreshToken)
	if err != nil {
		if strings.Contains(err.Error(), "invalid refresh token") || strings.Contains(err.Error(), "expired") {
			return c.JSON(http.StatusUnauthorized, map[string]string{
				"message": err.Error(),
			})
		}
		log.Printf("[ERROR] - %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"message": "Error refreshing token",
		})
	}

	return c.JSON(http.StatusOK, tokens)
}

// GetService exposes the service for middleware usage.
func (h *Handler) GetService() *Service {
	return h.service
}
