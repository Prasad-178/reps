.DEFAULT_GOAL := help
SHELL := /bin/zsh

REPS_BIN := ./bin/reps
BACKEND_ADDR ?= :7777
WEB_DIR := web

# ---- memory caps (per process) ------------------------------------------
# Node heap ceiling for `next dev`. Default V8 is ~50% of system RAM, so on
# 16 GB Macs Node will happily balloon to 8 GB if it can. 1 GB is plenty for
# a 12-route Next.js dev session with HMR.
NEXT_NODE_OPTIONS ?= --max-old-space-size=1024
# Go runtime soft memory ceiling. Triggers earlier GC and prevents accidental
# spikes on big batch embed / crawl jobs.
GOMEMLIMIT ?= 256MiB

# Common env applied to dev:
DEV_ENV := \
  NODE_OPTIONS='$(NEXT_NODE_OPTIONS)' \
  NEXT_TELEMETRY_DISABLED=1 \
  GOMEMLIMIT=$(GOMEMLIMIT)

.PHONY: help build install dev dev-slim dev-prod backend web web-install clean reset fresh setup

help: ## Show this help.
	@awk 'BEGIN {FS = ":.*##"; printf "Usage:\n  make \033[36m<target>\033[0m\n\nTargets:\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

build: ## Build the reps Go binary into ./bin/reps
	@mkdir -p bin
	go build -ldflags "-s -w" -trimpath -o $(REPS_BIN) ./cmd/reps
	@echo "built $(REPS_BIN)"

install: ## go install reps into $GOBIN
	go install ./cmd/reps

backend: build ## Run the HTTP API only (low RAM ~50 MB)
	$(DEV_ENV) $(REPS_BIN) serve --addr $(BACKEND_ADDR)

web-install: ## Install frontend dependencies
	cd $(WEB_DIR) && bun install

web: ## Run the Next.js dev server only (capped heap)
	cd $(WEB_DIR) && $(DEV_ENV) bun dev

dev: build ## Run backend + frontend dev together (Turbopack, ~1-1.5 GB)
	@$(MAKE) --no-print-directory _free-ports
	@echo "==> backend  $(BACKEND_ADDR)   (GOMEMLIMIT=$(GOMEMLIMIT))"
	@echo "==> frontend http://localhost:3000  (Node heap cap = $(NEXT_NODE_OPTIONS))"
	@echo "press Ctrl-C to stop both"
	@trap 'echo; echo "==> stopping..."; pkill -P $$$$ 2>/dev/null; kill 0 2>/dev/null; exit 0' INT TERM; \
	$(DEV_ENV) $(REPS_BIN) serve --addr $(BACKEND_ADDR) & \
	BACKEND_PID=$$!; \
	(cd $(WEB_DIR) && $(DEV_ENV) bun dev) & \
	FRONTEND_PID=$$!; \
	wait $$BACKEND_PID $$FRONTEND_PID

dev-slim: build ## Backend only — skip the Next dev server, save ~1 GB
	@$(MAKE) --no-print-directory _free-ports
	@echo "==> backend  $(BACKEND_ADDR)   (no frontend)"
	@echo "    open http://localhost:7777/healthz to verify; build the web prod bundle with make dev-prod."
	$(DEV_ENV) $(REPS_BIN) serve --addr $(BACKEND_ADDR)

dev-prod: build ## Build frontend once, serve it (no Turbopack; ~300 MB total)
	@$(MAKE) --no-print-directory _free-ports
	@echo "==> building frontend (one-time)"
	cd $(WEB_DIR) && $(DEV_ENV) bun run build
	@echo "==> backend  $(BACKEND_ADDR)"
	@echo "==> frontend http://localhost:3000  (production bundle)"
	@trap 'echo; echo "==> stopping..."; pkill -P $$$$ 2>/dev/null; kill 0 2>/dev/null; exit 0' INT TERM; \
	$(DEV_ENV) $(REPS_BIN) serve --addr $(BACKEND_ADDR) & \
	BACKEND_PID=$$!; \
	(cd $(WEB_DIR) && $(DEV_ENV) bun run start) & \
	FRONTEND_PID=$$!; \
	wait $$BACKEND_PID $$FRONTEND_PID

# Internal: kill stale processes on our ports so a previous crash doesn't
# block today's run.
.PHONY: _free-ports
_free-ports:
	@for port in 7777 3000; do \
		pids=$$(lsof -ti:$$port 2>/dev/null); \
		if [ -n "$$pids" ]; then echo "   freeing :$$port (was held by $$pids)"; kill -9 $$pids 2>/dev/null || true; fi; \
	done

clean: ## Remove build artefacts
	rm -rf bin/ web/.next/ web/.turbo/ web/node_modules/.cache

reset: build ## Wipe local reps data (DB + sources). DESTRUCTIVE.
	$(REPS_BIN) reset --yes --all

setup: build ## Run the setup wizard
	$(REPS_BIN) init

fresh: build ## Wipe ~/.reps/* AND re-run the wizard from scratch
	$(REPS_BIN) init --reset
