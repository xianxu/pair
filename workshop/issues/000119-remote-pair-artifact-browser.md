---
id: 000119
status: open
deps: ["#121"]
github_issue:
created: 2026-07-26
updated: 2026-07-26
estimate_hours:
---

# Remote Pair artifact browser

## Problem

Ariadne and Pair are repo-centered, but remote Pair control would initially show
only sessions. The user still needs local editor/browser access to inspect
issues, plans, targets, atlas docs, continuations, markdown previews, CSVs, SVGs,
and other local artifacts. A remote web interface can make those artifacts easier
to browse from a tablet, but it also creates the highest-risk security boundary:
local files must not become generally exposed over the internet.

## Spec

- Add a read-only artifact browser served through the local daemon and #121
  relay/auth model.
- Limit browsing to explicitly allowed repo roots in the peer-repo structure.
  Resolve and authorize every path through canonical filesystem paths; define the
  symlink policy before serving any content.
- Start with Ariadne/Pair artifact classes: `workshop/issues`, `workshop/plans`,
  `workshop/targets`, `atlas`, continuations, markdown, CSV, SVG, and plain text.
- Provide native web previews for supported formats and a safe fallback for
  unknown text files. Binary files, huge files, dotfiles/secrets, and generated
  caches are denied unless an explicit allowlist says otherwise.
- Keep writes, annotation, shell commands, and arbitrary file downloads out of
  scope for this issue.
- Keep artifact discovery as reusable pure logic over a repo root and an allowlist
  (`ARCH-PURE`), with rendering/HTTP as a thin IO layer.

## Done when

- An authenticated browser can browse allowed artifacts in the current repo and
  selected peer repos without direct filesystem access to the relay.
- Markdown, CSV, SVG, and plain text artifacts render usefully in the browser.
- Path traversal, symlink escape, oversized file, denied extension/class, and
  hidden-secret cases are rejected and tested.
- The relay stores and sees only request metadata and proxied responses required
  for transport; local authorization remains with the daemon.

## Plan

- [ ] Define repo-root allowlisting, path authorization, and symlink policy.
- [ ] Add artifact inventory/discovery for Ariadne-style repo structures.
- [ ] Add read-only content serving through the daemon/relay protocol.
- [ ] Add markdown/CSV/SVG/text web renderers.
- [ ] Add security tests for traversal, symlink, hidden, large, and denied files.

## Log

### 2026-07-26

Seeded as the artifact layer after remote control. The issue is read-only by
design; annotation/write behavior belongs in #122.
