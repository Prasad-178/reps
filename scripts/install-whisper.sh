#!/usr/bin/env bash
# Install whisper.cpp + base.en model under ~/.reps/.
# Usage: scripts/install-whisper.sh [model]
#   model defaults to base.en. Other choices: tiny.en, small.en, medium.en.
set -euo pipefail

MODEL="${1:-base.en}"
HOME_DIR="${REPS_HOME:-$HOME/.reps}"
MODELS_DIR="$HOME_DIR/models"
mkdir -p "$MODELS_DIR"

echo "==> Installing whisper.cpp via Homebrew..."
if ! command -v whisper-cli >/dev/null 2>&1; then
  brew install whisper-cpp
else
  echo "    whisper-cli already on PATH"
fi

echo "==> Installing sox via Homebrew..."
if ! command -v sox >/dev/null 2>&1; then
  brew install sox
else
  echo "    sox already on PATH"
fi

MODEL_FILE="$MODELS_DIR/ggml-${MODEL}.bin"
if [ -f "$MODEL_FILE" ]; then
  echo "==> Model already exists at $MODEL_FILE"
else
  URL="https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-${MODEL}.bin"
  echo "==> Downloading $URL"
  curl -L --fail -o "$MODEL_FILE" "$URL"
fi

WBIN="$(command -v whisper-cli)"
echo
echo "Done. Add this to ~/.reps/config.toml under [voice]:"
echo "  enabled       = true"
echo "  whisper_bin   = \"$WBIN\""
echo "  whisper_model = \"$MODEL_FILE\""
echo "Or run:"
echo "  reps config voice.enabled true"
echo "  reps config voice.whisper_bin \"$WBIN\""
echo "  reps config voice.whisper_model \"$MODEL_FILE\""
