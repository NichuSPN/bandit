#!/bin/bash
set -e

echo "======================================================="
echo "   Bandit 2-Step Hybrid Build (Go + Rust)"
echo "   Target: Single Executable Binary (./bandit)"
echo "======================================================="

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR"

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
