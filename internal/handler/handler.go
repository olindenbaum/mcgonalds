package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/olindenbaum/mcgonalds/internal/config"
	"github.com/olindenbaum/mcgonalds/internal/model"
	"github.com/olindenbaum/mcgonalds/internal/server_manager"
	"gorm.io/gorm"
)

// @title Server Manager API
// @version 1.0
// @description This is a server management service API
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /api/v1

type Handler struct {
	DB            *gorm.DB
	ServerManager *server_manager.ServerManager
	Config        *config.Config
}

func NewHandler(db *gorm.DB, sm *server_manager.ServerManager, cfg *config.Config) *Handler {
	return &Handler{
		DB:            db,
		ServerManager: sm,
		Config:        cfg,
	}
}

func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/servers", h.ListServers).Methods("GET")
	r.HandleFunc("/servers", h.CreateServer).Methods("POST")
	r.HandleFunc("/servers/{name}", h.GetServer).Methods("GET")
	r.HandleFunc("/servers/{name}", h.DeleteServer).Methods("DELETE")
	r.HandleFunc("/servers/{name}/start", h.StartServer).Methods("POST")
	r.HandleFunc("/servers/{name}/stop", h.StopServer).Methods("POST")
	r.HandleFunc("/servers/{name}/restart", h.RestartServer).Methods("POST")
	r.HandleFunc("/servers/{name}/command", h.SendCommand).Methods("POST")
	r.HandleFunc("/jar-files", h.UploadJarFile).Methods("POST")
	r.HandleFunc("/additional-files", h.UploadAdditionalFile).Methods("POST")
}

// ListServers godoc
// @Summary List all Minecraft servers
// @Description Get a list of all Minecraft servers
// @Tags servers
// @Produce json
// @Success 200 {array} model.Server
// @Failure 500 {object} ErrorResponse
// @Router /servers [get]
func (h *Handler) ListServers(w http.ResponseWriter, r *http.Request) {
	servers, err := h.ServerManager.ListServers()
	if err != nil {
		http.Error(w, "Failed to list servers", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(servers)
}

func (h *Handler) CreateServer(w http.ResponseWriter, r *http.Request) {
	var serverData struct {
		Name              string `json:"name"`
		Path              string `json:"path"`
		JarFileID         uint   `json:"jar_file_id"`
		AdditionalFileIDs []uint `json:"additional_file_ids"`
	}

	if err := json.NewDecoder(r.Body).Decode(&serverData); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.ServerManager.CreateServer(serverData.Name, serverData.Path, serverData.JarFileID, serverData.AdditionalFileIDs); err != nil {
		http.Error(w, "Failed to create server", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

// GetServer godoc
// @Summary Get a Minecraft server
// @Description Get details of a specific Minecraft server by name
// @Tags servers
// @Produce json
// @Param name path string true "Server name"
// @Success 200 {object} model.Server
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /servers/{name} [get]
func (h *Handler) GetServer(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	server, err := h.ServerManager.GetServer(name)
	if err != nil {
		http.Error(w, "Server not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(server)
}

// DeleteServer godoc
// @Summary Delete a Minecraft server
// @Description Delete a specific Minecraft server by name
// @Tags servers
// @Produce json
// @Param name path string true "Server name"
// @Success 204 "No Content"
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /servers/{name} [delete]
func (h *Handler) DeleteServer(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	if err := h.ServerManager.DeleteServer(name); err != nil {
		http.Error(w, "Failed to delete server", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// StartServer godoc
// @Summary Start a Minecraft server
// @Description Start a specific Minecraft server by name
// @Tags servers
// @Produce json
// @Param name path string true "Server name"
// @Success 200 {object} SuccessResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /servers/{name}/start [post]
func (h *Handler) StartServer(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	if err := h.ServerManager.StartServer(name); err != nil {
		http.Error(w, "Failed to start server", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// StopServer godoc
// @Summary Stop a Minecraft server
// @Description Stop a specific Minecraft server by name
// @Tags servers
// @Produce json
// @Param name path string true "Server name"
// @Success 200 {object} SuccessResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /servers/{name}/stop [post]
func (h *Handler) StopServer(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	if err := h.ServerManager.StopServer(name); err != nil {
		http.Error(w, "Failed to stop server", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// RestartServer godoc
// @Summary Restart a Minecraft server
// @Description Restart a specific Minecraft server by name
// @Tags servers
// @Produce json
// @Param name path string true "Server name"
// @Success 200 {object} SuccessResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /servers/{name}/restart [post]
func (h *Handler) RestartServer(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	if err := h.ServerManager.RestartServer(name); err != nil {
		http.Error(w, "Failed to restart server", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// SendCommand godoc
// @Summary Send a command to a Minecraft server
// @Description Send a command to a specific Minecraft server by name
// @Tags servers
// @Accept json
// @Produce json
// @Param name path string true "Server name"
// @Param command body CommandRequest true "Command to send"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /servers/{name}/command [post]
func (h *Handler) SendCommand(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	var commandData struct {
		Command string `json:"command"`
	}

	if err := json.NewDecoder(r.Body).Decode(&commandData); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.ServerManager.SendCommand(name, commandData.Command); err != nil {
		http.Error(w, "Failed to send command", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// UploadJarFile godoc
// @Summary Upload a jar file
// @Description Upload a new Minecraft server jar file
// @Tags jar-files
// @Accept mpfd
// @Produce json
// @Param name formData string true "Name of the jar file"
// @Param version formData string true "Version of the jar file"
// @Param file formData file true "The jar file to upload"
// @Success 201 {object} model.JarFile
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /jar-files [post]
func (h *Handler) UploadJarFile(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(10 << 20) // 10 MB max
	if err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	version := r.FormValue("version")
	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Failed to get file from form", http.StatusBadRequest)
		return
	}
	defer file.Close()

	jarFile, err := h.ServerManager.UploadJarFile(name, version, file, fileHeader.Size)
	if err != nil {
		fmt.Println("Error uploading jar file:", err)
		http.Error(w, "Failed to upload jar file", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(jarFile)
}

// UploadAdditionalFile godoc
// @Summary Upload an additional file
// @Description Upload an additional file (e.g., modpack)
// @Tags additional-files
// @Accept mpfd
// @Produce json
// @Param name formData string true "Name of the additional file"
// @Param type formData string true "Type of the additional file"
// @Param file formData file true "The additional file to upload"
// @Success 201 {object} model.AdditionalFile
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /additional-files [post]
func (h *Handler) UploadAdditionalFile(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(100 << 20) // 100 MB max
	if err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	fileType := r.FormValue("type")
	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Failed to get file from form", http.StatusBadRequest)
		return
	}
	defer file.Close()

	additionalFile, err := h.ServerManager.UploadAdditionalFile(name, fileType, file, fileHeader.Size)
	if err != nil {
		http.Error(w, "Failed to upload additional file", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(additionalFile)
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Message string `json:"message"`
}

// SuccessResponse represents a success response
type SuccessResponse struct {
	Message string `json:"message"`
}

// CommandRequest represents the request body for sending a command
type CommandRequest struct {
	Command string `json:"command"`
}

// CreateServer godoc
// @Summary Create a new Minecraft server
// @Description Create a new Minecraft server with specified jar file and additional files
// @Tags servers
// @Accept json
// @Produce json
// @Param server body CreateServerRequest true "Server creation request"
// @Success 201 {object} model.Server
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /servers [post]
type CreateServerRequest struct {
	Name              string `json:"name"`
	Path              string `json:"path"`
	JarFileID         uint   `json:"jar_file_id"`
	AdditionalFileIDs []uint `json:"additional_file_ids"`
}

// JarFile represents a Minecraft server jar file
type JarFile struct {
	model.SwaggerGormModel
	Name    string `json:"name"`
	Version string `json:"version"`
	Path    string `json:"path"`
}

// AdditionalFile represents an additional file for a Minecraft server
type AdditionalFile struct {
	model.SwaggerGormModel
	Name string `json:"name"`
	Type string `json:"type"`
	Path string `json:"path"`
}

// Server represents a Minecraft server
type Server struct {
	model.SwaggerGormModel
	Name              string           `json:"name"`
	Path              string           `json:"path"`
	JarFileID         uint             `json:"jar_file_id"`
	JarFile           JarFile          `json:"jar_file"`
	AdditionalFileIDs []uint           `json:"additional_file_ids"`
	AdditionalFiles   []AdditionalFile `json:"additional_files"`
}
