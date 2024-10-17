package config

import (
	"errors"
	"os"

	"gopkg.in/yaml.v2"
)

type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
	SSLMode  bool   `yaml:"sslmode"`
}

type Config struct {
	Server struct {
		Port string `yaml:"port"`
	} `yaml:"server"`

	Database DatabaseConfig `yaml:"database"`

	// Added local storage paths
	Storage struct {
		CommonDir string `yaml:"common_dir"`
	} `yaml:"storage"`

	JWTConfig JWTConfig `yaml:"jwt"`
}

type JWTConfig struct {
	Secret     string `yaml:"secret"`
	Expiration string `yaml:"expiration"`
}

func LoadConfig() (*Config, error) {
	cfg := &Config{}

	file, err := os.Open("config.global.yaml")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(cfg); err != nil {
		return nil, err
	}

	// Validate configurations
	if cfg.Storage.CommonDir == "" {
		return nil, errors.New("storage.common_dir must be set in config.yml")
	}

	return cfg, nil
}
