package model

type JarFile struct {
	SwaggerGormModel
	Name    string `gorm:"not null"`
	Version string `gorm:"not null"`
	Path    string `gorm:"not null"`
}
