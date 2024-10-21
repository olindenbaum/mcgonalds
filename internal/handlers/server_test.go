package handlers_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/olindenbaum/mcgonalds/internal/config"
	"github.com/olindenbaum/mcgonalds/internal/handlers"
	"github.com/olindenbaum/mcgonalds/internal/model"
	"github.com/olindenbaum/mcgonalds/internal/server_manager"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestEnvironment(t *testing.T) (*handlers.Handler, *gorm.DB, func()) {
	// Create a temporary directory for the test
	tempDir, err := os.MkdirTemp("", "mcgonalds_test")
	assert.NoError(t, err)

	// Set up a test database
	db, err := gorm.Open(sqlite.Open(filepath.Join(tempDir, "test.db")), &gorm.Config{})
	assert.NoError(t, err)

	// Run migrations
	err = db.AutoMigrate(&model.Server{}, &model.JarFile{}, &model.AdditionalFile{})
	assert.NoError(t, err)

	// Create a test configuration
	cfg := &config.Config{
		Storage: config.Storage{
			CommonDir: tempDir,
		},
	}

	// Create a server manager
	sm, err := server_manager.NewServerManager(db, cfg.Storage.CommonDir)
	assert.NoError(t, err)

	// Create a handler
	h := handlers.NewHandler(db, sm, cfg)

	// Return a cleanup function
	cleanup := func() {
		os.RemoveAll(tempDir)
	}

	return h, db, cleanup
}

func TestCreateAndStopServer(t *testing.T) {
	h, db, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// Create a test JAR file
	jarFile := &model.JarFile{
		Name:    "test.jar",
		Version: "1.0",
		Path:    filepath.Join(h.Config.Storage.CommonDir, "test.jar"),
	}
	db.Create(jarFile)

	// Test creating a server
	createServerRequest := model.CreateServerRequest{
		Name:              "test_server",
		JarFileID:         jarFile.ID,
		ExecutableCommand: "java -jar test.jar",
	}
	reqBody, _ := json.Marshal(createServerRequest)
	req, _ := http.NewRequest("POST", "/api/v1/servers", bytes.NewBuffer(reqBody))
	rr := httptest.NewRecorder()
	h.CreateServer(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)

	var createdServer model.Server
	json.Unmarshal(rr.Body.Bytes(), &createdServer)
	assert.Equal(t, "test_server", createdServer.Name)

	// Test starting the server
	req, _ = http.NewRequest("POST", fmt.Sprintf("/api/v1/servers/%s/start", createdServer.Name), nil)
	rr = httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/servers/{name}/start", h.StartServer)
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	// Wait for the server to start
	time.Sleep(2 * time.Second)

	// Test stopping the server
	req, _ = http.NewRequest("POST", fmt.Sprintf("/api/v1/servers/%s/stop", createdServer.Name), nil)
	rr = httptest.NewRecorder()
	router = mux.NewRouter()
	router.HandleFunc("/api/v1/servers/{name}/stop", h.StopServer)
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestDeleteServer(t *testing.T) {
	h, db, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// Create a test server
	server := &model.Server{
		Name: "test_server",
		Path: filepath.Join(h.Config.Storage.CommonDir, "test_server"),
		// JarFile: model.JarFile{Name: "test.jar", Version: "1.0"},
	}
	db.Create(server)

	// Test deleting the server
	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/v1/servers/%s", server.Name), nil)
	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/servers/{name}", h.DeleteServer)
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	// Verify the server was deleted from the database
	var deletedServer model.Server
	result := db.First(&deletedServer, "name = ?", server.Name)
	assert.Error(t, result.Error)
	assert.Nil(t, result.Error)
}

func TestVerifyJarOutput(t *testing.T) {
	h, _, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// Create a test JAR file
	jarContent := []byte{
		0xCA, 0xFE, 0xBA, 0xBE, // Magic number
		0x00, 0x00, 0x00, 0x34, // Java class file version
		// ... (rest of the JAR file content)
	}
	jarPath := filepath.Join(h.Config.Storage.CommonDir, "test_output.jar")
	err := os.WriteFile(jarPath, jarContent, 0644)
	assert.NoError(t, err)

	// Create a server with the test JAR
	server := &model.Server{
		Name: "test_output_server",
		Path: filepath.Join(h.Config.Storage.CommonDir, "test_output_server"),
		// JarFile: model.JarFile{Name: "test_output.jar", Version: "1.0", Path: jarPath},
	}
	h.DB.Create(server)

	// Start the server
	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/servers/%s/start", server.Name), nil)
	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/servers/{name}/start", h.StartServer)
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	// Wait for the server to start and produce output
	time.Sleep(2 * time.Second)

	// Get the server output
	req, _ = http.NewRequest("GET", fmt.Sprintf("/api/v1/servers/%s/output", server.Name), nil)
	rr = httptest.NewRecorder()
	router = mux.NewRouter()
	router.HandleFunc("/api/v1/servers/{name}/output", h.GetServerOutput)
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var output string
	json.Unmarshal(rr.Body.Bytes(), &output)
	assert.Contains(t, output, "Hello from test JAR!")

	// Stop the server
	req, _ = http.NewRequest("POST", fmt.Sprintf("/api/v1/servers/%s/stop", server.Name), nil)
	rr = httptest.NewRecorder()
	router = mux.NewRouter()
	router.HandleFunc("/api/v1/servers/{name}/stop", h.StopServer)
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}
