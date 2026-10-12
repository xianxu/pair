# Boundary Review — pair#471 (whole-issue close)

| field | value |
|-------|-------|
| issue | 471 — broadcast-web: replay a recording in the viewer with a timeline scrubber |
| repo | pair |
| issue file | workshop/issues/000471-broadcast-web-replay-a-recording-in-the-viewer-with-a-timeline-scrubber.md |
| boundary | whole-issue close |
| milestone | — |
| window | 5803b69ffaa13fe705e12fc8a120a2fb55ec6ee4..4eb72d2fa668a27c1ead7cc52ad1b878c8304931 |
| command | sdlc close --issue 471 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T03:49:55-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

I'm checking the JS catch-up port against the Go `CatchUp` it claims to mirror. Since #458 merged, Go drops hover/ptr records except the latest; the JS port appears not to.

Most of the server and CLI logic checks out. Now I'm looking at how the replay CLI gets its store: it calls `prepareRuntime` and holds a lease until Ctrl-C, which could conflict with a running Couch.

The lease is a read-only, empty lease for non-owners, so that's fine. Last check is whether the asset proxy validates the `<hash>` before reading from disk.

Docs are in place. I'll confirm the hover/ptr divergence with a quick script.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The replay works and is well built: the pure core is cleanly separated, the HTTP surface is confined, and the tests are real. Node passes 61/61, and the Go packages `webcast`, `webcast/assets` and `webcast/recording` pass. One thing blocks SHIP. The merge of origin/main (#458) changed Go's `recording.CatchUp`, and the JS port that claims to match it "record for record" was not updated. Go now keeps only the latest `hover` and `ptr` in the latest segment; the JS `catchUp` keeps all of them. The live parity check still passes only because no fixture contains a `hover` or `ptr` record. The drawn state is unaffected, because both kinds are last-writer-wins and sit outside the canonical state. But the claimed contract is broken, and the parity test would go red on any real recording with pointer movement. The fix is cheap.

**Inspection limit:** `couchcmd` and `artifactpath` don't build in this checkout. `runtimebundle/embed.go` needs generated `assets/runtime/files`, which isn't there; that's an environment issue, not #471. So I read `couchcmd/replay_test.go` but couldn't run it.

### 1. Strengths
- **Pure core:** `catchUp`, `timeline`, `placer`, `parseMoment` and `Player` are pure. `Player` has no clock (`tick(dt)` is driven by animation frames), and `replaySource` takes all its IO as deps. The node tests drive all of it with fakes (`replay.js:140-235`).
- **The seek invariant test is independent of the implementation.** It reduces the catch-up of every prefix and compares with reducing the whole prefix, over every fixture plus a synthetic recording. It also has a "check bites" test where a broken catch-up fails (`tests/webcast/replay.test.mjs:46-96`).
- **Server confinement is tight.**
  - The id regex plus `Lstat` refuses symlinks; `..`, `%2e%2e`, `a/b`, plain files and unknown ids all return 404 (`replay.go:103-110`, `replay_server_test.go` ConfinesToRoot).
  - The token check, CSP, `html/template` for the listing and `ValidHash` in the asset path are all in place.
- **Reuse instead of new code (ARCH-DRY):**
  - `assets.Reopen` reuses `Recorded` and `Proxy`, so a replay gets the same CSS rewrite and MIME allow-list as a live cast.
  - Viewer file serving was pulled into `serveViewerFile`, shared by `Server` and `ReplayServer`.
  - `recordingsDir` is now one helper shared by the recorder settings and the replay.
- **The shell change is small.** `viewer.js` has one branch point in `mount`, and the README, atlas and couch skill are all updated.

### 2. Critical findings
None.

### 3. Important findings
- **JS `catchUp` has drifted from Go's `CatchUp` after the merge** (`cmd/internal/webcast/web/replay.js:16-48` vs `recording/catchup.go:42-100`). ARCH-DRY / ARCH-PURPOSE.
  - Go (via #458's `9c288070`, brought in by merge `4eb72d2f`) drops every `hover` and `ptr` record from the latest segment except the latest of each.
  - The JS port has no such rule. Checked: for `head, seg, hover@2, ptr@3, hover@4, ptr@5`, JS returns all six records; Go returns only `head, seg, hover@4, ptr@5`.
  - **Fix:** port the hover/ptr selection into `catchUp`.
- **The parity check is vacuous for this class of record** (`replay_live_test.go:79-91`, `tests/webcast/replay.test.mjs:64`). No fixture (stress, app, blog, segments, driven) contains a `hover` or `ptr` record, so "JS `catchUp` takes the records Go's does" passes without testing that rule.
  - **Fix:** add hover and ptr records, two of each in the last segment, to the node `synthetic` recording. Then assert the expected index list directly in node, or add a Go-side test that encodes the expected result.
  - This would have caught the drift above at merge time.

### 4. Minor findings
- `ListRecordings`/`endLines` (`replay.go:215-240`): if the last events line is longer than 64 KB, the tail read cuts it, `json.Unmarshal` fails silently, and the duration shows as 0. A first line over 1 MB silently drops the recording from the listing.
- The listing has no segment count, though the plan's ReplayServer routes say it shows "segments".
- `resolveReplay` checks a DIR with `os.Stat`, which follows symlinks, but `ReplayServer.exists` uses `Lstat`. So a DIR that is a symlink prints a link that returns 404. Refuse symlinks at resolve time, or resolve with `EvalSymlinks` first.
- `run.go:286` calls `prepareRuntime` before argument resolution. Replaying an explicit DIR therefore still needs a resolvable Couch singleton, although it uses no store.
- `drawMarks` rebuilds the whole marks layer every frame while playing. That's fine at current note and stroke counts.
- `timeline()` reads `r.pts.length` with no guard. Records come through Go's decode and re-encode, so the risk is low. But a throw there would reject `v.loaded` without setting the bar's status.

### 5. Test coverage notes
- The node suite (61 tests) covers the catch-up invariant at every cut, pins reached forward, backward and shuffled, rebuild versus apply, tick and gap skipping, the refusal and error state, and `replaySource` with hash, clock and frames.
- The Go server tests cover routes, confinement, 405, CSP, the listing and assets before records.
- The live Chrome tests weren't run here; they need `WEBCAST_LIVE_CHROME=1`, unsandboxed. As written, they would still pass despite the drift above, because the fixtures have no `hover` or `ptr`.
- `couchcmd/replay_test.go` reads correctly (argument parsing, resolution, link format, stop on cancel, refusals), but I couldn't run it here.

### 6. Architectural notes
- **ARCH-DRY: flag.** The held JS port of `CatchUp` has already drifted once; see the finding above. Every future change to the catch-up rule has to touch both copies. Consider a shared table-driven case file, records plus expected kept indexes, that both the Go and node tests consume. That way neither side depends on fixture content to exercise a rule.
- **ARCH-PURE: pass.**
- **ARCH-PURPOSE: pass**, with one note. The Host mount is deferred by a recorded TL decision. The operator smoke test, a Done-when item, is still pending and is the TL's to schedule.
- **ARCH-MOCK: pass.** The only external binary is Chrome, through the existing live harness.
- **ARCH-CONSTRAINTS: pass.** Records are read per request, bounded by file size.
- **ARCH-SECURE: pass.** Recorded text only reaches the page through `textContent` and `title`, and colors go through `safeColor`.
- **ARCH-ORDER: pass.** `Player`'s states are explicit and errors are terminal, and the shuffled seeks come from a fixed seed, so a failing order can be reproduced.
- **ARCH-FUNERAL: pass.** Nothing durable is created. The proxies map is bounded by the number of directories and dies with the process.

### 7. Plan revision recommendations
- Add a "## Revisions" entry: "After the merge of #458, `recording.CatchUp` keeps only the latest hover and ptr of the latest segment. `catchUp` in `replay.js` ports that rule, and the parity coverage includes hover and ptr records, which no fixture has."

```findings
findings:
  - id: new
    severity: Important
    family: held-port-parity
    title: |
      JS catchUp no longer matches Go CatchUp after the #458 merge: it keeps every hover and ptr record
    detail: |
      Since 9c288070, recording.CatchUp keeps only the latest hover and ptr in the latest segment (catchup.go:42-100). replay.js:16-48 keeps all of them, despite claiming to match "record for record". Checked: for head,seg,hover,ptr,hover,ptr, JS returns 6 records and Go returns 4. The drawn state is unaffected (both kinds are last-writer-wins and outside the canonical state), but TestLiveReplayServerSeeks's index check fails on any real recording with pointer movement. Fix: port the hover/ptr selection.
  - id: new
    severity: Important
    family: vacuous-fixture-coverage
    title: |
      The catch-up parity check never exercises hover or ptr, because no fixture contains them
    detail: |
      None of stress/app/blog/segments/driven has hover or ptr records, so replay_live_test.go:79-91 and the node synthetic recording pass whatever the rule is. Add hover and ptr records (two of each in the last segment) to the synthetic recording in replay.test.mjs and assert the indexes kept; this would have caught the drift at merge time.
  - id: new
    severity: Minor
    family: silent-parse-fallback
    title: |
      ListRecordings silently shows a duration of 0 when the last events line is over 64 KB
    detail: |
      endLines reads a 64 KB tail, so a longer last line is cut and json.Unmarshal(last) fails silently (replay.go:215-240). A first line over 1 MB drops the recording from the listing without a word.
  - id: new
    severity: Minor
    family: symlink-policy-mismatch
    title: |
      resolveReplay accepts a symlinked DIR that ReplayServer then answers with 404
    detail: |
      resolveReplay uses os.Stat, which follows links; ReplayServer.exists uses Lstat. A symlinked DIR prints a link that 404s. Refuse it in resolveReplay or resolve it with EvalSymlinks.
  - id: new
    severity: Minor
    family: plan-surface-drift
    title: |
      The replay listing omits the segment count the plan lists
    detail: |
      The plan says the listing shows id, started, duration, segments and notes; Listed and listTemplate have no segment count.
```

---

## Re-review — 2026-10-12T03:56:20-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 471 — broadcast-web: replay a recording in the viewer with a timeline scrubber |
| repo | pair |
| issue file | workshop/issues/000471-broadcast-web-replay-a-recording-in-the-viewer-with-a-timeline-scrubber.md |
| boundary | whole-issue close |
| milestone | — |
| window | 5803b69ffaa13fe705e12fc8a120a2fb55ec6ee4..664ebbd789f60ddc4368343f95b1f7978e0b9e8e |
| command | sdlc close --issue 471 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T03:56:20-07:00 |
| verdict | SHIP |

## Review

Focus on the fix commit first.

Still reviewing: the webcast tests pass. Next I'm generating the runtime bundle so the couchcmd tests can build.

Now the revert checks: remove each fix and see whether its test fails.

```verdict
verdict: SHIP
confidence: high
```

**VERDICT: SHIP.** I checked all five round-1 findings against the code at the pinned head. All five are fixed. For each of the three behavior fixes I undid the fix in the disposable checkout, and its new test then failed:
- **BR-1:** with the `replay.js` change undone, the corpus test fails at `cut 23`.
- **BR-3:** with the `replay.go` change undone, `TestListRecordingsLongLastLine` shows a duration of `0s`.
- **BR-4:** with the `couchcmd/replay.go` change undone, `TestResolveReplay` fails on the linked case.

All Go webcast packages and the `couchcmd` replay tests pass. The `couchcmd` tests only build after the runtime bundle is generated with `go run ./cmd/internal/runtimebundle/generatecmd …`; that is a setup step, not a defect. `tests/webcast/replay.test.mjs` passes 28 of 28. One new Minor finding remains, in a family that has come up before. It does not block.

1. **Strengths**
   - The shared corpus `tests/webcast/catchup_corpus.json` is the right fix for the whole class, not just the one case. `TestCatchUpCorpus` (`cmd/internal/webcast/replay_corpus_test.go:53-62`) fails if any kind the rule special-cases is missing from the corpus. It holds the corpus answers to Go's rule, and the node test holds the JS port to the same answers. A future change to Go's rule now fails at once, not on the first real recording.
   - The corpus has two `hover` and two `ptr` records in the last segment (indexes 19/22 and 20/23), and one of each in the first segment. So every branch of the selection is reached.
   - `headAndEnd`/`lastLineStart` (`replay.go:218-280`) read the file backwards in chunks. Last lines of any length now work, and the full file is never loaded.
   - `resolveReplay` now resolves links with `EvalSymlinks` (`couchcmd/replay.go:69-77`). The printed path is now the one the server will actually serve.

2. **Critical findings:** none.

3. **Important findings:** none.

4. **Minor findings**
   - `replay.go:240-244`: the duration comes from a regex, `^\{"t":(\d+)`, on the last line. Encode does put `t` first for known kinds. But an `Unknown` record is written back exactly as read (`codec.go:37-38`), so a newer or foreign last line that doesn't start that way silently shows as 0 s. This is the second finding in family `silent-parse-fallback`, so it should be fixed as a rule, not as an instance (see the findings block).

5. **Test coverage notes**
   - The JS corpus test now covers what BR-2 asked for.
   - The live check (`replay_live_test.go`) still compares record for record on the browser fixtures only. The corpus is the oracle for the hover/ptr branches.

6. **Architecture**
   - **ARCH-DRY:** flag, but accepted. The catch-up rule exists twice by necessity (Go and JS), and it is now pinned to one corpus with Go's answers.
   - **ARCH-PURE:** pass. `catchUp`, the timeline and `Player` are pure and have no clock. The IO is in `ReplayServer` and the CLI.
   - **ARCH-PURPOSE:** pass. BR-1/BR-2 were fixed as a class: every branch is required in the corpus, not just hover/ptr.
   - **ARCH-MOCK:** pass. No new external dependencies; tests use temporary directories.
   - **ARCH-CONSTRAINTS:** pass. Listing now costs a head read plus a backward chunked scan, bounded by the size of the last line.
   - **ARCH-SECURE:** pass. Links are resolved before the server's `Lstat` gate. The 0-duration fallback on an unreadable last line is the one remaining gap (Minor above).
   - **ARCH-ORDER:** pass. `Player` state goes through `seek`/`tick`. Corpus cuts test every prefix ordering of the rule.
   - **ARCH-FUNERAL:** pass. Replay creates nothing durable; it only reads recordings.

7. **Plan revisions:** none needed. The `## Revisions` entries for BR-1/BR-2 and BR-5 match the code.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      replay.js:29-58 ports the hover/ptr selection; undoing it fails the node corpus test at cut 23.
  - id: BR-2
    disposition: addressed
    note: |
      The corpus has 2 hover and 2 ptr in the last segment plus 1 of each earlier; TestCatchUpCorpus requires every special-cased kind.
  - id: BR-3
    disposition: addressed
    note: |
      headAndEnd scans backwards to the start of the last line; TestListRecordingsLongLastLine (200 KB line) fails when the fix is undone.
  - id: BR-4
    disposition: addressed
    note: |
      resolveReplay resolves links with EvalSymlinks; the linked case in TestResolveReplay fails when the fix is undone.
  - id: BR-5
    disposition: addressed
    note: |
      The plan's Revisions entry drops the segment count with a reason: counting needs a full scan; Listed matches.
findings:
  - id: new
    severity: Minor
    family: silent-parse-fallback
    title: |
      The listing shows a duration of 0 when the last events line doesn't start with {"t":N
    detail: |
      2nd finding in silent-parse-fallback. Rule: a listing field read from a recording that can't be parsed shows as unknown (for example "?"), never as a made-up zero. Apply it to both of headAndEnd's reads (replay.go:240-244), which is every parse site in ListRecordings. Trigger: an Unknown record written back exactly as read (codec.go:37-38) as the last line.
```
