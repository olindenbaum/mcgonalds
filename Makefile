# Database connection string (replace with your actual details)
DB_CONNECTION_STRING := "postgres://postgres:changeme@localhost:5432/mcgonalds_db?sslmode=disable"

# Default target
.PHONY: all
all: setup run

# Run the application
.PHONY: run
run:
	go run main.go

# Run database migrations
.PHONY: migrate
migrate:
	goose -dir migrations postgres $(DB_CONNECTION_STRING) up

# Generate Swagger documentation
.PHONY: swagger
swagger:
	swag init

# Build the application
.PHONY: build
build:
	go build -o mcgonalds cmd/server/main.go


# Run tests
.PHONY: test
test:
	go test ./...

# Full development cycle: migrate, generate swagger, and run
.PHONY: dev
dev: migrate swagger run

# Help command to list available targets
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  run      - Run the application"
	@echo "  migrate  - Run database migrations"
	@echo "  swagger  - Generate Swagger documentation"
	@echo "  build    - Build the application"
	@echo "  clean    - Remove built binaries"
	@echo "  test     - Run tests"
	@echo "  dev      - Run migrations, generate Swagger docs, and start the app"
	@echo "  help     - Show this help message"

# Create necessary directories
.PHONY: setup
setup:
	mkdir game_servers/shared/jar_files
	mkdir /game_servers/shared/additional_files
	mkdir /game_servers/servers


.PHONY: migration
migration:
	goose -dir migrations create $(name) sql
