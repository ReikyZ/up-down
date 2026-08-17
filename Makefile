COMMAND ?= up
BIN_DIR ?= $(HOME)/.local/bin
DIST_DIR ?= dist
SERVER_URL ?= $(shell sed -n 's/^[[:space:]]*SERVER_URL[[:space:]]*=[[:space:]]*//p' .env | head -n 1)

.PHONY: build-client install test

build-client:
	@test -n "$(SERVER_URL)" || (echo "SERVER_URL is required; set it in .env or pass SERVER_URL=..." >&2; exit 1)
	mkdir -p "$(DIST_DIR)"
	stage_dir=$$(mktemp -d); \
	trap 'rm -rf "$$stage_dir"' EXIT; \
	mkdir "$$stage_dir/updown"; \
	cp updown/*.py "$$stage_dir/updown/"; \
	printf '%s\n' "DEFAULT_SERVER_URL = '$(SERVER_URL)'" > "$$stage_dir/updown/build_config.py"; \
	PYTHONDONTWRITEBYTECODE=1 python3 -m zipapp "$$stage_dir" --main updown.client:main --python '/usr/bin/env python3' --compress --output "$(DIST_DIR)/$(COMMAND)"; \
	chmod 755 "$(DIST_DIR)/$(COMMAND)"

install: build-client
	mkdir -p "$(BIN_DIR)"
	cp "$(DIST_DIR)/$(COMMAND)" "$(BIN_DIR)/$(COMMAND)"
	chmod 755 "$(BIN_DIR)/$(COMMAND)"

test:
	python3 -m unittest discover -s tests -v
