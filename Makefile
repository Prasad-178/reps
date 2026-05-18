.DEFAULT_GOAL := help
SHELL := /bin/bash

REPS_BIN := ./bin/reps
BACKEND_ADDR ?= :7777
WEB_DIR := web

.PHONY: help build install dev backend web web-install clean reset

help: ## Show this help.
	@awk 'BEGIN {FS = ":.*##"; printf "Usage:\n  make \033[36m<target>\033[0m\n\nTargets:\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

build: ## Build the reps Go binary into ./bin/reps
	@mkdir -p bin
	go build -o $(REPS_BIN) ./cmd/reps
	@echo "built $(REPS_BIN)"

install: ## go install reps into $GOBIN
	go install ./cmd/reps

backend: build ## Run the HTTP API only
	$(REPS_BIN) serve --addr $(BACKEND_ADDR)

web-install: ## Install frontend dependencies
	cd $(WEB_DIR) && bun install

web: ## Run the Next.js dev server only
	cd $(WEB_DIR) && bun dev

dev: build ## Run backend + frontend together; auto-loads ./.env
	@echo "==> backend  $(BACKEND_ADDR)"
	@echo "==> frontend http://localhost:3000"
	@echo "press Ctrl-C to stop both"
	@trap 'kill 0' INT TERM EXIT; \
	$(REPS_BIN) serve --addr $(BACKEND_ADDR) & \
	(cd $(WEB_DIR) && bun dev) & \
	wait

clean: ## Remove build artefacts
	rm -rf bin/ web/.next/ web/.turbo/

reset: build ## Wipe local reps data (DB + sources). DESTRUCTIVE.
	$(REPS_BIN) reset --yes --all
