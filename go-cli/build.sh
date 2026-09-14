#!/usr/bin/env bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

GOPATH_BIN="$(go env GOPATH)/bin"
mkdir -p "$GOPATH_BIN"

DEST="$GOPATH_BIN/leet"

echo "Building leet for Linux..."
go build -ldflags="-s -w" -o "$DEST" .

echo "✔ Successfully installed to: $DEST"

# Check if destination directory is in PATH
if [[ ":$PATH:" != *":$GOPATH_BIN:"* ]]; then
    echo ""
    echo "ℹ Note: '$GOPATH_BIN' is not in your current PATH."
    echo "  Add it by running:"
    echo "    echo 'export PATH=\"\$PATH:$GOPATH_BIN\"' >> ~/.bashrc  # (or ~/.zshrc)"
    echo "    source ~/.bashrc"
    echo ""
fi

echo "Run 'leet --version' or 'leet init' to get started!"
