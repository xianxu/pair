"""Run native conformance with a fresh candidate and require actual execution.

Candidate and JSON evidence are invocation-owned, on both macOS and Linux.
Native CI currently qualifies macOS; this runner makes no Linux pass claim.
"""
import json
import os
from pathlib import Path
import shutil
import signal
import subprocess
import tempfile

REQUIRED = {
    "TestNativeConsoleWrapperZellij",
    "TestNativeConsoleWrapperZellij/direct-zellij-baseline",
    "TestNativeConsoleWrapperZellij/wrapped-zellij",
    "TestLiveConsoleNvimPreservesContentAndChrome",
    "TestLiveReservedRowSurvivesRealScrolling",
    "TestNotificationPTYConformance",
    "TestTerminalRecoveryZellijConformance",
}


def require_execution(events):
    passed = set()
    for event in events:
        action, name = event.get("Action"), event.get("Test")
        if action in ("skip", "fail"):
            raise RuntimeError(f"native conformance {action}: {name or 'package'}")
        if action == "pass" and name:
            passed.add(name)
    missing = REQUIRED - passed
    if missing:
        raise RuntimeError(f"native conformance did not execute: {sorted(missing)}")


def run_tests(command, env, evidence):
    # The child process group must join before invocation scratch is removed,
    # including malformed output, output errors, and operator interruption.
    with evidence.open("w") as output:
        process = subprocess.Popen(command, env=env, stdout=subprocess.PIPE, text=True, start_new_session=True)
        try:
            for line in process.stdout:
                output.write(line)
                event = json.loads(line)
                if "Output" in event:
                    print(event["Output"], end="", flush=True)
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


def qualify(command, env, scratch):
    evidence = scratch / "events.jsonl"
    try:
        status = run_tests(command, env, evidence)
        if status:
            raise RuntimeError(f"native go test exited {status}")
        with evidence.open() as source:
            require_execution(json.loads(line) for line in source)
    except BaseException:
        # Go preserves -artifacts here; publish before owned scratch is deleted.
        # These are only the oracle's known evidence files, never the private HOME.
        for path in sorted((scratch / "artifacts").rglob("node-oracle-*/*")):
            if path.name in {"request.json", "stdout.raw", "stderr.txt", "node-report.json"} and path.is_file():
                print(f"\n--- oracle artifact {path.relative_to(scratch)} ({path.stat().st_size} bytes) ---", flush=True)
                print(path.read_bytes().decode("utf-8", errors="backslashreplace"), flush=True)
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
    print(f"Native baseline: {version}", flush=True)
    subprocess.run(["npm", "ci", "--prefix", "tests/terminal-oracle", "--ignore-scripts", "--no-audit", "--no-fund"], check=True, timeout=120)
    subprocess.run(["node", "--test", "tests/terminal-oracle/driver_test.cjs"], check=True, timeout=30)
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
        pattern = "^(TestNativeConsoleWrapperZellij|TestLiveConsoleNvimPreservesContentAndChrome|TestLiveReservedRowSurvivesRealScrolling|TestNotificationPTYConformance|TestTerminalRecoveryZellijConformance)$"
        artifacts = Path(scratch) / "artifacts"
        artifacts.mkdir()
        command = ["go", "test", "-p", "1", "-race", "-json", "./cmd/internal/couchtty", "./cmd/internal/layoutcmd", "-run", pattern, "-count=1", "-timeout=5m", "-artifacts", "-outputdir", str(artifacts)]
        qualify(command, env, Path(scratch))
        print(f"Verified execution of all {len(REQUIRED)} required native tests/subtests.")


if __name__ == "__main__":
    main()
