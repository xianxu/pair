#!/usr/bin/env python3
"""Canonical native replies for private Zellij test doubles, never production IO."""
import argparse
import json
from pathlib import Path
import sys


def render(kind, payload=None):
    """Serialize semantic fixture data; only explicit faults bypass the schema."""
    stdout, stderr, code = "", "", 0
    if kind == "panes":
        rows = []
        for source in payload or []:
            if "id" not in source:
                raise ValueError("fixture pane requires an explicit identity")
            row = dict(is_plugin=False, is_focused=False, is_floating=False,
                       is_fullscreen=False, tab_id=2, tab_position=1,
                       pane_x=0, pane_y=0, pane_columns=80, pane_rows=24,
                       terminal_command="", pane_command="/bin/sh", title="", cwd="/fixture")
            row.update(source)
            rows.append(row)
        stdout = json.dumps(rows, ensure_ascii=False) + "\n"
    elif kind == "clients":
        stdout = "CLIENT_ID ZELLIJ_PANE_ID RUNNING_COMMAND\n"
        for row in payload or []:
            stdout += f'{row["id"]} {row["pane_id"]} {row["running_command"]}\n'
    elif kind == "tab":
        tab = dict(tab_id=2, position=1, active_swap_layout_name="small",
                   is_swap_layout_dirty=False, is_fullscreen_active=False,
                   are_floating_panes_visible=False)
        tab.update(payload or {})
        stdout = json.dumps(tab, ensure_ascii=False) + "\n"
    elif kind in {"sessions-short", "sessions-full"}:
        rows = payload or []
        for value in rows:
            row = {"name": value} if isinstance(value, str) else value
            stdout += row["name"]
            if kind == "sessions-full":
                stdout += " [Created 1s ago]"
                if row.get("exited"):
                    stdout += " (EXITED - attach to resurrect)"
            stdout += "\n"
        if not rows:
            return render("no-sessions")
    elif kind == "no-sessions":
        stderr, code = "No active zellij sessions found.\n", 1
    elif kind == "no-session":
        stderr, code = "There is no active session!\n", 1
    elif kind == "not-found":
        stderr = f'Session \'{payload["name"]}\' not found. The following sessions are active:\n'
        stderr += "".join(name + "\n" for name in payload["active"])
        code = 1
    elif kind == "name-invalid":
        stderr = (f'error: invalid value \'{payload["name"]}\' for \'--session <SESSION>\': '
                  f'session name must be less than {payload["limit"]} characters\n\n'
                  "For more information, try '--help'.\n")
        code = 2
    elif kind == "visibility":
        if not isinstance(payload, bool):
            raise ValueError("visibility requires an explicit boolean")
        stdout = str(payload).lower() + "\n"
        code = 0 if payload else 1
    elif kind == "fault":
        if not isinstance(payload, dict) or not {"stdout", "stderr", "code"} <= payload.keys():
            raise ValueError("fault requires explicit stdout, stderr and code")
        stdout, stderr, code = payload["stdout"], payload["stderr"], payload["code"]
    else:
        raise ValueError("unknown fixture reply kind: " + kind)
    return dict(stdout=stdout, stderr=stderr, code=code)


def emit(kind, payload=None):
    result = render(kind, payload)
    sys.stdout.write(result["stdout"])
    sys.stderr.write(result["stderr"])
    raise SystemExit(result["code"])


def catalog():
    entries = []
    names = set()
    for path in sorted(Path(__file__).parent.glob("zellij_fixture_*_cases.json")):
        for entry in json.loads(path.read_text()):
            if entry["name"] in names:
                raise ValueError("duplicate fixture case: " + entry["name"])
            names.add(entry["name"])
            entries.append(entry)
    return entries


def case_payload(name):
    entries = [entry for entry in catalog() if entry["name"] == name]
    if len(entries) != 1:
        raise ValueError("unknown fixture case: " + name)
    return entries[0]["payload"]


def emit_case(name):
    entries = [entry for entry in catalog() if entry["name"] == name]
    if len(entries) != 1:
        raise ValueError("unknown fixture case: " + name)
    emit(entries[0]["kind"], entries[0].get("payload"))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--kind")
    parser.add_argument("--data", default="null")
    parser.add_argument("--case")
    parser.add_argument("--catalog", action="store_true")
    parser.add_argument("--envelope", action="store_true")
    args = parser.parse_args()
    if args.catalog:
        print(json.dumps([dict(entry, reply=render(entry["kind"], entry.get("payload")))
                          for entry in catalog()], ensure_ascii=False))
        return 0
    if args.case:
        found = [entry for entry in catalog() if entry["name"] == args.case]
        if len(found) != 1:
            parser.error("unknown fixture case " + args.case)
        kind, data = found[0]["kind"], found[0].get("payload")
    else:
        if not args.kind:
            parser.error("--kind or --case is required")
        kind, data = args.kind, json.loads(args.data)
    result = render(kind, data)
    if args.envelope:
        print(json.dumps(result, ensure_ascii=False))
        return 0
    sys.stdout.write(result["stdout"])
    sys.stderr.write(result["stderr"])
    return result["code"]


if __name__ == "__main__":
    sys.exit(main())
