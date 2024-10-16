package model

import (
	"gorm.io/gorm"
)

type JarFile struct {
	gorm.Model
	Name    string `gorm:"not null"`
	Version string `gorm:"not null"`
	Path    string `gorm:"not null"`
}
