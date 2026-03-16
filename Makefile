.PHONY: build build-server build-web dev dev-server dev-web clean migrate

# Variables
BINARY_NAME=ideasaver
CLI_BINARY=ideactl
GO_FLAGS=-ldflags="-s -w"

# Build all
build: build-server build-web

# Build Go server
build-server:
	cd server && go build $(GO_FLAGS) -o ../bin/$(BINARY_NAME) cmd/main.go

# Build CLI tool
build-cli:
	cd cli && go build $(GO_FLAGS) -o ../bin/$(CLI_BINARY) main.go

# Build React frontend
build-web:
	cd web && npm run build

# Development
dev-server:
	cd server && go run cmd/main.go

dev-web:
	cd web && npm run dev

# Database migration
migrate:
	cd server && go run cmd/main.go migrate

# Clean build artifacts
clean:
	rm -rf bin/
	rm -rf web/dist/

# Install dependencies
deps:
	cd server && go mod download
	cd web && npm install

# Run tests
test:
	cd server && go test ./...
	cd web && npm test

# Lint
lint:
	cd server && golangci-lint run
	cd web && npm run lint
