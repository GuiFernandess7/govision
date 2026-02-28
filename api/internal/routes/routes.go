package routes

import (
	"govision/api/internal/modules/file"
	"govision/api/internal/modules/job"

	"github.com/labstack/echo/v4"
)

func InitRoutes(e *echo.Echo, fileHandler *file.Handler, jobHandler *job.Handler) {
	v1 := e.Group("/v1")
	v1.POST("/image/upload", fileHandler.UploadFileImage)
	v1.GET("/jobs/:id", jobHandler.GetJobStatus)
}
