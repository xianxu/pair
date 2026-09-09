#!/usr/bin/env bash
# Every production reader of the reserved-row paint gate must ask SafeToPaint().
#
# WHY THIS IS A TEST AND NOT CARE. pair#199 BR-77 found couch reading the gate
# through a narrower question than `pair term` did; the fix updated four call
# sites and missed the fifth, which became BR-78 -- a hang. The lesson is not
# "check the fifth site". It is that a predicate with N consumers cannot be
# corrected by editing the consumers you happen to remember: the SET has to be
# enumerated mechanically, or the next one added is unguarded by default.
#
# SCOPE IS DERIVED, NOT LISTED (BR-84). The first cut of this guard hardcoded
# `FILES=(couchtty/console.go termcmd/run.go)` and matched the literal identifier
# `hostScan.` -- so a third consumer file, or the same scanner held under another
# name, was unguarded by default. That is the very defect this guard exists to
# prevent, moved out of the code and into the test. Both halves now come from the
# tree: the files are every non-test .go under cmd/ that declares a
# ptychild.Screen, and the identifier is read off each declaration.
#
# Allowed references:
#   FeedFraming(...)         -- a WRITE into the scanner, not a read of the gate
#   = ptychild.Screen{}      -- the reset on child swap; zero value is safe
#   SafeToPaint()            -- the shared question, the whole point
#   ...ForTest / ...forTest  -- probes that deliberately observe one half
#
# Anything else -- notably a bare MidSequence() or HoldsCursorSave() in
# production flow -- is a consumer asking half the question.
#
# Run: bash tests/paint-gate-consumers-test.sh   (wired into `make test`)
set -uo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
fails=0

# DERIVED: every non-test Go file under cmd/ that holds a ptychild.Screen.
# (Word-split over command substitution rather than arrays or temp files: macOS
# ships bash 3.2, which has no mapfile, and a sandboxed run may not get a TMPDIR.)
FILES="$(grep -rl --include='*.go' 'ptychild\.Screen' "$ROOT/cmd" |
  grep -v '_test\.go$' | sed "s|^$ROOT/||" | sort)"

case "$FILES" in
  *\ *) echo "  FAIL a derived path contains a space; this guard's word-splitting is unsafe"; exit 1 ;;
esac

nfiles=$(printf '%s\n' "$FILES" | grep -c .)
if [ "$nfiles" -lt 2 ]; then
  echo "  FAIL derived $nfiles consumer file(s); the scope went blind"
  echo "       (expected at least couchtty/console.go and termcmd/run.go)"
  exit 1
fi

checked=0
for rel in $FILES; do
  f="$ROOT/$rel"
  # DERIVED: whatever this file calls its scanner, not a remembered name.
  names="$(grep -oE '^[[:space:]]*[a-zA-Z_][a-zA-Z0-9_]*[[:space:]]+ptychild\.Screen$' "$f" |
    awk '{print $1}' | sort -u)"
  for name in $names; do
    checked=$((checked + 1))
    bad="$(awk -v scanner="$name" '
      /^func / { fn = $0 }
      index($0, scanner ".") {
        if (fn ~ /ForTest|forTest/) next
        if ($0 ~ /FeedFraming|= ptychild\.Screen\{\}|SafeToPaint\(\)/) next
        printf "%d: %s\n", NR, $0
      }
    ' "$f")"
    if [ -n "$bad" ]; then
      while IFS= read -r line; do
        echo "  FAIL $rel:$line"
        echo "       ^ reads the paint gate ($name) without asking SafeToPaint()"
        fails=$((fails + 1))
      done <<< "$bad"
    fi
  done
done

if [ "$checked" -eq 0 ]; then
  echo "  FAIL derived no scanner field in any consumer file; the guard is blind"
  exit 1
fi
if [ "$fails" -ne 0 ]; then
  echo "paint-gate-consumers: $fails failure(s)"
  exit 1
fi
echo "paint-gate-consumers: $checked gate(s) across $nfiles file(s); all consumers ask the shared question"
