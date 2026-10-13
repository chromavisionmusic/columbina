BIN_DIR := build
BIN_NAME := bot
BOT_SRC := cmd/main.go

.PHONY: build run dev test test-race lint fmt fmt-check clean deploy

build:
	@mkdir -p $(BIN_DIR)
	@go build -o $(BIN_DIR)/$(BIN_NAME) $(BOT_SRC)

run: build
	@./$(BIN_DIR)/$(BIN_NAME) production

dev: build
	@./$(BIN_DIR)/$(BIN_NAME) development

test:
	@go test -v ./...

test-race:
	@go test -v -race -coverprofile=coverage.out -covermode=atomic ./...

fmt:
	@gofmt -w -s .

fmt-check:
	@test -z "$$(gofmt -l .)" || (echo "Unformatted files found:" && gofmt -l . && exit 1)

lint:
	@golangci-lint run ./...

clean:
	@rm -rf $(BIN_DIR)

deploy:
	@go run cmd/deploy/main.go
