package main

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
	_ "github.com/olindenbaum/mcgonalds/docs" // This line is important
	"github.com/olindenbaum/mcgonalds/internal/config"
	"github.com/olindenbaum/mcgonalds/internal/db"
	"github.com/olindenbaum/mcgonalds/internal/handlers"
	"github.com/olindenbaum/mcgonalds/internal/server_manager"
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

// @title Minecraft Server Manager API
// @version 1.0
// @description This is a Minecraft server management service API
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /api/v1
func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	database, err := db.NewDatabase(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Initialize ServerManager with local storage directory (e.g., "/game_servers/shared")
	sharedDir := "/game_servers/shared"
	sm, err := server_manager.NewServerManager(database, sharedDir)
	if err != nil {
		log.Fatalf("Failed to create server manager: %v", err)
	}

	h := handlers.NewHandler(database, sm, cfg)

	r := mux.NewRouter()

	// API routes
	api := r.PathPrefix("/api/v1").Subrouter()
	h.RegisterRoutes(api)

	// Serve Swagger UI
	r.PathPrefix("/swagger/").Handler(httpSwagger.Handler(
		httpSwagger.URL("http://localhost:8080/swagger/doc.json"), // The URL pointing to API definition
		httpSwagger.DeepLinking(true),
		httpSwagger.DocExpansion("none"),
		httpSwagger.DomID("swagger-ui"),
	)).Methods(http.MethodGet)

	log.Printf("Starting server on port %s", cfg.Server.Port)
	log.Printf("API documentation available at http://localhost:%s/swagger/index.html", cfg.Server.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Server.Port, r))
}
