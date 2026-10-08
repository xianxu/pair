# Grok Harness Integration Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Bring xAI's `grok` CLI (Grok Build TUI 1.0.46) to parity with `claude`/`codex`/`agy`/`muse`/`qoder` across every surface of `atlas/how-to-bring-up-a-new-harness-cli.md` (§0 registry, aspects 1–7, §2 checklist), in standalone pair and under Couch.

**Architecture:** Every surface is a registration in an existing per-agent seam, following the qoder bring-up (pair#300) and its review-round rules. Grok is **not** claude-family on disk, so it gets its own small scanner over its ACP `updates.jsonl`. It is modeled on `scan_muse.go`: path identity plus a per-record identity check. The one structural change is that Codex's forced inline mode becomes a per-harness launcher field shared by Codex and Grok (ARCH-DRY), placed before any `--`. TTY recognizers, overlay markers and prompt glyphs are **capture-first**: no byte, marker or glyph is written from imagination.

**Tech Stack:** Go (`cmd/internal/*`), Neovim Lua (`nvim/scrollback.lua`), JSONL fixtures, live PTY capture harness (`PAIR_LIVE_HARNESS`).

**Verified ground facts (2026-10-07, grok 1.0.46):**

- Binary `~/.local/bin/grok`; `grok --version` prints `grok 1.0.46 (2765805b9442) [stable]`. Login is in `~/.grok/auth.json` (operator logged in 2026-10-07).
- Sessions live at `~/.grok/sessions/<url-encoded-cwd>/<uuidv7>/`. Each has `summary.json` (`info.id`, `info.cwd`, `created_at`, `updated_at`, `head_branch`, …) and `updates.jsonl`, one JSON-RPC-shaped record per line:
  `{"timestamp":<unix-s>,"method":"session/update"|"_x.ai/session/update","params":{"sessionId":"<uuid>","update":{"sessionUpdate":"<kind>",…},"_meta":{…}}}`.
  Kinds observed: `user_message_chunk` (`content.text` = the prompt), `agent_message_chunk`, `agent_thought_chunk`, `tool_call`, `tool_call_update`, and `turn_completed` (method `_x.ai/session/update`, with `stop_reason`). There are also sibling `events.jsonl`, `chat_history.jsonl`, `signals.json` and others, which pair does not read.
- **Stub sessions:** a dir with `summary.json` and no `updates.jsonl` exists after `grok login` (`num_messages: 0`).
- The per-cwd `prompt_history.jsonl` lists `{timestamp, session_id, prompt, is_bash}`.
- TUI flags: `-s/--session-id <uuid>` creates a **new** conversation with that ID (measured: `grok -s <uuid>`, no prompt, created the dir at once; `--help` agrees). `-r/--resume [id-or-title]` takes an optional value. `-c/--continue`. `--fork-session`. `-w/--worktree [name]` takes an optional value. `--no-alt-screen`; `--minimal` is experimental scrollback-native rendering (not used). `-p/--single <PROMPT>` is headless and **takes a value**. Positionals are `[PROMPT] [COMMAND]`, and subcommands include `agent`, `sessions`, `login` and others.
- Keys (README): Enter sends; Shift+Enter or Alt+Enter inserts a newline.
- Permissions: `~/.grok/config.toml`, `--allow <RULE>`/`--deny`, `--permission-mode {default,acceptEdits,auto,dontAsk,bypassPermissions,plan}`, `--always-approve`.

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `supportedAgents` grok row | `cmd/internal/launcher/agent_defaults.go` | modified |
| `AgentGrok` + session `supportedAgents` row | `cmd/internal/sessioninventory/model.go`, `runcli.go` | modified |
| `resumeform` grok row | `cmd/internal/resumeform/resumeform.go` | modified |
| `freshAgentSpecs`/`freshValueOption` grok rows | `cmd/internal/launcher/fresh_args.go` | modified |
| `inlineModes` / `inlineModeArgs` / `InlineOptOuts` (per-harness inline flag + opt-out env) | `cmd/internal/launcher/agentargs.go` | new (replaces `codexAltScreenArgs`) |
| `insertBeforeDoubleDash` | `cmd/internal/launcher/agentargs.go` | new |
| `resumeToken`/`composeResumeArgs`/`MintsSessionID` grok case | `cmd/internal/launcher/agentargs.go` | modified |
| `ValidateGrokDelta` + `grokPathFact` | `cmd/internal/sessioninventory/scan_grok.go` | new |
| `normalizeGrokEvent` + `grokMappedKinds`/`grokIgnoredKinds` | `cmd/internal/sessioninventory/event.go` | new |
| `ruledBoxComposerSpec.ruleCol`/`sideGlyph` | `cmd/internal/wrapcmd/composer_recognizers.go` | modified |
| `promptGlyphAuthorities` | `cmd/internal/wrapcmd/orientation.go` | new |
| `detectRawCarryOverlay` (shared qoder/grok) | `cmd/internal/wrapcmd/wrap.go` | new |
| `resumeform.Form.SessionID`/`Continue`, `ContextSelector`, `ContextShortLetters`, `HasSessionID` | `cmd/internal/resumeform/resumeform.go` | modified |
| Grok composer recognizer + `grokPromptGlyphs` | `cmd/internal/wrapcmd/composer_recognizers.go` | new (capture-gated) |
| `grokPickerMarkers` / `detectGrokOverlayText` | `cmd/internal/wrapcmd/wrap.go` | new (capture-gated) |
| `DefaultGrokModel` | `cmd/internal/model/model.go` | new |
| Prompt-glyph consumers | `nvim/scrollback.lua`, `cmd/internal/changelogcmd/distill.go`, `cmd/internal/wrapcmd/orientation.go` | modified (capture-gated) |

- **`inlineModeFor`**: returns `(flag, optOutEnv)` per agent, with codex `{--no-alt-screen, PAIR_CODEX_ALT_SCREEN}` and grok `{--no-alt-screen, PAIR_GROK_ALT_SCREEN}`. `inlineModeArgs(agent, args, optOut)` strips an existing flag and inserts it with `insertBeforeDoubleDash`.
  - **Relationships:** 1 table : N harnesses. The launcher reads the opt-out env per agent (`LaunchOptions.InlineOptOut map[string]bool` replaces `CodexAltScreenOptOut`).
  - **DRY rationale:** a second `grokAltScreenArgs` copy would fork the strip/insert rule (ARCH-DRY).
  - **Fixes codex:** today the codex flag is appended after `--`, so codex reads it as prompt text (`peer_launch_test.go:59` pins that wrong order). That test row changes to `--model model --no-alt-screen -- prompt`.
- **`insertBeforeDoubleDash`**: inserts tokens before the first `--`, or appends when there is none. It is shared by the inline flag, the minted `--session-id` (createflow) and grok's `composeResumeArgs`, because Grok reads every word after `--` as `[PROMPT]`.
- **`ValidateGrokDelta`**: a deterministic transition over supplied records. A record is valid when it decodes strictly into the envelope above **and** `params.sessionId == path ID`. Anything else disputes the file with a diagnostic. Chronology is the first record's `timestamp`, with `fallbackTime` otherwise. Role is always root (see non-goals).
  - **DRY rationale:** this is the first ACP-shaped harness. The decode is written grok-specific on purpose (an ACP-generic scanner was rejected in the Spec as speculative).
- **`normalizeGrokEvent`**: `user_message_chunk` → `EventOperator` (text). `agent_message_chunk` → `EventAssistant`. `tool_call` → `EventToolCall`. `tool_call_update` with `status: completed|failed` → `EventToolResult`, other statuses → ignored. `turn_completed` → `EventTerminal`. The ACP spec's documented bookkeeping kinds (`agent_thought_chunk`, `plan`, `available_commands_update`, `current_mode_update`, `config_option_update`, `usage_update`, `session_info_update`) → ignored, listed in one `grokIgnoredKinds` set. A kind in neither the mapped nor the ignored set, an unknown `method`, or a malformed envelope → `EventNearMiss`. That is the drift signal for genuinely new shapes, not for ordinary ACP kinds that our sample sessions didn't happen to contain.
  - **Chunked prompts (ARCH-ORDER):** the normalizer stays per-record and stateless. If Task 4 measures more than one `user_message_chunk` per prompt, the join is a grok-only fold inside `NativeEventsFromRecords` (`events.go:62`), the one batch consumer that already sees ordered records. The state is the pending group keyed by `_meta.promptIndex`. It is emitted at the group's first record position when a record with a different kind or index arrives. A group still open at the end of the batch is held, not emitted, so an incremental tail never yields half a prompt. This branch is taken only on evidence, and it opens a plan `## Revisions` entry before any code.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `ScanGrok` / `scanGrokFile` | `cmd/internal/sessioninventory/scan_grok.go` | new | `Runtime` roots, file reads |
| OSRuntime grok root `grok-sessions` | `cmd/internal/sessioninventory/runtime_os.go` | new row | `~/.grok/sessions` |
| `ProviderGrokACPV1` contract | `cmd/internal/sessioninventory/provider_contract.go` | new row | append-only producer contract |
| Incremental + target + watcher/ledger membership | `incremental_inventory.go`, `target.go`, `conformance.go`, `sessionwatch/sessionwatch.go`, `sessionledger/record.go` | modified | per-agent dispatch |
| `grok` TTY profile | `cmd/internal/wrapcmd/harness_tty.go` | new entry | proxy keymap, recognizer, overlay |
| `detectGrokOverlayOpen` | `cmd/internal/wrapcmd/wrap.go` | new | proxy-owned overlay carry |
| Live TTY fixtures | `cmd/internal/wrapcmd/testdata/tty/grok/1.0.46/` | new | real grok PTY bytes |
| Scanner fixtures | `cmd/internal/sessioninventory/testdata/native/grok/v1/grok-sessions/` | new | sanitized real `updates.jsonl` |
| `runGrok` | `cmd/internal/model/model.go` | new | `grok -p` subprocess (slug) |
| Grok permission allowlist | `~/.grok/config.toml` or repo-local equivalent (evidence-gated) | config | grok permission engine |

- **`ScanGrok`**: walks `grok-sessions`. It recognizes `<enc-cwd>/<uuid>/updates.jsonl` only. Other files in a session dir (`summary.json`, `events.jsonl`, …) produce no diagnostic. A session dir without `updates.jsonl` (a stub) yields nothing, which is the "no conversation" reading.
  - **Injected into:** every inventory consumer through `Runtime`. Tests use `sessioninventorytest.NewFakeRuntime` with fixture trees (the existing stateful fake, ARCH-MOCK).
- **`runGrok`**: runs `grok -p <prompt>` with `cmd.Dir = os.TempDir()` and `PAIR_SLUG_NESTED=1`. Grok has no `--no-session-persistence`, so each slug call leaves a session dir under the TempDir's encoded cwd (see Durable state).

### Durable state (ARCH-FUNERAL)

- Pair's own families (adapt log, inventory catalog, ledger) gain `grok` rows under their existing retention. There is no new family.
- **Slug residue:** each `runGrok` call creates one Grok session dir under `~/.grok/sessions/<enc-TMPDIR>/`. Grok owns that storage. The creator is pair, and nothing removes it. M3 Task 13 must measure whether a flag or config suppresses persistence (`grok -p --help`, README "Session Persistence"). If one exists, use it. If none does, `runGrok` removes the session dir it created, under these confinement rules (the input is untrusted output from a subprocess):
  - Run with `--output-format json` and parse `sessionId`. It must match `uuidPattern`, or nothing is deleted.
  - Build the target only as `<home>/.grok/sessions/<url-encode(cmd.Dir)>/<uuid>`: the encoded cwd is derived from pair's own `cmd.Dir`, never from Grok's output.
  - After `filepath.Clean`, refuse unless the target's parent equals that encoded-cwd dir, and refuse symlinks (`os.Lstat`).
  - Use `os.RemoveAll` only on that confined path.
  - On any refusal or error, leave the residue and emit an adapt `slug-parse` near-miss with the reason. The slug result is still returned; cleanup failure never fails the slug.
  - Task 9's fake-binary test feeds hostile IDs (`../x`, an absolute path, a non-UUID, a symlinked dir) and asserts that nothing outside the confined dir is touched.
- Fixtures are versioned test data; only the newest `grok/<version>/` dir is kept.

### Operating envelope (ARCH-CONSTRAINTS)

- `updates.jsonl` grows with every turn, and tool output is inline (`tool_call_update.content`/`rawOutput`). The scanner reads it through the shared `frameJSONLArtifact` with `unlimitedRecordSize`, the same framing and budget as claude/qoder/codex transcripts. A record has no per-line cap, which matches the writer (Grok has none either). Incremental inventory reads only the appended tail, as for the other harnesses. No new budget.
- `grok -p` slug latency is bounded by the shared `Request.timeout()`, like every `runX`. A timeout yields no slug (existing degradation). The headless probe measured ~2 s per `-p` call.

### Non-goals

Grok subagent transcripts (resume targets root sessions only; `subagents/` contents are skipped silently until a capture shows their shape), Couch peer delivery (follow-up issue), the context-meter usage surface, `--minimal` rendering, remote and worktree sessions.

---

## Chunk 1: M1 — usable in pair (registry, launcher args, TTY)

**Boundary:** `pair-dev grok` launches Grok inline with the Return remap active and pickers handled. The registry is joined. Couch can launch Grok (not resume it). `make test` is green with `sessionInventoryKnownGaps["grok"]` present.

### Task 1: Join the registry

**Files:** `cmd/internal/launcher/agent_defaults.go:19`, `cmd/internal/launcher/agent_defaults_test.go`, `cmd/internal/sessioninventory/model.go` (enum), `cmd/internal/sessioninventory/runcli.go` (`supportedAgents`, usage), `cmd/internal/sessioninventory/testdata/golden/cli-result-matrix.json`, `cmd/internal/launcher/agent_parity_test.go:17`.

- [ ] Extend `TestAgentInventoryIsTheSingleDefensiveHarnessSet`'s `want` with `"grok"`. Run `go test ./cmd/internal/launcher -run TestAgentInventoryIsTheSingleDefensiveHarnessSet`; expect FAIL.
- [ ] Add `AgentGrok Agent = "grok"`, the launcher row, and the session-side `supportedAgents` row (the usage line derives from it).
- [ ] Add `sessionInventoryKnownGaps["grok"] = "#410 M1: scanner, watcher and ledger membership land in M2"`. Name only sites the parity test probes. The orientation interim (M3) is recorded in the issue Spec, not in this message.
- [ ] `go test ./cmd/internal/launcher ./cmd/internal/sessioninventory ./cmd/internal/sessionwatch ./cmd/internal/sessionledger ./cmd/internal/couchcore ./cmd/internal/couchtty`. Extend each test that pins the agent roster (the failure names it, e.g. the golden CLI matrix's known-gap row) in this same commit.
- [ ] Commit `#410 M1: grok joins the agent registry (known gap: session side)`.

### Task 2: Resume spellings + fresh-launch table

**Files:** `cmd/internal/resumeform/resumeform.go`, `cmd/internal/launcher/fresh_args.go`, `cmd/internal/launcher/fresh_launch_test.go`, `cmd/internal/resumeform/*_test.go`.

- [ ] Test strategy (failing first): the fresh validator is attacked where its parsing is subtle: optional-valued `-r`/`-w`, value letters ending a cluster (`-pr` is `-p r` and accepted), and forbidden letters hidden mid-cluster. Use one `TestValidateFreshAgentArgs` table for those shapes, plus a grok seed set in the existing fresh-args fuzz (clusters drawn from `cspmwr`) asserting that the validator never accepts argv that `resumeform.Selector` or the forbidden sets would flag.
- [ ] `resumeform` grok row: `Space: --resume, -r`, `Inline: --resume=, -r=`, `Glued: -r`. `TestEveryTableSpellingRoundTrips` covers it automatically. Run it.
- [ ] `freshAgentSpecs["grok"]`: `forbiddenFlags: "--continue --session-id --fork-session"`, `forbiddenShort: "cs"`, `valueShort: "mpw"`. `freshValueOption` grok: `--agent --agents --allow --cwd --debug-file --deny --disallowed-tools --json-schema --leader-socket --model --max-turns --output-format --single --permission-mode --prompt-file --prompt-json --reasoning-effort --effort --rules --sandbox --system-prompt-override --system-prompt --tools --worktree --worktree-ref --ref -m -p -w`. `--worktree`/`-w` take an optional value: skip the next arg only when it doesn't start with `-` (extend the qoder guard at `fresh_args.go:112` to read a per-agent optional-value set rather than adding a second `agent ==` branch).
- [ ] `go test ./cmd/internal/launcher ./cmd/internal/resumeform`; commit `#410 M1: grok resume spellings + fresh-launch selectors`.

### Task 3: Shared inline mode + before-`--` placement + resume compose + mint

**Files:** `cmd/internal/launcher/agentargs.go`, `createflow.go:639-660`, `runcli.go:29`, `runtime.go:289`, `agentargs_test.go`, `createflow_test.go`, `peer_launch_test.go:59`, `createlogic_test.go`.

- [ ] Test strategy (failing first): every launcher-inserted token (inline flag, minted ID, resume token) must land before the first `--` and must be idempotent across Alt+n restarts. One property test ranges `AgentInventory()` × {no `--`, `--` mid, `--` first} × {fresh, resume, restart} and asserts that nothing pair inserts follows `--` and that no flag appears twice. The existing codex couch row flips to `--model model --no-alt-screen -- prompt` (the codex bug fix). Mint collision retry for grok mirrors `TestRunLaunchForcedCreateQoderMintProbesQoderSessions`.
- [ ] Implement:
  - `insertBeforeDoubleDash`.
  - `inlineModeFor`/`inlineModeArgs`, replacing `codexAltScreenArgs`.
  - `LaunchOptions.InlineOptOut` read once from `PAIR_<AGENT>_ALT_SCREEN` env (via `inlineModeFor`).
  - createflow: `if flag, _, ok := inlineModeFor(agent); ok { … }`.
  - `resumeToken`: `case "claude", "qoder", "grok"`.
  - `composeResumeArgs`: the append branch uses `insertBeforeDoubleDash`. Claude and qoder accept that order too, so they are unaffected for args without `--` and fixed for args with it.
  - `MintsSessionID`: add grok, with a comment citing the measurement.
  - The mint append uses `insertBeforeDoubleDash`.
- [ ] **Context-selector spellings join the `resumeform` table** (its home: the leaf package both the launcher and sessionwatch already import). `resumeform.Form` gains `SessionID []string` (claude/qoder `--session-id`; grok `--session-id`, `-s`) and `Continue []string` (grok `--continue`, `-c`). `resumeform.Strip` removes all three groups. `persistedConfigArgs` drops its separate `stripFlagAllForms(out, "--session-id")` in favor of the table. `shouldMintSessionID` asks `resumeform.HasSessionID(agent, args)` instead of `hasFlag(…, "--session-id")`, so a user-typed `-s <uuid>` both suppresses the mint (no two IDs on one command line) and is stripped from persisted config (no pinning the same ID on every relaunch). `sessionwatch.StripResumeArgs` reads the same table.
- [ ] Test strategy: `TestEveryTableSpellingRoundTrips` ranges every group of every agent through extract, strip and validate, so a new spelling is covered by construction. Add a grok `-s` row to the mint test (user `-s` → no mint) and to the persist test (`-s` and `-c` dropped).
- [ ] `go test ./cmd/internal/launcher ./cmd/internal/sessionwatch`; commit `#410 M1: per-harness inline mode before --; grok resume compose + minted session id`.

### Task 4: TTY profile + live capture (one atomic commit)

Follow pair#300 Task 9 exactly (its plan is archived; the rules it states are binding here). Summary:

- [ ] Add `"grok": {"grok", "--no-alt-screen"}` to the `commands` map in `harness_tty_live_test.go`.
- [ ] Hand-run `grok --no-alt-screen` in a real terminal and note the composer chrome (prompt glyph, rules, cursor, any OSC). Check whether Grok pushes the Kitty keyboard protocol (`CSI > … u`). That decides the newline bytes: `ESC CR` (Alt+Enter) without Kitty, `\x1b[13;2u` (Shift+Enter, muse precedent) with it, and in that case add an `assertKittyKeyboardPrecondition` row.
- [ ] Profile: `keymap{plainCR: <newline bytes>, altCR: '\r', altBS: 0x15}`, `composerGate: composerGatePositive`, `recognize: grokComposerActive`, `overlay: detectGrokOverlayOpen`, plus `progressShapes` from the live footer. Recognizer preference: a native OSC, then a `ruledBoxComposerSpec` registration if Grok paints a ruled box, then a new function.
- [ ] Iterate `PAIR_LIVE_HARNESS=grok go test ./cmd/internal/wrapcmd -run TestHarnessTTYLiveConformance -count=1 -v` until it reports `recognized`. Then capture with `PAIR_LIVE_CAPTURE_OUT=cmd/internal/wrapcmd/testdata/tty/grok/1.0.46/composer.raw`.
- [ ] Driven scenarios (`harnessTTYDrivenScenarios["grok"]`): a permission picker (`overlay.raw`), a selection/question picker (`selection.raw`) if reachable, and a `pressesReturn` scenario (retires the reaction gap). Write `metadata.json` (version string, argv, time, SHA-256 per raw).
- [ ] Register every oracle row: `TestHarnessTTYProfileRegistry`, `TestComposerReturnExpectationMatchesProfile`, and the negative/discrimination/reaction ledgers, each only for what is truly unproven.
- [ ] `go test ./cmd/internal/wrapcmd`; ONE commit `#410 M1: grok 1.0.46 live TTY capture — profile, recognizer, fixtures (atomic)`.
- [ ] **Measure and log:** submit a two-line prompt in the TUI, then count `user_message_chunk` records for it in that session's `updates.jsonl`. If there is more than one per prompt, Task 7's normalizer must join chunks per `promptIndex`. Record the finding in the issue `## Log`.

### Task 5: Overlay markers + recognizer spec

- [ ] From the captured `overlay.raw`/`selection.raw`, take the verbatim stripped picker strings and build `grokPickerMarkers` + `detectGrokOverlayOpen`. Use the shared `firstMarker`/`overlayVisible` helpers, scan before re-bounding the carry (BR-35), and keep the raw tail proxy-owned and cleared on the confirming Enter (BR-28). Avoid ordinary-English markers; any exemption must be stated and pinned with a prose-negative row.
- [ ] Failing-first tests: marker rows in `TestOverlayDetectorByAgent`, a stale-picker no-rearm test (the qoder counterpart), and a recognizer spec over the frozen captures (`composer_recognizers_test.go`).
- [ ] `go test ./cmd/internal/wrapcmd`; commit `#410 M1: grok overlay markers + composer recognizer spec`.

### Task 6: M1 boundary

- [ ] Full suite per the memory rule: `make -k test`, then scratchpad-TMPDIR `test-changelog`, then `go test ./...`. Compare any failure against main.
- [ ] Live: `pair-dev grok` in this checkout (after `make build`). Enter inserts a newline, Alt+Enter sends, the picker confirms with plain Enter, and scrollback flows. Record this in `## Log`.
- [ ] Couch interim check: while the gap is open, `ledgerRejectsAgent` makes every grok launch record parse as malformed. Launch one Couch-hosted grok thread (operator), then confirm that the thread runs, that Couch keeps serving other slots, and that `couch --recover-plan-from-sdlc` shows the row as unknown/malformed, not as a crash or a scope-wide refusal. If Couch degrades, move the launcher registry row to M2 (with `pair-dev grok` working unregistered in the meantime) and record a `## Revisions` entry.
- [ ] `sdlc milestone-close --issue 410 --milestone M1`; fix Critical/Important findings; log the verdict.

---

## Chunk 2: M2 — Couch-resumable (session side)

**Boundary:** `pair session-inventory --agent grok --json` lists real sessions as resumable roots. `AgentSessionExists`/`EstablishedSessionID` read them. A cold resume passes `--resume <id>`. The known-gap entry is deleted.

### Task 7: Scanner + event normalizer + every dispatch site (lands together, guide item 4)

**Files:** create `cmd/internal/sessioninventory/scan_grok.go` and `scan_grok_test.go`. Modify `model.go` (`validAgent` derives from the list; verify), `runtime_os.go` (root `grok-sessions` → `~/.grok/sessions`), `conformance.go` (`ScannerForAgent`), `incremental_inventory.go` (`artifactScannerShape` + both delta switches), `provider_contract.go` (`ProviderGrokACPV1 = "grok-acp-v1"`), `event.go` (`NormalizeNativeEvent` + `normalizeGrokEvent`), `target.go:197-227` (`observationNativeID`), `sessionwatch/sessionwatch.go:37`, `sessionledger/record.go:537`, `cmd/internal/artifactpath/manifest.go` (list `scan_grok.go`), and `cmd/internal/launcher/agent_parity_test.go` (delete the gap).

- [ ] Fixtures: copy one real TUI session's `updates.jsonl` (from Task 6's live run) into `testdata/native/grok/v1/grok-sessions/%2Frepo/11111111-1111-4111-8111-111111111111/updates.jsonl` (real-shaped URL-encoded cwd dir). Sanitize: placeholder UUID, `%2Frepo` cwd, text replaced, `_meta` IDs rewritten. Add a stub dir `22222222-…/summary.json` with no `updates.jsonl`, and an unrelated `events.jsonl`.
- [ ] Test strategy (failing first):
  - The scanner is attacked with what an untrusted append-only file can hold: truncated tails, foreign `sessionId`, malformed JSON, huge tool-output records. Grok joins the existing `scan_fuzz_test.go`, `native_large_record_test.go` and `TestAppendOnlyProviderConformance` tables rather than getting bespoke cases.
  - The fixture tree (real session + stub + sibling files) pins the happy path and the stub-is-silent rule in one `TestScanGrokV1`.
  - The normalizer table is generated from the mapped and ignored sets, plus one unknown kind, so the sets and the test cannot drift.
  - Dispatch completeness comes from the tables that range `SupportedAgents()` (`TestEveryAgentDispatchParity`, `TestAdvanceTargetValidationPerAgent`, `TestProviderContractFor`): adding grok to the list makes every missing arm fail.
  - `osruntime_test.go`: a present/absent pair for `AgentSessionExists("grok", …)` and `EstablishedSessionID` reading the inventory.
- [ ] Implement `scan_grok.go` on the `scan_muse.go` shape (`ScanGrok`, `scanGrokFile`, `ValidateGrokDelta`, `applyGrokRecord`, `grokPathFact` where `parts == [enc-cwd, uuid, "updates.jsonl"]`, uuid via `uuidPattern`) and every dispatch row above.
- [ ] `go test ./cmd/internal/sessioninventory ./cmd/internal/sessionwatch ./cmd/internal/sessionledger ./cmd/internal/launcher ./cmd/internal/couchcore ./cmd/internal/couchtty`. The parity test passes with `sessionInventoryKnownGaps` empty.
- [ ] Live conformance: `PAIR_LIVE_NATIVE_SESSIONS=1 go test ./cmd/internal/sessioninventory -run TestLiveNativeSessionShapeConformance -count=1 -v`. Also run `pair session-inventory --agent grok --json` against the real `~/.grok`. Log the node and diagnostic counts.
- [ ] Commit `#410 M2: grok session scanner, events, watcher/ledger membership`.

### Task 8: M2 boundary

- [ ] Full suite (as Task 6). Live: in `pair-dev grok`, run a turn, press Alt+n (restart in place), and the conversation continues. Then `pair resume <tag>` cold-resumes it. Log this.
- [ ] `sdlc milestone-close --issue 410 --milestone M2`; fix findings; log.

---

## Chunk 3: M3 — slug, glyphs, permissions, Couch, docs

### Task 9: Slug generation

- [ ] Live: `cd $TMPDIR && grok -p "Reply with exactly: ok" -m <model>`, plus `grok models`, to choose the cheapest fast model, stated as an alias if one exists. Measure persistence suppression (see Durable state) and log both.
- [ ] Failing test: `Run(Request{Agent:"grok"})` routes to `runGrok`. Use the fake-binary-on-PATH seam (`TestRunCodexCLIWithoutAPIKey` pattern), and pin argv, cwd and env, including the residue cleanup if that path is taken.
- [ ] Implement `runGrok` + `DefaultGrokModel`; add a gated live conformance `TestRunGrokLiveConformance` (`PAIR_LIVE_GROK_MODEL=1`). Commit `#410 M3: grok slug generation via grok -p`.

### Task 10: Prompt glyph — one authority, three consumers

- [ ] `grokPromptGlyphs` (+ column) lives in `composer_recognizers.go` from Task 4's capture. `orientationPromptOK` reads it (replace the per-agent `if` chain with a profile- or map-driven lookup if a third map-reading branch would be added; ARCH-DRY). `nvim/scrollback.lua` `PROMPT_PATTERN_BY_AGENT` gets a grok row, pinned by a parity test (generalize `TestScrollbackQoderPatternTracksPromptAuthority` to range the authoritative maps rather than copying it), and `nvim/scrollback_test.lua` gets rows. `distill.go` `promptGlyphChar` gets a grok row only if the consumption chain reaches grok (`model.Run` now supports it, so it does), extending `TestPromptGlyphRowsDriveBothReaders`.
- [ ] Orientation test: grok's captured `composer.raw` auto-submits (`TestOrientationCapturedComposersWithoutReturnRemap` row).
- [ ] Commit `#410 M3: grok prompt glyph (orientation, scrollback, distill)`.

### Task 11: Permission allowlist (aspect 6)

- [ ] Read `~/.grok/README.md`'s permission rules section and `grok inspect` in the repo. Register the standard set (`git`, `make`, `sdlc`, `lsof`, `zellij`) at the narrowest scope Grok supports, preferring repo-local over user scope (the qoder BR-F lesson). Measure before and after: one prompt that runs `make --version` should no longer ask. Log the evidence.

### Task 12: Docs + atlas sweep

- [ ] `rg -n -i 'qoder' README.md atlas/ doctor/ CHANGELOG.md`. Every hit that enumerates sibling harnesses gains grok in one commit; each skipped hit gets a reason in `## Log`.
- [ ] Guide (`atlas/how-to-bring-up-a-new-harness-cli.md`): add Grok to the parity list; add an orientation-glyph checklist item; note `insertBeforeDoubleDash` + `inlineModeFor` as bring-up steps; record the ACP transcript shape in aspect 3.
- [ ] `CHANGELOG.md` entry. Commit `#410 M3: docs sweep — grok joins every roster; guide gains orientation item`.

### Task 13: Follow-up issue + live smoke + close

- [ ] `sdlc issue new "Couch peer-delivery receiver profile for grok"` (leave details local, per the operator preference).
- [ ] `make build`, then hand to the operator for the Done-when smoke: standalone (`pair-dev grok`: multi-line prompt, picker, scroll, Alt+b, `doctor/doctor.sh` showing `return-remap`/`session-id`/`slug-parse` fired), then Couch (start and switch-agent menus list Grok, switching a slot to Grok auto-submits orientation, then park and cold resume).
- [ ] Log the evidence; `sdlc close --issue 410 --verified '<evidence>'` (the M3 boundary is the close; no separate milestone-close).

---

## Notes for the executor

- **Capture-first is absolute** (Tasks 4, 5, 10): no recognizer, marker or glyph without the capture beside it, and no hand-authored `.raw` byte.
- **Don't touch couch code** beyond its pinning tests. A grok-specific couch change means the registry contract broke.
- **Every per-agent dispatch reachable from `AgentInventory()`** either derives from one list or is probed by the parity test (pair#300 BR-16). Grep `qoder` after each task and confirm every hit has a grok counterpart or a logged reason.
- The fixture dir is `grok/1.0.46/`. If Grok self-updates before capture, use the new version string and say so in the plan's `## Revisions`.

---

## Revisions

### 2026-10-07 — plan-quality round 1 (sdlc change-code): PQ-1 Important + 7 Minors addressed

- **PQ-1 (untrusted-input-drives-destructive-op):** the slug residue cleanup is now confined. The ID is UUID-validated, the target is built only from pair's own `cmd.Dir`, with parent-equality and no-symlink checks, residue is left on any refusal, and the fake-binary test feeds hostile IDs (Durable state).
- **Known-gap message** names only probed sites (Task 1).
- **Couch interim claim** gets a live check, with a stated fallback (Task 6).
- **Context-selector table home** is `resumeform` (`SessionID`, `Continue` groups); grok `-s` is stripped and suppresses the mint (Task 3).
- **Test prose** replaced by one adversarial strategy per risky function (Tasks 2, 3, 7).
- **Drift signal**: documented ACP bookkeeping kinds are ignored, not near-miss. The chunk-join state location and flush rule are stated (Core concepts).
- **Fixture** cwd dir is real-shaped (`%2Frepo`).
- **Operating envelope** section added.

### 2026-10-07 — M1 execution: milestones re-cut (M1+M2 merged), deltas as landed

- **Re-cut (milestone-boundary-granularity):** the "registry joined, session side absent" interim is not a coherent state in today's tree. Every launch encodes a session-ledger record for its agent, so a registered agent the ledger rejects cannot launch through the fresh path or Alt+n (`agent_restart_test.go`). `TestFreshAgentInvocationHandsTheWatcherToTheReplacementWrap` requires a watcher for every registry agent. Adding grok to the session-side list early breaks every table that ranges `SupportedAgents()`. So **M1 now covers Chunk 1 and Chunk 2** (registry, arg plumbing, TTY, the full session side) and closes once, with no known-gap entry at the boundary. The old M3 (Chunk 3) becomes **M2**, closed by the final `sdlc close`. There are two review boundaries instead of three.
- **Ledger membership** (`sessionledger.isSupportedAgent`) landed with the registry row, for the reason above.
- **Inline mode as landed:** the plan's `inlineModeFor` is an `inlineModes` table plus `inlineModeArgs` and `InlineOptOuts` (`agentargs.go`); `LaunchOptions.InlineOptOut map[string]bool` replaced `CodexAltScreenOptOut`.
- **Selector groups:** `resumeform.Form` gained `SessionID`/`Continue` for grok only. Claude's and qoder's selectors stay in their fresh specs (migrating them would change sibling persisted-arg behavior). `persistedConfigArgs` keeps its agent-agnostic `--session-id` floor.
- **Fresh-args fuzz:** no fresh-args fuzz existed to join. `TestGrokFreshClusterRule` enumerates every cluster up to length 3 over `cspmwrv` instead. `TestFreshValidatorRefusesResumeLettersInClusters` now skips rows led by a value letter (grok's `-p` takes a value).
- **Optional-value flags** became a per-agent `freshAgentSpec.optionalValue` field, replacing the `agent ==` branches (claude/qoder behavior unchanged).
- **TTY:** Grok's composer is a full rounded box at column 2. `ruledBoxComposerSpec` gained `ruleCol` and `sideGlyph` (zero values keep every existing spec's behavior), so grok is a spec registration (`grokComposerActive`), not a new loop. Grok pushes no Kitty keyboard flags unless the terminal answers its `CSI ? u` query, so plain Return maps to `ESC CR` (Alt+Enter). `ttyFixtureVersionDir` now prefers the last dotted token, because grok's version string ends in a build hash.
- **Orientation glyph pulled forward from Task 10:** `promptGlyphAuthorities` (muse, qoder, grok) replaces the per-agent `if` chain in `orientationPromptOK`. Grok's profile sets `orientationPromptCol: grokPromptCol`.
