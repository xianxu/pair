#!/usr/bin/env bash
# pair-session-watch.sh — capture the agent's session-id by inspecting
# files held open by the agent's process tree, then write a per-(tag,agent)
# restart config under $PAIR_DATA_DIR.
#
# Usage:
#   pair-session-watch.sh <agent> <tag> <cwd> [agent-args...]
#
# Spawned in the background by bin/pair right before zellij launch on the
# new-session path. Issue #000020 replaced the earlier "first new file in
# the watch dir" snapshot with PID-bound discovery: two pair sessions in
# the same cwd previously raced to claim whichever agent's session file
# appeared first, occasionally cross-wiring tags' configs.
#
# Per-agent surface:
#   claude  — no-op. bin/pair pre-injects `--session-id <uuid>` and writes
#             config-<tag>-claude.json synchronously, so there's nothing
#             left to discover at runtime.
#   codex   — open file under ~/.codex/sessions/.../rollout-*-<uuid>.jsonl.
#             id = trailing UUID in filename.
#
#
# The pidfile ($PAIR_DATA_DIR/agent-pid-<tag>) is dropped by pair-wrap
# right after pty.Start; we wait briefly for it, then poll
# `lsof -p <pid>` against that pid + descendants.
#
# Config write is atomic (tmp + rename). Watcher self-times-out after 60s
# so a stuck agent doesn't leak the background process.

set -uo pipefail

agent="${1:-}"
tag="${2:-}"
cwd="${3:-}"
[ -z "$agent" ] || [ -z "$tag" ] || [ -z "$cwd" ] && exit 0
shift 3
args=( "$@" )

# Claude is fully handled at launch time. Codex and agy need lsof.
case "$agent" in
    codex|agy) ;;
    *) exit 0 ;;
esac

DATA_DIR="${PAIR_DATA_DIR:-${XDG_DATA_HOME:-$HOME/.local/share}/pair}"
mkdir -p "$DATA_DIR"
out="$DATA_DIR/config-$tag-$agent.json"
pid_file="$DATA_DIR/agent-pid-$tag"

# Aspect 3 flight recorder (atlas §3). PAIR_TAG is normally exported by bin/pair;
# fall back to our positional tag so logging works even if run standalone.
: "${PAIR_TAG:=$tag}"
export PAIR_TAG
_swdir="$(cd -P "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=bin/lib/adapt-log.sh
. "$_swdir/lib/adapt-log.sh" 2>/dev/null || true
nm_logged=0 # dedup the session-id near-miss across poll iterations

# Per-agent: directory we walk + the find pattern. Used by both the
# PID-bound primary path (for lsof path matching) and the legacy
# snapshot-diff fallback (for pair-wrap binaries that don't publish
# the pidfile yet).
case "$agent" in
    codex)
        watch_dir="$HOME/.codex/sessions"
        find_args=(-type f -name 'rollout-*.jsonl')
        ;;

    agy)
        watch_dir="$HOME/.gemini/antigravity-cli/conversations"
        find_args=(-type f -name '*.db')
        ;;
esac
mkdir -p "$watch_dir"

# Wait briefly for pair-wrap to publish this launch's agent PID. A pidfile from
# a prior run can survive until pair-wrap overwrites it; binding to that stale
# PID makes the watcher exit before the current agent starts.
watch_start=$(date +%s)
pid_wait_seconds="${PAIR_SESSION_WATCH_PID_WAIT_SECONDS:-2}"
case "$pid_wait_seconds" in
    ''|*[!0-9]*) pid_wait_seconds=2 ;;
esac
pid_deadline=$(( watch_start + pid_wait_seconds ))
pid_fresh() {
    [ -s "$pid_file" ] || return 1
    mt=$(stat -f %m "$pid_file" 2>/dev/null || echo 0)
    [ "$mt" -ge "$watch_start" ]
}
while ! pid_fresh && [ "$(date +%s)" -lt "$pid_deadline" ]; do
    sleep 0.1
done

root_pid=""
agent_start=0
if pid_fresh; then
    root_pid=$(cat "$pid_file" 2>/dev/null)
    # pair-wrap writes the pidfile right after pty.Start, so its mtime
    # is a tight upper bound on the agent's start epoch. Used as a
    # birth-time floor: any session file whose birth predates this was
    # created by an earlier pair session, not ours.
    agent_start=$(stat -f %m "$pid_file" 2>/dev/null || echo 0)
fi

# Legacy fallback state: snapshot the watch dir at start. Only consulted
# when the PID-bound path can't bind (no pidfile) — preserves the
# pre-#000020 behavior so old pair-wrap installs still capture sessions
# in the single-session case. Cross-tag races re-emerge in that path;
# the proper fix is to rebuild pair-wrap so the pidfile shows up.
legacy_existing=""
if [ -z "$root_pid" ]; then
    legacy_existing=$(find "$watch_dir" "${find_args[@]}" 2>/dev/null | sort)
fi

# pid + descendants. Codex/agy may keep the session file open in a native
# child below a JS launcher process. Use ps instead of pgrep -P: on macOS,
# pgrep can miss children that ps still reports, which makes the PID-bound
# discovery path inspect only the launcher and miss Codex's rollout fd.
descendants() {
    local root="$1"
    ps -axo pid=,ppid= 2>/dev/null | awk -v root="$root" '
        { children[$2] = children[$2] " " $1 }
        END {
            queue[1] = root
            seen[root] = 1
            head = 1
            tail = 1
            while (head <= tail) {
                p = queue[head++]
                print p
                n = split(children[p], kids, " ")
                for (i = 1; i <= n; i++) {
                    k = kids[i]
                    if (k != "" && !seen[k]) {
                        seen[k] = 1
                        queue[++tail] = k
                    }
                }
            }
        }'
}

match_path() {
    local line="$1"
    case "$agent" in
        codex)
            case "$line" in
                "$HOME/.codex/sessions/"*"/rollout-"*".jsonl") echo "$line" ;;
            esac
            ;;

        agy)
            case "$line" in
                "$HOME/.gemini/antigravity-cli/conversations/"*".db") echo "$line" ;;
            esac
            ;;
    esac
}

extract_id() {
    case "$agent" in
        codex)
            local fn
            fn=$(basename "$1" .jsonl)
            if [[ "$fn" =~ ([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})$ ]]; then
                echo "${BASH_REMATCH[1]}"
            fi
            ;;

        agy)
            # Path is like: ~/.gemini/antigravity-cli/conversations/<uuid>.db
            # The UUID is the basename of the file without the .db extension.
            local fn
            fn=$(basename "$1" .db)
            if [[ "$fn" =~ ^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$ ]]; then
                echo "$fn"
            fi
            ;;
    esac
}

deadline=$(( $(date +%s) + 60 ))
while [ "$(date +%s)" -lt "$deadline" ]; do
    # If we have a root_pid and the agent's gone, nothing more to do.
    if [ -n "$root_pid" ]; then
        kill -0 "$root_pid" 2>/dev/null || exit 0
    fi

    sid=""
    matched_file=""
    nm_candidate="" # a file that matched our pattern but yielded no id this pass

    if [ -n "$root_pid" ]; then
        # Primary path: lsof against the agent's PID tree. Race-free
        # because lsof output is scoped to specific PIDs, so a peer pair
        # session in the same cwd can't masquerade as ours.
        while IFS= read -r p; do
            [ -z "$p" ] && continue
            # `lsof -Fn` emits one record per fd: 'p' header line then 'n'
            # for the path. We only care about the n-prefixed lines.
            while IFS= read -r line; do
                [ "${line:0:1}" = "n" ] || continue
                path="${line:1}"
                hit=$(match_path "$path")
                [ -z "$hit" ] && continue
                cand=$(extract_id "$hit")
                if [ -n "$cand" ]; then
                    sid="$cand"
                    matched_file="$hit"
                    break 2
                fi
                nm_candidate="$hit"
            done < <(lsof -p "$p" -Fn 2>/dev/null)
        done < <(descendants "$root_pid")

        # Birth-time fallback: lsof can miss agents that close the fd
        # between writes. Walk the watch dir for files born at-or-after
        # our agent's start epoch (so files from earlier pair sessions
        # can't match), and accept only when there's exactly one
        # candidate — multiple = concurrent race, refuse rather than
        # guess wrong.
        if [ -z "$sid" ] && [ -n "${watch_dir:-}" ] && [ "$agent_start" -gt 0 ]; then
            candidates=()
            while IFS= read -r f; do
                [ -z "$f" ] && continue
                bt=$(stat -f %B "$f" 2>/dev/null || echo 0)
                [ "$bt" -ge "$agent_start" ] && candidates+=("$f")
            done < <(find "$watch_dir" "${find_args[@]}" 2>/dev/null)
            if [ "${#candidates[@]}" -eq 1 ]; then
                cand=$(extract_id "${candidates[0]}")
                if [ -n "$cand" ]; then
                    sid="$cand"
                    matched_file="${candidates[0]}"
                else
                    nm_candidate="${candidates[0]}"
                fi
            fi
        fi
    else
        # Legacy snapshot-diff path: pair-wrap didn't publish a pidfile
        # (older binary). Behaves identically to pre-#000020 — first new
        # file in the watch dir wins. Cross-tag race re-emerges here, but
        # we'd rather capture in the common single-session case than fail
        # silently. Rebuild pair-wrap to upgrade to the PID-bound path.
        current=$(find "$watch_dir" "${find_args[@]}" 2>/dev/null | sort)
        new=$(comm -13 <(printf '%s\n' "$legacy_existing") <(printf '%s\n' "$current"))
        if [ -n "$new" ]; then
            while IFS= read -r f; do
                [ -z "$f" ] && continue
                cand=$(extract_id "$f")
                if [ -n "$cand" ]; then
                    sid="$cand"
                    matched_file="$f"
                    break
                fi
                nm_candidate="$f"
            done <<< "$new"
        fi
    fi

    # Drift fingerprint: we found a file matching the agent's session-file
    # pattern but extract_id couldn't pull an id out of it — likely the
    # filename/format changed. Log once per session (the poll loop reruns).
    if [ -z "$sid" ] && [ -n "$nm_candidate" ] && [ "$nm_logged" = 0 ]; then
        adapt_log session-watch "$agent" 3 session-id near-miss \
            "matched session file but no id extracted: $(basename "$nm_candidate")"
        nm_logged=1
    fi

    if [ -n "$sid" ]; then
        # Strip --resume <id> / `resume <id>` so saved args don't carry
        # the resume binding into future relaunches — session_id below is
        # the canonical store. Same shape as bin/pair's stripping; keep
        # in sync.
        stripped=()
        n=${#args[@]}
        i=0
        if [ "$agent" = "codex" ] && [ $n -ge 2 ] \
            && [ "${args[0]}" = "resume" ]; then
            i=2
        fi
        while [ $i -lt $n ]; do
            if [ "${args[$i]}" = "--resume" ]; then
                i=$((i+2))
            else
                stripped+=("${args[$i]}")
                i=$((i+1))
            fi
        done

        tmp=$(mktemp "$out.XXXXXX") || exit 0
        if jq -n \
              --arg agent "$agent" \
              --arg sid "$sid" \
              '{ agent: $agent, args: $ARGS.positional, session_id: $sid }' \
              --args -- ${stripped[@]+"${stripped[@]}"} > "$tmp"
        then
            mv "$tmp" "$out"
            adapt_log session-watch "$agent" 3 session-id fired "session_id=$sid"
        else
            rm -f "$tmp"
        fi
        exit 0
    fi
    sleep 0.1
done

# Watched the full window without resolving an id: the session file never
# appeared where/how we expect — the strongest drift signal for this aspect.
adapt_log session-watch "$agent" 3 session-id fail "no session id within 60s deadline (agent=$agent)"
exit 0
