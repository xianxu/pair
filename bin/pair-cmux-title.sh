#!/usr/bin/env bash
# pair-cmux-title.sh — periodically prefix the cmux workspace title with
# an activity emoji. Convention: one pair session per cmux workspace
# (see bin/pair's cmux_rename_workspace).
#
# Usage:
#   pair-cmux-title.sh <tag> <agent>
#
# Spawned in the background by bin/pair after cmux_rename_workspace, on
# both the create and attach paths. Single-instance per tag enforced by
# pidfile (a second invocation finds the running poller and exits).
# Self-terminates when the pair-<tag> zellij session disappears (Alt+x,
# host shutdown, etc.).
#
# Activity sources (most-recent mtime wins):
#   - agent session file: claude jsonl / codex rollout / gemini chat
#   - nvim draft: $PAIR_DATA_DIR/draft-<tag>.md
# Both move on user typing AND agent output, so this captures both sides.
#
# Buckets:
#   < 1 day  → 🚧 prefix  (active)
#   < 7 days → 📦 prefix  (recent)
#   else     → no prefix  (cold)
#
# We always set the title to "<prefix> pair-<tag>" (or "pair-<tag>" for
# cold), mirroring the zellij SESSION name the user sees everywhere
# else (`pair list`, the terminal title bar, etc.). Manual renames in
# the cmux sidebar are overridden on the next poll — matches bin/pair's
# existing rename-on-launch behavior.

set -uo pipefail

TAG="${1:-}"
AGENT="${2:-}"
[ -z "$TAG" ] || [ -z "$AGENT" ] && exit 0

# Outside cmux: nothing to do.
[ -n "${CMUX_WORKSPACE_ID:-}" ] || exit 0
command -v cmux >/dev/null 2>&1 || exit 0

DATA_DIR="${PAIR_DATA_DIR:-${XDG_DATA_HOME:-$HOME/.local/share}/pair}"
SESSION="pair-$TAG"
DRAFT="$DATA_DIR/draft-$TAG.md"
PIDFILE="$DATA_DIR/cmux-title-pid-$TAG"

POLL_INTERVAL=60
ONE_DAY=86400
ONE_WEEK=604800
PREFIX_BUSY=$'\xf0\x9f\x9a\xa7'   # 🚧
PREFIX_IDLE=$'\xf0\x9f\x93\xa6'   # 📦

# Single-instance: bail if a prior poller for this tag is still alive.
if [ -f "$PIDFILE" ]; then
    old_pid=$(cat "$PIDFILE" 2>/dev/null || true)
    if [ -n "$old_pid" ] && kill -0 "$old_pid" 2>/dev/null; then
        exit 0
    fi
fi
mkdir -p "$DATA_DIR"
echo "$$" > "$PIDFILE"
trap 'rm -f "$PIDFILE"' EXIT

# Resolve the agent's session file path. Cached after first hit since the
# path is stable for the session's lifetime (claude --session-id pre-
# injection, codex/gemini single-file model). /clear in claude rotates
# the file, leaving the cache pointed at the old jsonl — that file's
# mtime freezes, which is the desired "no recent activity" signal anyway.
agent_file_cache=""
agent_session_file() {
    if [ -n "$agent_file_cache" ] && [ -f "$agent_file_cache" ]; then
        echo "$agent_file_cache"
        return
    fi
    local cfg="$DATA_DIR/config-$TAG-$AGENT.json"
    local sid path=""
    sid=$(jq -r '.session_id // empty' "$cfg" 2>/dev/null)
    [ -n "$sid" ] || return
    case "$AGENT" in
        claude)
            local enc
            enc=$(printf '%s' "$PWD" | tr ./ -)
            path="$HOME/.claude/projects/$enc/$sid.jsonl"
            [ -f "$path" ] || path=""
            ;;
        codex)
            path=$(find "$HOME/.codex/sessions" -type f -name "*$sid*.jsonl" 2>/dev/null | head -1)
            ;;
        gemini)
            path=$(grep -rl --include='*.json' "\"sessionId\":\"$sid\"" "$HOME/.gemini/tmp" 2>/dev/null | head -1)
            ;;
    esac
    if [ -n "$path" ] && [ -f "$path" ]; then
        agent_file_cache="$path"
        echo "$path"
    fi
}

# Most recent mtime across all activity sources.
latest_activity() {
    local latest=0 m
    local af
    af=$(agent_session_file)
    for f in "$DRAFT" "$af"; do
        [ -n "$f" ] && [ -f "$f" ] || continue
        m=$(stat -f %m "$f" 2>/dev/null || echo 0)
        [ "$m" -gt "$latest" ] && latest="$m"
    done
    echo "$latest"
}

# Bucket an age (seconds) into a prefix string. Empty = no prefix.
prefix_for_age() {
    local age="$1"
    if [ "$age" -lt "$ONE_DAY" ]; then
        printf '%s ' "$PREFIX_BUSY"
    elif [ "$age" -lt "$ONE_WEEK" ]; then
        printf '%s ' "$PREFIX_IDLE"
    fi
}

# Avoid hammering cmux with redundant rename calls. Only fire when the
# bucket changes (or on the very first iteration to set the prefix on a
# freshly-renamed workspace).
last_prefix="__init__"

while true; do
    # Self-terminate when the pair zellij session is gone — covers Alt+x,
    # host reboot, manual `zellij kill-session`, pair upgrade.
    if ! zellij list-sessions --short 2>/dev/null | grep -qx "$SESSION"; then
        exit 0
    fi

    now=$(date +%s)
    latest=$(latest_activity)
    if [ "$latest" -gt 0 ]; then
        age=$(( now - latest ))
    else
        # No activity sources resolved yet (config not written, agent
        # crashed pre-startup). Skip this iteration; try again next tick.
        sleep "$POLL_INTERVAL"
        continue
    fi

    prefix=$(prefix_for_age "$age")
    if [ "$prefix" != "$last_prefix" ]; then
        cmux rename-workspace "${prefix}${SESSION}" >/dev/null 2>&1 || true
        last_prefix="$prefix"
    fi
    sleep "$POLL_INTERVAL"
done
