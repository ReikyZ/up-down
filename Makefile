COMMAND ?= up
BIN_DIR ?= $(HOME)/.local/bin

.PHONY: install test

install:
	mkdir -p "$(BIN_DIR)"
	printf '%s\n' '#!/bin/sh' 'exec python3 "$(CURDIR)/cmd/up.py" "$$@"' > "$(BIN_DIR)/$(COMMAND)"
	chmod 755 "$(BIN_DIR)/$(COMMAND)"

test:
	python3 -m unittest discover -s tests -v
