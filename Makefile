.PHONY: dev build build-frontend build-backend build-app build-all clean

APP_NAME := octai-app
GO_BACKEND := go-backend/cmd/octai-app
FRONTEND_DIR := frontend
TAURI_DIR := src-tauri

VERSION := 0.1.0

dev:
	cd $(TAURI_DIR) && cargo tauri dev

build-frontend:
	cd $(FRONTEND_DIR) && pnpm install && pnpm build

build-backend:
	cd go-backend && CGO_ENABLED=0 go build -o ../$(TAURI_DIR)/binaries/octai-backend-$(shell uname -s | tr '[:upper:]' '[:lower:]')-x86_64 ./cmd/octai-app/

build: build-backend build-frontend

build-app: build
	cd $(TAURI_DIR) && cargo tauri build

build-all:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o $(TAURI_DIR)/binaries/octai-backend-linux-x86_64 ./$(GO_BACKEND)
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o $(TAURI_DIR)/binaries/octai-backend-linux-aarch64 ./$(GO_BACKEND)
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -o $(TAURI_DIR)/binaries/octai-backend-darwin-x86_64 ./$(GO_BACKEND)
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -o $(TAURI_DIR)/binaries/octai-backend-darwin-aarch64 ./$(GO_BACKEND)
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o $(TAURI_DIR)/binaries/octai-backend-windows-x86_64.exe ./$(GO_BACKEND)

clean:
	rm -rf $(FRONTEND_DIR)/dist $(FRONTEND_DIR)/node_modules
	rm -rf $(TAURI_DIR)/target $(TAURI_DIR)/binaries/*
	rm -rf build/ dist/
	cd go-backend && go clean
