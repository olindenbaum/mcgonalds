package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	_ "github.com/olindenbaum/mcgonalds/docs" // This line is important
	"github.com/olindenbaum/mcgonalds/internal/config"
	"github.com/olindenbaum/mcgonalds/internal/db"
	"github.com/olindenbaum/mcgonalds/internal/handlers"
	"github.com/olindenbaum/mcgonalds/internal/middleware"
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

	err = db.InitDatabase(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	database := db.GetDB()
	// Initialize ServerManager with local storage directory (e.g., "/game_servers/shared")
	sharedDir := "/game_servers/shared"
	sm, err := server_manager.NewServerManager(database, sharedDir)
	if err != nil {
		log.Fatalf("Failed to create server manager: %v", err)
	}

	h := handlers.NewHandler(database, sm, cfg)

	r := mux.NewRouter()
	r.Use(middleware.DebugMiddleware)
	// API routes
	authApi := r.PathPrefix("/api/v1").Subrouter()
	authApi.Use(middleware.AuthMiddleware(&cfg.JWTConfig))
	h.RegisterAuthenticatedRoutes(authApi)

	// Create a separate subrouter for unauthenticated routes
	unauthApi := r.PathPrefix("/api/v1").Subrouter()
	h.RegisterUnauthenticatedRoutes(unauthApi)

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

	err = r.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		pathTemplate, err := route.GetPathTemplate()
		if err == nil {
			fmt.Println("ROUTE:", pathTemplate)
		}
		pathRegexp, err := route.GetPathRegexp()
		if err == nil {
			fmt.Println("Path regexp:", pathRegexp)
		}
		queriesTemplates, err := route.GetQueriesTemplates()
		if err == nil {
			fmt.Println("Queries templates:", strings.Join(queriesTemplates, ","))
		}
		queriesRegexps, err := route.GetQueriesRegexp()
		if err == nil {
			fmt.Println("Queries regexps:", strings.Join(queriesRegexps, ","))
		}
		methods, err := route.GetMethods()
		if err == nil {
			fmt.Println("Methods:", strings.Join(methods, ","))
		}
		fmt.Println()
		return nil
	})

	if err != nil {
		fmt.Println(err)
	}

}
