# Boundary Review — pair#372 (whole-issue close)

| field | value |
|-------|-------|
| issue | 372 — couch focus view: show pair-slug inline |
| repo | pair |
| issue file | workshop/issues/000372-couch-focus-view-inline-pair-slug.md |
| boundary | whole-issue close |
| milestone | — |
| window | 23f311fb48d5a3bc9e5909a05669c2d9d241bbe0..ee291842863ca023c6ae1a35080e8068975a1e3a |
| command | sdlc close --issue 372 |
| reviewer | claude |
| timestamp | 2026-10-01T16:00:28-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

**Verdict: SHIP.** The branch does what the Spec asks, and I found nothing Critical or Important. Each focus-view row now renders `name ◆ description ◆ slug` on one line, clipped to the terminal width. The slug comes only from `slug-proposed-<tag>`, read with a bounded, no-follow read. It is filled in once per live row during the existing inventory refresh, so the render path does no IO. Read errors and bad files show as no slug. Every Done-when unit and inventory test exists. The touched packages pass for me: `slugline`, `slugcmd` and `couchtty` filtered to the slug/focus tests, and `couchcore -run Slug`. I did not run the full suite. The operator smoke test is logged as passed. All the findings below are Minor.

**Note:** while inspecting I ran `git checkout ee291842`, which detached the worktree. That was a slip, since this review is meant to be read-only. It was the branch HEAD, so nothing changed, and I switched straight back to `000372-couch-focus-view-inline-pair-slug`. The working tree is as I found it, with only the untracked #369 issue file.

### 1. What's done well
- **One owner for the slug format in Go.** `cmd/internal/slugline/slugline.go` now holds `Valid`, `Format`, `Focus` and `Unfenced`. `slugcmd` uses them, and its old regex, `rightOf` and their tests moved there instead of being copied (ARCH-DRY).
- **Follows the `ApplyRepositoryAliases` precedent.** `ApplySlugs` (`actionableinventory.go:694`) is a pure fold over the rows with the reader passed in. It resets `Slug` on every row, so a parked row can't keep a stale slug, and the test checks that. It skips non-live rows and rows with a zero address.
- **Safe file read.** `OSSlugReader` reuses `openSwitchRegularFile` (O_NOFOLLOW, non-blocking, regular files only) and caps the read at 1 KiB+1. The slug is also passed through `rowtext.Sanitize` at render time (`menu_reattach.go:368`). The test injects `\x1b[2J` and a newline and checks both are stripped.
- **"Not a notification" is tested through the real reducer.** `TestMenuFocusSlugChangeIsNotAttention` sends an inventory event through `ReduceMenu` and checks there are no effects, no attention and no notice, and that the new slug renders.
- **Artifact manifest updated.** The binding `composite-slug-proposed` and the source classifications were added in `artifactpath/manifest.go`.

### 2. Critical
None.

### 3. Important
None.

### 4. Minor
- **"One definition" claim is too strong** (`slugline.go:1-3`). `nvim/slug.lua:11-15` and its runtimebundle copy still parse the `=== … ===` fence on their own. That second definition is reasonable across languages, but the package comment should say it is the Go definition and that `nvim/slug.lua` mirrors it (ARCH-DRY).
- **Plan names `Unfence`; the code exports `Unfenced`.**
- **The constant `close` in `slugline.go:13` shadows Go's built-in `close`.** Harmless in this package; renaming it `closing` or `fence` would avoid the trap.
- **Slugs are read on paths that don't show them** (ARCH-CONSTRAINTS). `ActionableThreadInventoryContext` is also called from start and slot paths (`couch.go:206,265`, `slotstart.go:179,322`). Each call now does one extra ≤1 KiB read per live row, which is negligible, but those paths never use the slug.

### 5. Test coverage
- Every Done-when bullet has a test.
- The reader test covers:
  - a missing file;
  - `slug-<tag>` being ignored;
  - a valid file being unfenced;
  - a malformed file (`KEEP`) reading as no slug;
  - an oversized file returning an error.
- The oversized error is swallowed by `ApplySlugs`; that pairing is covered by the injected-reader failure case, not end to end. That is acceptable.
- The implementor's log reports a mutation check: removing `menuFocusSlug` makes the new tests fail.

### 6. Architecture
| Principle | Result | Why |
|---|---|---|
| ARCH-DRY | pass | One Lua/Go note under Minor. |
| ARCH-PURE | pass | `ApplySlugs` is pure with the reader injected; file IO stays in `OSSlugReader`. |
| ARCH-PURPOSE | pass | The full Spec is delivered, with no consumer left over. |
| ARCH-MOCK | pass | The filesystem read is tested against a real temp directory, and the seam is a function. |
| ARCH-CONSTRAINTS | pass | Reads are bounded and stay off the render path; extra-read note under Minor. |
| ARCH-SECURE | pass | The agent-written file is treated as untrusted: no-follow open, size cap, format validation, control characters stripped. Bad input shows as no slug rather than invented text. |
| ARCH-ORDER | pass | It holds no new state between events: the slug is recomputed from scratch on every inventory refresh and is a plain display field. |
| ARCH-FUNERAL | pass | It creates nothing durable; it only reads a file the existing pair-slug flow already owns. |

**For upcoming work:** the operator noted that the slug's branch segment often repeats the description. If that follow-up happens, put a `Left` accessor in `slugline` rather than splitting the string at the call site.

### 7. Plan revisions
- Add a `## Revisions` note: the export is named `slugline.Unfenced`, not `Unfence` as the Plan says.

```findings
findings:
  - id: new
    severity: Minor
    family: single-source-claim-overstated
    title: |
      slugline doc claims to be "the one definition" of the slug format while nvim/slug.lua independently parses the fence
    detail: |
      nvim/slug.lua:11-15 (plus the runtimebundle copy) re-implements the === L | R === recognition; reword the package doc to say it is the Go definition and nvim/slug.lua mirrors it.
  - id: new
    severity: Minor
    family: plan-code-name-drift
    title: |
      Plan names slugline.Unfence; the exported function is Unfenced
    detail: |
      Add a Revisions line so the plan matches the code.
  - id: new
    severity: Minor
    family: builtin-shadowing
    title: |
      slugline.go const close shadows Go's built-in close
  - id: new
    severity: Minor
    family: display-only-work-on-non-display-paths
    title: |
      ApplySlugs runs on start/slot inventory calls that never show slugs
    detail: |
      couch.go:206,265 and slotstart.go:179,322 now do one bounded read per live row; negligible cost, but the work is unused there.
```
