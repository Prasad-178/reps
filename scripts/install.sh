#!/usr/bin/env bash
# reps — one-line installer.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/Prasad-178/reps/main/scripts/install.sh | bash
#
# Detects OS+arch, downloads the matching release tarball from GitHub, drops
# `reps` into ~/.local/bin (creating + warning-about-PATH if needed). Falls
# back to `go install github.com/Prasad-178/reps/cmd/reps@latest` if no
# matching release is available (helpful for unreleased commits).

set -euo pipefail

REPO="Prasad-178/reps"
BIN_NAME="reps"
INSTALL_DIR="${REPS_INSTALL_DIR:-$HOME/.local/bin}"
VERSION="${REPS_VERSION:-latest}"

BOLD=$'\033[1m'
DIM=$'\033[2m'
GREEN=$'\033[32m'
YELLOW=$'\033[33m'
RED=$'\033[31m'
RESET=$'\033[0m'

say() { printf "${BOLD}::${RESET} %s\n" "$*"; }
ok()  { printf "${GREEN}✓${RESET}  %s\n" "$*"; }
warn(){ printf "${YELLOW}!${RESET}  %s\n" "$*"; }
err() { printf "${RED}✗${RESET}  %s\n" "$*" >&2; }

detect_target() {
  local os arch
  case "$(uname -s)" in
    Darwin) os="darwin" ;;
    Linux)  os="linux"  ;;
    *) err "Unsupported OS: $(uname -s) (try `go install github.com/${REPO}/cmd/reps@latest`)"; return 1 ;;
  esac
  case "$(uname -m)" in
    x86_64|amd64) arch="amd64" ;;
    arm64|aarch64) arch="arm64" ;;
    *) err "Unsupported arch: $(uname -m)"; return 1 ;;
  esac
  echo "${os}_${arch}"
}

ensure_dir() {
  mkdir -p "$INSTALL_DIR"
}

ensure_path_hint() {
  case ":$PATH:" in
    *":$INSTALL_DIR:"*) return 0 ;;
  esac
  warn "$INSTALL_DIR is not on \$PATH. Add this to your shell profile:"
  printf "    ${DIM}export PATH=\"\$PATH:%s\"${RESET}\n" "$INSTALL_DIR"
}

resolve_release_url() {
  local target="$1"
  local tag="$VERSION"

  if [[ "$tag" == "latest" ]]; then
    tag=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null \
          | grep -m1 '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/' || true)
    [[ -z "$tag" ]] && return 1
  fi
  tag="${tag#v}"
  echo "https://github.com/${REPO}/releases/download/v${tag}/${BIN_NAME}_${tag}_${target}.tar.gz"
}

install_release() {
  local target url tmp
  target=$(detect_target) || return 1
  url=$(resolve_release_url "$target" || true)
  if [[ -z "$url" ]]; then
    return 1
  fi
  say "Downloading $url"
  tmp=$(mktemp -d)
  if ! curl -fsSL -o "$tmp/reps.tar.gz" "$url"; then
    rm -rf "$tmp"
    return 1
  fi
  tar -xzf "$tmp/reps.tar.gz" -C "$tmp"
  install -m 0755 "$tmp/${BIN_NAME}" "$INSTALL_DIR/${BIN_NAME}"
  rm -rf "$tmp"
  ok "Installed $INSTALL_DIR/${BIN_NAME}"
}

install_via_go() {
  if ! command -v go >/dev/null 2>&1; then
    err "No release tarball matched your platform and Go isn't installed."
    err "Install Go: https://go.dev/dl/  — then re-run this script."
    return 1
  fi
  say "Falling back to: go install github.com/${REPO}/cmd/reps@latest"
  GOBIN="$INSTALL_DIR" go install "github.com/${REPO}/cmd/reps@latest"
  ok "Installed $INSTALL_DIR/${BIN_NAME}"
}

main() {
  printf "%s\n" "${BOLD}reps installer${RESET}"
  ensure_dir
  if ! install_release; then
    warn "No matching release found; trying go install instead."
    install_via_go || exit 1
  fi
  ensure_path_hint
  printf "\n${BOLD}Next step:${RESET}\n"
  printf "  ${DIM}\$${RESET} reps init\n\n"
  printf "  Wizard walks you through API key, model, sources, and ingestion.\n"
  printf "  Press SPACE to select items in multi-select; ENTER to confirm.\n\n"
}

main "$@"
