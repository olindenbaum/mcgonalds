# Database connection string (replace with your actual details)
DB_CONNECTION_STRING := "postgres://username:password@localhost:5432/database_name?sslmode=disable"

# Default target
.PHONY: all
all: run

# Run the application
.PHONY: run
run:
	go run cmd/server/main.go

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

# Clean built binaries
.PHONY: clean
clean:
	rm -f mcgonalds

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
