#!/bin/sh
# Shape tests for doctor/perf.sh — run via `make test-perf-capture`.
#
# perf.sh cannot be tested for its VALUES (they are whatever the machine is
# doing), so this pins the report's SHAPE, which is what consumers parse and
# what the collector bugs actually corrupted: a stray bare `0` line from
# `grep -c || echo 0`, and a failed collector rendering as a value.
set -e
here=$(cd "$(dirname "$0")" && pwd)
repo=$(cd "$here/.." && pwd)
fails=0
bad() { echo "FAIL $*" >&2; fails=$((fails + 1)); }

out=$(PAIR_HOME="$repo" sh "$here/perf.sh" 2>/dev/null)

# Every non-blank, non-comment line must be `key=value` or a `##` section.
# The bug this catches: `grep -c` prints 0 AND exits 1, so `|| echo 0` emitted a
# SECOND bare line that no parser could attribute to a key.
printf '%s\n' "$out" | while IFS= read -r line; do
	case "$line" in
		''|'#'*|'##'*|'###'*) continue ;;
		*=*) continue ;;
		*[!0-9]*) continue ;;   # sample rows are tab-separated ps output
		*) echo "STRAY $line" ;;
	esac
done | grep -q STRAY && bad "report contains a bare value line with no key"

for key in captured_at window_seconds load cpu_idle_pct pair_family_procs \
           build_procs windowserver_cpu_pct elapsed_seconds budget_seconds; do
	printf '%s\n' "$out" | grep -q "^$key=" || bad "missing key: $key"
done

# The budget is ENFORCED, not declared.
elapsed=$(printf '%s\n' "$out" | sed -n 's/^elapsed_seconds=//p')
budget=$(printf '%s\n' "$out" | sed -n 's/^budget_seconds=//p')
[ "$elapsed" -le "$budget" ] || bad "capture took ${elapsed}s over a ${budget}s budget"

# A missing probe degrades to n/a rather than vanishing or printing an empty
# value a reader would mistake for zero.
missing=$(PAIR_HOME=/nonexistent sh "$here/perf.sh" 2>/dev/null)
printf '%s\n' "$missing" | grep -q 'probes=n/a' \
	|| bad "absent probe binary did not degrade to n/a"

if [ "$fails" -gt 0 ]; then echo "$fails failure(s)" >&2; exit 1; fi
echo "perf.sh shape tests passed"
