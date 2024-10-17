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
	servers   map[string]*server.Server
	mutex     sync.RWMutex
	commonDir string
}

func NewServerManager(db *gorm.DB, commonDir string) (*ServerManager, error) {
	return &ServerManager{
		db:        db,
		servers:   make(map[string]*server.Server),
		commonDir: commonDir,
	}, nil
}

func (sm *ServerManager) CreateServer(name, path string, jarFileID uint, additionalFileIDs []uint) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	if _, exists := sm.servers[name]; exists {
		return fmt.Errorf("server %s already exists", name)
	}

	serverModel := &model.Server{
		Name:              name,
		Path:              path,
		JarFileID:         jarFileID,
		AdditionalFileIDs: additionalFileIDs,
	}

	if err := sm.db.Create(serverModel).Error; err != nil {
		return fmt.Errorf("failed to create server in database: %w", err)
	}

	// Load the related JarFile and AdditionalFiles
	if err := sm.db.Preload("JarFile").Preload("AdditionalFiles").First(serverModel, serverModel.ID).Error; err != nil {
		return fmt.Errorf("failed to load server details: %w", err)
	}

	// Create server directory
	serverEnvDir := filepath.Join(path, "env")
	if err := os.MkdirAll(serverEnvDir, 0755); err != nil {
		return fmt.Errorf("failed to create server environment directory: %w", err)
	}

	// Handle symbolic links for common resources
	if serverModel.JarFileID != 0 {
		jarSource := filepath.Join(sm.commonDir, "jar_files", serverModel.JarFile.Path)
		jarDest := filepath.Join(serverEnvDir, "file.jar")
		if err := utils.CreateSymlink(jarSource, jarDest); err != nil {
			return fmt.Errorf("failed to create symlink for jar file: %w", err)
		}
	}

	for _, additionalFile := range serverModel.AdditionalFiles {
		source := filepath.Join(sm.commonDir, "additional_files", additionalFile.Path)
		dest := filepath.Join(serverEnvDir, additionalFile.Name)
		if err := utils.CreateSymlink(source, dest); err != nil {
			return fmt.Errorf("failed to create symlink for additional file: %w", err)
		}
	}

	sm.servers[name] = server.NewServer(serverModel)
	return nil
}

func (sm *ServerManager) GetServer(name string) (*server.Server, error) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	srv, exists := sm.servers[name]
	if !exists {
		var dbServer model.Server
		if err := sm.db.Preload("JarFile").Preload("AdditionalFiles").Where("name = ?", name).First(&dbServer).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil, fmt.Errorf("server %s not found", name)
			}
			return nil, fmt.Errorf("failed to fetch server from database: %w", err)
		}
		srv = server.NewServer(&dbServer)
		sm.servers[name] = srv
	}
	return srv, nil
}

func (sm *ServerManager) DeleteServer(name string) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	if _, exists := sm.servers[name]; !exists {
		return fmt.Errorf("server %s not found", name)
	}

	delete(sm.servers, name)
	return sm.db.Where("name = ?", name).Delete(&model.Server{}).Error
}

func (sm *ServerManager) StartServer(name string) error {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	srv, exists := sm.servers[name]
	if !exists {
		return fmt.Errorf("server %s not found", name)
	}

	return srv.Start()
}

func (sm *ServerManager) StopServer(name string) error {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	srv, exists := sm.servers[name]
	if !exists {
		return fmt.Errorf("server %s not found", name)
	}

	return srv.Stop()
}

func (sm *ServerManager) RestartServer(name string) error {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	srv, exists := sm.servers[name]
	if !exists {
		return fmt.Errorf("server %s not found", name)
	}

	return srv.Restart()
}

func (sm *ServerManager) SendCommand(name, command string) (string, error) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	srv, exists := sm.servers[name]
	if !exists {
		return "", fmt.Errorf("server %s not found", name)
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

	// Fetch server model from database
	var serverModel model.Server
	if err := sm.db.Where("name = ?", serverName).First(&serverModel).Error; err != nil {
		return fmt.Errorf("server %s not found: %w", serverName, err)
	}

	// Fetch server configuration
	var config model.ServerConfig
	if err := sm.db.Preload("JarFile").Preload("ModPack").Where("server_id = ?", serverModel.ID).First(&config).Error; err != nil {
		return fmt.Errorf("failed to fetch server config: %w", err)
	}

	envDir := filepath.Join(serverModel.Path, "env")
	if err := os.MkdirAll(envDir, 0755); err != nil {
		return fmt.Errorf("failed to create environment directory: %w", err)
	}

	// Symlink or copy JAR file
	if config.JarFile.ID != 0 {
		jarSource := config.JarFile.Path
		jarDest := filepath.Join(envDir, "server.jar")
		if err := utils.CreateSymlink(jarSource, jarDest); err != nil {
			return fmt.Errorf("failed to symlink jar file: %w", err)
		}
	}

	// Symlink or copy Mod Pack
	if config.ModPack != nil {
		modPackSource := config.ModPack.Path
		modPackDest := filepath.Join(envDir, "mods")
		if err := utils.CreateSymlink(modPackSource, modPackDest); err != nil {
			return fmt.Errorf("failed to symlink mod pack: %w", err)
		}
	}

	// Create or update the server instance in the servers map
	srv := server.NewServer(&serverModel)
	sm.servers[serverName] = srv

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
