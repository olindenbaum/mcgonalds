package server_manager

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/olindenbaum/mcgonalds/internal/model"
	"github.com/olindenbaum/mcgonalds/internal/server"
	"gorm.io/gorm"
)

type ServerManager struct {
	db          *gorm.DB
	servers     map[string]*server.Server
	mutex       sync.RWMutex
	minioClient *minio.Client
}

func NewServerManager(db *gorm.DB, minioEndpoint, minioAccessKey, minioSecretKey string, minioUseSSL bool) (*ServerManager, error) {

	minioClient, err := minio.New(minioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(minioAccessKey, minioSecretKey, ""),
		Secure: minioUseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MinIO client: %w", err)
	}

	return &ServerManager{
		db:          db,
		servers:     make(map[string]*server.Server),
		minioClient: minioClient,
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

	sm.servers[name] = server.NewServer(serverModel)
	return nil
}

func (sm *ServerManager) UploadJarFile(name, version string, file io.Reader, size int64) (*model.JarFile, error) {
	bucketName := "mcgonalds-jar-files"
	objectName := fmt.Sprintf("%s-%s.jar", name, version)

	// Ensure the bucket exists
	fmt.Println("Uploading jar file to MinIO")
	fmt.Println("Bucket name:", bucketName)

	exists, errBucketExists := sm.minioClient.BucketExists(context.Background(), bucketName)
	if errBucketExists == nil && exists {
		// We already own this bucket
	} else {
		return nil, fmt.Errorf("failed to create bucket: %w", errBucketExists)
	}

	// Upload the file
	_, err := sm.minioClient.PutObject(context.Background(), bucketName, objectName, file, size, minio.PutObjectOptions{ContentType: "application/java-archive"})
	if err != nil {
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}

	// Generate the URL for the uploaded object
	url := fmt.Sprintf("http://%s/%s/%s", sm.minioClient.EndpointURL().Host, bucketName, objectName)

	// Create the database record
	jarFile := &model.JarFile{
		Name:    name,
		Version: version,
		Path:    url,
	}

	if err := sm.db.Create(jarFile).Error; err != nil {
		return nil, fmt.Errorf("failed to create jar file record: %w", err)
	}

	return jarFile, nil
}

func (sm *ServerManager) UploadAdditionalFile(name, fileType string, file io.Reader, size int64) (*model.AdditionalFile, error) {
	bucketName := "mcgonalds-additional-files"
	objectName := fmt.Sprintf("%s.zip", name)

	// Ensure the bucket exists
	err := sm.minioClient.MakeBucket(context.Background(), bucketName, minio.MakeBucketOptions{})
	if err != nil {
		// Check to see if we already own this bucket
		exists, errBucketExists := sm.minioClient.BucketExists(context.Background(), bucketName)
		if errBucketExists == nil && exists {
			// We already own this bucket
		} else {
			return nil, fmt.Errorf("failed to create bucket: %w", err)
		}
	}

	// Upload the file
	_, err = sm.minioClient.PutObject(context.Background(), bucketName, objectName, file, size, minio.PutObjectOptions{ContentType: "application/zip"})
	if err != nil {
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}

	// Generate the URL for the uploaded object
	url := fmt.Sprintf("http://%s/%s/%s", sm.minioClient.EndpointURL().Host, bucketName, objectName)

	// Create the database record
	additionalFile := &model.AdditionalFile{
		Name: name,
		Type: fileType,
		Path: url,
	}

	if err := sm.db.Create(additionalFile).Error; err != nil {
		return nil, fmt.Errorf("failed to create additional file record: %w", err)
	}

	return additionalFile, nil
}

func (sm *ServerManager) GetServer(name string) (*server.Server, error) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	srv, exists := sm.servers[name]
	if !exists {
		var dbServer model.Server
		if err := sm.db.Where("name = ?", name).First(&dbServer).Error; err != nil {
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

func (sm *ServerManager) ListServers() ([]*server.Server, error) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	var servers []*server.Server
	for _, srv := range sm.servers {
		servers = append(servers, srv)
	}

	return servers, nil
}

func (sm *ServerManager) DeleteServer(name string) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	srv, exists := sm.servers[name]
	if !exists {
		return fmt.Errorf("server %s not found", name)
	}

	if srv.IsRunning() {
		return fmt.Errorf("cannot delete a running server: %s", name)
	}

	if err := sm.db.Delete(&model.Server{}, "name = ?", name).Error; err != nil {
		return fmt.Errorf("failed to delete server from database: %w", err)
	}

	delete(sm.servers, name)
	return nil
}

func (sm *ServerManager) StartServer(name string) error {
	srv, err := sm.GetServer(name)
	if err != nil {
		return err
	}

	return srv.Start()
}

func (sm *ServerManager) StopServer(name string) error {
	srv, err := sm.GetServer(name)
	if err != nil {
		return err
	}

	return srv.Stop()
}

func (sm *ServerManager) RestartServer(name string) error {
	srv, err := sm.GetServer(name)
	if err != nil {
		return err
	}

	if err := srv.Stop(); err != nil {
		return err
	}

	return srv.Start()
}

func (sm *ServerManager) SendCommand(name, command string) error {
	srv, err := sm.GetServer(name)
	if err != nil {
		return err
	}

	return srv.SendCommand(command)
}

func (sm *ServerManager) ListFiles(name string) ([]string, error) {
	server, err := sm.GetServer(name)
	if err != nil {
		return nil, err
	}
	return server.ListFiles()
}
