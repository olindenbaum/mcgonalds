package server

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/olindenbaum/mcgonalds/internal/db"
	"github.com/olindenbaum/mcgonalds/internal/model"
)

// Server represents a Minecraft server instance.
type Server struct {
	model     *model.Server
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	stdout    io.ReadCloser
	console   chan string
	mutex     sync.Mutex
	isRunning bool
	stopOnce  sync.Once
	doneOnce  sync.Once
	done      chan struct{}
}

// NewServer initializes a new Server instance.
func NewServer(model *model.Server) *Server {
	return &Server{
		model:   model,
		console: make(chan string, 100),
		done:    make(chan struct{}),
	}
}

// GetConsole returns a read-only channel for server console output.
func (s *Server) GetConsole() <-chan string {
	return s.console
}

// Start launches the server process.
func (s *Server) Start() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.isRunning {
		return fmt.Errorf("server is already running")
	}

	config, err := s.GetConfig()
	if err != nil {
		return fmt.Errorf("failed to get server config: %w", err)
	}

	log.Printf("ExecutableCommand: %s", config.ExecutableCommand)

	// Split the command into the executable and its arguments
	parts := strings.Fields(config.ExecutableCommand)
	if len(parts) == 0 {
		return fmt.Errorf("invalid executable command")
	}

	executable := parts[0]
	args := parts[1:]

	s.cmd = exec.Command(executable, args...)
	s.cmd.Dir = s.model.Path

	var errBuffer bytes.Buffer
	s.cmd.Stderr = &errBuffer

	s.stdin, err = s.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	s.stdout, err = s.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := s.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	s.isRunning = true

	go s.readConsole()
	go s.monitorProcess()

	return nil
}

// readConsole reads the server's stdout and sends it to the console channel.
func (s *Server) readConsole() {
	scanner := bufio.NewScanner(s.stdout)
	for scanner.Scan() {
		line := scanner.Text()
		select {
		case s.console <- line:
		case <-s.done:
			return
		}
		log.Printf("[%s] %s", s.model.Name, line)
	}
	if err := scanner.Err(); err != nil {
		log.Printf("Error reading server output: %v", err)
	}
	// Close console channel only once
	s.doneOnce.Do(func() {
		close(s.console)
	})
}

// monitorProcess waits for the server process to exit and handles cleanup.
func (s *Server) monitorProcess() {
	err := s.cmd.Wait()

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if err != nil {
		log.Printf("Server %s exited with error: %v", s.model.Name, err)
	} else {
		log.Printf("Server %s stopped gracefully", s.model.Name)
	}

	s.isRunning = false

	// Signal done once
	s.doneOnce.Do(func() {
		close(s.done)
	})
}

// Stop gracefully stops the server process.
func (s *Server) Stop() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.isRunning {
		return fmt.Errorf("server is not running")
	}

	// Ensure stop is called only once
	s.stopOnce.Do(func() {
		if err := s.cmd.Process.Signal(os.Interrupt); err != nil {
			log.Printf("Failed to send interrupt signal: %v", err)
		}
	})

	s.isRunning = false

	return nil
}

// Restart stops and then starts the server.
func (s *Server) Restart() error {
	if err := s.Stop(); err != nil {
		return err
	}
	return s.Start()
}

// SendCommand sends a command to the server's stdin.
func (s *Server) SendCommand(command string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.isRunning {
		return fmt.Errorf("server is not running")
	}

	_, err := io.WriteString(s.stdin, command+"\n")
	if err != nil {
		return fmt.Errorf("failed to send command: %w", err)
	}
	return nil
}

// ListFiles lists all files in the server's environment directory.
func (s *Server) ListFiles() ([]string, error) {
	dirPath := filepath.Join(s.model.Path, "env")
	files, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read server directory: %w", err)
	}

	var fileNames []string
	for _, file := range files {
		fileNames = append(fileNames, file.Name())
	}

	return fileNames, nil
}

// UploadFile uploads a file to the server's environment directory.
func (s *Server) UploadFile(fileName string, content io.Reader) error {
	filePath := filepath.Join(s.model.Path, "env", fileName)
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	_, err = io.Copy(file, content)
	if err != nil {
		return fmt.Errorf("failed to write file content: %w", err)
	}

	return nil
}

// DeleteFile deletes a file from the server's environment directory.
func (s *Server) DeleteFile(fileName string) error {
	filePath := filepath.Join(s.model.Path, "env", fileName)
	err := os.Remove(filePath)
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	return nil
}

// IsRunning returns whether the server is currently running.
func (s *Server) IsRunning() bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.isRunning
}

// GetName returns the server's name.
func (s *Server) GetName() string {
	return s.model.Name
}

// GetPath returns the server's path.
func (s *Server) GetPath() string {
	return s.model.Path
}

// GetConfig retrieves the server's configuration from the database.
func (s *Server) GetConfig() (*model.ServerConfig, error) {
	var config model.ServerConfig
	if err := db.GetDB().Where("server_id = ?", s.GetServerId()).First(&config).Error; err != nil {
		return nil, fmt.Errorf("failed to get server config: %w", err)
	}
	return &config, nil
}

// GetServerId returns the server's ID as uint8.
func (s *Server) GetServerId() uint8 {
	return uint8(s.model.ID)
}

// String returns a string representation of the server.
func (s *Server) String() string {
	return fmt.Sprintf("Server{Name: %s, Path: %s, IsRunning: %t}", s.model.Name, s.model.Path, s.isRunning)
}

// GetServerDetails returns detailed information about the server.
func (s *Server) GetServerDetails() *ServerDetails {
	config, err := s.GetConfig()
	if err != nil {
		log.Printf("Failed to get server config: %v", err)
	}
	return &ServerDetails{
		Name:      s.model.Name,
		Path:      s.model.Path,
		IsRunning: s.isRunning,
		ServerId:  uint8(s.model.ID),
		Config:    *config,
	}
}
