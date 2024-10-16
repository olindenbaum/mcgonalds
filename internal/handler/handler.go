package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/olindenbaum/mcgonalds/internal/config"
	"github.com/olindenbaum/mcgonalds/internal/server_manager"
	"gorm.io/gorm"
)

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

func (h *Handler) GetServer(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	server, err := h.ServerManager.GetServer(name)
	if err != nil {
		http.Error(w, "Server not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(server)
}

func (h *Handler) DeleteServer(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	if err := h.ServerManager.DeleteServer(name); err != nil {
		http.Error(w, "Failed to delete server", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) StartServer(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	if err := h.ServerManager.StartServer(name); err != nil {
		http.Error(w, "Failed to start server", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) StopServer(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	if err := h.ServerManager.StopServer(name); err != nil {
		http.Error(w, "Failed to stop server", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) RestartServer(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	if err := h.ServerManager.RestartServer(name); err != nil {
		http.Error(w, "Failed to restart server", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

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
