#!/usr/bin/env bash
. "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib/pair-test-bootstrap.sh"
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
. "$ROOT/tests/lib/run-headless.sh"
tmp="$(mktemp -d "${TMPDIR:-/tmp}/pair-route-poke-admission.XXXXXX")"
trap 'rm -rf "${tmp:?}"' EXIT
mkdir -p "$tmp/bin"
cp "$ROOT/tests/lib/fake-zellij.py" "$tmp/bin/zellij"
cp "$ROOT/tests/lib/zellij_fixture_replies.py" "$tmp/bin/zellij_fixture_replies.py"
chmod +x "$tmp/bin/zellij"
PATH="$tmp/bin:$PATH" PAIR_HOME="$ROOT" PAIR_FAKE_ZELLIJ_STATE="$tmp/state.json" \
 ZELLIJ_SESSION_NAME=test ZELLIJ_PANE_ID=9 PAIR_DRAFT_PANE_PATH='' \
 run_headless -- nvim --headless -u NONE -l "$ROOT/nvim/route_poke_admission_test.lua"
printf 'nvim-route-poke-admission-test ok\n'
