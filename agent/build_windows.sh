#!/usr/bin/env bash
set -euo pipefail

# Build all Windows agent variants from macOS/Linux.
# Usage: ./build_windows.sh [output_dir] [name_prefix]

OUT_DIR="${1:-.}"
NAME_PREFIX="${2:-remote_pc_agent}"
mkdir -p "$OUT_DIR"

echo "Building Windows x64..."
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o "$OUT_DIR/${NAME_PREFIX}_windows_x64.exe" .

echo "Building Windows x86 (32-bit)..."
GOOS=windows GOARCH=386 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o "$OUT_DIR/${NAME_PREFIX}_windows_x86.exe" .

echo "Building Windows ARM64..."
GOOS=windows GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o "$OUT_DIR/${NAME_PREFIX}_windows_arm64.exe" .

echo "Done. Generated binaries:"
ls -lh "$OUT_DIR"/"${NAME_PREFIX}"_windows_*.exe

echo
echo "SHA256 checksums:"
shasum -a 256 "$OUT_DIR"/"${NAME_PREFIX}"_windows_*.exe
