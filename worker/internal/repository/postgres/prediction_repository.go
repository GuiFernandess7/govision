package postgres

import (
	"context"
	"fmt"
	"log"

	"govision/worker/internal/domain"
	"govision/worker/internal/repository"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PredictionRepository implements repository.PredictionRepository
// using PostgreSQL as the backing store.
type PredictionRepository struct {
	pool *pgxpool.Pool
}

// NewPredictionRepository creates a new PostgreSQL-backed prediction repository.
func NewPredictionRepository(pool *pgxpool.Pool) repository.PredictionRepository {
	return &PredictionRepository{pool: pool}
}

// SaveJobResult persists a completed job and all its predictions in a single
// database transaction. If any step fails, the entire operation is rolled back.
func (r *PredictionRepository) SaveJobResult(ctx context.Context, result domain.JobResult) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	_, err = tx.Exec(ctx,
		`INSERT INTO jobs (job_id, image_url, status, processed_at)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (job_id) DO UPDATE SET
		     status = EXCLUDED.status,
		     processed_at = EXCLUDED.processed_at`,
		result.JobID, result.ImageURL, result.Status, result.ProcessedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert job: %w", err)
	}

	if len(result.Predictions) > 0 {
		batch := &pgx.Batch{}
		for _, p := range result.Predictions {
			batch.Queue(
				`INSERT INTO predictions (job_id, x, y, width, height, confidence, class, class_id)
				 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
				result.JobID, p.X, p.Y, p.Width, p.Height, p.Confidence, p.Class, p.ClassID,
			)
		}

		br := tx.SendBatch(ctx, batch)
		for i := 0; i < len(result.Predictions); i++ {
			if _, err = br.Exec(); err != nil {
				_ = br.Close()
				return fmt.Errorf("failed to insert prediction %d: %w", i, err)
			}
		}
		if err = br.Close(); err != nil {
			return fmt.Errorf("failed to close batch: %w", err)
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Printf("[POSTGRES] - Job %s saved with %d prediction(s)", result.JobID, len(result.Predictions))
	return nil
}
