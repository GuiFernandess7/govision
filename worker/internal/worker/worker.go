package worker

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"time"

	"govision/worker/internal/domain"
	"govision/worker/internal/repository"
	"govision/worker/internal/services/roboflow"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Worker processes jobs from the RabbitMQ queue by sending images
// to the Roboflow API for inference and persisting results to the database.
type Worker struct {
	roboflow *roboflow.Client
	repo     repository.PredictionRepository
}

// New creates a new Worker with the given Roboflow client and prediction repository.
func New(rf *roboflow.Client, repo repository.PredictionRepository) *Worker {
	return &Worker{roboflow: rf, repo: repo}
}

// ProcessMessages listens for incoming AMQP deliveries, decodes the
// job message, sends the image to Roboflow and logs the results.
// Each message is acknowledged individually after processing.
func (w *Worker) ProcessMessages(ctx context.Context, msgs <-chan amqp.Delivery) {
	for {
		select {
		case <-ctx.Done():
			log.Println("[WORKER] - Context cancelled, stopping worker.")
			return

		case msg, ok := <-msgs:
			if !ok {
				log.Println("[WORKER] - Delivery channel closed, stopping worker.")
				return
			}

			w.handleMessage(ctx, msg)
		}
	}
}

func (w *Worker) handleMessage(ctx context.Context, msg amqp.Delivery) {
	var job domain.JobMessage
	if err := json.Unmarshal(msg.Body, &job); err != nil {
		log.Printf("[WORKER] - Failed to decode message: %v", err)
		_ = msg.Nack(false, false)
		return
	}

	log.Printf("[WORKER] - Processing job %s | Image: %s", job.JobID, job.ImageURL)

	result, err := w.roboflow.Detect(ctx, job.ImageURL)
	if err != nil {
		var apiErr *roboflow.APIError
		if errors.As(err, &apiErr) {
			log.Printf("[WORKER] - Job %s failed with non-retryable error (HTTP %d): %s", job.JobID, apiErr.StatusCode, apiErr.Body)
			_ = msg.Nack(false, false)
			return
		}
		log.Printf("[WORKER] - Job %s failed: %v", job.JobID, err)
		_ = msg.Nack(false, false)
		return
	}

	jobResult := domain.JobResult{
		JobID:       job.JobID,
		ImageURL:    job.ImageURL,
		Status:      "completed",
		ProcessedAt: time.Now(),
		Predictions: result.Predictions,
	}

	if err := w.repo.SaveJobResult(jobResult); err != nil {
		log.Printf("[WORKER] - Job %s: failed to save results to database: %v", job.JobID, err)
		_ = msg.Nack(false, true)
		return
	}

	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	log.Printf("[WORKER] - Job %s completed and saved. Result:\n%s", job.JobID, string(resultJSON))

	_ = msg.Ack(false)
}
