#!/usr/bin/env bash
# #255 replaces raw stream paint gates with typed endpoint/presenter ownership.
# Keep the existing check entrypoint, defending removal of the old authority.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
if rg -n --glob '*.go' --glob '!**/*_test.go' \
  'ptychild\.Screen|hostScan|SafeToPaint\(|RequestRepaint\(|ReplayThrough\(|\.host\.Write\(' \
  "$ROOT/cmd/internal/couchtty" "$ROOT/cmd/internal/termcmd"; then
  echo 'FAIL: compositor reintroduced raw replay/scanner/parent-write authority'
  exit 1
fi
if rg -n 'screen\.Feed|RepaintModes\(|RequestRepaint\(' \
  "$ROOT/cmd/internal/ptychild/child.go" "$ROOT/cmd/internal/ptychild/terminal.go"; then
  echo 'FAIL: Child reintroduced duplicate terminal authority'
  exit 1
fi
echo 'terminal ownership: migrated compositors have no raw replay or paint gate'
