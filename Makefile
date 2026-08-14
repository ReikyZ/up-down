COMMAND ?= up
BIN_DIR := $(shell go env GOBIN)

ifeq ($(strip $(BIN_DIR)),)
BIN_DIR := $(shell go env GOPATH)/bin
endif

.PHONY: install

install:
	mkdir -p "$(BIN_DIR)"
	go build -o "$(BIN_DIR)/$(COMMAND)" ./cmd/up
