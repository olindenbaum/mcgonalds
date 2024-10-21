package model

type Server struct {
	SwaggerGormModel
	Name   string `gorm:"not null" json:"name"`
	Path   string `gorm:"not null" json:"path"`
	Status string `json:"status"`
	UserID uint   `json:"user_id"`
	User   User   `json:"-"`
}
