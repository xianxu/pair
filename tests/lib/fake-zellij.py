#!/usr/bin/env python3
"""Private stateful Zellij CLI used by subprocess integration tests."""

import fcntl
import json
import os
import sys
import tempfile


def option(args, name, default=None):
    if name in args:
        index = args.index(name)
        if index + 1 >= len(args):
            raise ValueError("missing " + name + " value")
        return args[index + 1]
    return default


def identity(pane):
    return ("plugin_" if pane["is_plugin"] else "terminal_") + str(pane["id"])


def target_pane(session, target):
    if target is None:
        matches = [p for p in session.get("panes") or [] if p.get("is_focused")]
    else:
        target = str(target)
        if not target.startswith(("plugin_", "terminal_")):
            target = "terminal_" + target
        matches = [p for p in session.get("panes") or [] if identity(p) == target]
    if len(matches) != 1:
        raise ValueError("pane target missing or ambiguous")
    return matches[0]


def dispatch(state, args, query):
    model = state["model"]
    sessions = model["sessions"]
    if "list-sessions" in args or "ls" in args:
        return "".join(name + ("" if "--short" in args or "-s" in args else " [Created 1s ago]") + "\n" for name in sorted(sessions))
    session_name = query["session"]
    if session_name not in sessions:
        raise ValueError("session does not exist: " + session_name)
    session = sessions[session_name]
    if "action" not in args:
        # Attach is observable, but does not invent an OS process identity.
        if "attach" in args or "a" in args:
            state["effects"].append(dict(query, action="attach", target=session_name, is_plugin=False))
            return ""
        raise ValueError("unsupported CLI request")
    action_index = args.index("action") + 1
    action, params = args[action_index], args[action_index + 1:]
    if action == "list-clients":
        return "CLIENT_ID ZELLIJ_PANE_ID RUNNING_COMMAND\n" + "".join(
            str(c["id"]) + " " + c["pane_id"] + " " + c["running_command"] + "\n"
            for c in session.get("clients") or [])
    if action == "list-panes":
        return json.dumps(session.get("panes") or []) + "\n"
    if action == "current-tab-info":
        return json.dumps(session["tab"]) + "\n"
    if action == "are-floating-panes-visible":
        return str(session.get("floating_visible", False)).lower() + "\n"

    supported = {"focus-pane-id", "write-chars", "write", "toggle-fullscreen",
                 "toggle-pane-embed-or-floating", "change-floating-pane-coordinates",
                 "show-floating-panes", "hide-floating-panes", "toggle-floating-panes",
                 "next-swap-layout", "previous-swap-layout", "clear", "close-pane", "set-pane-color"}
    if action not in supported:
        raise ValueError("unsupported action: " + action)
    pane = None
    if action in {"focus-pane-id", "write-chars", "write", "toggle-fullscreen",
                  "toggle-pane-embed-or-floating", "change-floating-pane-coordinates", "clear", "close-pane", "set-pane-color"}:
        pane = target_pane(session, params[0] if action == "focus-pane-id" else option(params, "--pane-id"))
    requested_tab = option(params, "--tab-id")
    if requested_tab is not None and requested_tab != str(session["tab"]["tab_id"]):
        raise ValueError("requested tab is absent")
    target = identity(pane) if pane is not None else "tab_" + str(session["tab"]["tab_id"])
    if action == "set-pane-color":
        pane["color"] = None if "--reset" in params else option(params, "--bg")
    elif action == "focus-pane-id":
        for other in session.get("panes") or []:
            other["is_focused"] = other is pane
        for client in session.get("clients") or []:
            client["pane_id"] = target
    elif action in {"write-chars", "write"}:
        payload = params[:]
        if "--pane-id" in payload:
            index = payload.index("--pane-id")
            del payload[index:index + 2]
        text = " ".join(payload) if action == "write-chars" else "".join(chr(int(n)) for n in payload)
        pane["input"] = pane.get("input", "") + text
    elif action == "toggle-fullscreen":
        pane["is_fullscreen"] = not pane.get("is_fullscreen", False)
        session["tab"]["is_fullscreen_active"] = pane["is_fullscreen"]
    elif action == "toggle-pane-embed-or-floating":
        pane["is_floating"] = not pane.get("is_floating", False)
    elif action == "change-floating-pane-coordinates":
        for flag, key in (("-x", "pane_x"), ("-y", "pane_y"), ("--width", "pane_columns"), ("--height", "pane_rows"), ("--pinned", "is_pinned")):
            value = option(params, flag)
            if value is not None:
                # Percent coordinates are retained separately; fake does not claim native geometry conformance.
                pane[key if value.isdecimal() or flag == "--pinned" else key + "_requested"] = (value == "true") if flag == "--pinned" else int(value) if value.isdecimal() else value
    elif action in {"show-floating-panes", "hide-floating-panes", "toggle-floating-panes"}:
        visible = session.get("floating_visible", False)
        if (action == "show-floating-panes" and visible) or (action == "hide-floating-panes" and not visible):
            raise ValueError("floating visibility already set")
        session["floating_visible"] = not visible if action == "toggle-floating-panes" else action == "show-floating-panes"
        session["tab"]["are_floating_panes_visible"] = session["floating_visible"]
    elif action in {"next-swap-layout", "previous-swap-layout"}:
        layouts = session.get("layouts") or [session["tab"].get("active_swap_layout_name")]
        current = session["tab"].get("active_swap_layout_name")
        index = layouts.index(current) if current in layouts else 0
        session["tab"]["active_swap_layout_name"] = layouts[(index + (1 if action == "next-swap-layout" else -1)) % len(layouts)]
        session["tab"]["is_swap_layout_dirty"] = False
    elif action == "clear":
        pane["input"] = ""
    elif action == "close-pane":
        session["panes"].remove(pane)
    state["effects"].append(dict(query, action=action, target=target, is_plugin=pane["is_plugin"] if pane is not None else False))
    return ""


def main():
    path = os.environ["PAIR_FAKE_ZELLIJ_STATE"]
    args = sys.argv[1:]
    output, error = "", None
    with open(os.path.join(os.path.dirname(path), "state.lock"), "a") as lock:
        fcntl.flock(lock, fcntl.LOCK_EX)
        with open(path) as stream:
            state = json.load(stream)
        state["ordinal"] = state.get("ordinal", 0) + 1
        query = dict(ordinal=state["ordinal"], args=args, session=option(args, "--session", option(args, "-s", os.environ.get("ZELLIJ_SESSION_NAME", "test"))))
        state.setdefault("queries", [])
        state.setdefault("effects", [])
        state["queries"] = state["queries"] or []
        state["effects"] = state["effects"] or []
        state["queries"].append(query)
        faults = [f for f in state.get("faults") or [] if f["ordinal"] == state["ordinal"]]
        try:
            fault = faults[0] if faults else {}
            if "model" in fault:
                state["model"] = fault["model"]
            if fault.get("error"):
                raise ValueError(fault["error"])
            output = fault["reply"] if "reply" in fault else dispatch(state, args, query)
        except (ValueError, KeyError, IndexError) as exc:
            error = str(exc)
        fd, temporary = tempfile.mkstemp(prefix="state-", dir=os.path.dirname(path))
        try:
            with os.fdopen(fd, "w") as stream:
                json.dump(state, stream)
            os.replace(temporary, path)
        finally:
            if os.path.exists(temporary):
                os.unlink(temporary)
    if error is not None:
        print(error, file=sys.stderr)
        return 2
    sys.stdout.write(output)
    return 0


if __name__ == "__main__":
    sys.exit(main())
