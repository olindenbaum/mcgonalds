package utils

import (
	"log"
	"os"
	"path/filepath"
)

// CreateSymlink creates a symbolic link from source to destination
func CreateSymlink(source, destination string) error {
	// Ensure the destination directory exists
	destDir := filepath.Dir(destination)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		log.Fatalf("Failed to create destination directory: %v", err)
		return err
	}

	// Remove existing symlink if it exists
	if _, err := os.Lstat(destination); err == nil {
		os.Remove(destination)
		return err
	}

	// Create the symlink
	err := os.Symlink(source, destination)
	if err != nil {
		log.Fatalf("Failed to create symlink from %s to %s: %v", source, destination, err)
		return err
	}

	log.Printf("Symlink created: %s -> %s", destination, source)
	return nil
}
