package domain

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Job represents the jobs table in the database.
type Job struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	JobID       string         `gorm:"column:job_id;type:varchar(255);uniqueIndex;not null" json:"job_id"`
	ImageURL    string         `gorm:"column:image_url;type:text;not null" json:"image_url"`
	Status      string         `gorm:"column:status;type:varchar(50);not null;default:'completed'" json:"status"`
	ProcessedAt time.Time      `gorm:"column:processed_at;not null;default:now()" json:"processed_at"`
	CreatedAt   time.Time      `gorm:"column:created_at;not null;default:now()" json:"created_at"`
	Predictions []DBPrediction `gorm:"foreignKey:JobID;references:JobID" json:"predictions,omitempty"`
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

// DBPrediction represents the predictions table in the database.
type DBPrediction struct {
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

func (DBPrediction) TableName() string {
	return "predictions"
}

// BeforeCreate generates a UUID before inserting a new DBPrediction.
func (p *DBPrediction) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}
