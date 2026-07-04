#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
tmp="$(mktemp -d "${TMPDIR:-/tmp}/pair-embedded-runtime.XXXXXX")"
trap 'rm -rf "$tmp"' EXIT

bin_dir="$tmp/bin"
home="$tmp/home"
xdg="$tmp/xdg"
pair_data="$tmp/custom-data"
mkdir -p "$bin_dir" "$home" "$xdg" "$pair_data"
gomodcache="$(go env GOMODCACHE)"
gocache="$(go env GOCACHE)"

make -C "$repo_root" runtimebundle-generate >/dev/null
go build -o "$bin_dir/pair" "$repo_root/cmd/pair-go"

cat >"$bin_dir/zellij" <<'SH'
#!/usr/bin/env bash
set -eu
printf '%s\n' "$*" >> "${ZELLIJ_LOG:?}"
case "$*" in
  "list-sessions --no-formatting"|"list-sessions --short")
    exit 0
    ;;
  --session*" action list-clients")
    exit 0
    ;;
  --config-dir*)
    config=""
    layout=""
    prev=""
    for arg in "$@"; do
      if [ "$prev" = "--config-dir" ]; then config="$arg"; fi
      if [ "$prev" = "--new-session-with-layout" ]; then layout="$arg"; fi
      prev="$arg"
    done
    test -f "$config/config.kdl"
    test -f "$layout"
    case "$config" in */custom-data/runtime/*/pair-home/zellij) ;; *) printf 'bad config path: %s\n' "$config" >&2; exit 11 ;; esac
    case "$layout" in */custom-data/runtime/*/pair-home/zellij/layouts/main.kdl) ;; *) printf 'bad layout path: %s\n' "$layout" >&2; exit 12 ;; esac
    root="${config%/zellij}"
    test -x "$root/bin/pair-wrap"
    # #94 M2: the launcher spawns the Go binaries directly; the .sh shims are gone.
    test -x "$root/bin/pair-session-watch"
    test -x "$root/bin/pair-title"
    test ! -e "$root/bin/pair-session-watch.sh"
    test ! -e "$root/bin/pair-title.sh"
    test ! -e "$root/bin/copy-on-select.sh"
    test ! -e "$root/bin/flash-pane.sh"
    test ! -e "$root/bin/clipboard-to-pane.sh"
    test -f "$root/nvim/init.lua"
    # #95: the launcher must have prepended $root/bin to PATH, so a bundled helper
    # resolves by BARE NAME here (zellij execs pair-wrap/copy-on-select by bare
    # name). This stub's own PATH does NOT include $root/bin — only the launcher's
    # #95 prepend puts it there, so this is the regression guard for the dropped
    # PATH prepend (#99 M5c).
    command -v copy-on-select >/dev/null || { echo "copy-on-select not on PATH (launcher #95 PATH prepend missing)" >&2; exit 21; }
    command -v pair-wrap      >/dev/null || { echo "pair-wrap not on PATH" >&2; exit 22; }
    printf '%s\n' "$root" > "${PAIR_SMOKE_ROOT:?}"
    exit 0
    ;;
  *)
    exit 0
    ;;
esac
SH
chmod +x "$bin_dir/zellij"

cat >"$bin_dir/ps" <<'SH'
#!/usr/bin/env bash
case "$*" in
  "-o comm= -p "*)
    printf 'sh\n'
    ;;
  "-o ppid= -p "*)
    printf '1\n'
    ;;
  *)
    exec /bin/ps "$@"
    ;;
esac
SH
chmod +x "$bin_dir/ps"

# Stub agent CLI: the create path validates the agent resolves on PATH before the
# zellij handoff. A no-op stub suffices — the stub zellij intercepts the handoff
# before the agent would actually run.
printf '#!/bin/sh\nexit 0\n' > "$bin_dir/claude"
chmod +x "$bin_dir/claude"

export PATH="$bin_dir:$PATH"
export HOME="$home"
export XDG_DATA_HOME="$xdg"
export PAIR_DATA_DIR="$pair_data"
export GOMODCACHE="$gomodcache"
export GOCACHE="$gocache"
export ZELLIJ_LOG="$tmp/zellij.log"
export PAIR_SMOKE_ROOT="$tmp/root"
unset PAIR_DEV PAIR_HOME PAIR_TAG PAIR_AGENT PAIR_AGENT_ARGS ZELLIJ_SESSION_NAME ZELLIJ ZELLIJ_PANE_ID

help_out="$("$bin_dir/pair" --help)"
case "$help_out" in
  pair\ —*) ;;
  *)
    printf 'copied pair --help did not print the native usage; first bytes:\n%s\n' "$help_out" >&2
    exit 1
    ;;
esac

old_a="aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
old_b="bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
old_c="cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
mkdir -p "$pair_data/runtime/$old_a/pair-home" \
         "$pair_data/runtime/$old_b/pair-home" \
         "$pair_data/runtime/$old_c/pair-home"
printf '{"digest":"%s","asset_count":0}\n' "$old_a" > "$pair_data/runtime/$old_a/manifest.json"
printf '{"digest":"%s","asset_count":0}\n' "$old_b" > "$pair_data/runtime/$old_b/manifest.json"
printf '{"digest":"%s","asset_count":0}\n' "$old_c" > "$pair_data/runtime/$old_c/manifest.json"
touch -t 202001010000 "$pair_data/runtime/$old_a"
touch -t 202001020000 "$pair_data/runtime/$old_b"
touch -t 202001030000 "$pair_data/runtime/$old_c"

# Launch with a PATH containing ONLY the stub tools + system dirs (NOT the repo's
# bin/), so the bare-name helper resolution asserted inside the stub zellij proves
# the launcher's #95 PATH prepend — not a dev shell that already has repo/bin on
# PATH. Without the #95 prepend, copy-on-select/pair-wrap won't resolve → exit 21.
PATH="$bin_dir:/usr/bin:/bin" "$bin_dir/pair" resume smoke >/dev/null

test -s "$PAIR_SMOKE_ROOT"
root="$(cat "$PAIR_SMOKE_ROOT")"
case "$root" in */custom-data/runtime/*/pair-home) ;; *) printf 'bad extracted root: %s\n' "$root" >&2; exit 1 ;; esac
test -d "$root"
test ! -e "$pair_data/runtime/$old_a"
test -d "$pair_data/runtime/$old_b"
test -d "$pair_data/runtime/$old_c"
test ! -e "$xdg/pair/runtime"

printf 'pair embedded runtime smoke passed\n'
