package server_manager

import (
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
	db        *gorm.DB
	servers   map[uint8]*server.Server
	mutex     sync.RWMutex
	commonDir string
}

func NewServerManager(db *gorm.DB, commonDir string) (*ServerManager, error) {
	return &ServerManager{
		db:        db,
		servers:   make(map[uint8]*server.Server),
		commonDir: commonDir,
	}, nil
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

	srv, exists := sm.servers[id]
	if !exists {
		var dbServer model.Server
		if err := sm.db.Preload("JarFile").Where("id = ?", id).First(&dbServer).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil, fmt.Errorf("server %s not found", id)
			}
			return nil, fmt.Errorf("failed to fetch server from database: %w", err)
		}
		srv = server.NewServer(&dbServer)
		sm.servers[id] = srv
	}
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
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	srv, exists := sm.servers[id]
	if !exists {
		return fmt.Errorf("server %s not found", id)
	}

	return srv.Start()
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
