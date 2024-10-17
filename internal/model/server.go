package model

type Server struct {
	SwaggerGormModel
	Name string `gorm:"not null" json:"name"`
	Path string `gorm:"not null" json:"path"`
}
