package job

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Job represents the jobs table in the database.
type Job struct {
	ID          uuid.UUID    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	JobID       string       `gorm:"column:job_id;type:varchar(255);uniqueIndex;not null" json:"job_id"`
	ImageURL    string       `gorm:"column:image_url;type:text;not null" json:"image_url"`
	Status      string       `gorm:"column:status;type:varchar(50);not null;default:'completed'" json:"status"`
	ProcessedAt time.Time    `gorm:"column:processed_at;not null;default:now()" json:"processed_at"`
	CreatedAt   time.Time    `gorm:"column:created_at;not null;default:now()" json:"created_at"`
	Predictions []Prediction `gorm:"foreignKey:JobID;references:JobID" json:"predictions,omitempty"`
}

func (Job) TableName() string {
	return "jobs"
}

// BeforeCreate generates a UUID before inserting a new Job.
func (j *Job) BeforeCreate(tx *gorm.DB) error {
	if j.ID == uuid.Nil {
		j.ID = uuid.New()
	}
	return nil
}

// Prediction represents the predictions table in the database.
type Prediction struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	JobID      string    `gorm:"column:job_id;type:varchar(255);not null;index:idx_predictions_job_id" json:"job_id"`
	X          float64   `gorm:"column:x;type:double precision;not null" json:"x"`
	Y          float64   `gorm:"column:y;type:double precision;not null" json:"y"`
	Width      float64   `gorm:"column:width;type:double precision;not null" json:"width"`
	Height     float64   `gorm:"column:height;type:double precision;not null" json:"height"`
	Confidence float64   `gorm:"column:confidence;type:double precision;not null" json:"confidence"`
	Class      string    `gorm:"column:class;type:varchar(255);not null" json:"class"`
	ClassID    int       `gorm:"column:class_id;type:integer;not null" json:"class_id"`
	CreatedAt  time.Time `gorm:"column:created_at;not null;default:now()" json:"created_at"`
}

func (Prediction) TableName() string {
	return "predictions"
}

// JobStatusResponse is the DTO returned by the GET /jobs/:id endpoint.
type JobStatusResponse struct {
	JobID       string       `json:"job_id"`
	ImageURL    string       `json:"image_url"`
	Status      string       `json:"status"`
	ProcessedAt time.Time    `json:"processed_at"`
	CreatedAt   time.Time    `json:"created_at"`
	Predictions []Prediction `json:"predictions,omitempty"`
}
