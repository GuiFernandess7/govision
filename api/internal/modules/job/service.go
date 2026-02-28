package job

import (
	"fmt"
	"log"

	"gorm.io/gorm"
)

// Service handles the business logic for job queries.
type Service struct {
	repo JobRepository
}

// NewService creates a new job service.
func NewService(repo JobRepository) *Service {
	return &Service{repo: repo}
}

// GetJobStatus retrieves the current status and details of a job by its ID.
func (s *Service) GetJobStatus(jobID string) (*JobStatusResponse, error) {
	log.Printf("[RUNNING] - Querying job status for ID: %s", jobID)

	job, err := s.repo.FindByJobID(jobID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("job not found: %s", jobID)
		}
		return nil, fmt.Errorf("error querying job: %w", err)
	}

	response := &JobStatusResponse{
		JobID:       job.JobID,
		ImageURL:    job.ImageURL,
		Status:      job.Status,
		ProcessedAt: job.ProcessedAt,
		CreatedAt:   job.CreatedAt,
		Predictions: job.Predictions,
	}

	log.Printf("[SUCCESS] - Job %s found with status: %s", jobID, job.Status)
	return response, nil
}
