package middlewares

import (
	"log"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func LoggingMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			// Log the request with timestamp
			log.Printf("[%s] %s %s - Route called at: %s",
				c.Request().Method,
				c.Request().URL.Path,
				c.RealIP(),
				start.Format("2006-01-02 15:04:05.000"))

			// Process request
			err := next(c)

			// Log response time
			duration := time.Since(start)
			log.Printf("[%s] %s - Completed in %v - Status: %d",
				c.Request().Method,
				c.Request().URL.Path,
				duration,
				c.Response().Status)

			return err
		}
	}
}

func ApplySecurityMiddlewares(e *echo.Echo) *echo.Echo {
	e.Use(middleware.Recover())
	e.Use(LoggingMiddleware())
	e.Use(middleware.CORS())
	e.Use(middleware.Secure())
	e.Use(middleware.BodyLimit("20M"))

	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			w := c.Response().Writer
			w.Header().Set("X-DNS-Prefetch-Control", "off")
			w.Header().Set("X-Permitted-Cross-Domain-Policies", "none")
			w.Header().Set("Cross-Origin-Resource-Policy", "same-origin")
			w.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
			w.Header().Set("Cross-Origin-Embedder-Policy", "require-corp")
			w.Header().Set("Permissions-Policy", "geolocation=(self), microphone=()")
			w.Header().Set("X-Powered-By", "Django")
			w.Header().Set("Server", "")
			w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
			return next(c)
		}
	})
	return e
}