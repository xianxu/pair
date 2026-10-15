#!/usr/bin/env bash
# pair#436: a test goal drops the live session's PAIR_*/COUCH_*/ZELLIJ* identity
# and keeps every knob a caller may pass. Knobs are derived, not listed: every
# name test code reads that production never mentions, plus every name the docs
# pass to make or go test. Each must reach a test recipe.
set -euo pipefail
cd "$(dirname "$0")/.."

production=$(make -s --no-print-directory test-make-env-production-names)
[ -n "$production" ] || { echo "FAIL: production name scan found nothing"; exit 1; }

# find, not git ls-files: test-clean-bootstrap runs make test from a no-.git archive.
test_read=$({ find cmd nvim doctor tests probes -type f \( -name '*_test.go' -o -name '*_test.lua' -o -name '*test*.lua' \
  -o -name '*test*.sh' -o -path 'tests/*' -o -path 'probes/*' -o -path 'cmd/probes/*' \) -print0 2>/dev/null \
  | xargs -0 grep -IhoE '\b(PAIR|COUCH)_[A-Z0-9_]+' 2>/dev/null || true; } | sort -u)
test_only=$(comm -23 <(printf '%s\n' "$test_read") <(printf '%s\n' "$production"))
documented=$(grep -rhoE '\b(PAIR|COUCH)_[A-Z0-9_]+=[^ ]* +(\\[[:space:]]*)?(make|go test)' README.md atlas Makefile.local doctor 2>/dev/null \
  | grep -oE '^(PAIR|COUCH)_[A-Z0-9_]+' | sort -u || true)
knobs=$(printf '%s\n%s\n' "$test_only" "$documented" | grep -v '^$' | sort -u)
[ -n "$knobs" ] || { echo "FAIL: found no knobs (scan drifted?)"; exit 1; }
session="PAIR_TAG PAIR_DATA_DIR PAIR_SCOPE_KEY PAIR_SESSION_ID PAIR_LOG_PATH PAIR_HOME PAIR_RETENTION_PROTOCOL COUCH_PAIR_DATA_DIR COUCH_THREAD_TAG ZELLIJ ZELLIJ_SESSION_NAME"

args=()
for k in $knobs; do args+=("$k=knob-$k"); done
for s in $session; do args+=("$s=session-$s"); done
out=$(env "${args[@]}" make -s --no-print-directory test-make-env-probe)

fail=0
for k in $knobs; do
  grep -qx "$k=knob-$k" <<<"$out" || { echo "FAIL: knob $k did not reach the test recipe"; fail=1; }
done
for s in $session; do
  grep -q "^$s=" <<<"$out" && { echo "FAIL: session variable $s leaked into the test recipe"; fail=1; }
done
[ "$fail" -eq 0 ] && echo "PASS make-test-env ($(wc -w <<<"$knobs" | tr -d ' ') derived knobs kept, session identity dropped)"
# Exercise the real Make defaults through a nested Go command, without builds.
python3 - <<'PYFLAGS' || fail=1
import os, pathlib, subprocess, tempfile
with tempfile.TemporaryDirectory(prefix="pair-go-flags-") as tmp:
    mk = pathlib.Path(tmp) / "flags.mk"
    mk.write_text("include Makefile.local\n.PHONY: test-flags test-nested\ntest-flags:\n\t@go env GOFLAGS\ntest-nested:\n\t@$(MAKE) -s --no-print-directory -f " + str(mk) + " test-flags\n")
    base = dict(os.environ, GOFLAGS="-p=2", PAIR_GO_TRIMPATH="-trimpath", MAKEFLAGS="", MFLAGS="")
    cases = [([], "-p=2 -trimpath"), (["GOFLAGS=-p=1"], "-p=1 -trimpath"),
             (["GOFLAGS=-p=1 -trimpath"], "-p=1 -trimpath"),
             (["PAIR_GO_TRIMPATH="], "-p=2")]
    for args, want in cases:
        for target in ["test-flags", "test-nested"]:
            out = subprocess.check_output(["make", "-s", "--no-print-directory", "-f", str(mk), target, *args], env=base, text=True).strip()
            assert out == want, (target, args, out, want)
print("PASS additive Go flags, command-line overrides and nested Make")
PYFLAGS
exit "$fail"
