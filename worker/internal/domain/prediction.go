package domain

import "time"

type Prediction struct {
	X           float64 `json:"x"`
	Y           float64 `json:"y"`
	Width       float64 `json:"width"`
	Height      float64 `json:"height"`
	Confidence  float64 `json:"confidence"`
	Class       string  `json:"class"`
	ClassID     int     `json:"class_id"`
	DetectionID string  `json:"detection_id"`
	ParentID    string  `json:"parent_id"`
}

// RoboflowResponse represents the top-level response from the Roboflow Serverless API.
type RoboflowResponse struct {
	Predictions []Prediction `json:"predictions"`
}

// JobResult represents a completed job with its predictions,
// ready to be persisted in the database.
type JobResult struct {
	JobID        string
	ImageURL     string
	Status       string
	ProcessedAt  time.Time
	CountObjects int
	Predictions  []Prediction
}
