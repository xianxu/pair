#!/usr/bin/env bash
# doctor.sh — read the adaptation flight recorder and summarize harness drift.
#
# pair appends one JSON line per harness-adaptation trigger to
# $PAIR_DATA_DIR/adapt-<tag>.jsonl during normal use (see
# atlas/how-to-bring-up-a-new-harness-cli.md §3). This script tallies those
# lines per aspect/signal/outcome and surfaces the drift fingerprints
# (near-miss / fail) with their captured detail strings, so a human (or an
# agent reading doctor/README.md) can map each finding to an atlas aspect and
# propose a concrete fix.
#
# Usage: doctor.sh [path-to-adapt.jsonl]
#   No arg → $PAIR_DATA_DIR/adapt-$PAIR_TAG.jsonl, else newest adapt-*.jsonl.
# Always exits 0 (diagnostic); prints a NO-DATA notice if nothing is found.
set -euo pipefail

DATA_DIR="${PAIR_DATA_DIR:-${XDG_DATA_HOME:-$HOME/.local/share}/pair}"

f="${1:-}"
if [ -z "$f" ]; then
    if [ -n "${PAIR_TAG:-}" ] && [ -f "$DATA_DIR/adapt-${PAIR_TAG}.jsonl" ]; then
        f="$DATA_DIR/adapt-${PAIR_TAG}.jsonl"
    else
        # Newest session log as a fallback (e.g. invoked outside a live pane).
        f="$(ls -t "$DATA_DIR"/adapt-*.jsonl 2>/dev/null | head -1 || true)"
    fi
fi

if [ -z "$f" ] || [ ! -s "$f" ]; then
    echo "NO-DATA: no non-empty adaptation log found."
    echo "  looked in: $DATA_DIR (PAIR_TAG=${PAIR_TAG:-unset})"
    echo "  A log appears once you run a session: \$PAIR_DATA_DIR/adapt-<tag>.jsonl"
    exit 0
fi

if ! command -v jq >/dev/null 2>&1; then
    echo "ERROR: jq is required to parse the adaptation log ($f)." >&2
    exit 0
fi

echo "== pair-doctor =="
echo "log:   $f"
echo "lines: $(wc -l < "$f" | tr -d ' ')"
echo

echo "-- tallies (aspect · signal/outcome · count) --"
jq -rs '
  group_by([.aspect, .signal, .outcome])
  | map({aspect: .[0].aspect, signal: .[0].signal, outcome: .[0].outcome, n: length})
  | sort_by(.aspect, .signal, .outcome)
  | .[] | "  aspect \(.aspect)  \(.signal)/\(.outcome): \(.n)"
' "$f"
echo

echo "-- drift findings (near-miss / fail, deduped by detail) --"
findings="$(jq -rs '
  map(select(.outcome == "near-miss" or .outcome == "fail"))
  | unique_by([.signal, .detail])
  | sort_by(.aspect)
  | .[] | "  [\(.outcome)] aspect \(.aspect) \(.signal): \(.detail)"
' "$f")"
if [ -z "$findings" ]; then
    echo "  none — every adaptation that fired was recognized."
else
    echo "$findings"
fi
echo
echo "Map each finding to its aspect: see doctor/README.md and"
echo "atlas/how-to-bring-up-a-new-harness-cli.md §3."
