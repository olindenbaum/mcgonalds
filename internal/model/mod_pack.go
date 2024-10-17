package model

type ModPack struct {
	SwaggerGormModel
	Name     string `gorm:"not null" json:"name"`
	Version  string `gorm:"not null" json:"version"`
	Path     string `gorm:"not null" json:"path"`
	IsCommon bool   `gorm:"not null;default:false" json:"is_common"`
}
