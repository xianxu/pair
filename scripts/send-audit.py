#!/usr/bin/env python3
"""send-audit — did draft sends reach the agent intact? (pair#211)

Pairs every submitted entry in pair's send logs (the authored body, exact
bytes) with the agent transcript message that received it — Claude Code
(~/.claude/projects) and Codex (~/.codex/sessions) — and reports sends whose
arrived text is missing a run.

    scripts/send-audit.py                    # everything on disk
    scripts/send-audit.py --since 2026-09-30 # only sends after a fix shipped

pair#211 found, before this audit's fix, that Claude Code dropped whole ~1 KiB
middle reads of unbracketed sends: 8 of 11 sends spanning three reads, 0 of 31
spanning two. The fix sends the body as one bracketed paste; the verdict that
it worked is this table's 3+-read rows going to zero on sends after the fix.

What reaches the agent is not the logged body verbatim, and each difference is
normalized away rather than reported: pair strips `===` comment lines before
sending (nvim/normalization.lua); Claude Code strips trailing blanks per line
and wraps pasted runs in <pasted_content> tags.
"""
import argparse
import glob
import json
import os
import re
from datetime import datetime

LOG_ENTRY = re.compile(
    r"^## (\d{4}-\d\d-\d\d \d\d:\d\d:\d\d)\n"
    r"<!-- pair-log-v1 bytes=(\d+) [^>]*state=(\w+)[^>]*-->\n\n",
    re.M,
)
PASTE_TAG = re.compile(r"\n?</?pasted_content[^>]*>\n?")
LOSS_FLOOR = 100  # bytes; below this a difference is normalization, not a hole


def log_entries(since):
    for path in glob.glob(os.path.expanduser("~/.local/share/pair/repos/*/log-*.md")):
        text = open(path, encoding="utf-8", errors="replace").read()
        heads = list(LOG_ENTRY.finditer(text))
        for i, m in enumerate(heads):
            if m.group(3) != "submitted":
                continue
            ts = datetime.strptime(m.group(1), "%Y-%m-%d %H:%M:%S").astimezone()
            if since and ts < since:
                continue
            end = heads[i + 1].start() if i + 1 < len(heads) else len(text)
            body = re.sub(r"\n+---\n+$", "", text[m.end():end])
            yield path, ts, int(m.group(2)), body


def transcript_messages():
    """Yield (timestamp, text, agent) for every user message on disk."""
    claude = glob.glob(os.path.expanduser("~/.claude/projects/*/*.jsonl"))
    codex = glob.glob(os.path.expanduser("~/.codex/sessions/*/*/*/rollout-*.jsonl"))
    for paths, is_codex in ((claude, False), (codex, True)):
        needle = '"role":"user"' if is_codex else '"type":"user"'
        for path in paths:
            try:
                lines = open(path, encoding="utf-8", errors="replace")
            except OSError:
                continue
            for line in lines:
                if needle not in line:
                    continue
                try:
                    d = json.loads(line)
                except ValueError:
                    continue
                if "timestamp" not in d:
                    continue
                if is_codex:
                    p = d.get("payload", {})
                    if p.get("role") != "user":
                        continue
                    parts = p.get("content", [])
                    agent = "codex"
                else:
                    if d.get("type") != "user":
                        continue
                    parts = d.get("message", {}).get("content")
                    agent = "claude " + d.get("version", "?")
                if isinstance(parts, str):
                    text = parts
                else:
                    text = "".join(c.get("text", "") for c in parts or [] if isinstance(c, dict))
                if text:
                    yield datetime.fromisoformat(d["timestamp"].replace("Z", "+00:00")), text, agent


def sent_text(body):
    """What pair actually sends: the body minus `===` comment lines."""
    return "\n".join(l for l in body.split("\n") if not re.match(r"^\s*===", l))


def norm(s):
    s = PASTE_TAG.sub("\n", s.replace("\r\n", "\n"))
    s = "\n".join(l.rstrip() for l in s.split("\n"))
    return re.sub(r"\n{2,}", "\n\n", s).strip()


def main():
    ap = argparse.ArgumentParser(description=(__doc__ or "").split("\n")[0])
    ap.add_argument("--since", help="only sends at or after this local date (YYYY-MM-DD)")
    args = ap.parse_args()
    since = datetime.strptime(args.since, "%Y-%m-%d").astimezone() if args.since else None

    by_minute = {}
    for ts, text, agent in transcript_messages():
        by_minute.setdefault(int(ts.timestamp()) // 60, []).append((ts, text, agent))

    table, unmatched = {}, 0
    for path, ts, nbytes, body in sorted(log_entries(since), key=lambda e: e[1]):
        want = norm(sent_text(body))
        key = want[:40]
        if not key:
            continue
        minute = int(ts.timestamp()) // 60
        cands = [c for m in range(minute - 3, minute + 4) for c in by_minute.get(m, []) if key in c[1]]
        if not cands:
            unmatched += 1
            continue
        _, got, agent = min(cands, key=lambda c: abs((c[0] - ts).total_seconds()))
        got = norm(got)
        got = got[got.find(key):]
        reads = (nbytes + 1023) // 1024
        row = table.setdefault((agent.split()[0], min(reads, 4)), [0, 0])
        row[0] += 1
        if want in got or len(want) - len(got) < LOSS_FLOOR:
            continue
        row[1] += 1
        head = 0
        while head < min(len(got), len(want)) and got[head] == want[head]:
            head += 1
        print(f"{ts:%F %T}  {agent:14}  sent {len(want)}  got {len(got)}  "
              f"missing {len(want) - len(got)} from ~{head}  {os.path.basename(path)}")

    print(f"\n{'agent':8} {'~1 KiB reads':>12} {'sends':>6} {'lossy':>6}")
    for (agent, reads), (sends, lossy) in sorted(table.items()):
        print(f"{agent:8} {('4+' if reads == 4 else str(reads)):>12} {sends:6d} {lossy:6d}")
    print(f"unmatched log entries (no transcript found): {unmatched}")


if __name__ == "__main__":
    main()
