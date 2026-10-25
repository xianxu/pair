#!/usr/bin/env bash
. "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib/pair-test-bootstrap.sh"
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
. "$ROOT/tests/lib/run-headless.sh"
tmp="$(mktemp -d "${TMPDIR:-/tmp}/pair-route-poke-admission.XXXXXX")"
trap 'rm -rf "${tmp:?}"' EXIT
mkdir -p "$tmp/bin" "$tmp/home"
# The suite passes PAIR_TEST_BINARY; standalone runs share the same builder.
# Resolve the real guarded helper from the launch PATH, never an ambient Pair.
bash "$ROOT/tests/lib/pair-test-binary.sh" "$tmp/bin/pair"
cp "$ROOT/tests/lib/fake-zellij.py" "$tmp/bin/zellij"
cp "$ROOT/tests/lib/zellij_fixture_replies.py" "$tmp/bin/zellij_fixture_replies.py"
chmod +x "$tmp/bin/zellij"
PATH="$tmp/bin:$PATH" HOME="$tmp/home" ZELLIJ_SOCKET_DIR="$tmp/native" \
 PAIR_HOME="$ROOT" PAIR_FAKE_ZELLIJ_STATE="$tmp/state.json" \
 ZELLIJ_SESSION_NAME=test ZELLIJ_PANE_ID=9 PAIR_DRAFT_PANE_PATH='' \
 run_headless -- nvim --headless -u NONE -l "$ROOT/nvim/route_poke_admission_test.lua"
printf 'nvim-route-poke-admission-test ok\n'
