---
id: 000408
status: open
deps: []
github_issue:
created: 2026-10-07
updated: 2026-10-07
estimate_hours:
card_mirror: '527c62fe5f5cc01c2bd55294fce3c8fcc17591ec' # card fields mirrored from issue-cards; edit via sdlc
---

# Contribute UTF-8 parser fix evidence upstream

## Problem

#379 isolated and repaired Claude completion-notification text leaking into the focused nvim pane. The dependency parser treated UTF-8 continuation byte 0x9c inside ✻ (E2 9C BB) as the C1 string terminator. Pair now carries a narrow corrected streaming parser in `third_party/vt/ansiparser`, shipped through https://github.com/xianxu/pair/pull/211. Contributing the evidence and regression coverage upstream could eventually remove this maintenance burden.

## Spec

Deferred follow-up requested by the operator; leave open and unclaimed until there is time. Start by rechecking upstream state, then compare our repair and tests with existing work before choosing a contribution. Prefer strengthening an existing issue/PR over creating a duplicate competing fix.

Upstream references (last checked 2026-10-07):

- https://github.com/charmbracelet/x/issues/848 — exact UTF-8 control-string bug, open at that check.
- https://github.com/charmbracelet/x/pull/946 and https://github.com/charmbracelet/x/pull/976 — proposed fixes, unmerged at that check.
- https://github.com/charmbracelet/x/pull/886 — closed unmerged; dropped standalone C1 ST rather than distinguishing continuation bytes.

Useful contribution: a minimal synthetic Claude notification reproducer, confirmed real-world impact, and any missing coverage for OSC/DCS/SOS/PM/APC, split input, malformed UTF-8 prefixes, cancellation/reset, standalone C1 termination and bounded payload buffers. Preserve the permanent integration regression when changing dependencies. Share only synthetic fixtures and necessary technical summaries; private session captures and conversation content stay local. Filing this issue does not send an upstream message or start that work.

## Done when

- Current upstream implementation and existing proposals have been compared with Pair's repair and regression suite, with gaps recorded.
- An appropriate contribution is submitted under the operator's authorization when this task resumes, or a reasoned no-contribution outcome is recorded if upstream already resolves the problem.
- The result links to upstream evidence and records the criteria for retiring the local parser fork; future dependency updates must pass the retained Endpoint and observer regressions.

## Plan

- [ ] Recheck upstream issue/PR/release status and compare behavior and tests.
- [ ] Prepare a minimal synthetic reproducer and reusable missing tests; choose existing PR feedback, a test contribution, or a separate PR only if justified.
- [ ] Submit the agreed contribution when resumed and record links/outcome.
- [ ] Record the local-fork retirement path and dependency-update verification requirements.

## Log

### 2026-10-07

- Operator requested a separate deferred issue after #379 shipped; no upstream communication performed.
- Starting points: `third_party/vt/ansiparser/README.md`, `string_utf8.go`, `string_utf8_test.go`; `cmd/internal/terminal/notification_mapping_test.go` (TestEndpointClaudeFooterNotificationDoesNotPaintAtCursor); `cmd/internal/wrapcmd/parser_unicode_test.go`.
- #379 verification established red against the original dependency and green with the repair, plus a 23,025-feed captured replay without right-pane leakage. The regression fixture can be reconstructed entirely from the synthetic notification already in the committed test.
