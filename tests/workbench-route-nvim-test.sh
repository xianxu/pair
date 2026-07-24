#!/usr/bin/env bash
# Process-level coverage for global shortcut routing from Pair-owned nvim panes.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
. "$ROOT/tests/lib/run-headless.sh"
tmp="$(mktemp -d "${TMPDIR:-/tmp}/pair-nvim-route.XXXXXX")"
trap 'rm -rf "$tmp"' EXIT

mkdir -p "$tmp/bin"
cat > "$tmp/bin/zellij" <<EOF
#!/bin/sh
if [ "\$*" = "action list-panes --json --command --state" ]; then
  printf '%s\n' '[{"id":9,"title":"terminal","terminal_command":"pair term"},{"id":42,"title":"draft","terminal_command":"nvim -u $ROOT/nvim/init.lua /tmp/draft.md"}]'
  exit 0
fi
printf '%s\n' "\$*" >> "$tmp/actions"
EOF
chmod +x "$tmp/bin/zellij"

cat > "$tmp/driver.lua" <<'LUA'
local route = dofile(os.getenv('PAIR_HOME') .. '/nvim/workbench_route.lua')
assert(route.route('PairConfirmRestart'))
vim.cmd('qa!')
LUA

PATH="$tmp/bin:$PATH" PAIR_HOME="$ROOT" \
  run_headless -- nvim --headless -u NONE -l "$tmp/driver.lua"

want='action write --pane-id 42 28
action write --pane-id 42 14
action write-chars --pane-id 42 :lua PairConfirmRestart()
action write --pane-id 42 13'
got="$(cat "$tmp/actions")"
[ "$got" = "$want" ] || {
  printf 'FAIL addressed draft route:\n%s\n' "$got"
  exit 1
}

for init in init.lua review.lua scrollback.lua changelog.lua; do
  grep -Fq "workbench_route.lua" "$ROOT/nvim/$init" ||
    { printf 'FAIL %s does not load shared router\n' "$init"; exit 1; }
  grep -Fq "install_global_maps" "$ROOT/nvim/$init" ||
    { printf 'FAIL %s does not install global maps\n' "$init"; exit 1; }
done

printf 'workbench-route-nvim-test ok\n'
