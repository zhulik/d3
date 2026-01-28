#!/usr/bin/env bash

set -euo pipefail

go tool air &
PID=$!

cleanup() {
  kill -SIGINT $PID 2>/dev/null || true
  wait $PID 2>/dev/null || true
}

trap cleanup EXIT

sleep 1
task test:conformance
