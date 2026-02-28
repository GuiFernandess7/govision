package postgres

import (
	"log"

	"govision/worker/internal/domain"
	"govision/worker/internal/repository"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// PredictionRepository implements repository.PredictionRepository
// using PostgreSQL as the backing store via GORM.
type PredictionRepository struct {
	db *gorm.DB
}

// NewPredictionRepository creates a new PostgreSQL-backed prediction repository.
func NewPredictionRepository(db *gorm.DB) repository.PredictionRepository {
	return &PredictionRepository{db: db}
}

// CreatePendingJob inserts a new job with status "pending" before processing begins.
// Uses ON CONFLICT to avoid duplicates if the job already exists.
func (r *PredictionRepository) CreatePendingJob(jobID string, imageURL string) error {
	job := domain.Job{
		JobID:    jobID,
		ImageURL: imageURL,
		Status:   "pending",
	}

	if err := r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "job_id"}},
		DoNothing: true,
	}).Create(&job).Error; err != nil {
		return err
	}

	log.Printf("[POSTGRES] - Job %s created with status 'pending'", jobID)
	return nil
}

// SaveJobResult persists a completed job and all its predictions in a single
// database transaction. If any step fails, the entire operation is rolled back.
func (r *PredictionRepository) SaveJobResult(result domain.JobResult) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		job := domain.Job{
			JobID:       result.JobID,
			ImageURL:    result.ImageURL,
			Status:      result.Status,
			ProcessedAt: result.ProcessedAt,
		}

		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "job_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"status", "processed_at"}),
		}).Create(&job).Error; err != nil {
			return err
		}

		if len(result.Predictions) > 0 {
			dbPredictions := make([]domain.DBPrediction, len(result.Predictions))
			for i, p := range result.Predictions {
				dbPredictions[i] = domain.DBPrediction{
					JobID:      result.JobID,
					X:          p.X,
					Y:          p.Y,
					Width:      p.Width,
					Height:     p.Height,
					Confidence: p.Confidence,
					Class:      p.Class,
					ClassID:    p.ClassID,
				}
			}

			if err := tx.Create(&dbPredictions).Error; err != nil {
				return err
			}
		}

		log.Printf("[POSTGRES] - Job %s saved with %d prediction(s)", result.JobID, len(result.Predictions))
		return nil
	})
}
