---
gate: boundary-review
issue: 471
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-12T03:49:56-07:00"
      dispatched: "2026-10-12T03:46:25-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: 'JS catchUp no longer matches Go CatchUp after the #458 merge: it keeps every hover and ptr record'
          detail: 'Since 9c288070, recording.CatchUp keeps only the latest hover and ptr in the latest segment (catchup.go:42-100). replay.js:16-48 keeps all of them, despite claiming to match "record for record". Checked: for head,seg,hover,ptr,hover,ptr, JS returns 6 records and Go returns 4. The drawn state is unaffected (both kinds are last-writer-wins and outside the canonical state), but TestLiveReplayServerSeeks''s index check fails on any real recording with pointer movement. Fix: port the hover/ptr selection.'
          family: held-port-parity
          round: 1
        - id: BR-2
          severity: Important
          title: The catch-up parity check never exercises hover or ptr, because no fixture contains them
          detail: None of stress/app/blog/segments/driven has hover or ptr records, so replay_live_test.go:79-91 and the node synthetic recording pass whatever the rule is. Add hover and ptr records (two of each in the last segment) to the synthetic recording in replay.test.mjs and assert the indexes kept; this would have caught the drift at merge time.
          family: vacuous-fixture-coverage
          round: 1
        - id: BR-3
          severity: Minor
          title: ListRecordings silently shows a duration of 0 when the last events line is over 64 KB
          detail: endLines reads a 64 KB tail, so a longer last line is cut and json.Unmarshal(last) fails silently (replay.go:215-240). A first line over 1 MB drops the recording from the listing without a word.
          family: silent-parse-fallback
          round: 1
        - id: BR-4
          severity: Minor
          title: resolveReplay accepts a symlinked DIR that ReplayServer then answers with 404
          detail: resolveReplay uses os.Stat, which follows links; ReplayServer.exists uses Lstat. A symlinked DIR prints a link that 404s. Refuse it in resolveReplay or resolve it with EvalSymlinks.
          family: symlink-policy-mismatch
          round: 1
        - id: BR-5
          severity: Minor
          title: The replay listing omits the segment count the plan lists
          detail: The plan says the listing shows id, started, duration, segments and notes; Listed and listTemplate have no segment count.
          family: plan-surface-drift
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - "n": 2
      timestamp: "2026-10-12T03:56:20-07:00"
      dispatched: "2026-10-12T03:54:34-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: replay.js:29-58 ports the hover/ptr selection; undoing it fails the node corpus test at cut 23.
          round: 2
        - id: BR-2
          disposition: addressed
          note: The corpus has 2 hover and 2 ptr in the last segment plus 1 of each earlier; TestCatchUpCorpus requires every special-cased kind.
          round: 2
        - id: BR-3
          disposition: addressed
          note: headAndEnd scans backwards to the start of the last line; TestListRecordingsLongLastLine (200 KB line) fails when the fix is undone.
          round: 2
        - id: BR-4
          disposition: addressed
          note: resolveReplay resolves links with EvalSymlinks; the linked case in TestResolveReplay fails when the fix is undone.
          round: 2
        - id: BR-5
          disposition: addressed
          note: 'The plan''s Revisions entry drops the segment count with a reason: counting needs a full scan; Listed matches.'
          round: 2
      findings:
        - id: BR-6
          severity: Minor
          title: The listing shows a duration of 0 when the last events line doesn't start with {"t":N
          detail: '2nd finding in silent-parse-fallback. Rule: a listing field read from a recording that can''t be parsed shows as unknown (for example "?"), never as a made-up zero. Apply it to both of headAndEnd''s reads (replay.go:240-244), which is every parse site in ListRecordings. Trigger: an Unknown record written back exactly as read (codec.go:37-38) as the last line.'
          family: silent-parse-fallback
          round: 2
      recipe: milestone-review
      isolation: fenced
      reviewed: 664ebbd789f60ddc4368343f95b1f7978e0b9e8e
      blocked: false
---

# Gate ledger — pair#471 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-12T03:49:56-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `held-port-parity` JS catchUp no longer matches Go CatchUp after the #458 merge: it keeps every hover and ptr record
  Since 9c288070, recording.CatchUp keeps only the latest hover and ptr in the latest segment (catchup.go:42-100). replay.js:16-48 keeps all of them, despite claiming to match "record for record". Checked: for head,seg,hover,ptr,hover,ptr, JS returns 6 records and Go returns 4. The drawn state is unaffected (both kinds are last-writer-wins and outside the canonical state), but TestLiveReplayServerSeeks's index check fails on any real recording with pointer movement. Fix: port the hover/ptr selection.
- **BR-2** [Important] `vacuous-fixture-coverage` The catch-up parity check never exercises hover or ptr, because no fixture contains them
  None of stress/app/blog/segments/driven has hover or ptr records, so replay_live_test.go:79-91 and the node synthetic recording pass whatever the rule is. Add hover and ptr records (two of each in the last segment) to the synthetic recording in replay.test.mjs and assert the indexes kept; this would have caught the drift at merge time.
- **BR-3** [Minor] `silent-parse-fallback` ListRecordings silently shows a duration of 0 when the last events line is over 64 KB
  endLines reads a 64 KB tail, so a longer last line is cut and json.Unmarshal(last) fails silently (replay.go:215-240). A first line over 1 MB drops the recording from the listing without a word.
- **BR-4** [Minor] `symlink-policy-mismatch` resolveReplay accepts a symlinked DIR that ReplayServer then answers with 404
  resolveReplay uses os.Stat, which follows links; ReplayServer.exists uses Lstat. A symlinked DIR prints a link that 404s. Refuse it in resolveReplay or resolve it with EvalSymlinks.
- **BR-5** [Minor] `plan-surface-drift` The replay listing omits the segment count the plan lists
  The plan says the listing shows id, started, duration, segments and notes; Listed and listTemplate have no segment count.

## Round 2 — 2026-10-12T03:56:20-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — replay.js:29-58 ports the hover/ptr selection; undoing it fails the node corpus test at cut 23.
- BR-2 — addressed — The corpus has 2 hover and 2 ptr in the last segment plus 1 of each earlier; TestCatchUpCorpus requires every special-cased kind.
- BR-3 — addressed — headAndEnd scans backwards to the start of the last line; TestListRecordingsLongLastLine (200 KB line) fails when the fix is undone.
- BR-4 — addressed — resolveReplay resolves links with EvalSymlinks; the linked case in TestResolveReplay fails when the fix is undone.
- BR-5 — addressed — The plan's Revisions entry drops the segment count with a reason: counting needs a full scan; Listed matches.

### Raised

- **BR-6** [Minor] `silent-parse-fallback` The listing shows a duration of 0 when the last events line doesn't start with {"t":N
  2nd finding in silent-parse-fallback. Rule: a listing field read from a recording that can't be parsed shows as unknown (for example "?"), never as a made-up zero. Apply it to both of headAndEnd's reads (replay.go:240-244), which is every parse site in ListRecordings. Trigger: an Unknown record written back exactly as read (codec.go:37-38) as the last line.

## Open findings

- **BR-6** [Minor] `silent-parse-fallback` The listing shows a duration of 0 when the last events line doesn't start with {"t":N
