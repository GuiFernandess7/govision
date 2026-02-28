package routes

import (
	"govision/api/internal/middlewares"
	"govision/api/internal/modules/auth"
	"govision/api/internal/modules/file"
	"govision/api/internal/modules/job"

	"github.com/labstack/echo/v4"
)

func InitRoutes(e *echo.Echo, fileHandler *file.Handler, jobHandler *job.Handler, authHandler *auth.Handler) {
	v1 := e.Group("/v1")

	// Public routes
	v1.POST("/auth/register", authHandler.Register)
	v1.POST("/auth/login", authHandler.Login)
	v1.POST("/auth/refresh", authHandler.Refresh)

	// Protected routes
	protected := v1.Group("", middlewares.JWTAuth(authHandler.GetService()))
	protected.POST("/image/upload", fileHandler.UploadFileImage)
	protected.GET("/jobs/:id", jobHandler.GetJobStatus)
}
