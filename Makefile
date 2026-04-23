.PHONY: build build-linux build-darwin build-all clean docker-build docker-up docker-down

BINARY_SERVER=go-tunnel-server
BINARY_CLIENT=go-tunnel-client
BUILD_DIR=./bin

build:
	go build -o $(BUILD_DIR)/$(BINARY_SERVER) ./cmd/server
	go build -o $(BUILD_DIR)/$(BINARY_CLIENT) ./cmd/client

build-linux: build-linux-amd64

build-linux-amd64:
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_SERVER)-linux-amd64 ./cmd/server
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_CLIENT)-linux-amd64 ./cmd/client

build-darwin: build-darwin-amd64 build-darwin-arm64

build-darwin-amd64:
	GOOS=darwin GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_SERVER)-darwin-amd64 ./cmd/server
	GOOS=darwin GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_CLIENT)-darwin-amd64 ./cmd/client

build-darwin-arm64:
	GOOS=darwin GOARCH=arm64 go build -o $(BUILD_DIR)/$(BINARY_SERVER)-darwin-armd64 ./cmd/server
	GOOS=darwin GOARCH=arm64 go build -o $(BUILD_DIR)/$(BINARY_CLIENT)-darwin-arm64 ./cmd/client

build-all: build-linux-amd64 build-darwin-amd64 build-darwin-arm64

# Docker
docker-build:
	docker compose build

docker-up:
	docker compose up -d

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f

clean:
	rm -rf $(BUILD_DIR)
