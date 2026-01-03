// internal/models/job.go
package models

import (
	"time"

	"github.com/joshu-sajeev/goqueue/internal/config"
	"gorm.io/datatypes"
)

type Job struct {
	ID      uint `gorm:"primaryKey"`
	Queue   string
	Type    string
	Payload datatypes.JSON

	Status     config.JobStatus
	Attempts   int
	MaxRetries int

	AvailableAt time.Time
	LockedAt    *time.Time
	LockedBy    *uint

	Result datatypes.JSON
	Error  string

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (j *Job) IsLocked() bool {
	return j.LockedAt != nil && !j.LockedAt.IsZero()
}

func (j *Job) IsAvailable() bool {
	return !j.IsLocked() && j.AvailableAt.Before(time.Now())
}
