# SUDO Kanban Board Makefile

.PHONY: install dev build clean test docker-build docker-run

# Variables
APP_NAME=sudo-kanban
BINARY_NAME=bin/$(APP_NAME)
TEMPL_VERSION=v0.3.943
AIR_VERSION=v1.61.1

# Install development dependencies
install:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy
	npm install
	@echo "Installing development tools..."
	go install github.com/a-h/templ/cmd/templ@$(TEMPL_VERSION)
	go install github.com/air-verse/air@$(AIR_VERSION)
	@echo "Installation complete!"

# Development server with hot reload
dev:
	@echo "Starting development server..."
	@make -j3 dev-templ dev-tailwind dev-air

# Generate templ files and watch for changes
dev-templ:
	@echo "Watching Templ files..."
	templ generate --watch --proxy="http://localhost:8080" --proxyport=8081

# Build TailwindCSS and watch for changes
dev-tailwind:
	@echo "Building and watching TailwindCSS..."
	npx tailwindcss -i ./static/css/input.css -o ./static/css/styles.css --watch

# Run Air for Go hot reload
dev-air:
	@echo "Starting Air hot reload..."
	air

# Build for production
build:
	@echo "Building for production..."
	templ generate
	npx tailwindcss -i ./static/css/input.css -o ./static/css/styles.css --minify
	go build -ldflags="-s -w" -o $(BINARY_NAME) cmd/server/main.go
	@echo "Build complete: $(BINARY_NAME)"

# Clean generated files
clean:
	@echo "Cleaning generated files..."
	rm -rf bin/
	rm -f **/*_templ.go
	rm -f static/css/styles.css
	rm -rf node_modules/
	@echo "Clean complete!"

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...
	templ fmt .

# Lint code
lint:
	@echo "Linting code..."
	golangci-lint run

# Generate templ files
generate:
	@echo "Generating templ files..."
	templ generate

# Build CSS
build-css:
	@echo "Building TailwindCSS..."
	npx tailwindcss -i ./static/css/input.css -o ./static/css/styles.css --minify

# Docker commands
docker-build:
	@echo "Building Docker image..."
	docker build -t $(APP_NAME) .

docker-run:
	@echo "Running Docker container..."
	docker run -p 8080:8080 --env-file .env $(APP_NAME)

# Database commands
db-reset:
	@echo "Resetting database..."
	@echo "Run the schema.sql in your Supabase dashboard"

# Create .env file from template
env:
	@if [ ! -f .env ]; then \
		echo "Creating .env file..."; \
		cp .env.example .env; \
		echo ".env file created. Please fill in your values."; \
	else \
		echo ".env file already exists"; \
	fi

# Setup project (run after clone)
setup: install env
	@echo "Project setup complete!"
	@echo "Next steps:"
	@echo "   1. Fill in your .env file with Supabase credentials"
	@echo "   2. Run the schema.sql in your Supabase dashboard"
	@echo "   3. Run 'make dev' to start development server"

# Production deployment
deploy-prep: clean build
	@echo "Prepared for deployment!"
	@echo "Binary: $(BINARY_NAME)"
	@echo "Static files: static/"

# Help
help:
	@echo "SUDO Kanban Board - Available Commands:"
	@echo ""
	@echo "Development:"
	@echo "  make install    - Install all dependencies"
	@echo "  make dev        - Start development server with hot reload"
	@echo "  make setup      - Complete project setup after clone"
	@echo ""
	@echo "Building:"
	@echo "  make build      - Build for production"
	@echo "  make generate   - Generate templ files"
	@echo "  make build-css  - Build TailwindCSS"
	@echo ""
	@echo "Maintenance:"
	@echo "  make clean      - Clean generated files"
	@echo "  make test       - Run tests"
	@echo "  make fmt        - Format code"
	@echo "  make lint       - Lint code"
	@echo ""
	@echo "Docker:"
	@echo "  make docker-build - Build Docker image"
	@echo "  make docker-run   - Run Docker container"
	@echo ""
	@echo "Database:"
	@echo "  make db-reset   - Reset database (manual step)"
	@echo ""
	@echo "Deployment:"
	@echo "  make deploy-prep - Prepare for deployment"