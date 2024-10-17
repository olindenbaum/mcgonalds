package model

type Server struct {
	SwaggerGormModel
	Name              string           `gorm:"not null" json:"name"`
	Path              string           `gorm:"not null" json:"path"`
	JarFileID         uint             `gorm:"not null" json:"jar_file_id"`
	JarFile           JarFile          `gorm:"foreignKey:JarFileID" json:"jar_file"`
	AdditionalFileIDs []uint           `gorm:"-" json:"additional_file_ids"`
	AdditionalFiles   []AdditionalFile `gorm:"many2many:server_additional_files" json:"additional_files"`
	ServerConfig      ServerConfig     `gorm:"foreignKey:ServerID" json:"server_config"`
}
