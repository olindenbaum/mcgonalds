package server

import "github.com/olindenbaum/mcgonalds/internal/model"

type ServerDetails struct {
	ServerId  uint8              `json:"server_id"`
	Name      string             `json:"name"`
	Path      string             `json:"path"`
	IsRunning bool               `json:"is_running"`
	Config    model.ServerConfig `json:"config"`
}
