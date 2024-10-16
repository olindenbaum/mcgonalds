package main

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/olindenbaum/mcgonalds/internal/config"
	"github.com/olindenbaum/mcgonalds/internal/db"
	"github.com/olindenbaum/mcgonalds/internal/handler"
	"github.com/olindenbaum/mcgonalds/internal/server_manager"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	database, err := db.NewDatabase(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	sm, err := server_manager.NewServerManager(database, cfg.MinIO.Endpoint, cfg.MinIO.AccessKey, cfg.MinIO.SecretKey, cfg.MinIO.UseSSL)
	if err != nil {
		log.Fatalf("Failed to create server manager: %v", err)
	}

	h := handler.NewHandler(database, sm, cfg)

	r := mux.NewRouter()
	h.RegisterRoutes(r)

	log.Printf("Starting server on port %d", cfg.Server.Port)
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(cfg.Server.Port), r))
}
