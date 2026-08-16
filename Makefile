COMMAND ?= up
BIN_DIR ?= $(HOME)/.local/bin
DIST_DIR ?= dist

.PHONY: build-client install test

build-client:
	mkdir -p "$(DIST_DIR)"
	stage_dir=$$(mktemp -d); \
	trap 'rm -rf "$$stage_dir"' EXIT; \
	mkdir "$$stage_dir/updown"; \
	cp updown/*.py "$$stage_dir/updown/"; \
	PYTHONDONTWRITEBYTECODE=1 python3 -m zipapp "$$stage_dir" --main updown.client:main --python '/usr/bin/env python3' --compress --output "$(DIST_DIR)/$(COMMAND)"; \
	chmod 755 "$(DIST_DIR)/$(COMMAND)"

install:
	mkdir -p "$(BIN_DIR)"
	printf '%s\n' '#!/bin/sh' 'exec python3 "$(CURDIR)/cmd/up.py" "$$@"' > "$(BIN_DIR)/$(COMMAND)"
	chmod 755 "$(BIN_DIR)/$(COMMAND)"

test:
	python3 -m unittest discover -s tests -v
