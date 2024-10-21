package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/olindenbaum/mcgonalds/internal/config"
	"github.com/olindenbaum/mcgonalds/internal/middleware"
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

func (h *Handler) RegisterUnauthenticatedRoutes(r *mux.Router) {
	r.HandleFunc("/signup", h.Signup).Methods("POST")
	r.HandleFunc("/login", h.Login).Methods("POST")
}

func (h *Handler) RegisterAuthenticatedRoutes(r *mux.Router) {
	r.HandleFunc("/servers", h.CreateServer).Methods("POST")
	r.HandleFunc("/servers", h.ListServers).Methods("GET")
	r.HandleFunc("/servers/{id}", h.GetServer).Methods("GET")
	r.HandleFunc("/servers/{id}", h.DeleteServer).Methods("DELETE")
	r.HandleFunc("/servers/{id}/start", h.StartServer).Methods("POST")
	r.HandleFunc("/servers/{id}/stop", h.StopServer).Methods("POST")
	r.HandleFunc("/servers/{id}/restart", h.RestartServer).Methods("POST")
	r.HandleFunc("/servers/{id}/command", h.SendCommand).Methods("POST")
	r.HandleFunc("/servers/{id}/upload-jar", h.UploadJarFile).Methods("POST")
	r.HandleFunc("/servers/{id}/upload-modpack", h.UploadModPack).Methods("POST")
	r.HandleFunc("/jar-files", h.UploadSharedJarFile).Methods("POST")
	r.HandleFunc("/mod-packs", h.UploadSharedModPack).Methods("POST")
	r.HandleFunc("/jar-files", h.GetCommonJarFiles).Methods("GET")
	r.HandleFunc("/mod-packs", h.GetCommonModPacks).Methods("GET")
	r.HandleFunc("/servers/{id}/output", h.GetServerOutput).Methods("GET")
	r.HandleFunc("/servers/{id}/output/ws", h.GetServerOutputWS).Methods("GET")
}

// CreateServer godoc
// @Summary Create a new Minecraft server
// @Description Create a new Minecraft server with specified jar file and additional files
// @Tags servers
// @Accept json
// @Produce json
// @Param name formData string true "Server Name"
// @Param executable_command formData string true "Executable Command"
// @Param jar_file_id formData int false "JAR File ID"
// @Param jar_file formData file false "JAR File"
// @Param mod_pack_id formData int false "Mod Pack ID"
// @Param mod_pack formData file false "Mod Pack File"
// @Success 201 {object} model.Server
// @Failure 400 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /servers [post]
func (h *Handler) CreateServer(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form data with a maximum memory of 100MB
	userID, ok := r.Context().Value(middleware.ContextUserID).(uint)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	err := r.ParseMultipartForm(100 << 20)
	if err != nil {
		log.Printf("Error parsing multipart form: %v", err)
		http.Error(w, "Failed to parse form data", http.StatusBadRequest)
		return
	}

	// Extract form values
	name := r.FormValue("name")
	executableCommand := r.FormValue("executable_command")
	jarFileIDStr := r.FormValue("jar_file_id")
	modPackIDStr := r.FormValue("mod_pack_id")

	// Validate required fields
	if name == "" || executableCommand == "" {
		http.Error(w, "Name and executable_command are required", http.StatusBadRequest)
		return
	}

	// Initialize variables for jar file
	var jarFile *model.JarFile
	var jarFileID uint
	jarFileIDProvided := false

	// Check if jar_file_id is provided
	if jarFileIDStr != "" {
		id, err := strconv.Atoi(jarFileIDStr)
		if err != nil || id <= 0 {
			http.Error(w, "Invalid jar_file_id", http.StatusBadRequest)
			return
		}
		jarFileID = uint(id)
		jarFileIDProvided = true
	}

	// Handle jar_file upload if provided
	var uploadedJarFile *model.JarFile
	jarFileUploaded := false
	file, header, err := r.FormFile("jar_file")
	if err == nil {
		defer file.Close()
		jarFileUploaded = true
		uploadedJarFile, err = h.ServerManager.UploadJarFile(header.Filename, "default_version", file, header.Filename, header.Size, "TODOSERVERID", false)
		if err != nil {
			log.Printf("Error uploading JAR file: %v", err)
			http.Error(w, "Failed to upload JAR file", http.StatusInternalServerError)
			return
		}
	} else if err != http.ErrMissingFile {
		log.Printf("Error retrieving jar_file: %v", err)
		http.Error(w, "Failed to retrieve JAR file", http.StatusBadRequest)
		return
	}

	// Ensure either jar_file or jar_file_id is provided, but not both
	if jarFileUploaded && jarFileIDProvided {
		http.Error(w, "Provide either jar_file or jar_file_id, not both", http.StatusBadRequest)
		return
	}

	// If jar_file_id is provided, fetch the JarFile from the database
	if jarFileIDProvided {
		jarFile, err = h.ServerManager.GetJarFileByID(jarFileID)
		if err != nil {
			log.Printf("Error fetching JarFile by ID: %v", err)
			http.Error(w, "Invalid jar_file_id", http.StatusBadRequest)
			return
		}
	} else if jarFileUploaded {
		jarFile = uploadedJarFile
	} else {
		http.Error(w, "Either jar_file or jar_file_id must be provided", http.StatusBadRequest)
		return
	}

	// Initialize variables for mod pack
	var modPack *model.ModPack
	var modPackID uint
	modPackIDProvided := false

	// Check if mod_pack_id is provided
	if modPackIDStr != "" {
		id, err := strconv.Atoi(modPackIDStr)
		if err != nil || id <= 0 {
			http.Error(w, "Invalid mod_pack_id", http.StatusBadRequest)
			return
		}
		modPackID = uint(id)
		modPackIDProvided = true
	}

	// Handle mod_pack upload if provided
	var uploadedModPack *model.ModPack
	modPackUploaded := false
	file, header, err = r.FormFile("mod_pack")
	if err == nil {
		defer file.Close()
		modPackUploaded = true
		uploadedModPack, err = h.ServerManager.UploadModPack(header.Filename, file, header.Size, "TODOSERVERID", false)
		if err != nil {
			log.Printf("Error uploading mod pack: %v", err)
			http.Error(w, "Failed to upload mod pack", http.StatusInternalServerError)
			return
		}
	} else if err != http.ErrMissingFile {
		log.Printf("Error retrieving mod_pack: %v", err)
		http.Error(w, "Failed to retrieve mod pack", http.StatusBadRequest)
		return
	}

	// Ensure either mod_pack or mod_pack_id is provided, but not both
	if modPackUploaded && modPackIDProvided {
		http.Error(w, "Provide either mod_pack or mod_pack_id, not both", http.StatusBadRequest)
		return
	}

	// If mod_pack_id is provided, fetch the ModPack from the database
	if modPackIDProvided {
		modPack, err = h.ServerManager.GetModPackByID(modPackID)
		if err != nil {
			log.Printf("Error fetching ModPack by ID: %v", err)
			http.Error(w, "Invalid mod_pack_id", http.StatusBadRequest)
			return
		}
	} else if modPackUploaded {
		modPack = uploadedModPack
	} else {
		modPack = nil
	}

	// Create the server
	dir, err := os.Getwd()
	if err != nil {
		log.Printf("Error getting current working directory: %v", err)
		http.Error(w, "Failed to create server", http.StatusInternalServerError)
		return
	}
	serverPath := filepath.Join(dir, "game_servers", name)
	log.Printf("Creating server with path: %s", serverPath)
	id, err := h.ServerManager.CreateServer(name, serverPath, executableCommand, jarFile, modPack, nil, userID)
	if err != nil {
		log.Printf("Error creating server: %v", err)
		http.Error(w, "Failed to create server", http.StatusInternalServerError)
		return
	}
	log.Printf("Server created successfully with ID: %d", id)
	if err != nil {
		log.Printf("Error creating server: %v", err)
		http.Error(w, "Failed to create server", http.StatusInternalServerError)
		return
	}

	// Respond with the created server details
	server, err := h.ServerManager.GetServer(uint8(id), userID)
	if err != nil {
		log.Printf("Error fetching created server: %v", err)
		http.Error(w, "Server created but failed to fetch details", http.StatusInternalServerError)
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
	userID, ok := r.Context().Value(middleware.ContextUserID).(uint)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	servers, err := h.ServerManager.ListServers(userID)
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
// @Param id path uint8 true "Server ID"
// @Success 200 {object} model.Server
// @Failure 404 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /servers/{id} [get]
func (h *Handler) GetServer(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.ContextUserID).(uint)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	vars := mux.Vars(r)
	serverId := vars["id"]
	fmt.Printf("Getting server with ID: %s", serverId)
	id, err := strconv.ParseUint(serverId, 10, 8)
	if err != nil {
		http.Error(w, "Invalid server ID", http.StatusBadRequest)
		return
	}
	server, err := h.ServerManager.GetServer(uint8(id), userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "Server not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to fetch server", http.StatusInternalServerError)
		}
		return
	}
	serverDetails := server.GetServerDetails()
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(serverDetails)
}

// DeleteServer godoc
// @Summary Delete a Minecraft server
// @Description Delete a specific Minecraft server by name
// @Tags servers
// @Produce json
// @Param id path uint8 true "Server ID"
// @Success 200 {object} map[string]string
// @Failure 404 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /servers/{id} [delete]
func (h *Handler) DeleteServer(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.ContextUserID).(uint)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	vars := mux.Vars(r)
	serverId := vars["id"]

	id, err := strconv.ParseUint(serverId, 10, 8)
	if err != nil {
		http.Error(w, "Invalid server ID", http.StatusBadRequest)
		return
	}

	if err := h.ServerManager.DeleteServer(uint8(id), userID); err != nil {
		http.Error(w, "Failed to delete server: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Server deleted successfully"})
}

// StartServerRequest represents the payload for starting a server
type StartServerRequest struct {
}

// StartServer godoc
// @Summary Start a Minecraft server
// @Description Start a specific Minecraft server by name with customizable RAM and port
// @Tags servers
// @Accept json
// @Produce json
// @Param id path uint8 true "Server ID"
// @Param StartServerRequest body StartServerRequest true "RAM and Port"
// @Success 200 {object} map[string]string
// @Failure 404 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /servers/{id}/start [post]
func (h *Handler) StartServer(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.ContextUserID).(uint)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	vars := mux.Vars(r)
	serverId := vars["id"]

	id, err := strconv.ParseUint(serverId, 10, 8)
	if err != nil {
		http.Error(w, "Invalid server ID", http.StatusBadRequest)
		return
	}

	// Start the server
	err = h.ServerManager.StartServer(uint8(id), userID)
	if err != nil {
		log.Printf("Error starting server: %v", err)
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
// @Param id path uint8 true "Server ID"
// @Success 200 {object} map[string]string
// @Failure 404 {object{ model.ErrorResponse
// @Failure 500 {object{ model.ErrorResponse
// @Router /servers/{id}/stop [post]
func (h *Handler) StopServer(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.ContextUserID).(uint)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	vars := mux.Vars(r)
	serverId := vars["id"]

	id, err := strconv.ParseUint(serverId, 10, 8)
	if err != nil {
		http.Error(w, "Invalid server ID", http.StatusBadRequest)
		return
	}

	if err := h.ServerManager.StopServer(uint8(id), userID); err != nil {
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
// @Param id path uint8 true "Server ID"
// @Success 200 {object} map[string]string
// @Failure 404 {object{ model.ErrorResponse
// @Failure 500 {object{ model.ErrorResponse
// @Router /servers/{id}/restart [post]
func (h *Handler) RestartServer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serverId := vars["id"]

	id, err := strconv.ParseUint(serverId, 10, 8)
	if err != nil {
		http.Error(w, "Invalid server ID", http.StatusBadRequest)
		return
	}

	if err := h.ServerManager.RestartServer(uint8(id)); err != nil {
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
// @Param id path uint8 true "Server ID"
// @Param command body map[string]string true "Command to send"
// @Success 200 {object} map[string]string
// @Failure 400 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /servers/{id}/command [post]
func (h *Handler) SendCommand(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serverId := vars["id"]

	id, err := strconv.ParseUint(serverId, 10, 8)
	if err != nil {
		http.Error(w, "Invalid server ID", http.StatusBadRequest)
		return
	}
	var commandReq struct {
		Command string `json:"command"`
	}
	if err := json.NewDecoder(r.Body).Decode(&commandReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if _, err := h.ServerManager.SendCommand(uint8(id), commandReq.Command); err != nil {
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

// GetCommonJarFiles godoc
// @Summary Get common JAR files
// @Description Retrieve a list of common JAR files
// @Tags jar-files
// @Produce json
// @Param common query bool false "Filter by common JAR files"
// @Success 200 {array} model.JarFile
// @Failure 500 {object} model.ErrorResponse
// @Router /jar-files [get]
func (h *Handler) GetCommonJarFiles(w http.ResponseWriter, r *http.Request) {
	commonParam := r.URL.Query().Get("common")
	common, err := strconv.ParseBool(commonParam)
	if commonParam != "" && err != nil {
		http.Error(w, "Invalid 'common' query parameter", http.StatusBadRequest)
		return
	}

	jarFiles, err := h.ServerManager.GetJarFiles(common)
	if err != nil {
		http.Error(w, "Failed to fetch JAR files", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(jarFiles)
}

// GetCommonModPacks godoc
// @Summary Get common mod packs
// @Description Retrieve a list of common mod packs
// @Tags mod-packs
// @Produce json
// @Param common query bool false "Filter by common mod packs"
// @Success 200 {array} model.ModPack
// @Failure 500 {object} model.ErrorResponse
// @Router /mod-packs [get]
func (h *Handler) GetCommonModPacks(w http.ResponseWriter, r *http.Request) {
	commonParam := r.URL.Query().Get("common")
	common, err := strconv.ParseBool(commonParam)
	if commonParam != "" && err != nil {
		http.Error(w, "Invalid 'common' query parameter", http.StatusBadRequest)
		return
	}

	modPacks, err := h.ServerManager.GetModPacks(common)
	if err != nil {
		http.Error(w, "Failed to fetch mod packs", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(modPacks)
}

// GetServerOutput godoc
// @Summary Get server output
// @Description Retrieve the output stream of a specific Minecraft server
// @Tags servers
// @Produce json
// @Param id path uint8 true "Server ID"
// @Success 200 {object} map[string]string
// @Failure 404 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /servers/{id}/output [get]
func (h *Handler) GetServerOutput(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serverId := vars["name"]

	id, err := strconv.ParseUint(serverId, 10, 8)
	if err != nil {
		http.Error(w, "Invalid server ID", http.StatusBadRequest)
		return
	}

	output, err := h.ServerManager.GetServerOutput(uint8(id))
	if err != nil {
		log.Printf("Error fetching server output: %v", err)
		http.Error(w, "Failed to fetch server output", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"output": output})
}

// Upgrader specifies parameters for upgrading an HTTP connection to a WebSocket connection.
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Adjust this as needed for security
	},
}

// GetServerOutputWS godoc
// @Summary Get server output via WebSocket
// @Description Establish a WebSocket connection to receive real-time server output
// @Tags servers
// @Param id path uint8 true "Server ID"
// @Router /servers/{id}/output/ws [get]
func (h *Handler) GetServerOutputWS(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serverId := vars["id"]

	id, err := strconv.ParseUint(serverId, 10, 8)
	if err != nil {
		http.Error(w, "Invalid server ID", http.StatusBadRequest)
		return
	}

	// Get user ID from context
	userID, ok := r.Context().Value(middleware.ContextUserID).(uint)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Check server ownership
	var server model.Server
	if err := h.DB.First(&server, id).Error; err != nil {
		http.Error(w, "Server not found", http.StatusNotFound)
		return
	}

	if server.UserID != userID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	// Subscribe to server output
	outputChan, err := h.ServerManager.SubscribeOutput(uint8(id))
	if err != nil {
		log.Printf("Error subscribing to server output: %v", err)
		return
	}
	defer h.ServerManager.UnsubscribeOutput(uint8(id), outputChan)

	for msg := range outputChan {
		err := conn.WriteMessage(websocket.TextMessage, []byte(msg))
		if err != nil {
			log.Printf("WebSocket write error: %v", err)
			break
		}
	}
}

// // UploadAdditionalFile godoc
// // @Summary Upload an additional file for a server
// // @Description Upload an additional file to a specific server
// // @Tags servers
// // @Accept multipart/form-data
// // @Produce json
// // @Param name path string true "Server Name"
// // @Param file formData file true "Additional file to upload"
// // @Success 200 {object} map[string]string "File uploaded successfully"
// // @Failure 400 {object} model.ErrorResponse
// // @Failure 500 {object} model.ErrorResponse
// // @Router /servers/{name}/additional-files [post]
// func (h *Handler) UploadAdditionalFile(w http.ResponseWriter, r *http.Request) {
// 	vars := mux.Vars(r)
// 	serverName := vars["name"]

// 	file, header, err := r.FormFile("file")
// 	if err != nil {
// 		http.Error(w, "Failed to get file from form", http.StatusBadRequest)
// 		return
// 	}
// 	defer file.Close()

// 	// Call ServerManager's UploadAdditionalFile
// 	err = h.ServerManager.UploadAdditionalFile(serverName, file, header.Size)
// 	if err != nil {
// 		http.Error(w, "Failed to upload additional file: "+err.Error(), http.StatusInternalServerError)
// 		return
// 	}

// 	w.WriteHeader(http.StatusOK)
// 	json.NewEncoder(w).Encode(map[string]string{"message": "File uploaded successfully"})
// }
