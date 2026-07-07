#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
tmp="$(mktemp -d "${TMPDIR:-/tmp}/pair-embedded-runtime.XXXXXX")"
trap 'rm -rf "$tmp"' EXIT

bin_dir="$tmp/bin"
pairbin="$tmp/pairbin"   # pair lives APART from the stub tools (see the launch below)
home="$tmp/home"
xdg="$tmp/xdg"
pair_data="$tmp/custom-data"
mkdir -p "$bin_dir" "$pairbin" "$home" "$xdg" "$pair_data"
gomodcache="$(go env GOMODCACHE)"
gocache="$(go env GOCACHE)"

make -C "$repo_root" runtimebundle-generate >/dev/null
go build -o "$pairbin/pair" "$repo_root/cmd/pair-go"

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
    # #104 M3: the bundle carries config + shell shims ONLY — NO helper binaries
    # (every former helper is a `pair <sub>`). The shims are still bundled.
    test -x "$root/bin/pair-help"
    test -x "$root/bin/pair-notify"
    test ! -e "$root/bin/pair-wrap"
    test ! -e "$root/bin/pair-session-watch"
    test ! -e "$root/bin/pair-title"
    test ! -e "$root/bin/copy-on-select"
    test ! -e "$root/bin/pair-scrollback-render"
    test ! -e "$root/bin/pair"   # pair is never self-embedded
    test -f "$root/nvim/init.lua"
    # #104 M3: the launcher fronts the RUNNING pair's dir (dir os.Executable) on
    # PATH, so `pair` — and thus every `pair <sub>` — resolves inside the session
    # even though pair lives OUTSIDE this stub's PATH ($pairbin, not on PATH; see
    # the launch below). This is the regression guard for the "pair-on-PATH"
    # mechanism (no bundled helper binaries to resolve anymore).
    command -v pair >/dev/null || { echo "pair not on PATH (launcher pair-on-PATH prepend missing)" >&2; exit 21; }
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

help_out="$("$pairbin/pair" --help)"
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

# Launch with a PATH containing ONLY the stub tools + system dirs — crucially NOT
# $pairbin (where the pair binary lives). pair is invoked by absolute path, so it
# runs; the `command -v pair` check inside the stub zellij then proves the
# launcher fronted $pairbin (dir os.Executable) on the session PATH. Without that
# #104-M3 prepend, `pair` wouldn't resolve inside → exit 21.
PATH="$bin_dir:/usr/bin:/bin" "$pairbin/pair" resume smoke >/dev/null

test -s "$PAIR_SMOKE_ROOT"
root="$(cat "$PAIR_SMOKE_ROOT")"
case "$root" in */custom-data/runtime/*/pair-home) ;; *) printf 'bad extracted root: %s\n' "$root" >&2; exit 1 ;; esac
test -d "$root"
test ! -e "$pair_data/runtime/$old_a"
test -d "$pair_data/runtime/$old_b"
test -d "$pair_data/runtime/$old_c"
test ! -e "$xdg/pair/runtime"

printf 'pair embedded runtime smoke passed\n'
