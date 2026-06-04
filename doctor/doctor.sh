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
#
# Output adapts to the stream: an interactive terminal gets a color-coded,
# box-drawn table; a pipe/CI/test (or NO_COLOR / TERM=dumb) gets the stable
# plain text the regression suite greps for. Same data, two renderings.
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

# --- presentation mode -------------------------------------------------------
# TABLE renders box-drawn tables; COLOR adds ANSI. A non-tty (pipe, CI, the
# regression test's command substitution) falls back to the original plain
# format so its grep assertions keep matching. NO_COLOR / TERM=dumb drop color
# but keep the table on a real terminal.
TABLE=0; COLOR=0
if [ -t 1 ]; then
    TABLE=1
    if [ -z "${NO_COLOR:-}" ] && [ "${TERM:-dumb}" != "dumb" ]; then COLOR=1; fi
fi

RESET=$'\033[0m'; BOLD=$'\033[1m'; DIM=$'\033[2m'
GREEN=$'\033[32m'; YELLOW=$'\033[33m'; RED=$'\033[31m'; CYAN=$'\033[36m'

paint() { if [ "$COLOR" = 1 ]; then printf '%s%s%s' "$1" "$2" "$RESET"; else printf '%s' "$2"; fi; }

# Map an outcome to its semantic color: green=worked, cyan=normal alt path,
# yellow=soft drift, red=hard failure.
outcome_color() {
    case "$1" in
        fired)     printf '%s' "$GREEN" ;;
        bypass)    printf '%s' "$CYAN" ;;
        near-miss) printf '%s' "$YELLOW" ;;
        fail)      printf '%s' "$RED" ;;
        *)         printf '' ;;
    esac
}

hbar() { local n=$1; printf '─%.0s' $(seq "$n"); }

# print_table OUTCOME_COL "Tab\tSeparated\tHeader" <<< "tsv body"
# Computes per-column widths, draws a bordered table, color-tints the cell at
# OUTCOME_COL by its outcome value (use -1 for no tint).
print_table() {
    local outcome_col="$1" header="$2"
    local -a hcols; IFS=$'\t' read -r -a hcols <<< "$header"
    local ncol=${#hcols[@]}
    local -a rows=() w=(); local line i
    for ((i = 0; i < ncol; i++)); do w[i]=${#hcols[i]}; done
    while IFS= read -r line; do
        [ -n "$line" ] || continue
        rows+=("$line")
        local -a c; IFS=$'\t' read -r -a c <<< "$line"
        for ((i = 0; i < ncol; i++)); do
            if (( ${#c[i]} > w[i] )); then w[i]=${#c[i]}; fi
        done
    done

    # border J: glyphs for left/mid/right junctions
    _border() {
        local L="$1" M="$2" R="$3" j; local out="$L"
        for ((j = 0; j < ncol; j++)); do
            out+="$(hbar $((w[j] + 2)))"
            if (( j < ncol - 1 )); then out+="$M"; else out+="$R"; fi
        done
        paint "$DIM" "$out"; printf '\n'
    }

    _border '┌' '┬' '┐'
    # header row (bold)
    printf '%s' "$(paint "$DIM" '│')"
    for ((i = 0; i < ncol; i++)); do
        printf ' %s ' "$(paint "$BOLD" "$(printf '%-*s' "${w[i]}" "${hcols[i]}")")"
        printf '%s' "$(paint "$DIM" '│')"
    done
    printf '\n'
    _border '├' '┼' '┤'
    # body rows
    local r
    for r in "${rows[@]+"${rows[@]}"}"; do
        local -a c; IFS=$'\t' read -r -a c <<< "$r"
        printf '%s' "$(paint "$DIM" '│')"
        for ((i = 0; i < ncol; i++)); do
            local cell; cell="$(printf '%-*s' "${w[i]}" "${c[i]:-}")"
            if [ "$i" = "$outcome_col" ]; then
                printf ' %s ' "$(paint "$(outcome_color "${c[i]:-}")" "$cell")"
            else
                printf ' %s ' "$cell"
            fi
            printf '%s' "$(paint "$DIM" '│')"
        done
        printf '\n'
    done
    _border '└' '┴' '┘'
}

# Parse tolerantly: `fromjson? // empty` drops any malformed/partial line
# (a writer can crash mid-line, and O_APPEND only guarantees atomicity below
# PIPE_BUF), so one bad line never aborts the whole diagnostic. `|| true`
# keeps a jq hiccup from tripping `set -e` — this tool always exits 0.
records="$(jq -R 'fromjson? // empty' "$f" 2>/dev/null || true)"

if [ "$TABLE" = 1 ]; then
    printf '%s\n' "$(paint "$BOLD" '══ pair-doctor ══')"
else
    echo "== pair-doctor =="
fi
printf '%s %s\n' "$(paint "$DIM" 'log:  ')" "$f"
printf '%s %s\n' "$(paint "$DIM" 'lines:')" "$(wc -l < "$f" | tr -d ' ')"
echo

# --- tallies -----------------------------------------------------------------
tallies_tsv="$(printf '%s\n' "$records" | jq -rs '
  group_by([.aspect, .signal, .outcome])
  | map({aspect: .[0].aspect, signal: .[0].signal, outcome: .[0].outcome, n: length})
  | sort_by(.aspect, .signal, .outcome)
  | .[] | [(.aspect|tostring), .signal, .outcome, (.n|tostring)] | @tsv
' 2>/dev/null || true)"

if [ "$TABLE" = 1 ]; then
    echo "$(paint "$BOLD" 'tallies') $(paint "$DIM" '— every adaptation that triggered, by outcome')"
    if [ -z "$tallies_tsv" ]; then
        echo "  $(paint "$DIM" '(no parseable records)')"
    else
        print_table 2 $'Aspect\tSignal\tOutcome\tCount' <<< "$tallies_tsv"
    fi
else
    echo "-- tallies (aspect · signal/outcome · count) --"
    if [ -n "$tallies_tsv" ]; then
        while IFS=$'\t' read -r aspect signal outcome n; do
            [ -n "$aspect" ] || continue
            printf '  aspect %s  %s/%s: %s\n' "$aspect" "$signal" "$outcome" "$n"
        done <<< "$tallies_tsv"
    fi
fi
echo

# --- drift findings ----------------------------------------------------------
findings_tsv="$(printf '%s\n' "$records" | jq -rs '
  map(select(.outcome == "near-miss" or .outcome == "fail"))
  | unique_by([.signal, .detail])
  | sort_by(.aspect)
  | .[] | [.outcome, (.aspect|tostring), .signal, ((.detail // "-") | gsub("\t"; " "))] | @tsv
' 2>/dev/null || true)"

if [ "$TABLE" = 1 ]; then
    echo "$(paint "$BOLD" 'drift findings') $(paint "$DIM" '— near-miss / fail, deduped by detail')"
    if [ -z "$findings_tsv" ]; then
        echo "  $(paint "$GREEN" '✓ none') $(paint "$DIM" '— every adaptation that fired was recognized.')"
    else
        print_table 0 $'Outcome\tAspect\tSignal\tDetail' <<< "$findings_tsv"
    fi
    echo
    if [ "$COLOR" = 1 ]; then
        printf '  %s  %s  %s  %s\n' \
            "$(paint "$GREEN" '● fired')" "$(paint "$CYAN" '● bypass')" \
            "$(paint "$YELLOW" '● near-miss')" "$(paint "$RED" '● fail')"
    fi
else
    echo "-- drift findings (near-miss / fail, deduped by detail) --"
    if [ -z "$findings_tsv" ]; then
        echo "  none — every adaptation that fired was recognized."
    else
        while IFS=$'\t' read -r outcome aspect signal detail; do
            [ -n "$outcome" ] || continue
            printf '  [%s] aspect %s %s: %s\n' "$outcome" "$aspect" "$signal" "$detail"
        done <<< "$findings_tsv"
    fi
fi
echo
echo "Map each finding to its aspect: see doctor/README.md and"
echo "atlas/how-to-bring-up-a-new-harness-cli.md §3."
