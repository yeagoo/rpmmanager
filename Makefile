VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT)

.PHONY: dev dev-backend dev-frontend build build-backend web-build \
        test lint fmt check release docker docker-up docker-down clean

# ── Development ──────────────────────────────────────────────────
dev:
	@echo "Starting development servers..."
	@echo "Backend: http://localhost:8080"
	@echo "Frontend: http://localhost:5173 (with API proxy)"
	@$(MAKE) -j2 dev-backend dev-frontend

dev-backend:
	go run ./cmd/rpmmanager serve

dev-frontend:
	cd web && npm run dev

# ── Build ────────────────────────────────────────────────────────

# Full production build: frontend → embed → single binary
build: web-build
	rm -rf internal/embed/dist
	cp -r web/dist internal/embed/dist
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o bin/rpmmanager ./cmd/rpmmanager
	@echo "Built bin/rpmmanager ($(VERSION))"

# Quick backend build (skip frontend rebuild)
build-backend:
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o bin/rpmmanager ./cmd/rpmmanager

# Frontend only
web-build:
	cd web && npm ci && npm run build

# ── Test & Lint ──────────────────────────────────────────────────

# Run all checks (same as CI)
check: fmt test lint web-typecheck
	@echo "All checks passed."

# Go tests
test:
	go vet ./...
	go test -race -count=1 ./...

# Frontend lint
lint:
	cd web && npm run lint

# Frontend type check
web-typecheck:
	cd web && npx tsc --noEmit

# Go format check (fails if unformatted)
fmt:
	@test -z "$$(gofmt -l .)" || (echo "Run 'gofmt -w .' to fix formatting:" && gofmt -l . && exit 1)

# ── Release (cross-compile) ─────────────────────────────────────

# Build all platforms into dist/
release: web-build
	rm -rf dist
	rm -rf internal/embed/dist
	cp -r web/dist internal/embed/dist
	@echo "Building $(VERSION) for linux/amd64..."
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o dist/rpmmanager-linux-amd64 ./cmd/rpmmanager
	@echo "Building $(VERSION) for linux/arm64..."
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o dist/rpmmanager-linux-arm64 ./cmd/rpmmanager
	cd dist && sha256sum rpmmanager-* > checksums.txt
	@echo "Release artifacts:"
	@ls -lh dist/
	@echo ""
	@cat dist/checksums.txt

# ── Docker ───────────────────────────────────────────────────────
docker:
	docker build -f deploy/docker/Dockerfile \
		--build-arg VERSION=$(VERSION) --build-arg COMMIT=$(COMMIT) \
		-t rpmmanager:$(VERSION) .
	@echo "Built docker image rpmmanager:$(VERSION)"

docker-up:
	cd deploy/docker && docker compose up -d

docker-down:
	cd deploy/docker && docker compose down

docker-logs:
	cd deploy/docker && docker compose logs -f

# ── Clean ────────────────────────────────────────────────────────
clean:
	rm -rf bin/ dist/
	rm -rf web/dist web/node_modules
