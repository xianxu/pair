#!/usr/bin/env bash
# pair#436: a test goal drops the live session's PAIR_*/COUCH_*/ZELLIJ* identity
# and keeps every knob a caller may pass. Knobs are derived, not listed: every
# name test code reads that production never mentions, plus every name the docs
# pass to make or go test. Each must reach a test recipe.
set -euo pipefail
cd "$(dirname "$0")/.."

# Run the same assertions under real outer Make command-line precedence. The
# private fixture target prevents recursion; the temporary Makefile is removed
# even when a case fails. Neither invocation builds Go packages.
if [ "${1:-}" != --fixture ]; then
  fixture=$(mktemp "${TMPDIR:-/tmp}/pair-make-env.XXXXXX")
  trap 'rm -f "$fixture"' EXIT
  cat >"$fixture" <<'MAKEFIXTURE'
include Makefile
.PHONY: test-make-env-fixture
test-make-env-fixture:
	@bash tests/make-test-env-test.sh --fixture
MAKEFIXTURE
  for assignment in '' 'PAIR_GO_TRIMPATH=-trimpath'; do
    env -u MAKEFLAGS -u MAKEOVERRIDES -u MFLAGS make -s --no-print-directory \
      -f "$fixture" test-make-env-fixture ${assignment:+"$assignment"}
  done
  exit 0
fi

# This fixture supplies its own environment. Outer Make assignments otherwise
# outrank those synthetic values in every child make (pair#577). Clear only
# Make's inherited controls; the explicit override/recursive tests below still
# exercise real command-line precedence within their own invocations.
unset MAKEFLAGS MAKEOVERRIDES MFLAGS

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
# Lua/native recipes must resolve the candidate across the fixture's empty
# environment boundary even when an incompatible installed Pair fronts PATH.
python3 - <<'PYCANDIDATE' || fail=1
import os, pathlib, subprocess, tempfile
with tempfile.TemporaryDirectory(prefix="pair-native-candidate-", dir=os.environ.get("TMPDIR", "/tmp")) as tmp:
    root = pathlib.Path(tmp)
    for name in ("built", "shared", "installed"):
        folder = root / name
        folder.mkdir()
        binary = folder / ("plain-pair" if name == "shared" else "pair")
        binary.write_text("#!/bin/sh\nprintf '%s\\n' '" + name + "'\n")
        binary.chmod(0o700)
    (root / "shared/pair").write_text("#!/bin/sh\nprintf '%s\\n' 'wrong-sibling'\n")
    (root / "shared/pair").chmod(0o700)
    script = root / "probe.sh"
    script.write_text("#!/usr/bin/env bash\n. " + str(pathlib.Path.cwd() / "tests/lib/pair-test-bootstrap.sh") + "\npair --candidate-probe\n")
    mk = root / "probe.mk"
    mk.write_text("include Makefile.local\ntest-review-recipes test-lua-recipes:\n\t@bash " + str(script) + "\n")
    base = dict(os.environ, PATH=str(root / "installed") + os.pathsep + os.environ["PATH"],
                MAKEFLAGS="", MAKEOVERRIDES="", MFLAGS="")
    for shared in (False, True):
        base.pop("PAIR_TEST_BINARY", None)
        if shared:
            base["PAIR_TEST_BINARY"] = str(root / "shared/plain-pair")
        output = subprocess.check_output(["make", "-s", "--no-print-directory", "-f", str(mk),
            "-o", str(root / "built/pair"), "BIN_DIR=" + str(root / "built"),
            "MAKE=make -s --no-print-directory -f " + str(mk),
            "test-lua", "test-review"], env=base, text=True, stderr=subprocess.PIPE).splitlines()
        assert output == (["shared"] if shared else ["built"]) * 2, (shared, output)
print("PASS Lua/review recipes use built/shared candidate across fixture bootstrap, never installed Pair")
PYCANDIDATE
exit "$fail"
