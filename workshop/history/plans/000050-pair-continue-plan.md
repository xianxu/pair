# pair continue — Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Give pair a durable, portable session-handoff: produce `continuation` docs (the `ariadne#91` datatype) from a session's rendered scrollback, and resume from them with a `pair continue` verb — the human-understanding sibling of `pair resume`.

**Architecture:** Three milestones, each its own review boundary. **M1** adds a plain-text projection to the existing renderer (the *substrate*). **M2** builds `cmd/pair-continuation` — the deterministic *writer* that enforces `#91`'s invariants (ARCH-PURE: pure frontmatter/filename core + a thin injected clock/fs/git seam; the `xx-datatype` dispatcher does the distillation, the writer does the mechanics). **M3** wires the pair UX: the `pair continue` verb (consume) and the Alt+x park-nudge (preserve-on-quit), plus docs.

**Tech Stack:** Go (stdlib + `charmbracelet/x/vt`, `ultraviolet`; no git lib — shell `git` via `os/exec`, mirroring `cmd/pair-slug/main.go:70`); bash (`bin/pair`); existing `go test ./...` + `make test`.

---

## Core concepts

### Pure entities (the conceptual core)

| Name | Lives in | Status |
|------|----------|--------|
| `serializeRow(line, plain)` | `cmd/pair-scrollback-render/serialize_row.go`* | modified |
| `Fields` | `cmd/pair-continuation/continuation.go` | new |
| `RenderFrontmatter(Fields)` | `cmd/pair-continuation/continuation.go` | new |
| `AllocName(slug, ts, existing)` | `cmd/pair-continuation/continuation.go` | new |
| `Assemble(frontmatter, body)` | `cmd/pair-continuation/continuation.go` | new |
| `ValidateFields(Fields)` | `cmd/pair-continuation/continuation.go` | new |

\* `serializeRow` currently lives inline in `cmd/pair-scrollback-render/main.go:124`. Leave it there if extraction is noisy; the table row tracks the function, not a new file.

- **`serializeRow(line, plain bool)`** — renders one emulator row to a string. Today it emits SGR via `Style.Diff` (`main.go:165-166`) + a trailing `\x1b[0m` (`:176`), trimming to the last non-blank cell (`:134-150`). Add a `plain` parameter: when true, skip the SGR emission and the reset, emit `c.Content` only, but keep the `last`-cell trim and wide-grapheme skip.
  - **DRY rationale:** one row-serializer for both viewer (colored) and continuation (plain) — the `sessionView` abstraction with two decorations, not two renderers.
  - **Future extensions:** an `unwrap` mode that joins terminal hard-wraps (deferred polish per the Spec).

- **`Fields`** — the continuation's frontmatter inputs: `Slug, Agent, SessionID, Issues []string, Branch, Worktree, Supersedes string` + a `Created time.Time`. Pure data.
  - **DRY rationale:** single source of truth for the frontmatter contract `#91` defines; `RenderFrontmatter` and `ValidateFields` both read it.
  - **Future extensions:** a `Producer` field if we ever record the distilling agent separately from `Agent` (the original) — `#91` deliberately omitted it.

- **`RenderFrontmatter(Fields) string`** — emits the exact `continuation.md` frontmatter block (inline lists for `issues:`, ISO `created:`, omit-empty for optional fields). Pure; the conformance guarantee lives here.
- **`AllocName(slug string, ts time.Time, existing []string) string`** — returns `<YYYYMMDDTHHMMSS>-<slug>.md`, appending `-1`, `-2`, … on an exact clash against `existing`. Pure (clock + dir listing injected as args).
- **`Assemble(frontmatter, body string) string`** — `---\n{fm}\n---\n\n{body}\n` with newline hygiene. Pure.
- **`ValidateFields(Fields) error`** — rejects empty `Slug`/`Agent`/`Issues`; the structural guard the Spec's Done-when calls for. Pure.

### Integration points (where pure meets the world)

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `Clock` (func `() time.Time`) | `cmd/pair-continuation/main.go` | new | system clock |
| `dirList` / `writeFile` | `cmd/pair-continuation/io.go` | new | filesystem |
| `GitRunner` | `cmd/pair-continuation/git.go` | new | `exec.Command("git", …)` |
| `pair continue` verb | `bin/pair` | new | session launch |
| Alt+x park-nudge | `bin/pair` (`cleanup_quit_marker`) | modified | quit/scrollback `rm` |

- **`GitRunner`** — `interface { Run(args ...string) (string, error) }`; the default impl shells `git -C <root> …` (mirrors `cmd/pair-slug/main.go:70`). The writer calls `add`, `commit`, `push`.
  - **Injected into:** the writer's finalize step; pure core never touches it. Tests inject a real `git` against a **temp repo with a local bare `origin`** (process-level realism per AGENTS.md §5 — not a function mock), so `add/commit/push` are exercised for real.
  - **Future extensions:** a retry/offline-tolerant push (push failure shouldn't lose the local commit).
- **`Clock` / fs** — injected so `AllocName` and the write are deterministic in tests (fixed `ts`, temp dir).
- **`pair continue` verb** — bash; mirrors the `resume` block (`bin/pair:568-585`). Wraps pair's normal launch, seeding the draft to read the chosen doc.
- **Alt+x park-nudge** — bash; inserts a `[y/N]` prompt before the scrollback `rm` (`bin/pair:1159-1167`). On yes, **preserves** the `.raw` (skips the rm) so a live session can park it later — it does *not* distill (no live agent exists at quit; see M3 note).

---

## Milestones

- M1 — `--plain` substrate (renderer)
- M2 — `cmd/pair-continuation` writer (the robustness boundary)
- M3 — pair UX: `pair continue` verb + Alt+x park-nudge + docs

Each milestone closes via `sdlc milestone-close` (its own review boundary).

---

## M1 — `--plain` substrate

### Task 1.1: `serializeRow` gains a plain mode

**Files:**
- Modify: `cmd/pair-scrollback-render/main.go:124-178` (add `plain bool` param)
- Test: `cmd/pair-scrollback-render/serialize_row_test.go`

- [ ] **Step 1 — failing tests.** Add to `serialize_row_test.go`, following the existing `cell()/styledCell()/stripSGR()` helpers:

```go
func TestSerializeRow_Plain_NoSGR(t *testing.T) {
    line := uv.Line{styledCell("hi", 1, color.RGBA{R: 255, A: 255}), cell(" ", 1)}
    got := serializeRow(line, true) // plain=true
    if got != "hi" { t.Fatalf("plain: want %q got %q", "hi", got) }
    if strings.Contains(got, "\x1b") { t.Fatalf("plain row contains escape: %q", got) }
}

func TestSerializeRow_Plain_TrimsTrailingBlanks(t *testing.T) {
    line := uv.Line{cell("a", 1), cell(" ", 1), cell(" ", 1)}
    if got := serializeRow(line, true); got != "a" {
        t.Fatalf("want %q got %q", "a", got)
    }
}

func TestSerializeRow_Plain_TrimsTrailingBgBlank(t *testing.T) {
    // trailing blank visible ONLY via background (inverse-video / box fill);
    // build the bg-styled blank with the existing helper (cf. TestSerializeRow_PreservesNonDefaultBg).
    bgBlank := styledCellBg(" ", 1, color.RGBA{B: 255, A: 255}) // blank content, non-nil Style.Bg
    line := uv.Line{cell("a", 1), bgBlank}
    if got := serializeRow(line, true); got != "a" { // plain: bg is not emitted, so trim it
        t.Fatalf("plain should trim bg-only blank: got %q", got)
    }
    if got := serializeRow(line, false); strings.TrimRight(stripSGR(got), " ") == "a" {
        t.Fatalf("colored should keep the visible bg-blank, not collapse to %q", "a")
    }
}

func TestSerializeRow_Colored_Unchanged(t *testing.T) { // regression: plain=false == old behavior
    line := uv.Line{styledCell("hi", 1, color.RGBA{R: 255, A: 255})}
    if got := serializeRow(line, false); !strings.HasSuffix(got, "\x1b[0m") {
        t.Fatalf("colored row should still reset: %q", got)
    }
}
```

- [ ] **Step 2 — run, expect FAIL** (`serializeRow` takes one arg): `go test ./cmd/pair-scrollback-render/ -run TestSerializeRow_Plain -v`
- [ ] **Step 3 — implement.** Change the signature to `serializeRow(line uv.Line, plain bool) string`. Guard the SGR-emitting lines (`main.go:165-166`) and the reset (`:176`) with `if !plain`. In plain mode write `c.Content` only. **Make the trailing-blank (`last`-cell) computation `plain`-aware:** the current loop (`:144-149`) treats a bg-colored blank as visible (`else if c.Style.Bg != nil { last = i }`); in plain mode no bg is emitted, so a trailing inverse-video/border region would become space-padding toward terminal width (the box/status-bar noise the Spec says to trim). Skip the `Style.Bg != nil` branch when `plain==true` (plain trims to the last non-blank-*content* cell); keep it in colored mode. Wide-grapheme skip unchanged. **Update every existing caller** to pass `plain`: both `render()` call sites (`:240`, `:244` → `, false`) and all eight colored-mode test callers in `serialize_row_test.go` (lines 33, 42, 56, 69, 93, 110, 127, 148 → append `, false`). (`styledCellBg` may need adding as a test helper if only a fg-styled helper exists today.)
- [ ] **Step 4 — run, expect PASS** (incl. the unchanged colored tests): `go test ./cmd/pair-scrollback-render/ -v`
- [ ] **Step 5 — commit**: `#50 M1: serializeRow plain mode`

### Task 1.2: `--plain` and `--max-lines` flags; render honors them

**Files:**
- Modify: `cmd/pair-scrollback-render/main.go:43` (`historyRows`), `:201` (`SetMaxLines`), `:285-299` (flag parsing), `render()`
- Test: `cmd/pair-scrollback-render/render_test.go`

- [ ] **Step 1 — failing test.** In `render_test.go` (mirrors `TestRender_ViewportSidecar`'s temp-file setup): write a small `.raw` containing styled output, call `render(..., plain=true, maxLines=-1)`, assert the `.ansi` output contains the visible text and **no** `\x1b[` sequences.
- [ ] **Step 2 — run, expect FAIL**: `go test ./cmd/pair-scrollback-render/ -run TestRender_Plain -v`
- [ ] **Step 3 — implement.**
  - Add flags after `flag.Usage` (`:286`): `plain := flag.Bool("plain", false, "emit plain text (no SGR) for distillation")` and `maxLines := flag.Int("max-lines", historyRows, "scrollback history rows; <=0 = uncapped")`.
  - Thread `*plain` into `render()` → `serializeRow(line, *plain)`.
  - Replace the hardcoded `SetMaxLines(historyRows)` (`:201`) with `SetMaxLines(resolveMax(*maxLines))` where `resolveMax(n)` returns a very large value when `n <= 0` (uncapped) else `n`. Keep `historyRows` as the default so the viewer path is unchanged.
  - **Flag ordering / callers.** `main()` is *positional-only* today (3 args, no named flags); Go's `flag` package requires `--plain`/`--max-lines` to appear **before** the positionals. The existing viewer caller (`bin/pair-scrollback-open` / the Alt+/ path) passes **no** flags, so it's unaffected (3 positionals parse as before) — confirm this by re-running the viewer after the change. Only the new continuation-author caller invokes `pair-scrollback-render --plain --max-lines 0 <raw> <events> <out>` (flags first).
- [ ] **Step 4 — run, expect PASS**; also `make test` to confirm the viewer path still renders colored: `go test ./cmd/pair-scrollback-render/ -v`
- [ ] **Step 5 — commit**: `#50 M1: --plain + --max-lines flags`

- [ ] **Step 6 — real-`.raw` signal check (Spec Done-when).** Capture a real agent `.raw` (run a short pair session, or reuse one from `$PAIR_DATA_DIR`), render with `--plain --max-lines 0`, and eyeball: the conversation text is legible; box chrome/spinners are tolerable noise. Record the observation in the issue `## Log` (manual verification per AGENTS.md §5). Commit any fixture used as a testdata file.

**M1 close:** `sdlc milestone-close --issue 50 --milestone M1` (fix Critical/Important before crossing).

---

## M2 — `cmd/pair-continuation` writer

The robustness boundary. Pure core unit-tested without IO; the finalize step exercised against a **real temp git repo**. Modeled on `cmd/pair-slug/` (main.go + core + `*_test.go`; `buildBinary(t)` helper).

### Task 2.1: pure core — frontmatter, name allocation, assembly, validation

**Files:**
- Create: `cmd/pair-continuation/continuation.go`
- Test: `cmd/pair-continuation/continuation_test.go`

- [ ] **Step 1 — failing tests.**

```go
func TestRenderFrontmatter(t *testing.T) {
    f := Fields{Slug: "robotics", Agent: "claude", SessionID: "7f3a",
        Issues: []string{"000071", "000073"}, Branch: "main",
        Created: time.Date(2026, 6, 11, 14, 20, 0, 0, time.UTC)}
    got := RenderFrontmatter(f)
    want := "type: continuation\nslug: robotics\nagent: claude\n" +
        "session_id: 7f3a\ncreated: 2026-06-11T14:20:00\nbranch: main\n" +
        "issues: [000071, 000073]\n"
    if got != want { t.Fatalf("frontmatter:\n got %q\nwant %q", got, want) }
}

func TestRenderFrontmatter_OmitsEmptyOptionals(t *testing.T) {
    f := Fields{Slug: "x", Agent: "claude", Issues: []string{"000001"},
        Created: time.Date(2026, 6, 11, 0, 0, 0, 0, time.UTC)}
    got := RenderFrontmatter(f)
    for _, k := range []string{"session_id:", "supersedes:", "branch:", "worktree:"} {
        if strings.Contains(got, k) { t.Fatalf("optional %q should be omitted when empty: %q", k, got) }
    }
}

func TestAllocName_NoClash(t *testing.T) {
    ts := time.Date(2026, 6, 11, 14, 20, 5, 0, time.UTC)
    if got := AllocName("robotics", ts, nil); got != "20260611T142005-robotics.md" {
        t.Fatalf("got %q", got)
    }
}

func TestAllocName_Clash(t *testing.T) {
    ts := time.Date(2026, 6, 11, 14, 20, 5, 0, time.UTC)
    existing := []string{"20260611T142005-robotics.md"}
    if got := AllocName("robotics", ts, existing); got != "20260611T142005-robotics-1.md" {
        t.Fatalf("got %q", got)
    }
}

func TestValidateFields_RequiresCore(t *testing.T) {
    if ValidateFields(Fields{Agent: "claude", Issues: []string{"1"}}) == nil {
        t.Fatal("empty slug should error")
    }
}
```

- [ ] **Step 2 — run, expect FAIL**: `go test ./cmd/pair-continuation/ -v`
- [ ] **Step 3 — implement** `Fields`, `RenderFrontmatter`, `AllocName`, `Assemble`, `ValidateFields` — all pure, no imports beyond `fmt`/`strings`/`time`/`sort`. `AllocName` uses the compact `ts.Format("20060102T150405")`; clash loop appends `-N`. Emit frontmatter fields in the **exact order of `continuation.md`'s Frontmatter-shape table** (type, slug, agent, session_id, created, supersedes, branch, worktree, issues; omit-empty for the optionals) so the golden-string test stays stable.
- [ ] **Step 4 — run, expect PASS**: `go test ./cmd/pair-continuation/ -v`
- [ ] **Step 5 — commit**: `#50 M2: continuation pure core + tests`

### Task 2.2: IO seam + the writer CLI (finalize: write + git add/commit/push)

**Files:**
- Create: `cmd/pair-continuation/git.go` (`GitRunner` + default `exec` impl), `cmd/pair-continuation/main.go` (flag parsing, orchestration)
- Test: `cmd/pair-continuation/main_test.go`

- [ ] **Step 1 — failing integration test.** Following `cmd/pair-slug/main_test.go`'s `buildBinary(t)` + temp-dir pattern, set up a **real git repo with a bare origin**:

```go
func TestWriter_WritesCommitsPushes(t *testing.T) {
    bin := buildBinary(t)
    root := t.TempDir()
    gitInitWithBareOrigin(t, root) // git init; git remote add origin <temp bare>; initial commit on main
    body := "# Continuation: robotics\n\n## NEXT ACTION\nRun make test.\n"
    cmd := exec.Command(bin, "-repo-root", root, "-slug", "robotics", "-agent", "claude",
        "-issues", "000071,000073", "-branch", "main", "-body-file", writeTemp(t, body))
    out, err := cmd.CombinedOutput()
    if err != nil { t.Fatalf("writer failed: %v\n%s", err, out) }
    // 1. file exists at workshop/continuation/<ts>-robotics.md, conformant frontmatter + body
    // 2. it's committed (git log shows it) AND pushed (bare origin has the commit)
}
```

- [ ] **Step 2 — run, expect FAIL**: `go test ./cmd/pair-continuation/ -run TestWriter -v`
- [ ] **Step 3 — implement.**
  - `git.go`: `GitRunner` interface + `execGit{root}` calling `exec.Command("git", "-C", root, args...)` (pattern: `pair-slug/main.go:70`).
  - `main.go`: parse flags (`-repo-root`, `-slug`, `-agent`, `-session-id`, `-issues` (comma-split), `-branch`, `-worktree`, `-supersedes`, `-body-file`); default `-repo-root` to `git rev-parse --show-toplevel`; read body from `-body-file` (or stdin if `-`); `ValidateFields`; list `workshop/continuation/`; `AllocName`; `Assemble`; `MkdirAll` + write; then `git add <path>`, `git commit -m "continuation: <slug>"`, `git push`. A push failure prints a warning but does **not** delete the local commit (the recovery doc is still saved locally). Print the final path to stdout.
- [ ] **Step 4 — run, expect PASS**: `go test ./cmd/pair-continuation/ -v`
- [ ] **Step 5 — wire build/test**: add `pair-continuation` to `GO_BINS` (`Makefile.local:29`) + a build recipe whose target is `./cmd/pair-continuation` (so all `.go` files compile — mirror the **multi-file `pair-slug`** recipe, not the single-file `pair-scrollback-render` one) + an alias. Run `make build && make test`.
- [ ] **Step 6 — commit**: `#50 M2: pair-continuation writer (git seam + CLI) + integration test`

**M2 close:** `sdlc milestone-close --issue 50 --milestone M2`.

---

## M3 — pair UX: `pair continue` verb + Alt+x park-nudge + docs

Bash in `bin/pair` (no Go test harness; verify via the existing bash integration-test style under `tests/` + manual steps recorded in `## Log`).

### Task 3.1: `pair continue [slug] [agent]` verb

**Files:**
- Modify: `bin/pair` (verb parse near `:568`; launch path)
- Test: `tests/` bash test (mirror an existing one) + manual steps

- [ ] **Step 1 — verb parse.** Mirror the `resume` block (`bin/pair:568-585`): claim `continue` as a subcommand. **Placement matters:** insert the `continue` claim in the same *pre-loop* position as `resume` (before the agent-arg loop at ~`:588`), or `pair continue robotics` parses `continue` as the agent name. `pair continue` (no slug) → **list** mode: glob `$REPO_ROOT/workshop/continuation/*.md`, print one line per doc (slug from filename, first `## NEXT ACTION` line, `issues:` from frontmatter, mtime age); exit 0. `pair continue <slug> [agent]` → resolve the newest `*-<slug>.md`, set the optional agent, fall through to the launch path. **Don't literally mirror resume's positional handling:** once a forced tag is set, resume's positional loop (`:597-600`) routes any trailing arg to the `unexpected positional arg` error — so the `continue` block must capture `[agent]` itself (set `AGENT` from `$3` before its `shift`), or `pair continue <slug> <agent>` would reject the agent (the port feature).
- [ ] **Step 2 — seed the launch.** On `pair continue <slug>`: launch a fresh pair session (chosen agent, defaulting to the doc's `agent:`). **Seeding is a NEW mechanism** — there is no launch-time draft-seed today. The nvim draft is the on-disk file `$PAIR_DATA_DIR/draft-<tag>.md` (loaded by `nvim/init.lua` on tab open); write the seed text — `Read workshop/continuation/<file> and continue from its NEXT ACTION.` — to that file **before** the session launches, and confirm a fresh tab loads pre-existing draft content from disk (it does — the slot-load reads the file). Do **not** read `session_id` — `continue` never does a native resume.
- [ ] **Step 3 — bash test + manual verify.** Add a `tests/continue-list-test.sh` asserting the list output against a fixture continuation dir. Manually verify `pair continue <slug> codex` launches codex on the doc; record in `## Log`.
- [ ] **Step 4 — commit**: `#50 M3: pair continue verb (list + launch + port)`

### Task 3.2: Alt+x park-nudge (preserve scrollback on quit)

**Files:**
- Modify: `bin/pair` `cleanup_quit_marker()` (before the `rm -f` at `:1159-1167`)

- [ ] **Step 1 — prompt before rm.** Before the scrollback `rm -f` block, if the `.raw` is non-trivial (size > a small threshold), prompt on the tty (pattern from `bin/pair:1472`): `printf 'Park %s as a continuation? [y/N]: ' "$PAIR_TAG" >/dev/tty; read -r ans </dev/tty`.
- [ ] **Step 2 — on yes, preserve (do not distill).** There is **no live agent at quit**, so the nudge does not produce the doc. The scrollback paths are operands inside a *single* multi-line `rm -f … \` (`:1159-1167`); **split it into two** invocations — the scrollback `.raw`/`.events.jsonl`/`.ansi` conditional on the answer, everything else unconditional. On `y`: instead of removing the `.raw`, **rename it to a non-recyclable name** `parked-scrollback-<tag>-<ts>.raw` — the in-place `scrollback-<tag>-<agent>.raw` would be `O_TRUNC`'d by the next `pair <same-tag>` (Spec §3), so "preserve in place" is a latent data-loss bug. Drop a `parked-<tag>` marker; print: `Scrollback preserved (parked-scrollback-<tag>-<ts>.raw). Open a session and "park pair-<tag>" to finish the continuation.` On `n`/empty: rm as today.
- [ ] **Step 3 — manual verify.** Alt+x with `y` → scrollback `.raw` survives + marker present; with `n` → cleaned as before. Record in `## Log`.
- [ ] **Step 4 — commit**: `#50 M3: Alt+x park-nudge (preserve scrollback on opt-in)`

> **Design note (park-nudge scope).** Producing the full continuation at quit would need a headless distill (`claude -p` over the `--plain` render — there's precedent in `cmd/pair-slug`). v1 keeps the nudge to *preserve* (cheap, agent-free, robust against the immediate `rm`); the live-session park (the reliable, recommended author path) does the distillation + writer finalize. Headless-distill-at-quit is a noted v2.

### Task 3.3: docs

**Files:**
- Modify: `atlas/` (a `resume` vs `continue` note + pointer to the writer), `README.md` (the `pair continue` verb; the Alt+x nudge in the keybindings table)

- [ ] **Step 1** — atlas: short `resume` (machine state) vs `continue` (human understanding) entry; point at `cmd/pair-continuation` and `construct/datatype/continuation.md`.
- [ ] **Step 2** — README: document `pair continue [slug] [agent]` and the Alt+x park prompt.
- [ ] **Step 3 — commit**: `#50 M3: atlas + README for continue`

**M3 close:** `sdlc milestone-close --issue 50 --milestone M3`, then `sdlc close --issue 50`.

---

## Verification summary

- **M1:** `go test ./cmd/pair-scrollback-render/` (plain + colored-regression) + real-`.raw` signal eyeball.
- **M2:** `go test ./cmd/pair-continuation/` — pure core units + the write→commit→push integration test against a real temp repo with a bare origin (the deterministic anchor for the Spec's invariants). `make build && make test` clean.
- **M3:** bash list test + manual `pair continue` / Alt+x walkthroughs recorded in `## Log`.
- **End-to-end (records the Spec's headline):** in a live pair session, "park this" → `xx-datatype` applies `continuation.md` → shells `pair-continuation` → doc lands in `workshop/continuation/`, committed + pushed; then `pair continue <slug>` (optionally a different agent) launches a fresh session on it. Record the transcript-level walkthrough in `## Log`.

---

## Revisions

**2026-06-11 — reconcile Core concepts table with shipped code (M3 boundary review).**
The pure/IO-seam architecture held, but three table rows drifted from the implementation:
- `io.go` / `dirList` / `writeFile` were **folded into `main.go`** (`listMarkdown` + inline `os.WriteFile`) — no separate file.
- `GitRunner` is a concrete `gitRunner` struct (`git.go`), **not an injected interface** — the git seam is exercised via the **built binary against a real temp repo with a bare origin** (the subprocess integration test), which is stronger than an interface mock.
- `Clock` is a plain `func() time.Time` param to `run()`, not a named type.

**2026-06-11 — push target + dirty-index + NEXT-ACTION (M3 boundary review, FIX-THEN-SHIP).**
- **Critical:** the writer's `git commit` was un-scoped, so a pre-staged dirty index was swept into (and pushed with) the continuation commit. Fixed: `commit -- <rel>` (path-scoped) + `TestWriter_DoesNotSweepDirtyIndex`.
- **Push target:** the writer pushes `origin HEAD` (current branch), **not** a forced `main` — the doc reaches main when the feature branch merges. Docs (`continuation.md`, `atlas`, code comment) corrected from "to main".
- **NEXT-ACTION enforcement** now lives in the writer (`run()` rejects a body without `## NEXT ACTION`) — making the "structural guard" framing accurate; `TestWriter_RequiresNextAction`.
- Deferred minors: extract a `normalize_tag()` shell fn (resume/continue share the tag normalize+validate); a `pair park-render <tag>` lister for accumulated `parked-scrollback-*`.
