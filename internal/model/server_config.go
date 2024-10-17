package model

type ServerConfig struct {
	SwaggerGormModel
	ServerID          uint     `gorm:"uniqueIndex;not null" json:"server_id"`
	JarFileID         uint     `gorm:"not null" json:"jar_file_id"`
	JarFile           JarFile  `gorm:"foreignKey:JarFileID" json:"jar_file"`
	ModPackID         *uint    `json:"mod_pack_id"`
	ModPack           *ModPack `gorm:"foreignKey:ModPackID" json:"mod_pack,omitempty"`
	ExecutableCommand string   `gorm:"not null" json:"executable_command"`
}
