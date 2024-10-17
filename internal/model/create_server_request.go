package model

// CreateServerRequest represents the request payload for creating a new server
type CreateServerRequest struct {
	Name              string `json:"name" validate:"required"`
	Path              string `json:"path" validate:"required"`
	JarFileID         uint   `json:"jar_file_id" validate:"required"` // ID of the JAR file
	ModPackID         *uint  `json:"mod_pack_id,omitempty"`           // Optional ID of the mod pack
	ExecutableCommand string `json:"executable_command" validate:"required"`
}
