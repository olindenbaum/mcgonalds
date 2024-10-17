package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/olindenbaum/mcgonalds/internal/config"
	"github.com/olindenbaum/mcgonalds/internal/model"
	"github.com/olindenbaum/mcgonalds/internal/server_manager"
	"gorm.io/gorm"
)

type Handler struct {
	DB            *gorm.DB
	ServerManager *server_manager.ServerManager
	Config        *config.Config
}

func NewHandler(db *gorm.DB, sm *server_manager.ServerManager, config *config.Config) *Handler {
	return &Handler{
		DB:            db,
		ServerManager: sm,
		Config:        config,
	}
}

func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/servers", h.CreateServer).Methods("POST")
	r.HandleFunc("/servers", h.ListServers).Methods("GET")
	r.HandleFunc("/servers/{name}", h.GetServer).Methods("GET")
	r.HandleFunc("/servers/{name}", h.DeleteServer).Methods("DELETE")
	r.HandleFunc("/servers/{name}/start", h.StartServer).Methods("POST")
	r.HandleFunc("/servers/{name}/stop", h.StopServer).Methods("POST")
	r.HandleFunc("/servers/{name}/restart", h.RestartServer).Methods("POST")
	r.HandleFunc("/servers/{name}/command", h.SendCommand).Methods("POST")
	r.HandleFunc("/servers/{name}/upload-jar", h.UploadJarFile).Methods("POST")
	r.HandleFunc("/servers/{name}/upload-modpack", h.UploadModPack).Methods("POST")
	r.HandleFunc("/jar-files", h.UploadSharedJarFile).Methods("POST")
	r.HandleFunc("/mod-packs", h.UploadSharedModPack).Methods("POST")
}

// CreateServer godoc
// @Summary Create a new Minecraft server
// @Description Create a new Minecraft server with specified jar file and additional files
// @Tags servers
// @Accept json
// @Produce json
// @Param server body model.CreateServerRequest true "Server creation request"
// @Success 201 {object} model.Server
// @Failure 400 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /servers [post]
func (h *Handler) CreateServer(w http.ResponseWriter, r *http.Request) {
	var req model.CreateServerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	server := &model.Server{
		Name: req.Name,
		Path: req.Path,
	}

	if err := h.DB.Create(server).Error; err != nil {
		http.Error(w, "Failed to create server", http.StatusInternalServerError)
		return
	}

	if err := h.ServerManager.SetupServer(req.Name); err != nil {
		http.Error(w, "Failed to setup server: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(server)
}

// ListServers godoc
// @Summary List all Minecraft servers
// @Description Get a list of all Minecraft servers
// @Tags servers
// @Produce json
// @Success 200 {array} model.Server
// @Failure 500 {object{ model.ErrorResponse
// @Router /servers [get]
func (h *Handler) ListServers(w http.ResponseWriter, r *http.Request) {
	servers, err := h.ServerManager.ListServers()
	if err != nil {
		http.Error(w, "Failed to fetch servers", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(servers)
}

// GetServer godoc
// @Summary Get a specific Minecraft server
// @Description Get details of a specific Minecraft server by name
// @Tags servers
// @Produce json
// @Param name path string true "Server Name"
// @Success 200 {object} model.Server
// @Failure 404 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /servers/{name} [get]
func (h *Handler) GetServer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serverName := vars["name"]

	server, err := h.ServerManager.GetServer(serverName)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "Server not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to fetch server", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(server)
}

// DeleteServer godoc
// @Summary Delete a Minecraft server
// @Description Delete a specific Minecraft server by name
// @Tags servers
// @Produce json
// @Param name path string true "Server Name"
// @Success 200 {object} map[string]string
// @Failure 404 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /servers/{name} [delete]
func (h *Handler) DeleteServer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serverName := vars["name"]

	if err := h.ServerManager.DeleteServer(serverName); err != nil {
		http.Error(w, "Failed to delete server: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Server deleted successfully"})
}

// StartServer godoc
// @Summary Start a Minecraft server
// @Description Start a specific Minecraft server by name
// @Tags servers
// @Produce json
// @Param name path string true "Server Name"
// @Success 200 {object} map[string]string
// @Failure 404 {object{ model.ErrorResponse
// @Failure 500 {object{ model.ErrorResponse
// @Router /servers/{name}/start [post]
func (h *Handler) StartServer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serverName := vars["name"]

	if err := h.ServerManager.StartServer(serverName); err != nil {
		http.Error(w, "Failed to start server: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Server started successfully"})
}

// StopServer godoc
// @Summary Stop a Minecraft server
// @Description Stop a specific Minecraft server by name
// @Tags servers
// @Produce json
// @Param name path string true "Server Name"
// @Success 200 {object} map[string]string
// @Failure 404 {object{ model.ErrorResponse
// @Failure 500 {object{ model.ErrorResponse
// @Router /servers/{name}/stop [post]
func (h *Handler) StopServer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serverName := vars["name"]

	if err := h.ServerManager.StopServer(serverName); err != nil {
		http.Error(w, "Failed to stop server: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Server stopped successfully"})
}

// RestartServer godoc
// @Summary Restart a Minecraft server
// @Description Restart a specific Minecraft server by name
// @Tags servers
// @Produce json
// @Param name path string true "Server Name"
// @Success 200 {object} map[string]string
// @Failure 404 {object{ model.ErrorResponse
// @Failure 500 {object{ model.ErrorResponse
// @Router /servers/{name}/restart [post]
func (h *Handler) RestartServer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serverName := vars["name"]

	if err := h.ServerManager.RestartServer(serverName); err != nil {
		http.Error(w, "Failed to restart server: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Server restarted successfully"})
}

// SendCommand godoc
// @Summary Send a command to a Minecraft server
// @Description Send a command to a specific Minecraft server by name
// @Tags servers
// @Accept json
// @Produce json
// @Param name path string true "Server Name"
// @Param command body map[string]string true "Command to send"
// @Success 200 {object} map[string]string
// @Failure 400 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /servers/{name}/command [post]
func (h *Handler) SendCommand(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serverName := vars["name"]

	var commandReq struct {
		Command string `json:"command"`
	}
	if err := json.NewDecoder(r.Body).Decode(&commandReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if _, err := h.ServerManager.SendCommand(serverName, commandReq.Command); err != nil {
		http.Error(w, "Failed to send command: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Command sent successfully"})
}

// UploadJarFile godoc
// @Summary Upload JAR file for a server
// @Description Upload a JAR file to a specific server, either selecting a common JAR or uploading a new one
// @Tags servers
// @Accept multipart/form-data
// @Produce json
// @Param name formData string true "Server Name"
// @Param version formData string true "Version of the JAR file"
// @Param serverID path string true "Server ID"
// @Param file formData file true "JAR file to upload"
// @Success 200 {object} map[string]string "JAR file uploaded successfully"
// @Failure 400 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /servers/{serverId}/upload-jar [post]
func (h *Handler) UploadJarFile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serverID := vars["serverId"]
	nickname := r.FormValue("name")
	version := r.FormValue("version")
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Failed to parse file", http.StatusBadRequest)
		return
	}
	defer file.Close()
	// Extract the filename and extension
	filename := header.Filename
	extension := filepath.Ext(filename)                 // Get the file extension
	baseName := filename[:len(filename)-len(extension)] // Get the file name without extension

	log.Printf("Uploaded file: %s, with extension: %s", baseName, extension)

	jarFile, err := h.ServerManager.UploadJarFile(nickname, version, file, baseName, header.Size, serverID, false)
	if err != nil {
		http.Error(w, "Failed to upload JAR file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "JAR file uploaded successfully",
		"fileID":  strconv.Itoa(int(jarFile.ID)),
	})
}

// UploadModPack godoc
// @Summary Upload mod pack for a server
// @Description Upload a mod pack to a specific server, either selecting a common mod pack or uploading a new one
// @Tags servers
// @Accept multipart/form-data
// @Produce json
// @Param name path string true "Server Name"
// @Param file formData file true "Mod pack file to upload"
// @Success 200 {object} map[string]string "Mod pack uploaded successfully"
// @Failure 400 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /servers/{name}/upload-modpack [post]
func (h *Handler) UploadModPack(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serverName := vars["name"]

	// Parse the multipart form
	err := r.ParseMultipartForm(100 << 20) // 100 MB max size
	if err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Failed to get file from form", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Call ServerManager's UploadModPack
	modPack, err := h.ServerManager.UploadModPack(header.Filename, file, header.Size, serverName, false)
	if err != nil {
		http.Error(w, "Failed to upload mod pack: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return success response
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Mod pack uploaded successfully",
		"fileID":  strconv.Itoa(int(modPack.ID)),
	})
}

// UploadSharedJarFile godoc
// @Summary Upload a shared JAR file
// @Description Upload a shared JAR file to be used by multiple servers
// @Tags jar-files
// @Accept multipart/form-data
// @Produce json
// @Param name formData string true "Nickname of the JAR file"
// @Param version formData string true "Version of the JAR file"
// @Param file formData file true "The JAR file to upload"
// @Success 201 {object} model.JarFile
// @Failure 400 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /jar-files [post]
func (h *Handler) UploadSharedJarFile(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(10 << 20) // 10 MB max
	if err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	nickname := r.FormValue("name")
	version := r.FormValue("version")

	if nickname == "" || version == "" {
		http.Error(w, "Name and version are required", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Failed to get file from form", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Extract the filename and extension
	filename := header.Filename
	extension := filepath.Ext(filename)                 // Get the file extension
	baseName := filename[:len(filename)-len(extension)] // Get the file name without extension

	log.Printf("Uploaded file: %s, with extension: %s", baseName, extension)

	jarFile, err := h.ServerManager.UploadJarFile(nickname, version, file, baseName, header.Size, "", true)
	if err != nil {
		http.Error(w, "Failed to upload JAR file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(jarFile)
}

// UploadSharedModPack godoc
// @Summary Upload a shared mod pack
// @Description Upload a shared mod pack to be used by multiple servers
// @Tags mod-packs
// @Accept multipart/form-data
// @Produce json
// @Param name formData string true "Name of the mod pack"
// @Param version formData string true "Version of the mod pack"
// @Param type formData string true "Type of the mod pack (e.g., zip, folder)"
// @Param file formData file true "The mod pack file to upload"
// @Success 201 {object} model.ModPack
// @Failure 400 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /mod-packs [post]
func (h *Handler) UploadSharedModPack(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(100 << 20) // 100 MB max
	if err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	version := r.FormValue("version")
	fileType := r.FormValue("type")

	if name == "" || version == "" || fileType == "" {
		http.Error(w, "Name, version, and type are required", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Failed to get file from form", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Extract the filename and extension
	filename := header.Filename
	extension := filepath.Ext(filename)                 // Get the file extension
	baseName := filename[:len(filename)-len(extension)] // Get the file name without extension

	log.Printf("Uploaded file: %s, with extension: %s", baseName, extension)

	modPack, err := h.ServerManager.UploadModPack(header.Filename, file, header.Size, "", true)
	if err != nil {
		http.Error(w, "Failed to upload mod pack: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(modPack)
}
