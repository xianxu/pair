#!/bin/sh
set -eu
cd "$(dirname "$0")"
npm ci --ignore-scripts
cd ../..
PAIR_TERMINAL_ORACLE=1 go test ./cmd/internal/terminal -run 'TestRendererIndependent' -count=1
