APP_NAME := alipay-ai-service
BUILD_DIR := dist
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
VERSION ?= dev
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

.PHONY: build run test clean docker-build docker-up docker-down

build:
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build \
		-trimpath \
		-ldflags "-s -w -extldflags '-static' -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)" \
		-o $(BUILD_DIR)/$(APP_NAME) ./main.go
	@echo "built $(BUILD_DIR)/$(APP_NAME) for $(GOOS)/$(GOARCH)"

run: build
	./$(BUILD_DIR)/$(APP_NAME)

test:
	go test ./...

clean:
	rm -rf $(BUILD_DIR)

docker-build:
	docker build -t $(APP_NAME):local .

docker-up:
	docker compose up --build -d

docker-down:
	docker compose down
