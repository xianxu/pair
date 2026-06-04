#!/usr/bin/env bash
# Regression test for doctor/emitter-health.sh — the stale-emitter-binary probe
# (#000047).
#
# Covers BOTH halves the plan-quality review flagged: the marker check (is the
# adapt signal string present in a binary) AND the binary *selection* (running
# binary via the pidfile vs. PATH fallback) — the error-prone half. Selection is
# tested by overriding `_pid_exe` (a thin, portable wrapper) so the test needs no
# live process and no `ps`/`/proc` (the dev sandbox blocks `ps`).
#
# Run: bash tests/emitter-health-test.sh   (also wired into `make test`)
set -uo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
LIB="$ROOT/doctor/emitter-health.sh"
RT="$(mktemp -d "${TMPDIR:-/tmp}/pair-emitter-test.XXXXXX")"
trap 'rm -rf "$RT"' EXIT

fails=0
pass() { printf '  ok   %s\n' "$1"; }
fail() { printf '  FAIL %s\n' "$1"; fails=$((fails + 1)); }

# Fake "binaries" are plain text files: `strings` finds the marker if the file
# contains it, so no real Mach-O/ELF is needed to exercise the grep.
fresh_pw="$RT/fresh-pair-wrap"; printf 'header return-remap footer\n' > "$fresh_pw"
fresh_ps="$RT/fresh-pair-slug"; printf 'x slug-parse y\n'            > "$fresh_ps"
stale_pw="$RT/stale-pair-wrap"; printf 'no markers here\n'           > "$stale_pw"

. "$LIB"

# 1) marker check: present / absent / can't-tell
_binary_has_marker "$fresh_pw" return-remap; [ $? -eq 0 ] && pass "marker present ⇒ 0" || fail "marker present not 0"
_binary_has_marker "$stale_pw" return-remap; [ $? -eq 1 ] && pass "marker absent ⇒ 1"  || fail "marker absent not 1"
_binary_has_marker "$RT/nope"  return-remap; [ $? -eq 2 ] && pass "no file ⇒ 2 (can't-tell)" || fail "missing file not 2"

# 1b) pipefail + a LARGE binary with the marker near the top. Guards the SIGPIPE
#     regression: `grep -q` early-exits and SIGPIPEs `strings`, so under
#     `set -o pipefail` (doctor.sh's mode) a real match would mis-report absent.
#     This shell already runs `set -o pipefail`; the tiny fixtures above can't
#     trigger it (strings finishes before grep exits).
big="$RT/big-pair-wrap"
{ printf 'return-remap\n'; head -c 3000000 /dev/zero | tr '\0' 'A'; printf '\n'; } > "$big"
( set -o pipefail; _binary_has_marker "$big" return-remap ) \
    && pass "pipefail + large binary ⇒ marker still found (no SIGPIPE false-negative)" \
    || fail "pipefail/SIGPIPE regression: large binary reported marker absent"

# 2) selection: PATH fallback when no pidfile. Put a fake pair-wrap first on PATH.
mkdir -p "$RT/pathbin"; cp "$fresh_pw" "$RT/pathbin/pair-wrap"; chmod +x "$RT/pathbin/pair-wrap"
got="$( PATH="$RT/pathbin:$PATH" _resolve_emitter pair-wrap "$RT/empty-datadir" "notag" )"
[ "$got" = "$RT/pathbin/pair-wrap" ] && pass "no pidfile ⇒ resolves via PATH" || fail "PATH fallback wrong: $got"

# 3) selection: running binary preferred via pidfile. Override _pid_exe so the
#    test needs no live process; the pidfile must win over PATH.
printf '99999\n' > "$RT/pair-wrap-pid-mytag"
_pid_exe() { printf '%s\n' "$fresh_pw"; }          # pretend pid 99999's exe is fresh_pw
got="$( PATH="$RT/pathbin:$PATH" _resolve_emitter pair-wrap "$RT" "mytag" )"
[ "$got" = "$fresh_pw" ] && pass "pidfile present ⇒ prefers running binary" || fail "pidfile not preferred: $got"
unset -f _pid_exe

# 4) report: stale binary surfaces STALE-BINARY (well, [STALE]); healthy ⇒ [ok].
#    Override the resolver to inject controlled binaries.
_resolve_emitter() { case "$1" in pair-wrap) printf '%s\n' "$RT_PW";; pair-slug) printf '%s\n' "$RT_PS";; esac; }
RT_PW="$stale_pw" RT_PS="$fresh_ps"
out="$(emitter_health_report "$RT" "mytag")"
grep -q '\[STALE\] pair-wrap' <<<"$out" && pass "report flags a stale pair-wrap" || fail "stale not flagged:"$'\n'"$out"
grep -q '\[ok\]    pair-slug'  <<<"$out" && pass "report marks a fresh pair-slug ok" || fail "fresh not ok:"$'\n'"$out"
RT_PW="$fresh_pw"
out="$(emitter_health_report "$RT" "mytag")"
grep -q 'STALE' <<<"$out" && fail "false STALE on all-fresh:"$'\n'"$out" || pass "all-fresh ⇒ no STALE line"

if [ "$fails" -ne 0 ]; then
    printf '\n%d failure(s)\n' "$fails"
    exit 1
fi
printf '\nall emitter-health tests passed\n'
