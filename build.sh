#!/bin/bash
set -e

echo "======================================================="
echo "   Bandit 2-Step Hybrid Build (Go + Rust)"
echo "   Target: Single Executable Binary (./bandit)"
echo "======================================================="

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR"

# Auto-load Rust environment if cargo is not in PATH
if ! command -v cargo &> /dev/null; then
    if [ -f "$HOME/.cargo/env" ]; then
        source "$HOME/.cargo/env"
    elif [ -x "$HOME/.cargo/bin/cargo" ]; then
        export PATH="$HOME/.cargo/bin:$PATH"
    fi
fi

if ! command -v cargo &> /dev/null; then
    echo "❌ Error: 'cargo' command not found."
    echo "Rust is required to build Bandit's core engine."
    echo "Please install Rust by running:"
    echo "   curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh"
    echo "Then restart your terminal or run: source \$HOME/.cargo/env"
    exit 1
fi

if ! command -v go &> /dev/null; then
    echo "❌ Error: 'go' command not found."
    echo "Go 1.24+ is required to build Bandit."
    echo "Please download Go from https://go.dev/dl/"
    exit 1
fi

echo ""
echo "[1/2] Compiling Rust core engine static library (bandit_engine)..."
cd bandit_engine
cargo build --release
cd ..

echo ""
echo "[2/2] Statically linking Go CLI binary with CGO..."
CGO_ENABLED=1 go build -o bandit main.go

echo ""
echo "[3/3] Installing binary globally..."
GOPATH_BIN="$(go env GOPATH)/bin"
mkdir -p "$GOPATH_BIN"
cp -f ./bandit "$GOPATH_BIN/bandit" 2>/dev/null && echo " Updated $GOPATH_BIN/bandit" || true
if [ -d "$HOME/.cargo/bin" ]; then
    cp -f ./bandit "$HOME/.cargo/bin/bandit" 2>/dev/null && echo " Updated $HOME/.cargo/bin/bandit" || true
fi

echo ""
echo "======================================================="
echo " ✓ Build Successful! Single binary created: ./bandit"
echo "======================================================="
