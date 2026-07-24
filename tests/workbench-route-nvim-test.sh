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
printf '%s\n' "\$*" >> "$tmp/all-actions"
if [ "\$*" = "action list-panes --json --command --state" ]; then
  printf '%s\n' '[{"id":9,"title":"terminal","terminal_command":"pair term"},{"id":42,"title":"draft","terminal_command":"nvim -u $ROOT/nvim/init.lua /tmp/draft.md"}]'
  exit 0
fi
printf '%s\n' "\$*" >> "$tmp/actions"
EOF
chmod +x "$tmp/bin/zellij"

cat > "$tmp/driver.lua" <<'LUA'
local route = dofile(os.getenv('PAIR_HOME') .. '/nvim/workbench_route.lua')
vim.fn.writefile({ vim.json.encode({
  session = vim.env.ZELLIJ_SESSION_NAME,
  pane_id = '42',
  pid = vim.fn.getpid(),
}) }, vim.env.PAIR_DATA_DIR .. '/draft-pane-' .. vim.env.PAIR_TAG .. '.json')
assert(route.route('PairConfirmRestart'))
vim.cmd('qa!')
LUA

mkdir -p "$tmp/data"
PATH="$tmp/bin:$PATH" PAIR_HOME="$ROOT" PAIR_DATA_DIR="$tmp/data" PAIR_TAG=t \
  ZELLIJ_SESSION_NAME=pair-t \
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
if grep -Fq 'list-panes' "$tmp/all-actions"; then
  printf 'FAIL valid pane locator still invoked slow list-panes path\n'
  exit 1
fi

for init in init.lua review.lua scrollback.lua changelog.lua; do
  grep -Fq "workbench_route.lua" "$ROOT/nvim/$init" ||
    { printf 'FAIL %s does not load shared router\n' "$init"; exit 1; }
  grep -Fq "install_global_maps" "$ROOT/nvim/$init" ||
    { printf 'FAIL %s does not install global maps\n' "$init"; exit 1; }
done

grep -Fq "draft-pane-" "$ROOT/nvim/init.lua" ||
  { printf 'FAIL draft init does not publish pane locator\n'; exit 1; }

printf 'workbench-route-nvim-test ok\n'
