package middlewares

import (
	"net/http"
	"strings"

	"govision/api/internal/modules/auth"

	"github.com/labstack/echo/v4"
)

// JWTAuth returns a middleware that validates the Bearer token in the
// Authorization header using the provided auth service.
func JWTAuth(authService *auth.Service) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			header := c.Request().Header.Get("Authorization")
			if header == "" {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"message": "Missing authorization header",
				})
			}

			parts := strings.SplitN(header, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"message": "Invalid authorization format. Expected: Bearer <token>",
				})
			}

			claims, err := authService.ValidateAccessToken(parts[1])
			if err != nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"message": "Invalid or expired token",
				})
			}

			// Store claims in context for downstream handlers
			c.Set("user_id", claims.UserID)
			c.Set("email", claims.Email)

			return next(c)
		}
	}
}
