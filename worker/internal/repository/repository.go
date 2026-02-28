package repository

import (
	"govision/worker/internal/domain"
)

// PredictionRepository defines the contract for persisting job results
// and their associated predictions.
type PredictionRepository interface {
	CreatePendingJob(jobID string, imageURL string) error
	SaveJobResult(job domain.JobResult) error
}
