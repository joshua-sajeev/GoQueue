package models

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Job struct {
	gorm.Model
	ID         uint           `gorm:"primaryKey;autoIncrement"`
	Queue      string         `gorm:"type:varchar(255);not null"`
	Type       string         `gorm:"type:varchar(255);not null"`
	Payload    datatypes.JSON `gorm:"type:jsonb"`
	Status     string         `gorm:"type:varchar(50);not null;default:'pending'"`
	Attempts   int            `gorm:"default:0;not null"`
	MaxRetries int            `gorm:"default:5"`
	Result     datatypes.JSON `gorm:"type:jsonb"`
	Error      string         `gorm:"type:text"`
	CreatedAt  time.Time      `gorm:"autoCreateTime"`
	UpdatedAt  time.Time      `gorm:"autoUpdateTime"`
}
