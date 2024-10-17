package server

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/olindenbaum/mcgonalds/internal/model"
)

type Server struct {
	model      *model.Server
	cmd        *exec.Cmd
	stdin      io.WriteCloser
	stdout     io.ReadCloser
	console    chan string
	mutex      sync.Mutex
	isRunning  bool
	stopSignal chan struct{}
}

func NewServer(model *model.Server) *Server {
	return &Server{
		model:      model,
		console:    make(chan string, 100),
		stopSignal: make(chan struct{}),
	}
}

func (s *Server) Start() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.isRunning {
		return fmt.Errorf("server is already running")
	}

	jarPath := filepath.Join(s.model.Path, "env", "file.jar")
	s.cmd = exec.Command("java", "-jar", jarPath)

	s.cmd.Dir = s.model.Path

	var err error
	s.stdin, err = s.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	s.stdout, err = s.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	s.cmd.Stderr = s.cmd.Stdout // Redirect stderr to stdout

	if err := s.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	s.isRunning = true
	go s.readConsole()
	go s.monitorProcess()

	return nil
}

func (s *Server) readConsole() {
	scanner := bufio.NewScanner(s.stdout)
	for scanner.Scan() {
		line := scanner.Text()
		s.console <- line
		log.Printf("[%s] %s", s.model.Name, line) // Log server output
	}
	if err := scanner.Err(); err != nil {
		log.Printf("Error reading server output: %v", err)
	}
	close(s.console)
}

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
}

// Stop gracefully stops the server
func (s *Server) Stop() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.isRunning {
		return fmt.Errorf("server is not running")
	}

	if err := s.cmd.Process.Signal(os.Interrupt); err != nil {
		return fmt.Errorf("failed to send interrupt signal: %w", err)
	}

	s.isRunning = false
	return nil
}

func (s *Server) Restart() error {
	if err := s.Stop(); err != nil {
		return err
	}
	return s.Start()
}

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

func (s *Server) DeleteFile(fileName string) error {
	filePath := filepath.Join(s.model.Path, "env", fileName)
	err := os.Remove(filePath)
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	return nil
}

func (s *Server) IsRunning() bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.isRunning
}

func (s *Server) GetName() string {
	return s.model.Name
}

func (s *Server) GetPath() string {
	return s.model.Path
}
