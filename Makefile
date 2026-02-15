BIN_DIR := bin
BIN_NAME := tidskott-pi
BIN_PATH := $(BIN_DIR)/$(BIN_NAME)

.PHONY: build
build:
	mkdir -p $(BIN_DIR)
	go build -o $(BIN_PATH) ./cmd/tidskott-pi

.PHONY: build-pi
build-pi:
	mkdir -p $(BIN_DIR)
	GOOS=linux GOARCH=arm GOARM=7 go build -o $(BIN_PATH) ./cmd/tidskott-pi

.PHONY: clean
clean:
	rm -rf $(BIN_DIR)
