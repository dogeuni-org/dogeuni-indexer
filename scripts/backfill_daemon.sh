#!/usr/bin/env bash
set -euo pipefail

# Usage: scripts/backfill_daemon.sh [all|drc20|meme20] [batch] [config_path]
# Defaults: target=all, batch=2000, config.json in project root

TARGET="${1:-all}"
BATCH="${2:-2000}"
CFG="${3:-config.json}"

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
BIN="$ROOT_DIR/backfill"

if [[ ! -x "$BIN" ]]; then
  echo "Building backfill binary..."
  (cd "$ROOT_DIR" && go build -o backfill ./cmd/backfill)
fi

mkdir -p "$ROOT_DIR/logs"
LOGFILE="$ROOT_DIR/logs/backfill_$(date +%Y%m%d_%H%M%S).log"

echo "Starting backfill: target=$TARGET batch=$BATCH cfg=$CFG"
nohup "$BIN" -config "$CFG" -target "$TARGET" -batch "$BATCH" >> "$LOGFILE" 2>&1 &
PID=$!
echo $PID > "$ROOT_DIR/logs/backfill.pid"

echo "Backfill started. PID=$PID"
echo "Logs: $LOGFILE"
echo "To watch: tail -f $LOGFILE"
echo "To stop:  kill $(cat $ROOT_DIR/logs/backfill.pid)"


