package model

type AdditionalFile struct {
	SwaggerGormModel
	Name string `gorm:"not null"`
	Type string `gorm:"not null"` // e.g., "modpack", "config", etc.
	Path string `gorm:"not null"`
}
