package job

import "gorm.io/gorm"

// JobRepository defines the contract for querying job data.
type JobRepository interface {
	FindByJobID(jobID string) (*Job, error)
}

// postgresJobRepository implements JobRepository
// using PostgreSQL as backing store via GORM.
type postgresJobRepository struct {
	db *gorm.DB
}

// NewJobRepository creates a new PostgreSQL-backed job repository.
func NewJobRepository(db *gorm.DB) JobRepository {
	return &postgresJobRepository{db: db}
}

// FindByJobID retrieves a job and its associated predictions by the job_id.
func (r *postgresJobRepository) FindByJobID(jobID string) (*Job, error) {
	var job Job
	err := r.db.
		Preload("Predictions").
		Where("job_id = ?", jobID).
		First(&job).Error

	if err != nil {
		return nil, err
	}

	return &job, nil
}
