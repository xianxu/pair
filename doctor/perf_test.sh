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
# Under the SUCCESS key, not a `probes=n/a` of its own: a consumer that knows
# only the success key would otherwise drop the row entirely, and an absent line
# reads as "this tool has no probes" rather than "the probes were not measured".
for k in pipe_hop_ms fork_exec_ms zellij_action_ms; do
	printf '%s\n' "$missing" | grep -q "^$k=n/a" \
		|| bad "absent probe binary: $k did not degrade to n/a under its own key"
done

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

# BR-36: the Lua side parses a RECORDED capture, which pins delta against a
# snapshot but not against what perf.sh emits TODAY. This asserts the live
# grammar, so a change here fails immediately instead of at the next slowdown.
in_sample=0
rows=0
printf '%s\n' "$out" | while IFS= read -r line; do
	case "$line" in
		"### procs") in_sample=1; continue ;;
		"##"*|"#"*)  in_sample=0; continue ;;
	esac
	[ "$in_sample" = 1 ] || continue
	[ -n "$line" ] || continue
	# pid<TAB>etime<TAB>rss<TAB>comm — the exact shape doctor.parse_samples reads.
	printf '%s' "$line" | awk -F'\t' '
		NF != 4          { print "SHAPE wrong field count: " $0; exit }
		$1 !~ /^[0-9]+$/ { print "SHAPE pid not numeric: " $0; exit }
		$3 !~ /^[0-9]+$/ { print "SHAPE rss not numeric: " $0; exit }'
	rows=$((rows + 1))
done | grep -q SHAPE && bad "perf.sh sample rows no longer match the grammar doctor.parse_samples reads"

printf '%s\n' "$out" | grep -q '^### procs$' || bad "no ### procs section for delta to parse"

# BR-38: the grammar assertion above validates whatever rows exist -- ZERO of
# them wherever ps is denied, which is a pass that proves nothing. Require rows.
proc_rows=$(printf '%s\n' "$out" | awk '/^### procs$/{p=1;next} /^#/{p=0} p&&NF{n++} END{print n+0}')
if [ "$proc_rows" -lt 10 ]; then
	bad "only $proc_rows sample rows; the grammar pin validated almost nothing"
fi
printf '%s\n' "$out" | grep -q '^### cputime$' || bad "no ### cputime section for delta to parse"

# A DENIED ps must render n/a, not an empty section. `ps | awk` exits with awk's
# status, so a failed ps used to exit 0 with no output: `cputimes || say n/a`
# never fired and both sample sections came out blank -- which a reader parses
# as "no processes ran", a fabricated claim. Caught by a sandboxed `make test`
# where ps is denied outright.
fake_bin="${TMPDIR:-/tmp}/perf_test_fakeps.$$"
mkdir -p "$fake_bin"
cat > "$fake_bin/ps" <<'FAKE'
#!/bin/sh
exit 1
FAKE
chmod +x "$fake_bin/ps"
out=$(PATH="$fake_bin:$PATH" PAIR_PERF_WINDOW=0 sh "$here/perf.sh" 2>/dev/null)
for section in cputime procs; do
	body=$(printf '%s\n' "$out" | awk -v s="### $section" '$0==s{f=1;next} /^###|^##/{f=0} f')
	case "$body" in
		*n/a*) ;;
		*) bad "a failed ps left ### $section rendering as '$body' instead of n/a" ;;
	esac
done
rm -rf "$fake_bin"

# The ps CONTENT filter. The stub above only exits 1, so it pins the
# pipeline-exit-status half and nothing about what a process NAME may contain --
# reverting the awk redaction left every suite green. A comm is
# attacker-controlled text that reaches a terminal, a file the agent is told to
# open, and a committed fixture: ^N is SO and garbles every following line, and
# a newline would split a sample row and could forge a `key=value` line inside
# the reader's own evidence.
fake2="${TMPDIR:-/tmp}/perf_test_fakeps2.$$"
mkdir -p "$fake2"
cat > "$fake2/ps" <<'FAKE'
#!/bin/sh
# One row with a long path AND a control byte, mimicking WhatsApp's real argv.
printf '%s\n' "  501 01:02.03 12345 /Applications/$(printf '\016')Evil.app/Contents/MacOS/EvilName"
# A real comm can contain SPACES; taking one awk field basenames "Google Chrome"
# to "Google" -- a wrong name on a real pid.
printf '%s\n' "  502 01:02.03 12345 /Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
FAKE
chmod +x "$fake2/ps"
out2=$(PATH="$fake2:$PATH" PAIR_PERF_WINDOW=0 sh "$here/perf.sh" 2>/dev/null)
rows=$(printf '%s\n' "$out2" | awk '/^### procs/{f=1;next} /^###|^##/{f=0} f')
printf '%s\n' "$rows" | grep -q "$(printf '\016')" \
	&& bad "a control byte in a process name reached the report"
printf '%s\n' "$rows" | grep -q '/Applications/' \
	&& bad "a full path reached the report; comm must be reduced to its basename"
printf '%s\n' "$rows" | grep -q 'EvilName' \
	|| bad "the readable part of the process name did not survive redaction"
printf '%s\n' "$rows" | grep -q 'Google Chrome' \
	|| bad "a process name containing a space was truncated (Google Chrome -> Google)"
rm -rf "$fake2"

if [ "$fails" -gt 0 ]; then echo "$fails failure(s)" >&2; exit 1; fi
echo "perf.sh shape tests passed"
