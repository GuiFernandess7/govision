package routes

import (
	"govision/internal/modules/file"

	"github.com/labstack/echo/v4"
)

func InitRoutes(e *echo.Echo, fileHandler *file.Handler) {
	v1 := e.Group("/v1")
	v1.POST("/image/upload", fileHandler.UploadFileImage)
}
