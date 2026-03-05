.PHONY: dev dev-backend dev-frontend build clean test

# Development: run backend and frontend in parallel
dev:
	@echo "Starting development servers..."
	@echo "Backend: http://localhost:8080"
	@echo "Frontend: http://localhost:5173 (with API proxy)"
	@$(MAKE) -j2 dev-backend dev-frontend

dev-backend:
	go run ./cmd/rpmmanager serve

dev-frontend:
	cd web && npm run dev

# Production build: frontend → embed → single binary
build:
	cd web && npm ci && npm run build
	rm -rf internal/embed/dist
	cp -r web/dist internal/embed/dist
	CGO_ENABLED=0 go build -ldflags="-s -w" -o rpmmanager ./cmd/rpmmanager

# Quick backend build (skip frontend)
build-backend:
	CGO_ENABLED=0 go build -ldflags="-s -w" -o rpmmanager ./cmd/rpmmanager

# Run tests
test:
	go test ./...

# Clean build artifacts
clean:
	rm -f rpmmanager
	rm -rf internal/embed/dist
	rm -rf web/dist web/node_modules
