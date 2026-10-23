"""Run native conformance with a fresh candidate and require actual execution.

Candidate and JSON evidence are invocation-owned, on both macOS and Linux.
Native CI currently qualifies macOS; this runner makes no Linux pass claim.
"""
import importlib.util
import json
import os
from pathlib import Path
import shutil
import signal
import subprocess
import tempfile
import sys

_validator_spec = importlib.util.spec_from_file_location("conformance_vm", Path(__file__).with_name("conformance-vm.py"))
_validator = importlib.util.module_from_spec(_validator_spec)
_validator_spec.loader.exec_module(_validator)
REQUIRED_BY_PACKAGE = _validator.NATIVE_REQUIRED
REQUIRED = set().union(*REQUIRED_BY_PACKAGE.values())


def require_execution(events):
    # Compatibility for callers with decoded fixtures. Production streams below
    # feed the shared reducer directly and parse each event exactly once.
    return _validator.validate_results((json.dumps(event) for event in events), REQUIRED_BY_PACKAGE)


def run_tests(command, env, evidence, required=None):
    # The child process group must join before invocation scratch is removed,
    # including malformed output, output errors, and operator interruption.
    with evidence.open("w") as output:
        process = subprocess.Popen(command, env=env, stdout=subprocess.PIPE, text=True, start_new_session=True)
        try:
            def lines():
                for line in process.stdout:
                    output.write(line)
                    # Raw Go JSON is retained both here and by the VM stdout
                    # stream; human/tool diagnostics use stderr separately.
                    print(line, end="", flush=True)
                    yield line
            _validator.validate_results(lines(), REQUIRED_BY_PACKAGE if required is None else required)
            return process.wait(timeout=10)
        finally:
            if process.poll() is None:
                os.killpg(process.pid, signal.SIGTERM)
                try:
                    process.wait(timeout=3)
                except subprocess.TimeoutExpired:
                    os.killpg(process.pid, signal.SIGKILL)
                    process.wait()
            process.stdout.close()


def qualify(command, env, scratch, required=None):
    evidence = scratch / "events.jsonl"
    try:
        status = run_tests(command, env, evidence, required)
        if status:
            raise RuntimeError(f"native go test exited {status}")
    except BaseException:
        # Go preserves -artifacts here; publish before owned scratch is deleted.
        # These are only the oracle's known evidence files, never the private HOME.
        for path in sorted((scratch / "artifacts").rglob("node-oracle-*/*")):
            if path.name in {"request.json", "stdout.raw", "stderr.txt", "node-report.json"} and path.is_file():
                print(f"\n--- oracle artifact {path.relative_to(scratch)} ({path.stat().st_size} bytes) ---", file=sys.stderr, flush=True)
                print(path.read_bytes().decode("utf-8", errors="backslashreplace"), file=sys.stderr, flush=True)
        raise


def main():
    root = Path(__file__).resolve().parent.parent
    os.chdir(root)
    for command in ("go", "zellij", "nvim", "python3", "node", "npm", "tic", "infocmp"):
        if not shutil.which(command):
            raise RuntimeError(f"missing native dependency: {command}")
    version = subprocess.check_output(["zellij", "--version"], text=True, timeout=10).strip()
    if version != "zellij 0.45.1":
        raise RuntimeError(f"native baseline requires zellij 0.45.1, got {version!r}")
    print(f"Native baseline: {version}", file=sys.stderr, flush=True)
    subprocess.run(["npm", "ci", "--prefix", "tests/terminal-oracle", "--ignore-scripts", "--no-audit", "--no-fund"], check=True, stdout=sys.stderr, timeout=120)
    subprocess.run(["node", "--test", "tests/terminal-oracle/driver_test.cjs"], check=True, stdout=sys.stderr, timeout=30)
    with tempfile.TemporaryDirectory(prefix="pair-native-ci-") as scratch:
        candidate = str(Path(scratch) / "pair")
        # Native fixtures inherit the same empty allowlist as every Pair test.
        helper = root / "tests/lib/pair-test-env.sh"
        caches = [key + "=" + subprocess.check_output(["go", "env", key], text=True).strip()
                  for key in ("GOCACHE", "GOMODCACHE")]
        raw = subprocess.check_output(["/bin/sh", str(helper), "--print0", scratch, *caches],
                                      env={"PATH": os.environ.get("PATH", "/usr/bin:/bin")})
        build_env = dict(entry.decode().split("=", 1) for entry in raw.split(b"\0") if entry)
        subprocess.run(["bash", str(root / "tests/lib/pair-test-binary.sh"), candidate],
                       env=build_env, check=True, timeout=180)
        env = dict(build_env, PAIR_LIVE_COUCH="1", PAIR_LIVE_COUCH_NATIVE="1", PAIR_LIVE_ZELLIJ="1", PAIR_NATIVE_BINARY=candidate)
        pattern = _validator.test_pattern(REQUIRED_BY_PACKAGE)
        artifacts = Path(scratch) / "artifacts"
        artifacts.mkdir()
        command = ["go", "test", "-p", "1", "-race", "-json", "./cmd/internal/couchtty", "./cmd/internal/layoutcmd", "-run", pattern, "-count=1", "-timeout=5m", "-artifacts", "-outputdir", str(artifacts)]
        qualify(command, env, Path(scratch))
        print(f"Verified execution of all {len(REQUIRED)} required native tests/subtests.", file=sys.stderr)


if __name__ == "__main__":
    main()
