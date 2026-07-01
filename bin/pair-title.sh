#!/usr/bin/env bash
# pair-title.sh — the per-tag title poller. Always-on (spawned on every pair
# create/attach); owns TWO title surfaces:
#   1. The zellij FRAME title of each agent pane — "<agent> (<count>) [<cwd>]",
#      where <count> is the agent's current context-window size (#71). Runs
#      with or without cmux.
#   2. The cmux WORKSPACE title — an activity heat-ramp emoji prefix. Only
#      when running inside cmux (block-local gate in the main loop).
#
# Usage:
#   pair-title.sh <tag> <agent>
#
# Spawned in the background by bin/pair on both the create and attach paths.
# Single-instance per tag enforced by pidfile (a second invocation finds the
# running poller and exits). Self-terminates when the pair-<tag> zellij
# session disappears (Alt+x, host shutdown, etc.).
#
# Activity sources (most-recent mtime wins):
#   - agent session file: claude jsonl / codex rollout / agy transcript
#   - nvim draft: $PAIR_DATA_DIR/draft-<tag>.md
# Both move on user typing AND agent output, so this captures both sides.
#
# ── cmux workspace title (surface 2) ──
# Buckets (heat-ramp, hottest first):
#   < 1 day   → 🔴 prefix  (today)
#   < 3 days  → 🟠 prefix  (last few days)
#   < 10 days → 🟡 prefix  (this fortnight)
#   < 21 days → 🔵 prefix  (this month)
#   else      → no prefix  (cold)
# All four emoji are CJK-wide so the title alignment in cmux's sidebar
# stays uniform across buckets.
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

# Config paths — hoisted above the test hook so update_frame_titles (which the
# hook can invoke) sees DATA_DIR / SESSION.
DATA_DIR="${PAIR_DATA_DIR:-${XDG_DATA_HOME:-$HOME/.local/share}/pair}"
SESSION="pair-$TAG"
DRAFT="$DATA_DIR/draft-$TAG.md"
PIDFILE="$DATA_DIR/title-pid-$TAG"

# True iff $1 is a live pair-title.sh poller for THIS tag. A bare
# `kill -0` is not enough for the single-instance guard: after a host
# sleep/reboot the kernel can recycle a dead poller's PID onto an
# unrelated process, and a naive liveness check would then read that
# stranger as "poller still alive" and suppress the respawn — freezing
# the title even across a pair restart. Confirm the command line really
# is our poller for this tag. The trailing space keeps tag "21" from
# matching "211"; the agent arg always follows the tag. Mirrors bin/pair's
# existing `ps -p <pid> -o command=` identity probe.
poller_alive() {
    local pid="$1" argv
    [ -n "$pid" ] || return 1
    kill -0 "$pid" 2>/dev/null || return 1
    argv=$(ps -p "$pid" -o command= 2>/dev/null || true)
    case "$argv" in
        *"pair-title.sh $TAG "*) return 0 ;;
    esac
    return 1
}

# ── frame meter (#71): show each agent pane's context size in its zellij
# FRAME title. Always-on (the frame exists with or without cmux); the cmux
# WORKSPACE title (main loop, below) stays cmux-gated. ──

# Abbreviate a raw cwd to ~ on a path boundary (mirrors bin/pair:1154).
abbrev_cwd() {
    case "$1" in
        "$HOME")   printf '~' ;;
        "$HOME"/*) printf '~%s' "${1#"$HOME"}" ;;
        *)         printf '%s' "$1" ;;
    esac
}

# Per-pane last-title cache to skip redundant renames. macOS bash 3.2 has no
# associative arrays — use a flat "pane_id=title" newline list. pane ids are
# numeric, so the `^id=` regex is metachar-safe.
frame_titles=""
frame_title_cached() { printf '%s\n' "$frame_titles" | sed -n "s/^$1=//p" | head -1; }
frame_title_store() { frame_titles=$(printf '%s\n' "$frame_titles" | grep -v "^$1="; printf '%s=%s\n' "$1" "$2"); }

# Rename every agent pane's zellij frame to "<agent> (<count>) [<cwd>]" (or
# "<agent> [<cwd>]" when no count). Reads pane-<tag>-*.json (pane id + display
# cwd); the count comes from `pair-context <tag> <agent>`.
update_frame_titles() {
    local pf agent pane_id cwd_disp count title cached
    for pf in "$DATA_DIR"/pane-"$TAG"-*.json; do
        [ -f "$pf" ] || continue
        agent=$(basename "$pf" .json)
        agent=${agent#pane-"$TAG"-}
        pane_id=$(jq -r '.pane_id // empty' "$pf" 2>/dev/null)
        [ -n "$pane_id" ] || continue
        cwd_disp=$(jq -r '.cwd_display // empty' "$pf" 2>/dev/null)
        [ -n "$cwd_disp" ] || cwd_disp=$(abbrev_cwd "$(jq -r '.cwd // empty' "$pf" 2>/dev/null)")
        count=$(pair context "$TAG" "$agent" 2>/dev/null)
        if [ -n "$count" ]; then
            title="$agent ($count) [$cwd_disp]"
        else
            title="$agent [$cwd_disp]"
        fi
        cached=$(frame_title_cached "$pane_id")
        [ "$cached" = "$title" ] && continue
        zellij --session "$SESSION" action rename-pane --pane-id "$pane_id" "$title" >/dev/null 2>&1 || true
        frame_title_store "$pane_id" "$title"
    done
}

# Test-only: two ticks with the same state — exercises the unchanged-skip
# (second tick must emit no new rename). Invoked via PAIR_TITLE_TEST_CALL.
_test_frame_titles_twice() {
    update_frame_titles
    update_frame_titles
}

# Test hook (mirrors bin/pair's PAIR_TEST_CALL): invoke a single helper
# against the trailing args, then exit — so the harness can unit-test
# poller_alive / update_frame_titles without a live cmux/zellij. Never set
# in normal use.
if [ -n "${PAIR_TITLE_TEST_CALL:-}" ]; then
    _fn="$PAIR_TITLE_TEST_CALL"
    shift 2 2>/dev/null || true
    "$_fn" "$@"
    exit $?
fi

POLL_INTERVAL=60
# Grace period for the zellij session to appear after spawn — covers the
# create path in bin/pair, which spawns this poller right BEFORE calling
# `zellij --new-session-with-layout`. Without this, the very first
# liveness check loses the race and the poller exits before it ever
# renames anything.
STARTUP_GRACE=30
ONE_DAY=86400
THREE_DAYS=259200
TEN_DAYS=864000
TWENTYONE_DAYS=1814400
PREFIX_HOT=$'\xf0\x9f\x94\xb4'      # 🔴 < 1 day
PREFIX_WARM=$'\xf0\x9f\x9f\xa0'     # 🟠 < 3 days
PREFIX_LUKEWARM=$'\xf0\x9f\x9f\xa1' # 🟡 < 10 days
PREFIX_COOL=$'\xf0\x9f\x94\xb5'     # 🔵 < 21 days

# Ignore SIGHUP. bin/pair spawns this with `& disown`, which only
# removes the job from the shell's job table — the poller still
# shares a controlling tty with the launching shell, so when that
# terminal goes away (cmux pane close, ghostty quit) the kernel
# sends SIGHUP and the poller terminates. The downstream symptom:
# workspace titles freeze at whatever bucket was last written.
# Trapping HUP keeps the poller alive across terminal lifecycle
# changes; it only exits via the explicit "zellij session gone"
# branch below.
trap '' HUP

# Single-instance: bail only if a prior poller for this tag is genuinely
# still running. poller_alive() is identity-checked (not a bare kill -0),
# so a recycled PID left by a dead poller can't wedge the respawn — the
# next pair create/attach/restart reliably revives a poller that a host
# sleep/reboot killed.
if [ -f "$PIDFILE" ]; then
    old_pid=$(cat "$PIDFILE" 2>/dev/null || true)
    if poller_alive "$old_pid"; then
        exit 0
    fi
fi
mkdir -p "$DATA_DIR"
echo "$$" > "$PIDFILE"
trap 'rm -f "$PIDFILE"' EXIT

# Resolve the agent's session file path (used by the cmux activity-emoji
# mtime check, NOT the frame meter — that reads via pair-context). Cached
# after first hit since the path is stable for the session's lifetime
# (claude --session-id pre-injection, codex/agy single-file model). Note:
# claude's /clear and compaction continue writing the SAME pinned file
# in-place (verified against real transcripts, #71) — it does NOT rotate,
# so the cache stays valid and the mtime keeps tracking real activity.
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
        printf '%s ' "$PREFIX_HOT"
    elif [ "$age" -lt "$THREE_DAYS" ]; then
        printf '%s ' "$PREFIX_WARM"
    elif [ "$age" -lt "$TEN_DAYS" ]; then
        printf '%s ' "$PREFIX_LUKEWARM"
    elif [ "$age" -lt "$TWENTYONE_DAYS" ]; then
        printf '%s ' "$PREFIX_COOL"
    fi
}

# Avoid hammering cmux with redundant rename calls. Only fire when the
# bucket changes (or on the very first iteration to set the prefix on a
# freshly-renamed workspace).
last_prefix="__init__"

# Wait for the zellij session to appear (create-path race: bin/pair
# spawns us right before `zellij --new-session-with-layout`). After this
# block, "session missing" reliably means the user ended the session.
session_seen=0
deadline=$(( $(date +%s) + STARTUP_GRACE ))
while [ "$(date +%s)" -lt "$deadline" ]; do
    if zellij list-sessions --short 2>/dev/null | grep -qx "$SESSION"; then
        session_seen=1
        break
    fi
    sleep 1
done
[ "$session_seen" -eq 1 ] || exit 0

# Tolerate transient `zellij list-sessions` failures: after a system
# sleep/wake the first IPC call sometimes returns empty briefly even
# though the session is alive, and a single-miss exit here was the
# other observed cause of zombie titles. Require this many consecutive
# misses (= `SESSION_MISS_THRESHOLD * POLL_INTERVAL` seconds of
# unbroken signal-lost time) before deciding the session is really
# gone.
SESSION_MISS_THRESHOLD=5
session_misses=0

while true; do
    # Self-terminate when the pair zellij session is gone — covers Alt+x,
    # host reboot, manual `zellij kill-session`, pair upgrade. Counted
    # across multiple polls so a single flaky IPC read doesn't kill us.
    if zellij list-sessions --short 2>/dev/null | grep -qx "$SESSION"; then
        session_misses=0
    else
        session_misses=$(( session_misses + 1 ))
        if [ "$session_misses" -ge "$SESSION_MISS_THRESHOLD" ]; then
            exit 0
        fi
        sleep "$POLL_INTERVAL"
        continue
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

    # Frame meter (#71): refresh each agent pane's zellij FRAME title while the
    # session is active. Gated on recent activity so idle sessions stop
    # re-rendering; the per-pane unchanged-skip cache (NOT the cmux bucket
    # guard below) is what prevents churn during an active-but-stable stretch.
    # MUST be outside the cmux bucket-change guard: an active session keeps the
    # same heat bucket for a day, so the bucket guard fires once — gating the
    # meter on it would refresh the count once and then freeze.
    if [ "$age" -lt $(( 2 * POLL_INTERVAL )) ]; then
        update_frame_titles
    fi

    # cmux WORKSPACE title (cmux-only) — the heat-ramp emoji prefix. Block-local
    # gate (replaces the old whole-script `exit 0`): the frame meter above runs
    # with or without cmux; only this workspace-title block is cmux-specific.
    if [ -n "${CMUX_WORKSPACE_ID:-}" ] && command -v cmux >/dev/null 2>&1; then
        prefix=$(prefix_for_age "$age")
        if [ "$prefix" != "$last_prefix" ]; then
            # Workspace-title ownership (matches bin/pair's cmux_rename_workspace):
            # if another live pair owns this cmux workspace, leave the title
            # alone. If the owner crashed without cleanup, take over. If no
            # owner is set, claim it so a later 2nd pair defers to us.
            owner_file="$DATA_DIR/cmux-owner-${CMUX_WORKSPACE_ID}"
            owner=""
            [ -f "$owner_file" ] && owner=$(cat "$owner_file" 2>/dev/null)
            if [ -n "$owner" ] && [ "$owner" != "$TAG" ]; then
                if zellij list-sessions --short 2>/dev/null | grep -qx "pair-$owner"; then
                    sleep "$POLL_INTERVAL"
                    continue
                fi
                # Owner is stale; fall through and reclaim.
            fi
            printf '%s\n' "$TAG" > "$owner_file"
            # Personal display convention (matches bin/pair's cmux_rename_workspace):
            # 'brain' → 🧠, 'book' → 📗, 'pair' → ♋ anywhere in the title.
            title="${prefix}${SESSION}"
            title="${title//brain/🧠}"
            title="${title//book/📗}"
            title="${title//pair/♋}"
            cmux rename-workspace "$title" >/dev/null 2>&1 || true
            last_prefix="$prefix"
        fi
    fi
    sleep "$POLL_INTERVAL"
done
