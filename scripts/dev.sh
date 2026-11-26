#!/usr/bin/env bash
# Runs the Go API, the Python AI service and the web app; Ctrl-C stops all three.
set -euo pipefail
cd "$(dirname "$0")/.."
export PATH="$HOME/.local/share/mise/shims:$HOME/.local/bin:$HOME/go/bin:$PATH"

[[ -f .env ]] && set -a && source .env && set +a

pids=()
cleanup() { kill "${pids[@]}" 2>/dev/null || true; wait 2>/dev/null || true; }
trap cleanup EXIT INT TERM

run() {
  local name=$1; shift
  ( "$@" 2>&1 | sed -u "s/^/[$name] /" ) &
  pids+=($!)
}

run api bash -c 'cd go && go run ./cmd/core-api'
run ai  bash -c 'cd python/ai && uv run brandshake-ai'
run web bash -c 'cd web && npm run dev'

echo "web: http://localhost:5173   api: http://localhost:8080   ai: http://localhost:8090"
wait -n
