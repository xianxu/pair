#!/bin/bash
# Runs only inside the guest; all roots are runtime-owned guest paths.
set -u
export PATH=/opt/pair/bin:/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin
export LANG=C TERM=xterm-256color
fail() { printf '%s\n' 'guest bootstrap refused unsafe or unusable roots' >&2; exit 125; }
[ "$#" -ge 5 ] || fail
source=$1 work=$2 results=$3
shift 3
[ "$1" = -- ] || fail
shift
[ "$#" -gt 0 ] && [ -n "$1" ] || fail
safe_root() {
    local path=$1 part
    case "$path" in /|''|*//*|*/./*|*/../*|*/.|*/..|*/) return 1;; /*) ;; *) return 1;; esac
    case "$path" in *[$'\001'-$'\037'$'\177']*) return 1;; esac
    part=$path
    while [ "$part" != / ]; do
        [ ! -L "$part" ] || return 1
        part=${part%/*}
        [ -n "$part" ] || part=/
    done
}
overlaps() { case "$1/" in "$2/"*) return 0;; esac; case "$2/" in "$1/"*) return 0;; esac; return 1; }
for root in "$source" "$work" "$results"; do safe_root "$root" || fail; done
overlaps "$source" "$work" && fail
overlaps "$source" "$results" && fail
overlaps "$work" "$results" && fail
[ -d "$source" ] && [ ! -e "$results" ] && [ ! -L "$results" ] && [ ! -e "$work" ] && [ ! -L "$work" ] || fail
[ ! -e "$source/.git" ] || { [ -d "$source/.git" ] && [ ! -L "$source/.git" ]; } || fail
[ ! -e "$results/status" ] && [ ! -L "$results/status" ] || fail
# Atomic creation prevents reusing any previous run, including symlink roots.
umask 077
mkdir "$work" "$results" || fail
export HOME="$work/home" TMPDIR="$work/tmp"
export XDG_CONFIG_HOME="$work/config" XDG_DATA_HOME="$work/data"
export XDG_CACHE_HOME="$work/cache" XDG_STATE_HOME="$work/state"
export ZELLIJ_SOCKET_DIR="$work/sockets" PAIR_DATA_DIR="$work/pair"
export COUCH_ISOLATED_ROOT="$work/couch" WF_SDLC_QUEUE_ROOT="$work/queue"
mkdir "$HOME" "$TMPDIR" "$XDG_CONFIG_HOME" "$XDG_DATA_HOME" "$XDG_CACHE_HOME" "$XDG_STATE_HOME" "$ZELLIJ_SOCKET_DIR" "$PAIR_DATA_DIR" "$COUCH_ISOLATED_ROOT" "$WF_SDLC_QUEUE_ROOT" "$work/checkout" || fail
cp -R -P "$source/." "$work/checkout/" || fail
cd "$work/checkout" || fail
"$@"
command_exit=$?
collection_error=0
if mkdir "$results/diagnostics"; then
    # One overflow sentinel byte bounds discovery storage and makes overflow fail.
    # pipefail preserves find failures, including an upstream SIGPIPE at the cap.
    if (set -C -o pipefail; /usr/bin/find "$TMPDIR" "$HOME" -type f -iname '*zellij*' -print0 | /usr/bin/head -c 65537 > "$work/diagnostic-files") &&
        [ "$(/usr/bin/wc -c < "$work/diagnostic-files")" -le 65536 ]; then
        count=0
        while IFS= read -r -d '' log; do
            [ "$count" -lt 8 ] || break
            (set -C; /usr/bin/head -c 65536 "$log" > "$results/diagnostics/zellij-$count.log") || collection_error=1
            count=$((count + 1))
        done < "$work/diagnostic-files"
    else
        collection_error=1
    fi
else
    collection_error=1
fi
# No command arguments or environment are persisted in the bounded status.
(set -C; printf 'exit_code=%s\ncollection_error=%s\n' "$command_exit" "$collection_error" > "$results/status") || collection_error=1
# The fixed catalog is bounded independently of command-created guest files.
# No writable host share exists; the host pulls only this validated population.
(
    set -C
    {
    printf '{"version":1,"files":[{"path":"status","size":%s}' "$(/usr/bin/wc -c < "$results/status")"
    for ((i=0; i<8; i++)); do
        log="$results/diagnostics/zellij-$i.log"
        if [ -f "$log" ] && [ ! -L "$log" ]; then
            printf ',{"path":"diagnostics/zellij-%s.log","size":%s}' "$i" "$(/usr/bin/wc -c < "$log")"
        fi
    done
    printf ']}\n'
    } > "$results/manifest.json"
) || collection_error=1
[ "$command_exit" -ne 0 ] && exit "$command_exit"
[ "$collection_error" -eq 0 ] || exit 125
exit 0
