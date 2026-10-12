BIN_DIR := build
BIN_NAME := bot
BOT_SRC := cmd/main.go

.PHONY: build run dev test clean

build:
	@mkdir -p $(BIN_DIR)
	@go build -o $(BIN_DIR)/$(BIN_NAME) $(BOT_SRC)

run: build
	@./$(BIN_DIR)/$(BIN_NAME) production

dev: build
	@./$(BIN_DIR)/$(BIN_NAME) development

test:
	@go test -v ./...

clean:
	@rm -rf $(BIN_DIR)
