package job

import (
	"log"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

// Handler exposes HTTP endpoints for job queries.
type Handler struct {
	service *Service
}

// NewHandler creates a new job handler with its dependencies.
func NewHandler(db *gorm.DB) *Handler {
	repo := NewJobRepository(db)
	return &Handler{service: NewService(repo)}
}

// GetJobStatus handles GET /jobs/:id and returns the job status and predictions.
func (h *Handler) GetJobStatus(c echo.Context) error {
	log.Println("[STARTING] - calling route /jobs/:id...")

	jobID := strings.TrimSpace(c.Param("id"))
	if jobID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"message": "Job ID is required",
		})
	}

	result, err := h.service.GetJobStatus(jobID)
	if err != nil {
		if strings.Contains(err.Error(), "job not found") {
			return c.JSON(http.StatusNotFound, map[string]string{
				"message": err.Error(),
			})
		}

		log.Printf("[ERROR] - %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"message": "Error retrieving job status",
		})
	}

	return c.JSON(http.StatusOK, result)
}
