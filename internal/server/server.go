package server

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/olindenbaum/mcgonalds/internal/model"
)

type Server struct {
	model     *model.Server
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	stdout    io.ReadCloser
	console   chan string
	mutex     sync.Mutex
	isRunning bool
}

func NewServer(model *model.Server) *Server {
	return &Server{
		model:   model,
		console: make(chan string, 100),
	}
}

func (s *Server) Start() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.isRunning {
		return fmt.Errorf("server is already running")
	}

	s.cmd = exec.Command("java", "-jar", "server.jar")
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

	err = s.cmd.Start()
	if err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	s.isRunning = true
	go s.readConsole()

	return nil
}

func (s *Server) Stop() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.isRunning {
		return fmt.Errorf("server is not running")
	}

	err := s.SendCommand("stop")
	if err != nil {
		return fmt.Errorf("failed to send stop command: %w", err)
	}

	err = s.cmd.Wait()
	if err != nil {
		return fmt.Errorf("failed to wait for server to stop: %w", err)
	}

	s.isRunning = false
	s.cmd = nil
	s.stdin = nil
	s.stdout = nil
	close(s.console)
	s.console = make(chan string, 100)

	return nil
}

func (s *Server) SendCommand(cmd string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.isRunning {
		return fmt.Errorf("server is not running")
	}

	_, err := fmt.Fprintln(s.stdin, cmd)
	if err != nil {
		return fmt.Errorf("failed to send command: %w", err)
	}

	return nil
}

func (s *Server) readConsole() {
	scanner := bufio.NewScanner(s.stdout)
	for scanner.Scan() {
		s.console <- scanner.Text()
	}
}

func (s *Server) GetConsoleChannel() <-chan string {
	return s.console
}

func (s *Server) ListFiles() ([]string, error) {
	files, err := os.ReadDir(s.model.Path)
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
	filePath := filepath.Join(s.model.Path, fileName)
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
	filePath := filepath.Join(s.model.Path, fileName)
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
