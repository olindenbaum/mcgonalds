package db

import (
	"fmt"
	"log"

	"github.com/olindenbaum/mcgonalds/internal/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func NewDatabase(cfg *config.DatabaseConfig) (*gorm.DB, error) {
	var dsn string
	if cfg.SSLMode {
		dsn = fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=enable",
			cfg.Host, cfg.User, cfg.Password, cfg.DBName, cfg.Port)
	} else {
		dsn = fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable",
			cfg.Host, cfg.User, cfg.Password, cfg.DBName, cfg.Port)
	}

	fmt.Println(dsn)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	log.Println("Connected to database successfully")
	return db, nil
}
