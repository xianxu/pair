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
# The prior guard for this was scoped to ONE function (couchtty's notification
# drain, the site the reviewer named). It would have passed unchanged if the
# unguarded consumer had been any other function -- the same defect as the code
# it guards. This one enumerates every `hostScan.` reference in both packages.
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

FILES=(
  "cmd/internal/couchtty/console.go"
  "cmd/internal/termcmd/run.go"
)

for rel in "${FILES[@]}"; do
  f="$ROOT/$rel"
  if [ ! -f "$f" ]; then
    echo "  FAIL $rel does not exist; this guard is checking nothing"
    fails=$((fails + 1))
    continue
  fi
  if ! grep -q "hostScan" "$f"; then
    echo "  FAIL $rel has no hostScan reference; the gate moved and this guard went blind"
    fails=$((fails + 1))
    continue
  fi
  bad="$(awk '
    /^func / { fn = $0 }
    /hostScan\./ {
      if (fn ~ /ForTest|forTest/) next
      if ($0 ~ /FeedFraming|= ptychild\.Screen\{\}|SafeToPaint\(\)/) next
      printf "%d: %s\n", NR, $0
    }
  ' "$f")"
  if [ -n "$bad" ]; then
    while IFS= read -r line; do
      echo "  FAIL $rel:$line"
      echo "       ^ reads the paint gate without asking SafeToPaint()"
      fails=$((fails + 1))
    done <<< "$bad"
  fi
done

if [ "$fails" -ne 0 ]; then
  echo "paint-gate-consumers: $fails failure(s)"
  exit 1
fi
echo "paint-gate-consumers: every gate consumer asks the shared question"
