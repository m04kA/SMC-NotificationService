.PHONY: help build run test clean clean-all docker-build docker-up docker-down docker-restart docker-logs docker-clean docker-prune migrate-up migrate-down db-reset

# Variables
APP_NAME=notification-service
DB_NAME=smc_notificationservice
DOCKER_COMPOSE=docker-compose
GO=go

# Default target
help:
	@echo "Available commands:"
	@echo "  make build          - Build the application binary"
	@echo "  make run            - Run the application locally"
	@echo "  make test           - Run tests"
	@echo "  make clean          - Clean build artifacts and logs"
	@echo "  make clean-all      - Clean everything (artifacts, logs, Docker volumes, images)"
	@echo ""
	@echo "Docker commands:"
	@echo "  make docker-build   - Build Docker image"
	@echo "  make docker-up      - Start all services (app, db, migrate)"
	@echo "  make docker-down    - Stop all services"
	@echo "  make docker-restart - Restart all services"
	@echo "  make docker-logs    - Show logs from all containers"
	@echo "  make docker-logs-app - Show logs from app container only"
	@echo "  make docker-clean   - Stop services and remove volumes"
	@echo "  make docker-prune   - Remove all project Docker images"
	@echo ""
	@echo "Database commands:"
	@echo "  make migrate-up     - Apply database migrations"
	@echo "  make migrate-down   - Rollback database migrations"
	@echo "  make db-reset       - Reset database (down volumes + up)"
	@echo ""
	@echo "Development commands:"
	@echo "  make dev            - Start only database for local development"
	@echo "  make install        - Install Go dependencies"

# Build commands
build:
	@echo "Building application..."
	@$(GO) build -o bin/$(APP_NAME) ./cmd/main.go
	@echo "Build complete: bin/$(APP_NAME)"

run:
	@echo "Running application locally..."
	@$(GO) run cmd/main.go

test:
	@echo "Running tests..."
	@$(GO) test ./... -v

clean:
	@echo "Cleaning build artifacts and logs..."
	@rm -rf bin/
	@rm -rf logs/*.log
	@echo "Clean complete"

clean-all: clean docker-clean docker-prune
	@echo "Full cleanup complete"

# Docker commands
docker-build:
	@echo "Building Docker image..."
	@$(DOCKER_COMPOSE) build app
	@echo "Docker image built successfully"

docker-up:
	@echo "Starting all services..."
	@$(DOCKER_COMPOSE) up -d
	@echo "Services started. Access:"
	@echo "  - App:        http://localhost:8085"
	@echo "  - Health:     http://localhost:8085/health"
	@echo "  - Metrics:    http://localhost:8085/metrics"
	@echo "  - Webhook:    http://localhost:8085/webhook/telegram"
	@echo "  - PostgreSQL: localhost:5440"

docker-down:
	@echo "Stopping all services..."
	@$(DOCKER_COMPOSE) down
	@echo "Services stopped"

docker-restart:
	@echo "Restarting all services..."
	@$(DOCKER_COMPOSE) restart
	@echo "Services restarted"

docker-logs:
	@$(DOCKER_COMPOSE) logs -f

docker-logs-app:
	@$(DOCKER_COMPOSE) logs -f app

docker-clean:
	@echo "Stopping services and removing volumes..."
	@$(DOCKER_COMPOSE) down -v
	@echo "Docker volumes removed"

docker-prune:
	@echo "Removing project Docker images..."
	@docker images | grep smc-notificationservice | awk '{print $$3}' | xargs -r docker rmi -f || true
	@echo "Docker images removed"

# Database commands
migrate-up:
	@echo "Applying database migrations..."
	@$(DOCKER_COMPOSE) up -d postgres
	@sleep 3
	@$(DOCKER_COMPOSE) up migrate
	@echo "Migrations applied"

migrate-down:
	@echo "Rolling back database migrations..."
	@$(DOCKER_COMPOSE) run --rm migrate -path /migrations -database "postgres://postgres:postgres@postgres:5432/$(DB_NAME)?sslmode=disable" down
	@echo "Migrations rolled back"

db-reset:
	@echo "Resetting database..."
	@$(DOCKER_COMPOSE) down -v
	@$(DOCKER_COMPOSE) up -d postgres
	@sleep 5
	@$(DOCKER_COMPOSE) up migrate
	@echo "Database reset complete"

# Development helpers
dev:
	@echo "Starting development environment..."
	@$(DOCKER_COMPOSE) up -d postgres
	@sleep 3
	@$(DOCKER_COMPOSE) up migrate
	@echo "Database started. Run 'make run' to start the app locally"

install:
	@echo "Installing Go dependencies..."
	@$(GO) mod download
	@$(GO) mod tidy
	@echo "Dependencies installed"
