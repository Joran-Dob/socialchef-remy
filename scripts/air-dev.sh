#!/usr/bin/env bash
# Run both server and worker; on exit (e.g. Air rebuild) kill both.
set -e
trap 'kill $(jobs -p) 2>/dev/null' EXIT
cd "$(dirname "$0")/.."
./tmp/server &
./tmp/worker &
wait
