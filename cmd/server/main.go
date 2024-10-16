package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/olindenbaum/mcgonalds/internal/config"
	"github.com/olindenbaum/mcgonalds/internal/db"
	"github.com/olindenbaum/mcgonalds/internal/handler"
	"github.com/olindenbaum/mcgonalds/internal/server_manager"
	httpSwagger "github.com/swaggo/http-swagger"
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

	// API routes
	api := r.PathPrefix("/api/v1").Subrouter()
	h.RegisterRoutes(api)

	// Swagger route
	port := strconv.Itoa(cfg.Server.Port)

	swaggerURL := fmt.Sprintf("http://localhost:%s/swagger/doc.json", port)
	r.PathPrefix("/swagger/").Handler(httpSwagger.Handler(
		httpSwagger.URL(swaggerURL), // The url pointing to API definition
		httpSwagger.DeepLinking(true),
		httpSwagger.DocExpansion("none"),
		httpSwagger.DomID("swagger-ui"),
	)).Methods(http.MethodGet)

	log.Printf("Starting server on port %s", port)
	log.Printf("API documentation available at http://localhost:%s/swagger/index.html", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
