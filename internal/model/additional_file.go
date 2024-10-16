package model

import (
	"gorm.io/gorm"
)

type AdditionalFile struct {
	gorm.Model
	Name string `gorm:"not null"`
	Type string `gorm:"not null"` // e.g., "modpack", "config", etc.
	Path string `gorm:"not null"`
}
