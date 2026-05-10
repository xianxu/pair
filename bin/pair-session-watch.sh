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
# Per-agent discovery surface differs:
#   claude  — ~/.claude/projects/<encoded-cwd>/<id>.jsonl. id = filename.
#   codex   — ~/.codex/sessions/YYYY/MM/DD/rollout-<ts>-<id>.jsonl.
#             id = trailing UUID (8-4-4-4-12) in filename. Recursive scan.
#   gemini  — ~/.gemini/tmp/<project>/chats/session-<ts>-<short>.json,
#             where <short> is the first 8 chars of the id. The full id
#             needed for `--resume` lives inside the JSON body under
#             "sessionId" — we read the body, not just the filename.
# Other agents are silent no-ops.

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

# Per-agent: where session files show up, the find pattern that selects
# only those files, and how to pull the session id out of one.
extract_id() {
    case "$agent" in
        claude)
            basename "$1" .jsonl
            ;;
        codex)
            # Filename pattern: rollout-<ts>-<uuid>.jsonl. The session id
            # is the trailing UUID (8-4-4-4-12). bash 3.2 supports =~.
            local fn
            fn=$(basename "$1" .jsonl)
            if [[ "$fn" =~ ([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})$ ]]; then
                echo "${BASH_REMATCH[1]}"
            fi
            ;;
        gemini)
            # The filename has only an 8-char prefix of the id; the full
            # id required by `gemini --resume` is in the JSON body. Best-
            # effort: gemini may write the file before flushing sessionId,
            # so this can return empty on early reads — caller retries.
            jq -r '.sessionId // empty' "$1" 2>/dev/null
            ;;
    esac
}

case "$agent" in
    claude)
        # Claude encodes both `/` and `.` as `-` (so `/path/foo.nvim`
        # becomes `-path-foo-nvim`). Translating only `/` lands on an
        # empty dir for any cwd containing a dot.
        encoded=$(printf '%s' "$cwd" | tr ./ -)
        watch_dir="$HOME/.claude/projects/$encoded"
        find_args=(-maxdepth 1 -type f -name '*.jsonl')
        ;;
    codex)
        # Codex shards by date subdir, so the find is recursive. The
        # rollout- prefix narrows the match enough that scans are cheap
        # even with a long history.
        watch_dir="$HOME/.codex/sessions"
        find_args=(-type f -name 'rollout-*.jsonl')
        ;;
    gemini)
        # Gemini namespaces under ~/.gemini/tmp/<project>/chats/. The
        # `<project>` segment varies across runs (a hash of the cwd, by
        # observation), so we watch the whole tmp tree and filter by the
        # `*/chats/` path component.
        watch_dir="$HOME/.gemini/tmp"
        find_args=(-type f -name 'session-*.json' -path '*/chats/*')
        ;;
    *)
        exit 0
        ;;
esac

mkdir -p "$watch_dir"

# Snapshot the existing session files; new file = first one not in this
# list. `sort` keeps `comm` happy below; both inputs must be sorted.
existing=$(find "$watch_dir" "${find_args[@]}" 2>/dev/null | sort)

deadline=$(( $(date +%s) + 60 ))
while [ "$(date +%s)" -lt "$deadline" ]; do
    current=$(find "$watch_dir" "${find_args[@]}" 2>/dev/null | sort)
    new=$(comm -13 <(printf '%s\n' "$existing") <(printf '%s\n' "$current"))
    if [ -n "$new" ]; then
        # Walk all new files until one yields an id. Gemini in particular
        # may create the file before the JSON body is complete, so the
        # first hit can return empty — try the next, or wait for next tick.
        sid=""
        while IFS= read -r f; do
            [ -z "$f" ] && continue
            sid=$(extract_id "$f")
            [ -n "$sid" ] && break
        done <<< "$new"

        if [ -n "$sid" ]; then
            # Strip any --resume <id> (claude/gemini flag) or codex's
            # `resume <id>` subcommand prefix from args before serializing.
            # session_id below is the canonical storage for the resume
            # binding; retaining it in args makes every relaunch through
            # the picker compound the saved config — option 1 ("use saved
            # params + session") appends --resume <sid>, the watcher then
            # persists that combined argv, and on the NEXT run the picker
            # reads "args=[... --resume <id>] / resume=<id>" with the same
            # id duplicated. Stripping here keeps saved args = what the
            # user actually typed as launch flags, decoupled from the
            # resume target.
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

            # Build the JSON via jq (already a brew dep) so escaping handles
            # quotes/backslashes in args correctly. `--args` consumes the rest
            # of the argv as positional strings, exposed as $ARGS.positional.
            #
            # `${stripped[@]+"${stripped[@]}"}` safely expands to nothing when
            # the array is empty, working around bash 3.2's `set -u` treating
            # empty arrays as unset (macOS still ships bash 3.2 by default).
            tmp=$(mktemp "$out.XXXXXX") || exit 0
            if jq -n \
                  --arg agent "$agent" \
                  --arg sid "$sid" \
                  '{ agent: $agent, args: $ARGS.positional, session_id: $sid }' \
                  --args -- ${stripped[@]+"${stripped[@]}"} > "$tmp"
            then
                mv "$tmp" "$out"
            else
                rm -f "$tmp"
            fi
            exit 0
        fi
    fi
    sleep 0.1
done

exit 0
