.PHONY: all up down restart logs ps test build docs docs-build clean help

# Default target
all: help

## 🚀 Docker Compose Orchestration
up: ## Start all services with Docker Compose (Local Meilisearch + Harvester + API + Studio)
	docker compose up -d --build

down: ## Stop all running services
	docker compose down

restart: ## Restart all services
	docker compose restart

logs: ## Follow live logs from all services
	docker compose logs -f

ps: ## Check status of running containers
	docker compose ps

## ☁️ Meilisearch Cloud Deployment
up-cloud: ## Start services configured for external Meilisearch Cloud
	docker compose -f docker-compose.yml -f docker-compose.cloud.yml up -d --build

## 🧪 Testing & Verification
test: ## Run test suites across all sub-projects
	@echo "==> Testing geomilli-harvester..."
	@if [ -d "geomilli-harvester" ] && [ -f "geomilli-harvester/go.mod" ]; then (cd geomilli-harvester && go test ./...); fi
	@echo "==> Testing geomilli-api..."
	@if [ -d "geomilli-api" ] && [ -f "geomilli-api/go.mod" ]; then (cd geomilli-api && go test ./...); fi
	@echo "==> Verifying docs build..."
	@(cd docs && npm run build)

## 📦 Compilation & Builds
build: ## Build all Go binaries and frontend bundles
	@echo "==> Building geomilli-harvester..."
	@if [ -d "geomilli-harvester" ] && [ -f "geomilli-harvester/go.mod" ]; then (cd geomilli-harvester && go build ./...); fi
	@echo "==> Building geomilli-api..."
	@if [ -d "geomilli-api" ] && [ -f "geomilli-api/go.mod" ]; then (cd geomilli-api && go build ./...); fi
	@echo "==> Building docs..."
	@(cd docs && npm run build)
	@mkdir -p geomilli-studio/public/docs
	@cp -R docs/.vitepress/dist/ geomilli-studio/public/docs/

## 📚 Documentation Site
docs: ## Start local VitePress documentation server
	(cd docs && npm run dev)

docs-build: ## Build static VitePress documentation and sync to studio
	(cd docs && npm run build)
	mkdir -p geomilli-studio/public/docs
	cp -R docs/.vitepress/dist/ geomilli-studio/public/docs/

clean: ## Remove temporary build files and caches
	rm -rf docs/.vitepress/dist docs/.vitepress/cache
	rm -rf data/staging data/raw

help: ## Show this help menu
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-16s\033[0m %s\n", $$1, $$2}'
