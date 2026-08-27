# adapt-log.sh — shell emitter for the adaptation flight recorder.
#
# Sourced by shell components that need to record a harness-adaptation event
# (the Go orchestrators use cmd/internal/adapt instead). Writes ONE JSON line per call to
# the exact $PAIR_ADAPT_LOG_PATH selected by the launcher, byte-identical in
# schema + field order to
# the Go emitter (cmd/internal/adapt) and the Lua emitter (nvim/adapt.lua), so
# doctor/doctor.sh reads every component's lines uniformly. See
# atlas/how-to-bring-up-a-new-harness-cli.md §3.
#
# Usage:  adapt_log <comp> <agent> <aspect> <signal> <outcome> [detail]
#   outcome ∈ fired | bypass | near-miss | fail
# No-op (returns 0) when PAIR_ADAPT_LOG_PATH is unset or jq is missing —
# telemetry must never break the component it observes.

adapt_log() {
    [ -n "${PAIR_ADAPT_LOG_PATH:-}" ] || return 0
    command -v jq >/dev/null 2>&1 || return 0

    local comp="$1" agent="$2" aspect="$3" signal="$4" outcome="$5" detail="${6:-}"
    local ts
    ts="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
    # Cap detail. All shell call sites pass ASCII (session-file basenames, ids,
    # marker text), so this byte ≈ char cap matches the Go/Lua 200-byte rune-safe
    # cap exactly. (For multibyte detail it would differ — bash slices by chars
    # in a UTF-8 locale — but no shell emitter passes multibyte detail.)
    detail="${detail:0:200}"

    # Build with a fixed key order (ts,comp,agent,aspect,signal,outcome[,detail])
    # via jq so escaping is correct and detail is omitted when empty — exactly
    # what Go's struct marshalling (omitempty) produces.
    jq -cn \
        --arg ts "$ts" --arg comp "$comp" --arg agent "$agent" \
        --argjson aspect "$aspect" --arg signal "$signal" \
        --arg outcome "$outcome" --arg detail "$detail" \
        '{ts:$ts,comp:$comp,agent:$agent,aspect:$aspect,signal:$signal,outcome:$outcome}
         + (if $detail == "" then {} else {detail:$detail} end)' \
        >> "$PAIR_ADAPT_LOG_PATH" 2>/dev/null || true
}
