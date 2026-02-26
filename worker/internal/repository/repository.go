package repository

import (
	"context"

	"govision/worker/internal/domain"
)

// PredictionRepository defines the contract for persisting job results
// and their associated predictions.
type PredictionRepository interface {
	SaveJobResult(ctx context.Context, job domain.JobResult) error
}
