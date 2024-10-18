package server_manager

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/olindenbaum/mcgonalds/internal/model"
	"github.com/olindenbaum/mcgonalds/internal/server"
	"github.com/olindenbaum/mcgonalds/internal/utils"
	"gorm.io/gorm"
)

type ServerManager struct {
	db            *gorm.DB
	servers       map[uint8]*server.Server
	mutex         sync.RWMutex
	commonDir     string
	outputStreams map[uint8][]chan string
	streamMutex   sync.RWMutex
}

func NewServerManager(db *gorm.DB, commonDir string) (*ServerManager, error) {
	sm := &ServerManager{
		db:            db,
		servers:       make(map[uint8]*server.Server),
		commonDir:     commonDir,
		outputStreams: make(map[uint8][]chan string),
	}

	// Fetch all existing servers from the database
	var dbServers []model.Server
	if err := db.Find(&dbServers).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch existing servers: %w", err)
	}

	// Populate the servers map
	for _, dbServer := range dbServers {
		sm.servers[uint8(dbServer.ID)] = server.NewServer(&dbServer)
	}

	return sm, nil
}

func (sm *ServerManager) CreateServer(name, path, executableCommand string, jarFile *model.JarFile, modPack *model.ModPack, additionalFileIDs []uint) (uint8, error) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	// Check if the server already exists in the database
	var existingServer model.Server
	result := sm.db.Where("name = ?", name).First(&existingServer)
	if result.Error == nil {
		return 0, fmt.Errorf("server %s already exists", name)
	} else if result.Error != gorm.ErrRecordNotFound {
		return 0, fmt.Errorf("error checking for existing server: %w", result.Error)
	}

	// Start a transaction
	tx := sm.db.Begin()
	if tx.Error != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", tx.Error)
	}

	// Create server model
	serverModel := &model.Server{
		Name: name,
		Path: path,
	}

	// Create server in the database
	if err := tx.Create(serverModel).Error; err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("failed to create server in database: %w", err)
	}

	// Create server config
	serverConfig := &model.ServerConfig{
		ServerID:          serverModel.ID,
		ExecutableCommand: executableCommand,
		JarFileID:         jarFile.ID,
	}

	if modPack != nil {
		serverConfig.ModPackID = &modPack.ID
	}

	// Create server config in the database
	if err := tx.Create(serverConfig).Error; err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("failed to create server config in database: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Create server directory
	envDir := filepath.Join(path, "env")
	if err := os.MkdirAll(envDir, 0755); err != nil {
		return 0, fmt.Errorf("failed to create server environment directory: %w", err)
	}

	// Handle symbolic link for JAR file
	if jarFile != nil {
		jarSource := jarFile.Path
		jarDest := filepath.Join(envDir, "server.jar")
		if err := utils.CreateSymlink(jarSource, jarDest); err != nil {
			return 0, fmt.Errorf("failed to create symlink for jar file: %w", err)
		}
	}

	// Handle symbolic link for Mod Pack
	if modPack != nil {
		modPackSource := modPack.Path
		modPackDest := filepath.Join(envDir, "mods")
		if err := utils.CreateSymlink(modPackSource, modPackDest); err != nil {
			return 0, fmt.Errorf("failed to create symlink for mod pack: %w", err)
		}
	}

	// Save the executable command to a script file for easy execution
	execScriptPath := filepath.Join(envDir, "start.sh")
	execScriptContent := fmt.Sprintf("#!/bin/bash\n%s", executableCommand)
	if err := os.WriteFile(execScriptPath, []byte(execScriptContent), 0755); err != nil {
		return 0, fmt.Errorf("failed to create executable script: %w", err)
	}

	// Initialize the server instance
	srv := server.NewServer(serverModel)
	sm.servers[uint8(serverModel.ID)] = srv

	return uint8(serverModel.ID), nil
}

// GetJarFileByID retrieves a JarFile by its ID.
func (sm *ServerManager) GetJarFileByID(id uint) (*model.JarFile, error) {
	var jarFile model.JarFile
	if err := sm.db.First(&jarFile, id).Error; err != nil {
		return nil, err
	}
	return &jarFile, nil
}

// GetModPackByID retrieves a ModPack by its ID.
func (sm *ServerManager) GetModPackByID(id uint) (*model.ModPack, error) {
	var modPack model.ModPack
	if err := sm.db.First(&modPack, id).Error; err != nil {
		return nil, err
	}
	return &modPack, nil
}

func (sm *ServerManager) GetServer(id uint8) (*server.Server, error) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	fmt.Printf("Getting server with ID: %d", id)
	srv, exists := sm.servers[id]
	fmt.Printf("Exists: %v", exists)
	if !exists {
		var dbServer model.Server
		if err := sm.db.Where("id = ?", id).First(&dbServer).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil, fmt.Errorf("server %s not found", id)
			}
			return nil, fmt.Errorf("failed to fetch server from database: %w", err)
		}
		srv = server.NewServer(&dbServer)
		sm.servers[id] = srv
	}

	fmt.Printf("Server: %v\n", srv)
	details := srv.GetServerDetails()
	fmt.Printf("Details: %v\n", details)
	return srv, nil
}

func (sm *ServerManager) DeleteServer(id uint8) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	if _, exists := sm.servers[id]; !exists {
		return fmt.Errorf("server %s not found", id)
	}

	delete(sm.servers, id)
	return sm.db.Where("id = ?", id).Delete(&model.Server{}).Error
}

func (sm *ServerManager) StartServer(id uint8) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	srv, exists := sm.servers[id]
	if !exists {
		// Initialize the server instance
		var serverModel model.Server
		if err := sm.db.Where("id = ?", id).First(&serverModel).Error; err != nil {
			return fmt.Errorf("server not found: %w", err)
		}
		srv = server.NewServer(&serverModel)
		sm.servers[id] = srv
	}

	// Ensure required files are present
	if err := sm.verifyRequiredFiles(id); err != nil {
		return err
	}

	// Start the server
	if err := srv.Start(); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	// Optionally, manage output stream
	go sm.streamServerOutput(id, srv)

	return nil
}

func (sm *ServerManager) StopServer(id uint8) error {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	srv, exists := sm.servers[id]
	if !exists {
		return fmt.Errorf("server %s not found", id)
	}

	return srv.Stop()
}

func (sm *ServerManager) RestartServer(id uint8) error {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	srv, exists := sm.servers[id]
	if !exists {
		return fmt.Errorf("server %s not found", id)
	}

	return srv.Restart()
}

func (sm *ServerManager) SendCommand(id uint8, command string) (string, error) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	srv, exists := sm.servers[id]
	if !exists {
		return "", fmt.Errorf("server %s not found", id)
	}

	if err := srv.SendCommand(command); err != nil {
		return "", err
	}

	return "Command executed", nil
}

func (sm *ServerManager) UploadJarFile(name, version string, file io.Reader, baseName string, size int64, serverID string, isCommon bool) (*model.JarFile, error) {
	var jarDir string
	currentDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current working directory: %w", err)
	}

	if serverID != "" {
		// Server-specific jar file
		jarDir = filepath.Join(currentDir, sm.commonDir, "game_servers", serverID)
	} else if isCommon {
		// Common jar file
		jarDir = filepath.Join(currentDir, sm.commonDir, "jar_files")
	} else {
		return nil, fmt.Errorf("invalid arguments: must provide either serverID or set isCommon to true")
	}

	if err := os.MkdirAll(jarDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create jar directory: %w", err)
	}

	objectPath := filepath.Join(jarDir, baseName)
	// Log the jar file upload
	log.Printf("Uploading JAR file: %s (version: %s, size: %d bytes)", name, version, size)
	log.Printf("Destination path: %s", objectPath)

	if isCommon {
		log.Printf("Uploading as common JAR file")
	} else if serverID != "" {
		log.Printf("Uploading for server ID: %s", serverID)
	}

	destFile, err := os.Create(objectPath)
	if err != nil {
		log.Printf("Error creating JAR file: %v", err)
		return nil, fmt.Errorf("failed to create jar file: %w", err)
	}
	defer destFile.Close()

	bytesWritten, err := io.Copy(destFile, file)
	if err != nil {
		log.Printf("Error saving JAR file: %v", err)
		return nil, fmt.Errorf("failed to save jar file: %w", err)
	}
	log.Printf("Successfully wrote %d bytes to %s", bytesWritten, objectPath)

	jarFile := &model.JarFile{
		Name:     name,
		Version:  version,
		Path:     objectPath,
		IsCommon: isCommon,
	}

	if err := sm.db.Create(jarFile).Error; err != nil {
		log.Printf("Error creating JAR file record in database: %v", err)
		return nil, fmt.Errorf("failed to create jar file record: %w", err)
	}
	log.Printf("Successfully created JAR file record in database with ID: %d", jarFile.ID)

	return jarFile, nil
}

func (sm *ServerManager) UploadAdditionalFile(name, fileType string, file io.Reader, size int64) (*model.AdditionalFile, error) {
	additionalDir := filepath.Join(sm.commonDir, "additional_files")
	if err := os.MkdirAll(additionalDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create additional files directory: %w", err)
	}

	objectName := fmt.Sprintf("%s.zip", name)
	objectPath := filepath.Join(additionalDir, objectName)

	destFile, err := os.Create(objectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create additional file: %w", err)
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, file); err != nil {
		return nil, fmt.Errorf("failed to save additional file: %w", err)
	}

	additionalFile := &model.AdditionalFile{
		Name: name,
		Type: fileType,
		Path: objectName,
	}

	if err := sm.db.Create(additionalFile).Error; err != nil {
		return nil, fmt.Errorf("failed to create additional file record: %w", err)
	}

	return additionalFile, nil
}

func (sm *ServerManager) UploadModPack(originalFilename string, file io.Reader, size int64, serverID string, isCommon bool) (*model.ModPack, error) {
	var modPackDir string
	currentDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current working directory: %w", err)
	}
	if serverID != "" {
		// Server-specific mod pack
		modPackDir = filepath.Join(currentDir, sm.commonDir, "game_servers", serverID, "mods")
	} else if isCommon {
		// Common mod pack
		modPackDir = filepath.Join(currentDir, sm.commonDir, "mod_packs")
	} else {
		return nil, fmt.Errorf("invalid arguments: must provide either serverID or set isCommon to true")
	}

	if err := os.MkdirAll(modPackDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create mod pack directory: %w", err)
	}

	objectPath := filepath.Join(modPackDir, originalFilename)

	destFile, err := os.Create(objectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create mod pack file: %w", err)
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, file); err != nil {
		return nil, fmt.Errorf("failed to save mod pack file: %w", err)
	}

	modPack := &model.ModPack{
		Name:     originalFilename,
		Path:     objectPath,
		IsCommon: isCommon,
	}

	if err := sm.db.Create(modPack).Error; err != nil {
		return nil, fmt.Errorf("failed to create mod pack record: %w", err)
	}

	return modPack, nil
}

// SetupServer sets up the server by reading its config and loading necessary files
func (sm *ServerManager) SetupServer(serverName string) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	// Log the start of the setup process
	log.Printf("Starting setup for server: %s", serverName)

	// Fetch server model from database
	var serverModel model.Server
	if err := sm.db.Where("name = ?", serverName).First(&serverModel).Error; err != nil {
		log.Printf("Error fetching server model: %v", err)
		return fmt.Errorf("server %s not found: %w", serverName, err)
	}
	log.Printf("Fetched server model: %+v", serverModel)

	// Fetch server configuration
	var config model.ServerConfig
	if err := sm.db.Preload("JarFile").Preload("ModPack").Where("server_id = ?", serverModel.ID).First(&config).Error; err != nil {
		log.Printf("Error fetching server config for server %s: %v", serverName, err)
		return fmt.Errorf("failed to fetch server config: %w", err)
	}
	log.Printf("Fetched server config: %+v", config)

	envDir := filepath.Join(serverModel.Path, "env")
	if err := os.MkdirAll(envDir, 0755); err != nil {
		log.Printf("Error creating environment directory %s: %v", envDir, err)
		return fmt.Errorf("failed to create environment directory: %w", err)
	}
	log.Printf("Created environment directory: %s", envDir)

	// Symlink or copy JAR file
	if config.JarFile.ID != 0 {
		jarSource := config.JarFile.Path
		jarDest := filepath.Join(envDir, "server.jar")
		log.Printf("Creating symlink for JAR file from %s to %s", jarSource, jarDest)
		if err := utils.CreateSymlink(jarSource, jarDest); err != nil {
			log.Printf("Error symlinking JAR file: %v", err)
			return fmt.Errorf("failed to symlink jar file: %w", err)
		}
		log.Printf("Successfully symlinked JAR file to %s", jarDest)
	}

	// Symlink or copy Mod Pack
	if config.ModPack != nil {
		modPackSource := config.ModPack.Path
		modPackDest := filepath.Join(envDir, "mods")
		log.Printf("Creating symlink for Mod Pack from %s to %s", modPackSource, modPackDest)
		if err := utils.CreateSymlink(modPackSource, modPackDest); err != nil {
			log.Printf("Error symlinking mod pack: %v", err)
			return fmt.Errorf("failed to symlink mod pack: %w", err)
		}
		log.Printf("Successfully symlinked Mod Pack to %s", modPackDest)
	}

	// Create or update the server instance in the servers map
	srv := server.NewServer(&serverModel)
	sm.servers[uint8(serverModel.ID)] = srv
	log.Printf("Server instance for %s created/updated in the servers map", serverName)

	return nil
}

// ListServers returns a list of all servers
func (sm *ServerManager) ListServers() ([]model.Server, error) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	var servers []model.Server
	if err := sm.db.Find(&servers).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch servers from database: %w", err)
	}

	return servers, nil
}

// GetJarFiles retrieves JAR files. If common is true, only common JAR files are returned.
func (sm *ServerManager) GetJarFiles(common bool) ([]model.JarFile, error) {
	var jarFiles []model.JarFile
	query := sm.db
	if common {
		query = query.Where("is_common = ?", true)
	}
	if err := query.Find(&jarFiles).Error; err != nil {
		return nil, err
	}
	return jarFiles, nil
}

// GetModPacks retrieves mod packs. If common is true, only common mod packs are returned.
func (sm *ServerManager) GetModPacks(common bool) ([]model.ModPack, error) {
	var modPacks []model.ModPack
	query := sm.db
	if common {
		query = query.Where("is_common = ?", true)
	}
	if err := query.Find(&modPacks).Error; err != nil {
		return nil, err
	}
	return modPacks, nil
}

// UpdateServerCommand updates the executable command for a server
func (sm *ServerManager) UpdateServerCommand(id uint8, command string) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	var serverModel model.Server
	if err := sm.db.Where("id = ?", id).First(&serverModel).Error; err != nil {
		return fmt.Errorf("server not found: %w", err)
	}

	serverConfig, err := sm.getServerConfig(id)
	if err != nil {
		return fmt.Errorf("failed to get server config: %w", err)
	}

	serverConfig.ExecutableCommand = command
	if err := sm.db.Save(&serverConfig).Error; err != nil {
		return fmt.Errorf("failed to update server command: %w", err)
	}

	return nil
}

// verifyRequiredFiles checks if necessary files are present in the server directory
func (sm *ServerManager) verifyRequiredFiles(id uint8) error {
	var serverModel model.Server
	if err := sm.db.Where("id = ?", id).First(&serverModel).Error; err != nil {
		return fmt.Errorf("server not found: %w", err)
	}

	// envDir := fmt.Sprintf("%s/env", serverModel.Path)
	// requiredFiles := []string{"server.jar", "start.sh"}

	// for _, file := range requiredFiles {
	// 	filePath := fmt.Sprintf("%s/%s", envDir, file)
	// 	if _, err := os.Stat(filePath); os.IsNotExist(err) {
	// 		return fmt.Errorf("required file %s is missing", file)
	// 	}
	// }

	return nil
}

// updateServerOutput appends a new line to the server's output
func (sm *ServerManager) updateServerOutput(id uint8, line string) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	// This is a simplistic implementation.
	// For a scalable solution, consider using a concurrent safe data structure or streaming.
	// You might also implement WebSockets or Server-Sent Events (SSE) to push updates to the frontend.
	var serverOutput bytes.Buffer
	serverOutput.WriteString(line + "\n")
	// Save the output buffer to a map or database as needed
	// For demonstration, we'll skip persistent storage
}

// GetServerOutput retrieves the accumulated output for a server
func (sm *ServerManager) GetServerOutput(id uint8) (string, error) {
	// Implement a way to retrieve the server's output
	// This could be from an in-memory buffer, a file, or a database
	// For simplicity, we'll return a placeholder
	return "Server output would appear here...", nil
}

// getServerConfig retrieves the server's configuration
func (sm *ServerManager) getServerConfig(id uint8) (*model.ServerConfig, error) {
	var config model.ServerConfig
	if err := sm.db.Where("server_id = ?", id).First(&config).Error; err != nil {
		return nil, err
	}
	return &config, nil
}

// SubscribeOutput allows handlers to receive server output
func (sm *ServerManager) SubscribeOutput(id uint8) (chan string, error) {
	sm.streamMutex.Lock()
	defer sm.streamMutex.Unlock()

	ch := make(chan string, 100)
	sm.outputStreams[id] = append(sm.outputStreams[id], ch)
	return ch, nil
}

// UnsubscribeOutput removes a handler from receiving server output
func (sm *ServerManager) UnsubscribeOutput(id uint8, ch chan string) {
	sm.streamMutex.Lock()
	defer sm.streamMutex.Unlock()

	streams := sm.outputStreams[id]
	for i, subscriber := range streams {
		if subscriber == ch {
			sm.outputStreams[id] = append(streams[:i], streams[i+1:]...)
			close(ch)
			break
		}
	}
}

// streamServerOutput sends server output to all subscribers
func (sm *ServerManager) streamServerOutput(id uint8, srv *server.Server) {
	for line := range srv.GetConsole() {
		sm.streamMutex.RLock()
		subscribers := sm.outputStreams[id]
		sm.streamMutex.RUnlock()

		for _, ch := range subscribers {
			select {
			case ch <- line:
			default:
				// Handle slow consumers or drop messages
			}
		}
	}
}

func (sm *ServerManager) GetExecutableCommand(id uint8) (string, error) {
	_, ok := sm.servers[id]
	if !ok {
		return "", fmt.Errorf("server %d not found", id)
	}

	config, err := sm.GetServerConfig(id)
	if err != nil {
		return "", fmt.Errorf("failed to get server config: %w", err)
	}

	return config.ExecutableCommand, nil
}

func (sm *ServerManager) GetServerConfig(id uint8) (*model.ServerConfig, error) {
	var config model.ServerConfig
	if err := sm.db.Where("server_id = ?", id).First(&config).Error; err != nil {
		return nil, fmt.Errorf("failed to get server config: %w", err)
	}
	return &config, nil
}
