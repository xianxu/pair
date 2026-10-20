#!/usr/bin/env python3
"""Execute sdlc's already-planned selection without widening Go coverage."""
from dataclasses import dataclass
import os
import re
import shlex
import subprocess
import sys

GROUPS = frozenset({"lua", "review", "doctor", "build", "shell"})
PACKAGE = re.compile(r"github\.com/xianxu/pair(?:/[A-Za-z0-9_][A-Za-z0-9_.-]*)+")
RUNTIME = "github.com/xianxu/pair/cmd/internal/runtimebundle"


@dataclass(frozen=True)
class TestSelection:
    packages: tuple[str, ...]
    groups: tuple[str, ...]


def parse_selection(package_text: str, group_text: str) -> TestSelection:
    packages, groups = tuple(package_text.split()), tuple(group_text.split())
    if not packages and not groups:
        raise ValueError("selection is empty (sdlc handles docs-only changes)")
    if len(set(packages)) != len(packages) or len(set(groups)) != len(groups):
        raise ValueError("duplicate package or group")
    if any(not PACKAGE.fullmatch(package) or "..." in package for package in packages):
        raise ValueError("expected exact Pair Go import paths")
    if any(group not in GROUPS for group in groups):
        raise ValueError("unknown test group")
    return TestSelection(packages, groups)


def go_test_argv(selection: TestSelection) -> list[str]:
    packages = list(selection.packages)
    if "lua" in selection.groups and RUNTIME not in packages:
        packages.append(RUNTIME)
    return ["go", "test", *packages, "-count=1", "-timeout", "20m"] if packages else []


def run_selection(selection: TestSelection, make_argv: list[str]) -> None:
    # This guard scans source text across import boundaries. Build owns the
    # same recipe; every other affected selection needs it once up front.
    if "build" not in selection.groups:
        subprocess.run([*make_argv, "test-make-env"], check=True)
    argv = go_test_argv(selection)
    if argv:
        subprocess.run([*make_argv, "runtimebundle-generate"], check=True)
        subprocess.run(argv, check=True)
    if selection.groups:
        subprocess.run(["bash", "tests/lib/with-pair-test-binary.sh", *make_argv,
                        *("test-affected-" + group for group in selection.groups)], check=True)


def main() -> int:
    try:
        selection = parse_selection(os.environ.get("SDLC_AFFECTED_GO_PKGS", ""),
                                    os.environ.get("SDLC_AFFECTED_GROUPS", ""))
    except ValueError as error:
        print(f"invalid affected selection: {error}", file=sys.stderr)
        return 2
    try:
        run_selection(selection, shlex.split(os.environ.get("MAKE", "make")))
    except subprocess.CalledProcessError as error:
        return error.returncode if error.returncode > 0 else 128 - error.returncode
    return 0


if __name__ == "__main__":
    sys.exit(main())
