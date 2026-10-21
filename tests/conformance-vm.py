"""Run isolated VM conformance and validate native Go evidence in one pass."""
import json
import math


def _unique_object(pairs):
    value = {}
    for key, member in pairs:
        if key in value:
            raise ValueError(f"duplicate JSON member: {key}")
        value[key] = member
    return value


def _reject_constant(value):
    raise ValueError(f"non-JSON numeric constant: {value}")


def validate_results(lines, required_by_package):
    """Return ``{'events': count, 'passed': {package: [test, ...]}}``.

    Each input line is decoded once; required names bind to exact Go import
    paths. All observed tests, including additional tests, must finish. Unknown
    additive JSON members are accepted without changing their decoded values.
    Build diagnostics may name dependency import paths; they never prove test
    execution. Failure and skip events always refuse the entire population.
    """
    if not isinstance(required_by_package, dict) or not required_by_package:
        raise ValueError("required package/case mapping must be nonempty")
    required = {}
    for package, names in required_by_package.items():
        if not isinstance(package, str) or not package or isinstance(names, str):
            raise ValueError("invalid required package/case mapping")
        try:
            cases = set(names)
        except (TypeError, ValueError) as exc:
            raise ValueError("invalid required cases") from exc
        if not cases or any(not isinstance(name, str) or not name for name in cases):
            raise ValueError("required cases must be nonempty strings")
        required[package] = cases
    packages = {}
    count = 0
    string_fields = ("Time", "Action", "Package", "Test", "Output", "OutputType",
                     "FailedBuild", "ImportPath", "Path", "Key", "Value")
    for line in lines:
        count += 1
        event = json.loads(line, object_pairs_hook=_unique_object,
                           parse_constant=_reject_constant)
        if not isinstance(event, dict):
            raise ValueError(f"event {count}: expected JSON object")
        for key in string_fields:
            if key in event and not isinstance(event[key], str):
                raise ValueError(f"event {count}: {key} must be a string")
        if "Elapsed" in event:
            elapsed = event["Elapsed"]
            try:
                finite = type(elapsed) in (int, float) and math.isfinite(elapsed)
            except OverflowError:
                finite = False
            if not finite or elapsed < 0:
                raise ValueError(f"event {count}: invalid Elapsed")
        action = event.get("Action")
        if action in ("fail", "skip", "build-fail"):
            raise RuntimeError(f"conformance {action}: {event.get('Test') or event.get('Package') or event.get('ImportPath')}")
        if action == "build-output":
            if (not event.get("ImportPath") or "Output" not in event
                    or "Package" in event or "Test" in event):
                raise ValueError(f"event {count}: malformed build output")
            continue
        if action not in ("start", "run", "pause", "cont", "pass", "output", "bench", "artifacts", "attr"):
            raise ValueError(f"event {count}: unsupported Action {action!r}")
        package, name = event.get("Package"), event.get("Test")
        if package not in required:
            raise RuntimeError(f"event {count}: unexpected package {package!r}")
        if "Test" in event and not name:
            raise ValueError(f"event {count}: empty Test")
        if action == "start":
            if name or package in packages:
                raise RuntimeError(f"event {count}: duplicate or malformed package start")
            packages[package] = {"done": False, "tests": {}, "passed": []}
            continue
        state = packages.get(package)
        if state is None or state["done"]:
            raise RuntimeError(f"event {count}: package not running: {package}")
        tests = state["tests"]
        if action == "output":
            if "Output" not in event or (name and name not in tests):
                raise RuntimeError(f"event {count}: output without observed execution")
            continue
        if action == "pass" and not name:
            if any(status != "pass" for status in tests.values()):
                raise RuntimeError(f"event {count}: package passed with unfinished tests")
            state["done"] = True
            continue
        if not name:
            raise RuntimeError(f"event {count}: {action} requires Test")
        status = tests.get(name)
        if action in ("artifacts", "attr"):
            # test2json emits these from ArtifactDir/Attr. They carry metadata,
            # never execution/completion authority. Empty attr strings are
            # omitted by Go's omitempty codec, so absence means empty here.
            if status not in ("run", "pause") or (action == "artifacts" and not event.get("Path")):
                raise RuntimeError(f"event {count}: metadata without running test or artifact path")
        elif action == "run" and status is None:
            tests[name] = "run"
        elif action == "pause" and status == "run":
            tests[name] = "pause"
        elif action == "cont" and status == "pause":
            tests[name] = "run"
        elif action == "pass" and status == "run":
            tests[name] = "pass"
            state["passed"].append(name)
        elif action == "bench" and status == "run":
            # Benchmark observations do not count as required case completion.
            pass
        else:
            raise RuntimeError(f"event {count}: invalid {action} for {name} ({status})")
    for package, cases in required.items():
        state = packages.get(package)
        if state is None or not state["done"]:
            raise RuntimeError(f"incomplete package execution: {package}")
        missing = cases.difference(state["passed"])
        if missing:
            raise RuntimeError(f"conformance did not execute {package}: {list(missing)}")
    return {"events": count, "passed": {p: state["passed"] for p, state in packages.items()}}


NATIVE_REQUIRED = {
    'github.com/xianxu/pair/cmd/internal/couchtty': {
        'TestNativeConsoleWrapperZellij',
        'TestNativeConsoleWrapperZellij/direct-zellij-baseline',
        'TestNativeConsoleWrapperZellij/wrapped-zellij',
        'TestLiveConsoleNvimPreservesContentAndChrome',
        'TestLiveReservedRowSurvivesRealScrolling',
        'TestNotificationPTYConformance',
    },
    'github.com/xianxu/pair/cmd/internal/layoutcmd': {'TestTerminalRecoveryZellijConformance'},
}
LIFECYCLE_REQUIRED = {
    'github.com/xianxu/pair/cmd/internal/launcher': {
        'TestSessionQuiescenceLive', 'TestSessionDetachLive',
        'TestQuitLifecycleLive', 'TestTheAcceptorAgreesWithRealZellijLive',
    },
    'github.com/xianxu/pair/cmd/internal/couchcore': {
        'TestParkLifecycleLiveIntentOnlyMutation',
        'TestRecoveryRealHelperAndSessionConformanceLive',
        'TestRecoveryRealHelperAndSessionConformanceLive/warm',
    },
    'github.com/xianxu/pair/cmd/internal/couchcmd': {
        'TestAgentShortcutInputConformanceLive',
        *('TestAgentShortcutInputConformanceLive/' + name for name in (
            'inherited-alt-f', 'inherited-alt-bracket', 'inherited-alt-plus',
            'alt-up', 'alt-down', 'alt-left', 'alt-right', 'alt-h', 'alt-l', 'alt-x',
            'reserved-shift-alt-t', 'reserved-shift-alt-left', 'reserved-shift-alt-right')),
    },
}


def host_main(root, binary, env):
    """Invoke the generic runner once; it owns leases, artifacts and cleanup."""
    import os
    from pathlib import Path
    import subprocess
    command = [str(binary), 'vm', 'run', '--checkout', str(root), '--lane', 'train', '--timeout', '20m']
    artifacts = env.get('PAIR_VM_ARTIFACTS_DIR')
    if artifacts:
        if not Path(artifacts).is_absolute() or os.path.normpath(artifacts) != artifacts:
            raise ValueError('PAIR_VM_ARTIFACTS_DIR must be an absolute clean fresh path')
        command += ['--artifacts', artifacts]
    command += ['--', 'python3', 'tests/conformance-vm.py', 'guest']
    status = subprocess.run(command, env=env, check=False).returncode
    return status if status >= 0 else 128 - status


def guest_main():
    """Qualify both existing populations inside the runner's private guest."""
    import importlib.util
    import os
    from pathlib import Path
    import tempfile
    import subprocess
    import sys
    root = Path(__file__).resolve().parents[1]
    os.chdir(root)
    # Ignored generated runtime assets are deliberately absent from the snapshot.
    subprocess.run(["make", "runtimebundle-generate"], check=True, stdout=sys.stderr, timeout=180)
    # Reuse the native target's setup, candidate and strict streaming checker.
    spec = importlib.util.spec_from_file_location('native_ci', root / 'tests/native-terminal-ci.py')
    ci = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(ci)
    ci.main()
    names = sorted({name for cases in LIFECYCLE_REQUIRED.values() for name in cases if '/' not in name})
    env = dict(os.environ, PAIR_LIVE_COUCH='1')
    with tempfile.TemporaryDirectory(prefix='pair-lifecycle-ci-') as scratch:
        command = ['go', 'test', '-p', '1', '-json', *('./' + p.removeprefix('github.com/xianxu/pair/') for p in LIFECYCLE_REQUIRED),
                   '-run', '^(' + '|'.join(names) + ')$',
                   '-skip', '^TestRecoveryRealHelperAndSessionConformanceLive$/^checkpoint$',
                   '-count=1', '-timeout=5m', '-artifacts', '-outputdir', scratch]
        ci.qualify(command, env, Path(scratch), required=LIFECYCLE_REQUIRED)
    return 0


def main():
    import os
    from pathlib import Path
    import sys
    root = Path(__file__).resolve().parents[1]
    if sys.argv[1:] == ['guest']:
        return guest_main()
    if sys.argv[1:] == ['host']:
        return host_main(root, root / 'bin/pair', os.environ.copy())
    print('Usage: conformance-vm.py host|guest', file=sys.stderr)
    return 2


if __name__ == '__main__':
    raise SystemExit(main())
