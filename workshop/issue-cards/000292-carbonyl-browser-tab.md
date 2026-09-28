---
id: 000292
status: working
started: 2026-09-19T11:10:35-07:00
created: 2026-09-19
updated: 2026-09-19
estimate_hours: 9.46
github_issue:
---

# Carbonyl browser tab in the right pane, shared with the agent over DevTools

## Problem

While developing, the operator and the agent both need to look at a web page
(a local dev server, say `localhost:1111`), and today that means leaving the
terminal. Carbonyl is Chromium rendered into terminal cells, so it can run as an
ordinary child in a right-pane tab. The operator tried it and it works well
enough for a first version. Its Chromium can also be shared with the agent, so
the human and the agent operate the *same* browser during development and
testing.

Verified 2026-09-19:
- **Version:** Carbonyl 0.0.2 (`/opt/homebrew/bin/carbonyl`) bundles Chrome 111.
- **Options:** `--fps` (default 60) and `--zoom`, and it "supports most Chromium
  options".
- **DevTools:** launched with `--remote-debugging-port=0 --user-data-dir=<dir>`,
  it writes `<dir>/DevToolsActivePort`. The DevTools protocol answers there
  (`"Browser": "Google Chrome/111.0.5511.1 (Carbonyl)"`), and `/json/list`
  returns the page's live `url` and `title`.
