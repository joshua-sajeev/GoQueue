package models

import (
	"time"

	"gorm.io/datatypes"
)

type Job struct {
	ID         string         `gorm:"primaryKey"`
	Queue      string         `gorm:"type:varchar(255);not null"`
	Type       string         `gorm:"type:varchar(255);not null"`
	Payload    datatypes.JSON `gorm:"type:jsonb"`
	Status     string         `gorm:"type:varchar(50);not null"`
	Attempts   int            `gorm:"default:0"`
	MaxRetries int            `gorm:"default:5"`
	Result     datatypes.JSON `gorm:"type:jsonb"`
	Error      string         `gorm:"type:text"`
	CreatedAt  time.Time      `gorm:"autoCreateTime"`
	UpdatedAt  time.Time      `gorm:"autoUpdateTime"`
}
