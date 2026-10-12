# broadcast-web agent queries Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `couch --broadcast-web-query DIR dom|events|notes` answers an agent's three questions about a recording (the page at t, what happened in a window, every note with its element, context and the user's actions just before), in text sized for an agent's context, with `--json` for structure.

**Architecture:** The real `query.Surface` (#457's seam) is one pass of the reference reducer (`recording.State.Apply`) over the records; nothing is indexed or stored beside the recording (ARCH: one truth, the records). Rendering to text is a separate pure layer, so the Surface stays the contract and the CLI is only parse + render. The notes model is `notes.Model` behind its interface: `notes.FakeModel` until #461 lands, then a one-line swap at the CLI's construction site.

**Tech Stack:** Go, the `cmd/internal/webcast/recording` reducer, golden files under `query/testdata/golden/`.

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `Querier` (the real `Surface`) | `cmd/internal/webcast/query/querier.go` | new |
| `Select` (CSS-selector subset + `@id`) | `cmd/internal/webcast/query/selector.go` | new |
| `Page` (compact DOM text) | `cmd/internal/webcast/query/render.go` | new |
| `Describe` (one-line element name, `<button#save @12>`) | `cmd/internal/webcast/query/render.go` | new |
| `EventLines` (event window as lines, DOM runs summarized) | `cmd/internal/webcast/query/events.go` | new |
| `NoteText` (a NoteView as text) | `cmd/internal/webcast/query/render.go` | new |
| `Main` (argv → output; closed form) | `cmd/internal/webcast/query/cli.go` | new |
| `UserAction` | `cmd/internal/webcast/query/surface.go` | unchanged (#457's) |

- **Querier** — `New(recs, model) *Querier`; `DOMAt` = `Reduce(recs, t)` then `Select` then `Page`; `Events` = the records in `[t1,t2]`; `Notes(before)` = one pass applying records, capturing at each `note` record the element, `model.Context(st, at)` and the window of `UserAction`s, then patching each note's state to its final one (an agent wants "is it resolved now"). O(records), not O(notes × records) as the fake is.
  - **Relationships:** 1:1 with a recording; holds a `notes.Model` (1:1).
  - **DRY rationale:** the fake keeps its plain `Outline` (it is the seam's minimal fake); the real one adds selectors, state overlay and budgets. Both run `RunSurfaceContract`, so they can't drift on meaning. `UserAction` stays the one definition of "an action before a note".
  - **Future extensions:** a seek index from the latest `seg` (atlas: Lifecycle) when recordings get long.
- **Select** — selector grammar: compound parts `tag`, `#id`, `.class`, `[attr]`, `[attr=v]`, `[attr*=v]` (quoted or not), `:nth-child(n)`, joined by descendant (space) and child (`>`) combinators; plus `@N`, a recording node id (what every line of output names). Searches documents, shadow roots, template contents and iframe documents. Unknown syntax is an error that names the supported grammar. Returns all matches in document order.
  - **DRY rationale:** first selector matcher in the repo; `notes.FakeModel`'s `tag:nth-child(i) > …` selectors are in the grammar, so note anchors resolve through it.
- **Page** — the compact text: one line per element `@id <tag attrs>`, text inline when it's the only child, kept attributes (id, class, role, aria-*, name, type, href, src, alt, title, placeholder, for, value, data-testid; each trimmed to 80 chars), observer state overlaid (`=value "Ada"`, `=checked`, `=sel [1]`, `focused`, `scroll 0,100`), `#shadow-root (closed)`, iframe documents inline under their host, `<script>`/`<style>` bodies replaced by `(N chars)`, `<head>` collapsed to one summary line, `<svg>` collapsed to one line with its element count (unless selected). A line budget (`--max-lines`, default 300) ends with `… N more lines; narrow it with --sel`.
  - **Future extensions:** layout boxes on lines (`--boxes`), when the notes model needs them.
- **EventLines** — applies records in order from the start (so each target can be named as of its moment, `rm` named before it applies), emits a line per interaction (`click`, `input`, `focus`, `scroll`, `selection`, `nav`, `title`, `vp`, `seg`, `stop`, `doc`, `note`, `note-state`, `stroke`, `ctl`) with `by <viewer>` when attributed, and folds each run of consecutive DOM/style/asset/layout records into one `dom` line: counts by kind and up to three most-touched elements; a run of at most 3 records is written out one per line instead. Unknown kinds: one `? kind` line.
- **NoteText** — header (id, t, seg, state, author), the quoted text, the anchor (`@id`, selector, box), the page URL, the element (`Page` of it, 20-line budget), the focused context (landmark described + its text), and the "before" lines (`EventLines` of `Before`).
- **Main** — `Main(args, stdout, stderr) int`: `DIR dom [--at T] [--sel S] [--json] [--max-lines N]`, `DIR events --from T1 --to T2 [--json] [--max-lines N]`, `DIR notes [--before MS] [--note ID] [--json]`. Times are ms since the recording started (the format's unit), plain integers; `--at` defaults to the end of the recording. Validates the directory with `recording.ReadDir` and refuses a recording the reducer rejects, naming the record. `--json`: `dom` → `{t,url,title,seg,matches:[node]}` (node in the format's `Node` shape plus `frame`/`state`); `events` → the records as JSON lines (`recording.Encode`); `notes` → an array of `{note,state,seg,element,context,before:[records]}`.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `couch --broadcast-web-query` dispatch | `cmd/internal/couchcmd/cli.go`, `run.go` | modified | argv / filesystem (recording dir) |
| `notes.Model` | `cmd/internal/webcast/notes/model.go` | unchanged (#457) | #461's real model |

- **couch dispatch** — `ParseCLI` returns `cliWebQuery` with the argv tail when `args[0]` is the flag (a closed form, parsed before layout flags like `--session-sync`); `Run` calls `query.Main` without preparing the Couch runtime (a pure file read: works with no Couch running). `usage` gains two lines; `couch --skill` gains a paragraph. Injected: none; the recording dir is the IO.
- **notes.Model** — injected into `Querier`; `Main` constructs `notes.FakeModel{}` today.

## Tasks

- [ ] **Task 1: Querier passes the surface contract.** `query/querier_test.go`: `RunSurfaceContract(t, func(recs) Surface { return New(recs, notes.FakeModel{}) })` plus fixture smoke (every fixture: DOMAt at each canonical time, Events over all, Notes) and a test that `Notes` reports a resolved note as resolved. Red → implement `querier.go` (with a minimal `Page`) → green. Commit.
- [ ] **Task 2: Select.** `selector_test.go` table: each grammar part, combinators, `@N`, shadow/iframe reach, `FakeModel` selectors round-trip (anchor → select → same node on the `app` fixture), errors for unsupported syntax. Implement. Commit.
- [ ] **Task 3: Page rendering.** `render_test.go`: state overlay (contract recording: `=value "Ada"`, `focused`), script/style bodies elided, head collapsed, svg collapsed, closed shadow root shown, iframe doc inline (stress fixture), budget line. Commit.
- [ ] **Task 4: EventLines.** `events_test.go`: contract window 1500..2100 gives `input`, `click`, `attr` lines (a short run written out); stress's busy window folds to a `dom` line with counts; `by` shown on the `driven` fixture; `rm` target named. Commit.
- [ ] **Task 5: Notes text + CLI.** `cli_test.go`: argv errors (unknown verb, missing `--from/--to`, bad number, missing dir) with exit 2; `--json` decodes; a recording the reducer rejects exits 1 naming the record. Commit.
- [ ] **Task 6: Golden tests.** `golden_test.go`: for every fixture, `dom` at its first note's time, `events` over the 3 s before it, `notes`; compared with `testdata/golden/<fixture>.<query>.txt`; `-update` rewrites. Commit.
- [ ] **Task 7: Agent check (scripted).** `agent_test.go`: for each fixture note, `notes --note ID` output names the element (`@node`, its tag), the time, and — where the fixture has one in the window — the preceding action's kind and target. Plus a recorded manual run: a fresh subagent with only `couch --broadcast-web-query` and a fixture path answers what `app-n2` refers to; transcript summary in `## Log`.
- [ ] **Task 8: couch wiring + docs.** `couchcmd` dispatch, usage, skill text, a `couchcmd` parse test; `atlas/webcast-format.md` gains "Agent queries (#462)" (command, outputs, budgets, the FakeModel swap point); `atlas/index.md` if a new file. `make build`; run the three queries on the `blog` fixture from the built binary.

## Risks

- Output size on `stress` (9k records): budgets are the defence; the golden files make any regression visible in the diff.
- The selector grammar is a subset: an agent's real CSS (`:hover`, `~`, `+`) is refused with the grammar, never silently mismatched.
- The fixture notes carry selectors from fixturegen that aren't CSS (`h1~Reports`, `img[alt*=issue tracker]`); notes resolve by node id first, so they still work; `Select` reports those as unsupported if an agent passes them back.

## Revisions

- 2026-10-12: #461 landed on main mid-issue. `Main` now constructs `notes.Structural` instead of the fake (the planned one-line swap), and `Select` also reads the real model's anchor grammar (`:nth-of-type(n)`, the `>>>` deep step), so a note's selector works with `--sel`; `TestModelSelectorsSelectTheirElement` checks both models' anchors on two fixtures.

