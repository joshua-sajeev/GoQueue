package models

import (
	"time"

	"gorm.io/datatypes"
)

type Job struct {
	ID         uint `gorm:"primaryKey"`
	Queue      string
	Type       string
	Payload    datatypes.JSON
	Status     string
	Attempts   int
	MaxRetries int
	Result     datatypes.JSON
	Error      string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
