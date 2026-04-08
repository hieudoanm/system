.PHONY: lint format build install all

BIN_DIR := bin
BIN_NAME := system
SRC := ./main.go

VERSION := $(shell date +%Y.%m.%d.%H.%M.%S)
LDFLAGS := -ldflags="-X 'github.com/hieudoanm/system/src/cmd.Version=$(VERSION)'"


## Commands
lint:
	go vet ./...

format:
	go fmt ./...

## Build binary into ./bin
build: format lint
	@mkdir -p $(BIN_DIR)
	go build $(LDFLAGS) -o $(BIN_DIR)/$(BIN_NAME) $(SRC)
	@echo "✅ Binary: $$(pwd)/$(BIN_DIR)/$(BIN_NAME)"
	@echo "📦 Version: $(VERSION)"
	@echo "📏 Size: $$(du -h $(BIN_DIR)/$(BIN_NAME) | cut -f1)"

## Install binary into ~/bin
install: build
	@mkdir -p $(HOME)/bin
	cp $(BIN_DIR)/$(BIN_NAME) $(HOME)/bin/
	chmod +x $(HOME)/bin/$(BIN_NAME)
	@echo "Installed to $$HOME/bin"
	@echo "Make sure $$HOME/bin is in your PATH:"
	@echo '  export PATH="$$HOME/bin:$$PATH"'
