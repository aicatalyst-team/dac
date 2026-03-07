NAME=dac$(shell if [ "$(shell go env GOOS)" = "windows" ]; then echo .exe; fi)
BUILD_DIR ?= bin
BUILD_SRC=.

NO_COLOR=\033[0m
OK_COLOR=\033[32;01m
ERROR_COLOR=\033[31;01m
WARN_COLOR=\033[33;01m

# Suppress CGO linker warnings on macOS (not needed on Linux/Windows)
ifeq ($(shell go env GOOS),darwin)
export CGO_LDFLAGS=-Wl,-w
export LDFLAGS=-Wl,-w
endif

.PHONY: all clean test build frontend deps format dev-frontend dev-backend

all: clean deps test build

deps:
	@printf "$(OK_COLOR)==> Installing dependencies$(NO_COLOR)\n"
	@go mod tidy
	@cd frontend && npm install

# Build the frontend and copy to web/ for Go embedding
frontend:
	@echo "$(OK_COLOR)==> Building the frontend...$(NO_COLOR)"
	@cd frontend && npm run build
	@rm -rf web
	@cp -r frontend/dist web

# Build the Go binary (requires frontend to be built first)
build: frontend
	@echo "$(OK_COLOR)==> Building the application...$(NO_COLOR)"
	@go build -v -ldflags="-s -w" -o "$(BUILD_DIR)/$(NAME)" "$(BUILD_SRC)"

clean:
	@rm -rf ./bin ./web ./frontend/dist

test: test-unit

test-unit:
	@echo "$(OK_COLOR)==> Running the unit tests$(NO_COLOR)"
	@go test -race -cover -timeout 5m ./cmd/... ./pkg/...

format:
	@echo "$(OK_COLOR)>> [go vet] running$(NO_COLOR)"
	@go vet ./cmd/... ./pkg/...

# Run frontend dev server (with API proxy to Go backend on :8321)
dev-frontend:
	@cd frontend && npm run dev

# Run Go backend in dev mode
dev-backend:
	@go build -o "$(BUILD_DIR)/$(NAME)" "$(BUILD_SRC)"
	@./$(BUILD_DIR)/$(NAME) serve --dir ./testdata/dashboards --port 8321
