.PHONY: dev dev-backend dev-frontend dev-db build clean test help start stop up down

# Variables
BACKEND_DIR := backend
FRONTEND_DIR := frontend

help: ## Show this help message
	@echo "CT-SaaS - Development Commands"
	@echo ""
	@echo "ONE-LINERS:"
	@echo "  make up      - Start everything (production)"
	@echo "  make down    - Stop everything"
	@echo "  make dev     - Start development mode"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# === ONE-LINERS ===
up: ## Start CT-SaaS (production mode)
	@./start.sh prod

down: ## Stop CT-SaaS
	@./stop.sh all

start: up ## Alias for 'up'

stop: down ## Alias for 'down'

dev: ## Start development environment (databases + hot reload)
	@./start.sh dev

# Development
dev-db: ## Start development databases (PostgreSQL, Redis)
	docker-compose -f docker-compose.dev.yml up -d
	@echo "Waiting for databases to be ready..."
	@sleep 3
	@echo "PostgreSQL: localhost:5433"
	@echo "Redis: localhost:6380"

stop-db: ## Stop development databases
	docker-compose -f docker-compose.dev.yml down

dev-backend: ## Run backend in development mode
	cd $(BACKEND_DIR) && go run ./cmd/server

dev-frontend: ## Run frontend in development mode
	cd $(FRONTEND_DIR) && npm run dev

install: ## Install all dependencies
	cd $(BACKEND_DIR) && go mod download
	cd $(FRONTEND_DIR) && npm install

# Build
build: build-backend build-frontend ## Build both backend and frontend

build-backend: ## Build backend binary
	cd $(BACKEND_DIR) && CGO_ENABLED=0 go build -o ../bin/ct-saas ./cmd/server

build-frontend: ## Build frontend for production
	cd $(FRONTEND_DIR) && npm run build

# Docker
docker-build: ## Build Docker images
	docker-compose build

docker-up: ## Start all services with Docker
	docker-compose up -d

docker-down: ## Stop all Docker services
	docker-compose down

docker-logs: ## View Docker logs
	docker-compose logs -f

# Testing
test: test-backend ## Run all tests

test-backend: ## Run backend tests
	cd $(BACKEND_DIR) && go test -v ./...

# Linting
lint: lint-backend lint-frontend ## Run all linters

lint-backend: ## Lint backend code
	cd $(BACKEND_DIR) && go vet ./...

lint-frontend: ## Lint frontend code
	cd $(FRONTEND_DIR) && npm run lint

# Cleanup
clean: ## Clean build artifacts
	rm -rf bin/
	rm -rf $(FRONTEND_DIR)/dist/
	rm -rf $(FRONTEND_DIR)/node_modules/

# Database
migrate: ## Run database migrations
	cd $(BACKEND_DIR) && go run ./cmd/server -migrate

# Create admin user
create-admin: ## Create an admin user (EMAIL=admin@example.com PASSWORD=password make create-admin)
	@echo "Creating admin user..."
	@curl -X POST http://localhost:7842/api/v1/auth/register \
		-H "Content-Type: application/json" \
		-d '{"email":"$(EMAIL)","password":"$(PASSWORD)"}'
	@echo ""
	@echo "Note: You'll need to manually update the user role to 'admin' in the database"
