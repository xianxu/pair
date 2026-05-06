#!/usr/bin/env bash
# pair-session-watch.sh — capture the agent's session-id once it appears,
# then write a per-(tag,agent) restart config under $PAIR_DATA_DIR.
#
# Usage:
#   pair-session-watch.sh <agent> <tag> <cwd> [agent-args...]
#
# Spawned in the background by bin/pair right before zellij launch, on the
# new-session path only. Polls the agent's session dir at 100ms intervals
# for a freshly-created session file, extracts the id, and writes
# $PAIR_DATA_DIR/config-<tag>-<agent>.json:
#
#   { "agent": "<agent>", "args": [...], "session_id": "<id>" }
#
# The file is written atomically (tmp + rename) once the id is known, so a
# concurrent reader either sees the previous config or the new complete one
# — never a partial.
#
# One-shot: gives up after 60 seconds if no new file appears (agent crashed
# pre-startup, or wrote to an unexpected path). Bounded lifetime keeps a
# stuck watcher from leaking when the parent zellij session goes away.
#
# Per-agent discovery surface differs (see workshop/issues/000016); only
# claude is wired here for M1. Other agents are silent no-ops at this
# milestone.

set -uo pipefail

agent="${1:-}"
tag="${2:-}"
cwd="${3:-}"
[ -z "$agent" ] || [ -z "$tag" ] || [ -z "$cwd" ] && exit 0
shift 3
args=( "$@" )

DATA_DIR="${PAIR_DATA_DIR:-${XDG_DATA_HOME:-$HOME/.local/share}/pair}"
mkdir -p "$DATA_DIR"
out="$DATA_DIR/config-$tag-$agent.json"

# Per-agent: where the session file shows up, and how to extract its id.
case "$agent" in
    claude)
        # Claude encodes the cwd by replacing `/` with `-` and stores the
        # transcript at ~/.claude/projects/<encoded>/<session-id>.jsonl.
        # The filename minus extension IS the session id — no body parse.
        encoded=$(printf '%s' "$cwd" | tr / -)
        watch_dir="$HOME/.claude/projects/$encoded"
        ;;
    *)
        # codex/gemini come in M3.
        exit 0
        ;;
esac

mkdir -p "$watch_dir"

# Snapshot the existing session files; new file = first one not in this
# list. `sort` keeps `comm` happy below; both inputs must be sorted.
existing=$(find "$watch_dir" -maxdepth 1 -type f -name '*.jsonl' 2>/dev/null | sort)

deadline=$(( $(date +%s) + 60 ))
while [ "$(date +%s)" -lt "$deadline" ]; do
    current=$(find "$watch_dir" -maxdepth 1 -type f -name '*.jsonl' 2>/dev/null | sort)
    new=$(comm -13 <(printf '%s\n' "$existing") <(printf '%s\n' "$current"))
    if [ -n "$new" ]; then
        f=$(printf '%s\n' "$new" | head -1)
        sid=$(basename "$f" .jsonl)

        # Build the JSON via jq (already a brew dep) so escaping handles
        # quotes/backslashes in args correctly. `--args` consumes the rest
        # of the argv as positional strings, exposed as $ARGS.positional.
        tmp=$(mktemp "$out.XXXXXX") || exit 0
        # `${args[@]+"${args[@]}"}` safely expands to nothing when args is
        # empty, working around bash 3.2's `set -u` treating empty arrays
        # as unset (macOS still ships bash 3.2 by default).
        if jq -n \
              --arg agent "$agent" \
              --arg sid "$sid" \
              '{ agent: $agent, args: $ARGS.positional, session_id: $sid }' \
              --args -- ${args[@]+"${args[@]}"} > "$tmp"
        then
            mv "$tmp" "$out"
        else
            rm -f "$tmp"
        fi
        exit 0
    fi
    sleep 0.1
done

exit 0
