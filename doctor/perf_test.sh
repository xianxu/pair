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
#
# Hiding it takes BOTH a bogus PAIR_HOME and a PATH without it: since the probe
# is resolved on PATH first (a shipped pair has no $PAIR_HOME/bin at all), a
# bogus PAIR_HOME alone no longer hides an installed binary.
missing=$(PAIR_HOME=/nonexistent PATH=/usr/bin:/bin:/usr/sbin:/sbin sh "$here/perf.sh" 2>/dev/null)
printf '%s\n' "$missing" | grep -q 'probes=n/a' \
	|| bad "absent probe binary did not degrade to n/a"

# BR-9: nothing pinned a failing COLLECTOR or a failing PROBE, which is the
# whole point of the n/a rule. Both are exercised here, because "renders as n/a"
# is a claim that only a broken environment can test.
# A bare `mktemp -d` targets TMPDIR, which sandboxed agent shells deny -- that
# made `make test-perf-capture`, and therefore `make test`, fail for exactly the
# readers most likely to run it. Fall back to a repo-local dir.
stub=$(mktemp -d 2>/dev/null) || stub="$repo/.perf-test-stub.$$"
mkdir -p "$stub"
for t in ps top sysctl vm_stat iostat; do
	printf '#!/bin/sh\nexit 1\n' > "$stub/$t"; chmod +x "$stub/$t"
done
denied=$(PAIR_HOME="$repo" PATH="$stub:$PATH" sh "$here/perf.sh" 2>/dev/null)
rm -rf "$stub"

# Not one fabricated value. A bare `key=` or a `0` from a failed tool is
# indistinguishable from a real reading, which is the bug this rule exists for.
for key in load process_count cpu_idle_pct pair_family_procs build_procs; do
	val=$(printf '%s\n' "$denied" | sed -n "s/^$key=//p")
	case "$val" in
		"n/a"*) ;;
		"")     bad "$key vanished entirely under tool denial" ;;
		*)      bad "$key=$val is a fabricated value; a failed collector must render n/a" ;;
	esac
done

# A probe whose command always fails must render n/a, not a fast-looking number.
# `false` exits instantly, so timing it alone would report excellent latency.
if [ -x "$repo/bin/pair" ]; then
	if "$repo/bin/pair" hoprtt -spawn 3 -- /usr/bin/false >/dev/null 2>&1; then
		bad "a probe whose command always fails exited 0"
	fi
fi

if [ "$fails" -gt 0 ]; then echo "$fails failure(s)" >&2; exit 1; fi
echo "perf.sh shape tests passed"
