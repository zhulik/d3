#!/usr/bin/env bash

set -euo pipefail

go tool air &
PID=$!

trap 'kill -SIGINT $PID 2>/dev/null || true; wait $PID 2>/dev/null || true' EXIT

sleep 0.2
task test:conformance
