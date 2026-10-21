#!/usr/bin/env bash
# Standalone Zellij 0.45.1 diagnostic; no retries of measured queries.
set -eu
n=${1:-1000}; parallel=${2:-4}
case "$n:$parallel" in *[!0-9:]*|:*|*:) echo 'usage: bash zellij-false-miss.sh [queries=1000] [workers=4]' >&2; exit 2;; esac
(( n > 0 && n <= 100000 && parallel > 0 && parallel <= 16 )) || exit 2
[[ $(zellij --version) == 'zellij 0.45.1' ]] || { echo 'requires zellij 0.45.1' >&2; exit 2; }
root=$(mktemp -d "${TMPDIR:-/tmp}/zfm.XXXXXX") || exit 1
umask 077
export HOME="$root/h" XDG_CONFIG_HOME="$root/c" XDG_DATA_HOME="$root/d"
export XDG_CACHE_HOME="$root/k" XDG_STATE_HOME="$root/state" XDG_RUNTIME_DIR="$root/r"
export ZELLIJ_SOCKET_DIR="$root/s" ZELLIJ_CONFIG_DIR="$root/c/zellij" TMPDIR="$root/t" SHELL=/bin/sh
unset ZELLIJ ZELLIJ_SESSION_NAME ZELLIJ_PANE_ID ZELLIJ_CONFIG_FILE
mkdir -p "$HOME" "$XDG_DATA_HOME" "$XDG_CACHE_HOME" "$XDG_STATE_HOME" "$XDG_RUNTIME_DIR" "$ZELLIJ_SOCKET_DIR" "$ZELLIJ_CONFIG_DIR" "$TMPDIR"
cleanup() {
    trap - EXIT
    zellij kill-all-sessions --yes >/dev/null 2>&1 || :
    wait || :
    for (( tries=0; tries<30; tries++ )); do
        remaining=0
        for sock in "$ZELLIJ_SOCKET_DIR"/*/*; do [[ ! -S $sock ]] || remaining=1; done
        (( remaining == 0 )) && break
        zellij kill-all-sessions --yes >/dev/null 2>&1 || :
        sleep 0.1
    done
    if (( remaining )); then echo "cleanup=unconfirmed; private root retained: $root" >&2; exit 1; fi
    rm -rf -- "${root:?}"
    echo 'cleanup=pass (private session sockets absent; root removed)'
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
printf 'web_server false\nsession_serialization false\n' > "$ZELLIJ_CONFIG_DIR/config.kdl"
printf 'layout { pane command="/bin/sh"; }\n' > "$root/layout.kdl"
sessions=("zfm-$$-0" "zfm-$$-1" "zfm-$$-2")
for target in "${sessions[@]}"; do
    sock="$ZELLIJ_SOCKET_DIR/contract_version_1/$target"
    (( ${#sock} < 104 )) || { echo 'socket path too long; use a shorter TMPDIR' >&2; exit 2; }
    zellij --layout "$root/layout.kdl" attach --create-background "$target" > "$root/start-$target.log" 2>&1 || { cat "$root/start-$target.log" >&2; exit 1; }
    [[ -S "$ZELLIJ_SOCKET_DIR/contract_version_1/$target" ]] || { echo "session did not start: $target" >&2; exit 1; }
done
printf 'version=zellij-0.45.1 sessions=3 queries=%s workers=%s\n' "$n" "$parallel"
worker() {
    local w=$1 i target sock before rc a=0 b=0 uncertain=0 other=0 esc=$'\033'
    local out="$root/out.$w" err="$root/err.$w" samples="$root/samples.$w"
    : > "$samples"
    for (( i=w; i<n; i+=parallel )); do
        target=${sessions[$(( (i/4)%3 ))]}; sock="$ZELLIJ_SOCKET_DIR/contract_version_1/$target"
        before=0; [[ ! -S $sock ]] || before=1
        case $((i%4)) in
            0) set -- --session "$target" action list-panes --json --command;;
            1) set -- --session "$target" action list-panes --json --command --state;;
            2) set -- --session "$target" action list-clients;;
            3) set -- list-sessions --no-formatting;;
        esac
        rc=0; zellij "$@" > "$out" 2> "$err" || rc=$?
        if (( rc == 0 )) && grep -Fxq "Session '$target' not found. The following sessions are active:" "$err" &&
            sed "s/${esc}\[[0-9;]*m//g" "$out" | awk -v name="$target" '$1==name {found=1} END {exit !found}'; then
            a=$((a+1))
        elif (( rc == 1 )) && [[ ! -s $out ]] && grep -Fxq 'There is no active session!' "$err"; then
            if (( before )) && [[ -S $sock ]] && zellij --session "$target" action dump-layout > "$root/alive.$w" 2>/dev/null &&
                grep -Eq '^layout[[:space:]]*\{' "$root/alive.$w" && [[ -S $sock ]]; then b=$((b+1)); else uncertain=$((uncertain+1)); fi
        elif (( rc != 0 )); then other=$((other+1)); fi
        if [[ -s $err ]] || (( rc != 0 )); then
            { printf '\nquery=%s target=%s exit=%s argv=' "$i" "$target" "$rc"; printf '%q ' "$@"; printf '\nstdout:\n'; cat "$out"; printf '\nstderr:\n'; cat "$err"; } >> "$samples"
        fi
    done
    printf '%s %s %s %s\n' "$a" "$b" "$uncertain" "$other" > "$root/count.$w"
}
pids=()
for (( w=0; w<parallel; w++ )); do worker "$w" & pids+=("$!"); done
for pid in "${pids[@]}"; do wait "$pid"; done
cat "$root"/samples.*
awk '{a+=$1;b+=$2;u+=$3;o+=$4} END {printf "exit0_target_listed=%d\nexit1_alive_confirmed=%d\nexit1_liveness_unconfirmed=%d\nother_nonzero=%d\n",a,b,u,o}' "$root"/count.*
