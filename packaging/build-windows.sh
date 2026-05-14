#!/bin/sh
set -eu

PARALLELISM="${PARALLELISM:-$(nproc 2>/dev/null || sysctl -n hw.ncpu 2>/dev/null || echo 4)}"
OUT_DIR="${OUT_DIR:-dist/windows-amd64}"

mkdir -p "$OUT_DIR"

export GOOS=windows
export GOARCH=amd64
export CGO_ENABLED=1
export CC="${CC:-x86_64-w64-mingw32-gcc}"
export GOMAXPROCS="$PARALLELISM"

echo "Building with parallelism=${PARALLELISM}"

go build -p "$PARALLELISM" -tags nativegui -ldflags="-H=windowsgui" -o "$OUT_DIR/wol-relay.exe" ./cmd/wol-relay &
pid_app=$!

go build -p "$PARALLELISM" -tags nativegui -ldflags="-H=windowsgui" -o "$OUT_DIR/wol-relay-installer.exe" ./cmd/wol-relay-installer &
pid_installer=$!

go build -p "$PARALLELISM" -tags nativegui -ldflags="-H=windowsgui" -o "$OUT_DIR/wol-relay-uninstaller.exe" ./cmd/wol-relay-uninstaller &
pid_uninstaller=$!

wait "$pid_app"
wait "$pid_installer"
wait "$pid_uninstaller"

echo "Built:"
echo "  $OUT_DIR/wol-relay.exe"
echo "  $OUT_DIR/wol-relay-installer.exe"
echo "  $OUT_DIR/wol-relay-uninstaller.exe"
