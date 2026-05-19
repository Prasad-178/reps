.DEFAULT_GOAL := help
SHELL := /bin/bash

REPS_BIN := ./bin/reps
BACKEND_ADDR ?= :7777
WEB_DIR := web

.PHONY: help build install dev backend web web-install clean reset fresh setup

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
	@echo "==> freeing :7777 and :3000 if held by stale processes"
	@for port in 7777 3000; do \
		pids=$$(lsof -ti:$$port 2>/dev/null); \
		if [ -n "$$pids" ]; then echo "   killing $$pids on :$$port"; kill -9 $$pids 2>/dev/null || true; fi; \
	done
	@echo "==> backend  $(BACKEND_ADDR)"
	@echo "==> frontend http://localhost:3000"
	@echo "press Ctrl-C to stop both"
	@trap 'echo; echo "==> stopping..."; pkill -P $$$$ 2>/dev/null; kill 0 2>/dev/null; exit 0' INT TERM; \
	$(REPS_BIN) serve --addr $(BACKEND_ADDR) & \
	BACKEND_PID=$$!; \
	(cd $(WEB_DIR) && bun dev) & \
	FRONTEND_PID=$$!; \
	wait $$BACKEND_PID $$FRONTEND_PID

clean: ## Remove build artefacts
	rm -rf bin/ web/.next/ web/.turbo/

reset: build ## Wipe local reps data (DB + sources). DESTRUCTIVE.
	$(REPS_BIN) reset --yes --all

setup: build ## Run the setup wizard
	$(REPS_BIN) init

fresh: build ## Wipe ~/.reps/* AND re-run the wizard from scratch
	$(REPS_BIN) init --reset
