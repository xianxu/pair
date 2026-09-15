#!/bin/sh
# Disposable #250 acceptance: exact real-process/Zellij conformance followed by
# the real Couch UI over deterministic echo terminals. No paid agents or user
# Pair stores are opened. The Go fixtures own every temp directory and process.
set -eu
root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$root"
case "${1:-warm}" in
  warm|checkpoint|retired-checkpoint) mode=${1:-warm} ;;
  *) printf 'Usage: %s [warm|checkpoint|retired-checkpoint]\n' "$0" >&2; exit 2 ;;
esac
if [ ! -t 0 ] || [ ! -t 1 ]; then
  printf 'Run this smoke test in an interactive terminal.\n' >&2
  exit 2
fi
command -v zellij >/dev/null
printf 'First checking real disposable helper death, Zellij session ownership, and checkpoint transport.\n'
PAIR_LIVE_COUCH=1 go test ./cmd/internal/couchcore -run '^TestRecoveryRealHelperAndSessionConformanceLive$' -count=1 -v
fixture_dir=$(mktemp -d "${TMPDIR:-/tmp}/pair-recovery-ui.XXXXXX")
trap 'rm -f "$fixture_dir/recovery.test"; rmdir "$fixture_dir"' EXIT HUP INT TERM
go test -c -o "$fixture_dir/recovery.test" ./cmd/internal/couchcmd
printf '\nOpening disposable Couch UI (%s). Ctrl+Space opens the switcher; Ctrl+D exits the fixture agent.\n' "$mode"
PAIR_RECOVERY_INTERACTIVE="$mode" "$fixture_dir/recovery.test" -test.run '^TestRecoveryDisposableInteractive$' -test.timeout 6m -test.v
