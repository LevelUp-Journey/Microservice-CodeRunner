# ============================================================================
# Makefile for CodeRunner Microservice - Docker Operations
# ============================================================================

.PHONY: help build up down restart logs clean clean-all test health status shell db-shell env

# Default target
.DEFAULT_GOAL := help

# Variables
PROJECT_NAME := coderunner
DOCKER_COMPOSE := docker-compose
DOCKER := docker
SERVICE_NAME := coderunner-service
DB_SERVICE := coderunner-postgres

# Colors for output
COLOR_RESET := \033[0m
COLOR_BOLD := \033[1m
COLOR_GREEN := \033[32m
COLOR_YELLOW := \033[33m
COLOR_BLUE := \033[34m
COLOR_RED := \033[31m

# ============================================================================
# Help Target
# ============================================================================

help: ## Show this help message
	@echo "$(COLOR_BOLD)CodeRunner Microservice - Docker Management$(COLOR_RESET)"
	@echo ""
	@echo "$(COLOR_GREEN)Available targets:$(COLOR_RESET)"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  $(COLOR_BLUE)%-20s$(COLOR_RESET) %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""

# ============================================================================
# Setup and Configuration
# ============================================================================

env: ## Create .env file from .env.example
	@if [ ! -f .env ]; then \
		echo "$(COLOR_YELLOW)Creating .env file from .env.example...$(COLOR_RESET)"; \
		cp .env.example .env; \
		echo "$(COLOR_GREEN)✓ .env file created. Please edit it with your configuration.$(COLOR_RESET)"; \
	else \
		echo "$(COLOR_RED)✗ .env file already exists. Remove it first if you want to recreate it.$(COLOR_RESET)"; \
	fi

check-env: ## Check if .env file exists
	@if [ ! -f .env ]; then \
		echo "$(COLOR_RED)✗ .env file not found!$(COLOR_RESET)"; \
		echo "$(COLOR_YELLOW)Run 'make env' to create it from .env.example$(COLOR_RESET)"; \
		exit 1; \
	fi

# ============================================================================
# Docker Build and Run
# ============================================================================

build: check-env ## Build all Docker images
	@echo "$(COLOR_GREEN)Building Docker images...$(COLOR_RESET)"
	$(DOCKER_COMPOSE) build --no-cache
	@echo "$(COLOR_GREEN)✓ Build complete!$(COLOR_RESET)"

build-quick: check-env ## Build Docker images (with cache)
	@echo "$(COLOR_GREEN)Building Docker images (using cache)...$(COLOR_RESET)"
	$(DOCKER_COMPOSE) build
	@echo "$(COLOR_GREEN)✓ Build complete!$(COLOR_RESET)"

up: check-env ## Start all services
	@echo "$(COLOR_GREEN)Starting all services...$(COLOR_RESET)"
	$(DOCKER_COMPOSE) up -d
	@echo "$(COLOR_GREEN)✓ Services started!$(COLOR_RESET)"
	@echo "$(COLOR_BLUE)Run 'make logs' to view logs$(COLOR_RESET)"
	@echo "$(COLOR_BLUE)Run 'make status' to check service health$(COLOR_RESET)"

up-build: check-env ## Build and start all services
	@echo "$(COLOR_GREEN)Building and starting all services...$(COLOR_RESET)"
	$(DOCKER_COMPOSE) up -d --build
	@echo "$(COLOR_GREEN)✓ Services built and started!$(COLOR_RESET)"

down: ## Stop all services
	@echo "$(COLOR_YELLOW)Stopping all services...$(COLOR_RESET)"
	$(DOCKER_COMPOSE) down
	@echo "$(COLOR_GREEN)✓ Services stopped!$(COLOR_RESET)"

restart: down up ## Restart all services

restart-service: ## Restart only the CodeRunner service
	@echo "$(COLOR_YELLOW)Restarting CodeRunner service...$(COLOR_RESET)"
	$(DOCKER_COMPOSE) restart coderunner
	@echo "$(COLOR_GREEN)✓ Service restarted!$(COLOR_RESET)"

# ============================================================================
# Database Operations
# ============================================================================

db-up: check-env ## Start only database services (PostgreSQL + pgAdmin)
	@echo "$(COLOR_GREEN)Starting database services...$(COLOR_RESET)"
	$(DOCKER_COMPOSE) up -d postgres pgadmin
	@echo "$(COLOR_GREEN)✓ Database services started!$(COLOR_RESET)"
	@echo "$(COLOR_BLUE)PostgreSQL: localhost:5432$(COLOR_RESET)"
	@echo "$(COLOR_BLUE)pgAdmin: http://localhost:5050$(COLOR_RESET)"

db-shell: ## Connect to PostgreSQL database
	@echo "$(COLOR_GREEN)Connecting to PostgreSQL...$(COLOR_RESET)"
	$(DOCKER) exec -it $(DB_SERVICE) psql -U postgres -d code_runner_db

db-migrate: ## Run database migrations
	@echo "$(COLOR_GREEN)Running database migrations...$(COLOR_RESET)"
	$(DOCKER) exec -it $(DB_SERVICE) psql -U postgres -d code_runner_db -f /docker-entrypoint-initdb.d/001_create_execution_tables.sql
	@echo "$(COLOR_GREEN)✓ Migrations complete!$(COLOR_RESET)"

db-backup: ## Backup database to file
	@echo "$(COLOR_GREEN)Creating database backup...$(COLOR_RESET)"
	@mkdir -p ./backups
	$(DOCKER) exec -t $(DB_SERVICE) pg_dump -U postgres code_runner_db > ./backups/backup_$$(date +%Y%m%d_%H%M%S).sql
	@echo "$(COLOR_GREEN)✓ Backup created in ./backups/$(COLOR_RESET)"

db-restore: ## Restore database from latest backup (use BACKUP_FILE=path to specify)
	@if [ -z "$(BACKUP_FILE)" ]; then \
		echo "$(COLOR_RED)✗ Please specify BACKUP_FILE=path/to/backup.sql$(COLOR_RESET)"; \
		exit 1; \
	fi
	@echo "$(COLOR_YELLOW)Restoring database from $(BACKUP_FILE)...$(COLOR_RESET)"
	$(DOCKER) exec -i $(DB_SERVICE) psql -U postgres -d code_runner_db < $(BACKUP_FILE)
	@echo "$(COLOR_GREEN)✓ Database restored!$(COLOR_RESET)"

# ============================================================================
# Logs and Monitoring
# ============================================================================

logs: ## Show logs for all services
	$(DOCKER_COMPOSE) logs -f

logs-service: ## Show logs for CodeRunner service only
	$(DOCKER_COMPOSE) logs -f coderunner

logs-db: ## Show logs for PostgreSQL service
	$(DOCKER_COMPOSE) logs -f postgres

logs-tail: ## Show last 100 lines of logs
	$(DOCKER_COMPOSE) logs --tail=100

status: ## Show status of all services
	@echo "$(COLOR_BOLD)Service Status:$(COLOR_RESET)"
	@$(DOCKER_COMPOSE) ps
	@echo ""
	@echo "$(COLOR_BOLD)Docker Images Inside DinD:$(COLOR_RESET)"
	@$(DOCKER) exec $(SERVICE_NAME) docker images 2>/dev/null || echo "$(COLOR_YELLOW)Service not running or Docker not initialized$(COLOR_RESET)"

health: ## Check health of all services
	@echo "$(COLOR_BOLD)Health Check:$(COLOR_RESET)"
	@echo ""
	@echo "$(COLOR_BLUE)HTTP Health Check:$(COLOR_RESET)"
	@curl -f http://localhost:8084/health 2>/dev/null && echo " ✓" || echo " $(COLOR_RED)✗ Failed$(COLOR_RESET)"
	@echo ""
	@echo "$(COLOR_BLUE)gRPC Health Check:$(COLOR_RESET)"
	@grpcurl -plaintext localhost:9084 com.levelupjourney.coderunner.CodeExecutionService/HealthCheck 2>/dev/null && echo " ✓" || echo " $(COLOR_RED)✗ Failed or grpcurl not installed$(COLOR_RESET)"
	@echo ""
	@echo "$(COLOR_BLUE)Database Connection:$(COLOR_RESET)"
	@$(DOCKER) exec $(DB_SERVICE) pg_isready -U postgres 2>/dev/null && echo " ✓" || echo " $(COLOR_RED)✗ Failed$(COLOR_RESET)"

stats: ## Show resource usage statistics
	@echo "$(COLOR_BOLD)Resource Usage:$(COLOR_RESET)"
	$(DOCKER) stats --no-stream

# ============================================================================
# Shell Access
# ============================================================================

shell: ## Open shell in CodeRunner container
	@echo "$(COLOR_GREEN)Opening shell in CodeRunner container...$(COLOR_RESET)"
	$(DOCKER) exec -it $(SERVICE_NAME) sh

shell-db: ## Open shell in PostgreSQL container
	@echo "$(COLOR_GREEN)Opening shell in PostgreSQL container...$(COLOR_RESET)"
	$(DOCKER) exec -it $(DB_SERVICE) sh

shell-dind: ## Access Docker daemon inside DinD container
	@echo "$(COLOR_GREEN)Accessing Docker inside DinD container...$(COLOR_RESET)"
	$(DOCKER) exec -it $(SERVICE_NAME) docker ps

# ============================================================================
# Testing and Development
# ============================================================================

test: ## Run a test gRPC call
	@echo "$(COLOR_GREEN)Running test gRPC call...$(COLOR_RESET)"
	@echo ""
	@echo "$(COLOR_BLUE)Testing ExecuteCode with factorial:$(COLOR_RESET)"
	@grpcurl -plaintext -d '{ \
		"solution_id": "test_sol_001", \
		"challenge_id": "factorial", \
		"student_id": "test_student", \
		"code": "function factorial(n) { return n <= 1 ? 1 : n * factorial(n-1); }", \
		"language": "javascript" \
	}' localhost:9084 com.levelupjourney.coderunner.CodeExecutionService/ExecuteCode

test-health: ## Test health endpoints
	@echo "$(COLOR_GREEN)Testing health endpoints...$(COLOR_RESET)"
	@echo ""
	@echo "$(COLOR_BLUE)HTTP Health:$(COLOR_RESET)"
	@curl -v http://localhost:8084/health 2>&1 | grep "< HTTP"
	@echo ""
	@echo "$(COLOR_BLUE)gRPC Health:$(COLOR_RESET)"
	@grpcurl -plaintext localhost:9084 com.levelupjourney.coderunner.CodeExecutionService/HealthCheck

dev: db-up ## Start only database for local development
	@echo "$(COLOR_GREEN)Development mode: Database services started$(COLOR_RESET)"
	@echo "$(COLOR_BLUE)Run the application locally with: go run ./cmd/server$(COLOR_RESET)"

# ============================================================================
# Cleanup Operations
# ============================================================================

clean: ## Clean up containers and networks
	@echo "$(COLOR_YELLOW)Cleaning up containers and networks...$(COLOR_RESET)"
	$(DOCKER_COMPOSE) down
	@echo "$(COLOR_GREEN)✓ Cleanup complete!$(COLOR_RESET)"

clean-volumes: ## Clean up volumes (WARNING: deletes all data)
	@echo "$(COLOR_RED)WARNING: This will delete all volumes and data!$(COLOR_RESET)"
	@read -p "Are you sure? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		$(DOCKER_COMPOSE) down -v; \
		echo "$(COLOR_GREEN)✓ Volumes removed!$(COLOR_RESET)"; \
	else \
		echo "$(COLOR_YELLOW)Cancelled.$(COLOR_RESET)"; \
	fi

clean-all: clean-volumes ## Full cleanup including images
	@echo "$(COLOR_YELLOW)Removing all images...$(COLOR_RESET)"
	$(DOCKER_COMPOSE) down --rmi all
	@echo "$(COLOR_GREEN)✓ Full cleanup complete!$(COLOR_RESET)"

clean-dind: ## Clean up Docker resources inside DinD container
	@echo "$(COLOR_YELLOW)Cleaning up Docker resources inside DinD...$(COLOR_RESET)"
	$(DOCKER) exec $(SERVICE_NAME) docker system prune -af
	@echo "$(COLOR_GREEN)✓ DinD cleanup complete!$(COLOR_RESET)"

prune: ## Remove unused Docker resources (system-wide)
	@echo "$(COLOR_YELLOW)Pruning unused Docker resources...$(COLOR_RESET)"
	$(DOCKER) system prune -af
	@echo "$(COLOR_GREEN)✓ Prune complete!$(COLOR_RESET)"

# ============================================================================
# Production Operations
# ============================================================================

pull: ## Pull latest images
	@echo "$(COLOR_GREEN)Pulling latest images...$(COLOR_RESET)"
	$(DOCKER_COMPOSE) pull
	@echo "$(COLOR_GREEN)✓ Images updated!$(COLOR_RESET)"

update: pull up-build ## Update and restart services

deploy: check-env build up ## Full deployment (build and start)
	@echo "$(COLOR_GREEN)✓ Deployment complete!$(COLOR_RESET)"
	@make health

# ============================================================================
# Azure Operations
# ============================================================================

azure-login: ## Login to Azure Container Registry
	@echo "$(COLOR_GREEN)Logging into Azure Container Registry...$(COLOR_RESET)"
	@read -p "Registry name: " registry; \
	az acr login --name $$registry

azure-push: ## Push image to Azure Container Registry
	@echo "$(COLOR_GREEN)Pushing to Azure Container Registry...$(COLOR_RESET)"
	@read -p "Registry URL (e.g., myregistry.azurecr.io): " registry; \
	$(DOCKER) tag $(PROJECT_NAME)-service:latest $$registry/$(PROJECT_NAME)-service:latest; \
	$(DOCKER) push $$registry/$(PROJECT_NAME)-service:latest; \
	echo "$(COLOR_GREEN)✓ Image pushed to $$registry$(COLOR_RESET)"

# ============================================================================
# Information
# ============================================================================

info: ## Show project information
	@echo "$(COLOR_BOLD)CodeRunner Microservice Information$(COLOR_RESET)"
	@echo ""
	@echo "$(COLOR_BLUE)Project:$(COLOR_RESET) $(PROJECT_NAME)"
	@echo "$(COLOR_BLUE)Services:$(COLOR_RESET)"
	@echo "  - CodeRunner gRPC (Port 9084)"
	@echo "  - Health Check HTTP (Port 8084)"
	@echo "  - PostgreSQL (Port 5432)"
	@echo "  - pgAdmin (Port 5050)"
	@echo ""
	@echo "$(COLOR_BLUE)URLs:$(COLOR_RESET)"
	@echo "  - gRPC: localhost:9084"
	@echo "  - Health: http://localhost:8084/health"
	@echo "  - pgAdmin: http://localhost:5050"
	@echo ""
	@echo "$(COLOR_BLUE)Documentation:$(COLOR_RESET)"
	@echo "  - Setup: DOCKER_SETUP.md"
	@echo "  - README: README.md"

version: ## Show Docker and Docker Compose versions
	@echo "$(COLOR_BOLD)Versions:$(COLOR_RESET)"
	@$(DOCKER) --version
	@$(DOCKER_COMPOSE) --version
	@echo ""
	@echo "$(COLOR_BLUE)Go version inside container:$(COLOR_RESET)"
	@$(DOCKER) run --rm golang:1.25.1-alpine go version

# ============================================================================
# Quick Commands
# ============================================================================

quick-start: env up ## Quick start (create env and start services)
	@echo "$(COLOR_GREEN)✓ Quick start complete!$(COLOR_RESET)"
	@echo "$(COLOR_YELLOW)Don't forget to edit .env with your Azure credentials!$(COLOR_RESET)"

quick-stop: down ## Quick stop all services

quick-restart: restart ## Quick restart all services

quick-logs: logs-service ## Quick view service logs

# ============================================================================
# Advanced Operations
# ============================================================================

inspect: ## Inspect CodeRunner container configuration
	@echo "$(COLOR_BOLD)Container Inspection:$(COLOR_RESET)"
	$(DOCKER) inspect $(SERVICE_NAME) | jq '.[0] | {State, Config: .Config.Env, Mounts: .Mounts}'

network-inspect: ## Inspect Docker network
	@echo "$(COLOR_BOLD)Network Inspection:$(COLOR_RESET)"
	$(DOCKER) network inspect $(PROJECT_NAME)_coderunner-network

volume-inspect: ## Inspect Docker volumes
	@echo "$(COLOR_BOLD)Volume Inspection:$(COLOR_RESET)"
	$(DOCKER) volume ls | grep $(PROJECT_NAME)

troubleshoot: ## Run troubleshooting diagnostics
	@echo "$(COLOR_BOLD)Running Diagnostics...$(COLOR_RESET)"
	@echo ""
	@echo "$(COLOR_BLUE)1. Container Status:$(COLOR_RESET)"
	@$(DOCKER_COMPOSE) ps
	@echo ""
	@echo "$(COLOR_BLUE)2. Health Checks:$(COLOR_RESET)"
	@make health
	@echo ""
	@echo "$(COLOR_BLUE)3. Recent Logs (last 20 lines):$(COLOR_RESET)"
	@$(DOCKER_COMPOSE) logs --tail=20
	@echo ""
	@echo "$(COLOR_BLUE)4. Resource Usage:$(COLOR_RESET)"
	@$(DOCKER) stats --no-stream
	@echo ""
	@echo "$(COLOR_BLUE)5. Docker Info Inside DinD:$(COLOR_RESET)"
	@$(DOCKER) exec $(SERVICE_NAME) docker info 2>/dev/null || echo "$(COLOR_RED)DinD not available$(COLOR_RESET)"

validate: check-env ## Validate environment configuration
	@echo "$(COLOR_GREEN)Running configuration validation...$(COLOR_RESET)"
	@chmod +x scripts/validate-config.sh
	@./scripts/validate-config.sh
