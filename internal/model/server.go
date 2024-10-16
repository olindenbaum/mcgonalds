package model

type Server struct {
	SwaggerGormModel
	Name              string `gorm:"uniqueIndex;not null"`
	Path              string `gorm:"not null"`
	JarFileID         uint
	JarFile           JarFile `gorm:"foreignKey:JarFileID"`
	AdditionalFileIDs []uint
	AdditionalFiles   []AdditionalFile `gorm:"many2many:server_additional_files;"`
}
