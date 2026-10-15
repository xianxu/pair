---
id: 000571
status: working
deps: [pair#556]
github_issue:
target: slot-recovery
created: 2026-10-15
updated: 2026-10-15
estimate_hours: 5.9
card_mirror: 'ef90521f95d32c1c536635de27e3eddfa067a423' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-15T10:22:13-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
---

# Resolve Codex indirect resumes from faithful read-only configuration

## Problem

#556 M2 adds native conversation admission. Codex indirect resume selectors retain the native CLI's existing behavior, unreserved with a one-line notice, until faithful resolution can pin one exact conversation without a UX regression. The operator requires the three primary agents (Claude, Codex, Qoder) to keep working.

## Spec

Build a faithful read-only Codex latest/indirect selector adapter. Read effective configuration from files and documented source-backed precedence: CODEX_HOME (default ~/.codex), config.toml model_provider/sqlite_home, runtime profiles, CLI config overrides and relevant environment overrides. Do not start app-server as a configuration oracle: measurement proved startup initializes/migrates metadata and lacks runtime-profile parity. Honor the effective physical Codex transcript root as well as SQLite home; current inventory roots hardcode HOME/.codex/sessions.

Use the matching Codex source's selector semantics and existing native inventory: exact cwd/agent, provider/source filters, native DB ordering, path/ID validation and complete unique resolution. Resolve once, reserve that canonical ID, and execute an explicit ID so the native CLI cannot choose a newer/different conversation afterward. Preserve unrelated flags/prompts and saved defaults. Remove #556's unreserved compatibility exception only for selectors which the adapter handles faithfully; do not introduce a human UX regression or silently guess unsupported configuration.

Verification must include read-only comparison against the real DB/configuration on this machine, without starting/resuming any real conversation or changing native files/defaults. Headless tests use private real transcript/SQLite/config shapes, profile/env/CLI precedence, custom CODEX_HOME/sqlite_home, ties/incomplete/unsupported scopes, newer arrivals after selection, and exact-ID admission at wrapper handoff. Coordinate any live acceptance through the hub/operator.

## Done when

- Ordinary operator Codex resume --last and other supported indirect selectors resolve faithfully to the same canonical ID the native CLI would select, then use native admission with explicit pinned argv.
- Runtime profile, CLI/env precedence and custom native/state homes follow verified source semantics; unsupported mappings are honest and do not silently choose a different conversation.
- Read-only real-machine comparison and headless tests establish ordering/filter/path correctness, no native/config/default mutations, and no two live writers for the resolved ID.
- The one-line unguarded compatibility notice is removed where faithful guarded resolution is available; any remaining limitations are documented.

## Plan

Canonical approved implementation design: [durable plan](../plans/000571-codex-indirect-resume-adapter-plan.md). One atomic full-flow close; named pure/IO entities, source authority, budgets and function-level test strategies are in that plan.

- [x] Resolve plan-quality findings and estimate, enter change-code.
- [x] Build source-faithful read-only config projection and inventory-owned scoped roots; API-only local bootstrap proof or native compatibility.
- [x] Build separate TUI and exec collectors with portable real config/SQLite/transcript evidence.
- [x] Reuse pin/reserve/claim with namespace-bound handoff and unchanged compatibility UX.
- [x] Verify source/machine controls, targeted normal/race and full suite; update atlas and prepare close evidence.

- [x] Match native CLI leaf replacement and prove projected worktree scope/selected ID (BR-1).
- [ ] Validate native transcript record eligibility, replace incomplete positive fixtures, and rerun targeted/full verification for close (BR-2).

## Log

Operator decision relayed by ops:0, verified receipt b927c539-6b36-4710-b39e-8b9142e1721b. Split unassigned from #556. Existing evidence is on #556's branch in workshop/plans/000556-codex-config-probe.md and its plan revisions. Private config/read experiments establish that API is merged configuration, omits defaults/env and rejects runtime profiles; this follow-up explicitly uses read-only files instead.

### 2026-10-15 — read-only planning and machine measurement

Claimed by pair:7. Ran bare sdlc start-plan to load architectural principles; deferred start-plan --issue because its automatic dependency refresh compiles, and the operator has not released the disk/build restriction. The local Ariadne checkout trails its remote, so this is a real refresh risk. Shape and publish this record from main under the claim; do not commit on the resting branch. No builds, suites, Codex app-server or real resume started.

Installed npm metadata identifies Codex0.162.1. Read matching upstream rust-v0.162.1 source into bounded scratch files, plus #556's explicitly referenced probe and admission branch using git show only. Findings: profile-v2 uses NAME.config.toml above config.toml; legacy profile/[profiles.NAME] is not the runtime-profile implementation. Configured sqlite_home beats CODEX_SQLITE_HOME; defaults are Codex home and provider openai. Custom CODEX_HOME canonicalizes an existing directory and changes physical transcript scope independently of sqlite_home. Trusted project config may set sqlite_home but cannot set provider; higher managed requirements/layers must not be silently ignored.

Private real-DB measurement: cloned only state_5.sqlite and its WAL to a temporary directory on the main data volume using cp -c; source identity/size/mtime were stable before/after the bounded copy. SQLite opened only the copies in mode=ro/query_only; no source SHM opened or copied. The temporary snapshot was removed after measurement. Native config was parsed in memory, selecting no explicit provider/sqlite/profile override; CODEX_HOME and CODEX_SQLITE_HOME unset. The real DB snapshot contains2928 threads and59 migrations, with required selector columns. Interactive sources:214cli,22vscode;1696exec;996other structured. Two providers present. Source-derived latest query in this slot and main checkout each found a unique timestamp winner; each existing rollout lies in the selected native root and its session_meta ID agrees with the DB. No contents, full IDs, credentials or native path listings retained in this record. This compares source-derived query behavior with the real DB, not a claim to have run native selection or the future adapter.

Matching TUI latest requests state-DB-only first, UpdatedAt, limit1, active/visible threads and provider/source/cwd filters. It may fall back to scan-and-repair and eventually StartFresh. We must retain native passthrough when read-only evidence is absent/incomplete rather than fabricate absence. UpdatedAt with no relation does NOT request an ID tie breaker: equal maxima cannot be assigned invented ID ordering. A research suggestion to add ID DESC was rejected after inspecting the conditional in state/runtime/threads.rs:1270-1275,1579-1583.

Source links (matching tag):
- [Config/profile layering](https://github.com/openai/codex/blob/rust-v0.162.1/codex-rs/config/src/loader/mod.rs#L270)
- [SQLite precedence](https://github.com/openai/codex/blob/rust-v0.162.1/codex-rs/core/src/config/mod.rs#L4112)
- [CODEX_HOME](https://github.com/openai/codex/blob/rust-v0.162.1/codex-rs/utils/home-dir/src/lib.rs#L13)
- [TUI latest selection](https://github.com/openai/codex/blob/rust-v0.162.1/codex-rs/tui/src/lib.rs#L823)
- [Conditional ordering](https://github.com/openai/codex/blob/rust-v0.162.1/codex-rs/state/src/runtime/threads.rs#L1269)
- [History provider selection](https://github.com/openai/codex/blob/rust-v0.162.1/codex-rs/tui/src/app_server_session/provider_selection.rs#L40)
- [Official configuration reference](https://learn.chatgpt.com/docs/config-file/config-reference)

## Revisions

### 2026-10-15 — implementation and adversarial controls

#556 landed bcf73458 and the hub released integration (eeabc985). Merged main, materialized the canonical plan and cleared full-flow change-code after PQ-1/PQ-2 were addressed; estimate5.90h adopted after plan-quality CLEAN. Later estimate review was INFO only. Implementation uses a source-derived local forced-login-api bootstrap as positive evidence, never credentials or a caller completeness flag.

ARCH-DRY/ARCH-SECURE: shared bounded osfs reads enforce no-follow/nonblocking/identity checks for config and Git metadata; inventory owns CODEX_HOME for selector, ledger reuse, binding validation and recovery. Scope travels on existing admission grants and is reprojected before Claim without repeating latest selection. Typed scope is <=8KiB; existing grant limit increased16KiB→64KiB to carry the maximum encoded scope (bounded round-trip regression); durable store remains1MiB. Config total is tightened8MiB→4MiB, with64 reads,128 ancestry and256 worktree registrations. Limits fail to native compatibility, never produce selection authority.

Behavioral RED/GREEN: real private config/profile/SQLite/transcript integration failed against the old OS resolver through a Go overlay (both TUI and exec unsupported), then passed with scoped resolution/pinning. Distinct adversarial REDs covered CLI comment/newline keys forging API-only proof, nonexistent providers, Git-admin aliases inheriting trust native rejects, OFFSET skipping a candidate after an earlier row disappears, transcript mutation of the selected or excluded higher candidate, and custom-home ledger/recovery reads silently using the default root. Fixed literal native dotted-key grammar, provider membership, original Git path validation, timestamp keyset pagination and snapshot fences. Neither UID restrictions nor ID tie-breakers were invented.

Final integrated targeted normal tests passed launcher11.970s, wrapper19.894s, sessioninventory32.126s and selected authority guards0.149s (/tmp/pair571-integrated-final.log). Final integrated race passed launcher12.310s, wrapper20.839s, sessioninventory137.805s (/tmp/pair571-integrated-race.log). Agent config/Git/osfs race checks passed1.517s/1.179s; no-Cgo config/Git passed0.658s. The broader artifact inventory initially rejected new unclassified files; they are now explicitly recorded as non-Pair-artifact sources, and full artifactpath/osfs rerun passed28.243s/0.074s. Full suite and close review remain pending; no completion claim. PR #351; recorded suite queued on1e76b04c.

Explicit real-machine read-only config conformance opened only named noncredential files,12 bounded reads/47390 bytes across TUI and exec, no native startup/selection/admission. Both remain compatibility: observation-incomplete (secure project-file observation), stricter than the previously source-derived cloud-unknown result; no forced API-only policy was supplied. Read-only Lstat classification of ancestor .codex/config.toml paths confirms one symlink and one regular file: the shared reader refuses the static symlink before open (not ELOOP after open). Earlier source-DB private-snapshot controls remain conditional selection measurements, not guarded support on this machine. No native paths or contents printed.

### 2026-10-15 — final Couch argv regression

Source sweep found Couch injects --disable shell_snapshot after latest projection; wrapper reprojection initially rejected it. A real private API-only scope regression failed before the fix with configuration unavailable before Claim, then passed after allowing this exact selector-independent native flag alongside worktrees. Unknown feature names remain compatibility. Normal config/admission tests passed0.282s/4.417s; race1.298s/5.426s. The1e76b04c full-suite attempt was canceled while queued (exit130, no tests executed) so it cannot be confused with verification; queue the corrected code through the normal gate.

### 2026-10-15 — acceptance decision

TL receipt26e39323-7dfb-4957-9f86-82b9defff289: read-only acceptance suffices. This machine remains native-compatible, so a live check would only repeat unchanged pass-through; private API-only fixtures prove the guarded path. Finish the recorded suite onb32d23d8, close and land on SHIP. No pair:0 staging. Local make build passed (/tmp/pair571-build.log).

### 2026-10-15 — final source portability control

Independent exact-tag path-utils comparison confirms physical cwd canonicalization for exec; no nested-symlink mismatch on macOS. It also exposes WSL-only ASCII casefold under /mnt/<single-letter-drive>. A pure platform/path guard keeps that Linux physical cwd domain on native compatibility without extra environment/file probes; otherwise a newer native match could be skipped in favor of an older target. Existing macOS acceptance is unchanged. The b32d23d8 suite was canceled while still queued (exit130; no tests ran) to verify this final safety boundary on the actual final code, not a superseded head.

Final config/collector/selector normal tests passed16.770s and race28.153s; pure platform/path table0.179s; full artifactpath ownership25.701s. No further code edits pending; queue the final full suite next.

### 2026-10-15 — ordinary compatibility failure class

Pre-close launch-boundary audit found ordinary Codex latest refused adapter errors and scope encoding limits, contrary to the approved native fallback contract. Behavioral REDs cover adapter error/deadline/canceled and a real >8KiB scope: code1/no launch before the fix. Strict ResumeRequired, validated recovery and exact-ID controls retain refusal. The fix must preserve the original ordinary invocation and make no reservation; read-only resolution has no uncertain write to recover. The a1839f7f suite was canceled while queued (exit130; no tests executed), so verification must use the corrected head.

Fixed at the launch boundary: ordinary error/deadline/canceled/oversize/missing-scope keeps exact argv, launches once, publishes no native identity and makes no reservation; strict recovery/required/explicit counterparts refuse. Final launcher normal/race regex Test.*(CodexConfig|CodexLatestCreate|NativeResume|NativeCreate) passed7.746s/10.736s; includes existing native resume/create tests as well as the new matrix. Diff check clean.

### 2026-10-15 — full-suite classification finding and main integration

Recorded suite on12223899 completed every shell/editor prerequisite and all Go packages (the failure was in the final go test ./... recipe, so no remaining recipes/packages were skipped). Sole failure: TestEverySlotStateSiteIsAResource incorrectly classifies two native Git common/worktrees joins in codex_git_scope.go as Pair slot containers. No functional test failed. Runtime11m54s; log sha256:091ad05cf7b8, /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/2d5391b0bdfc0302904c3fd66f568868.log.

Merged origin/main after #573 landed, cleanly with no conflict resolutions (ba943894); integrated-head suite is running and will finish, not be stopped at the known classification finding. Corrected only the audit: exact native Git source/function/base-expression/component/arity exceptions, counted exactly once, with negative unrelated-slot and stale/changed/duplicate controls. Behavioral RED preceded fix; targeted normal0.583s/race4.402s now pass. Production code unchanged. Final fresh suite is required on the corrected audit head before landing; source ownership verification now explicitly includes both artifactpath and the independent slot-resource audit.

The ba943894 integrated suite completed all recipes/packages too: same sole pre-fix TestEverySlotStateSiteIsAResource failure, no other failures. Runtime14m15s, log sha256:d8d2bdbd2144 (/Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/b0cc324146edc517b8e0a1e7ab52f320.log). Corrected-head suite8bd4baa1 is now running after202s queue wait, log /Users/xianxu/workspace/pair/.git/sdlc/test-stats/inflight/b2dd3d164d5caad0fd4f624661697ade/run.log. Do not count either earlier failure as a passing suite or stop the final run early.

### 2026-10-15 — separate broadcast flake

Corrected-head recorded suite8bd4baa1 passed couchcore (407.388s), launcher (40.931s) and sessioninventory (33.836s), including the fixed inventory audit. While remaining packages finish, its only observed failure is TestQuickTunnelAPIFailureIsNotAURL: early fake-process exit reported no output instead of its stderr failure. Unchanged isolated count10 passed1.497s. At the TL's direction filed independent unassigned #585 with exact tested head, run log, first-seen date and the concurrent Wait/stderr-drain hypothesis; no broadcast fix belongs in #571. After the full run finishes, rerun the recorded suite unchanged on8bd4baa138c63cda357c3a447ddb2cba294f1df7 under the approved flake rule.

The8bd4baa1 run completed all recipes and Go packages in13m3s, with only the #585 broadcast failure. Saved log: /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/b2dd3d164d5caad0fd4f624661697ade.log; sha256 prefix40d6d761bed7. The unchanged recorded rerun is now executing on exactly8bd4baa138c63cda357c3a447ddb2cba294f1df7 (run119840be5020067f99a3413fd951e608), after #585 publication. No tests were skipped or aborted.

### 2026-10-15 — independent restoration observation flake

The unchanged recorded rerun on8bd4baa1 stopped in test-review after4m33s, before Go packages: tests/review-observation-test.sh Python line55 received branch changed during restoration; invoke Alt+C again in its repeated checkout/request loop. Log /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/119840be5020067f99a3413fd951e608.log, sha256 prefix58ed1e657115. Unchanged isolated shell test passed; the adapter changes none of that observation/controller/test code. Filed unassigned #586 with the first-check evidence and unproven incomplete-wanted-observation hypothesis. Preserve restore authorization. Started make -k to complete remaining recipes/packages and queued another recorded suite on the unchanged8bd4baa1 code. No production or test fixes added for either flake. Main's intervening issue documents merged cleanly; all files changed since the tested head remain issue Markdown only.

TL confirmed the same flake rule for #586: file independent unassigned evidence, then one unchanged recorded rerun. This is run23c46148480d5d752a0e3df4ef9e4435 on8bd4baa1. If it fails on a different flake, report before a fourth recorded attempt; suite flake rate is a test-economy concern. The keep-going run has now passed review-observation-test unchanged under suite load and continues through the remaining checks.

### 2026-10-15 — full keep-going suite green; close readiness

Full `make -k PAIR_GO_TRIMPATH=-trimpath test` completed exit0, every recipe and Go package, with no failures. Log /tmp/pair571-keep-going-suite.log; SHA256 f841adcce742565c0a5e2de6bb4e644a58ea9e31a9ecf9463a3373631cfe712e. It ran from branch head0cddd02c643b709823b16950d1abcd88b956efe4, whose code is identical to8bd4baa1; subsequent commits only record issue evidence. Couchcore382.008s, launcher43.574s, inventory16.317s, wrapper82.895s. Both independent flakes passed unchanged. The recorded same-code run23c46148 has also passed review observation and entered its Go suite; await its result for the required publish status.

Checklist clarification: the final design checkbox now describes deliverables complete before close; close/landing remain governed by the binary and will be recorded as outcomes, rather than incorrectly ticking future actions. No design, scope or implementation changed. Indexed atlas and compatibility docs are current. Read-only acceptance remains the TL-approved machine path; no native startup, credential reads or pair:0 action.

### 2026-10-15 — close REWORK; source-fidelity corrections

The unchanged recorded suite on8bd4baa138c63cda357c3a447ddb2cba294f1df7 passed exit0 in11m20s, all recipes/packages. Log /Users/xianxu/workspace/pair/.git/sdlc/local-checks/8bd4baa138c63cda357c3a447ddb2cba294f1df7/suite.log; sha256 prefix954cc1b5b1d8. Both suite processes completed; no superseded suite remains running. This is pre-review-fix evidence, not a substitute for final changed-code verification.

Close round1 on73a3a6a3 returned REWORK, findings BR-1 and BR-2 Critical; no close finalized. Measured candidate actual5.55h. BR-1: native config/src/overrides.rs assigns/replaces the final dotted key, with a special structured-feature exception, whereas the adapter recursively merges every CLI assignment. Keep inter-layer merging distinct from ordered CLI leaf assignment; sweep all supported table-valued overrides and verify worktree scope and selected ID after replacement. BR-2: native serde successfully decodes whole RolloutLine/SessionMeta/TurnContextItem before trusting it; payload.cwd alone is not eligibility. Add a conservative source-backed native schema admission at the adapter boundary, including optional fields when present, or retain native compatibility when eligibility cannot be established. Do not let a malformed newer record authorize a different ID; use native-valid positive fixtures across collectors and handoff tests. Verify against tagged source before changes, add REDs, fix each class, rerun normal/race plus full suite, then reclose. Hub informed, receipt472445a6 submitted. No approved credential, native-startup or staging boundary changes.

### 2026-10-15 — review fixes implemented; targeted evidence

BR-1 REDs demonstrated stale worktree, project and provider entries after CLI replacement. Separate native leaf assignment now replaces them; independent layers retain recursive merge. All four native structured-feature exceptions have boolean/table/intermediate controls. Config/Git normal3.656s, race3.302s, no-CGO4.481s; logs /tmp/pair571-br1-config-{normal,race}.log. A private real Git/SQLite/transcript integration proves worktrees=false selects the older current-checkout ID, while a later replacing features table restores native defaults and selects the newer linked-checkout ID. Source scope sweep also found root toggles precede subcommand config overrides: that ambiguous mixed worktree-toggle case preserves compatibility without changing argv or acquiring a reservation. Selector-independent shell_snapshot remains supported after scoped config, with RED/GREEN parser and real-reader controls.

BR-2 native schema fence is confined to the new latest adapter: required envelope/SessionMeta/TurnContext fields and supported optional-present shapes are validated; unsupported complex or unknown records preserve compatibility. Native defaults/omissions remain accepted, including native session_id backfill. Positive fixtures now use valid native records. Behavioral REDs previously resolved/reserved malformed metadata in both modes and cwd-only turn context in exec; corrected handoff preserves original argv and makes zero reservations/grants/published IDs. Go JSON Unicode replacement behavior is fenced with UTF8 validity and escape-parity-aware conservative surrogate rejection, including negative controls for invalid bytes and positive Unicode/literal-escape controls. Collector/config/schema normal16.017s/race20.708s. Launcher/wrapper initial integrated normal13.549s/11.597s and race13.398s/12.754s; final injection-preserving normal8.647s/8.252s and race9.425s/10.915s passed (logs /tmp/pair571-br-final-launch-{normal,race}.log). Artifact/source and slot-resource audits passed0.153s/0.650s; build passed. Repeat machine read-only config conformance passed (12 reads49510bytes), same native-compatible outcome, no credentials/native startup. Full changed-code recorded suite and second close remain required.

### 2026-10-15 — planning under the disk restriction

Append source-backed configuration/selector measurements and a proposal before implementation. No architecture or implementation gate is waived: issue-branch preparation, canonical durable plan and change-code wait until compilation is permitted; admission integration waits for #556 on main. Preserve native passthrough for unresolved scopes, picker/names, incomplete DB evidence and nondeterministic ties instead of changing ordinary resume behavior into a refusal.

### 2026-10-15 — selector/config scope refinement

The earlier flat-cwd real-DB query is only a lower-level shape/path/ID control, NOT full native selection parity: default Worktrees widens TUI scope, while exec selects from all sources and uses latest turn_context.cwd. Plan now explicitly models both. Native UpdatedAt-only ties remain compatibility; no invented ID ordering. Config source confirms system/managed files and forced MDM values absent here; cloud-cache absence alone is insufficient, so nonsecret eligibility classification is pending the hub's answer. This proposed fallback is a fidelity boundary, never a claim that cloud policy was observed.

### 2026-10-15 — source-derived effective-scope controls

Expanded the private-snapshot measurement to the ordinary Worktrees-on scope:24 existing registered checkouts cross-checked against the same common Git directory, source-derived active/visible/interactive/provider query, unique timestamp maximum and an existing winning rollout. Independently followed exec's all-source provider query and reverse latest-turn-context cwd: its first candidate was usable, matching current cwd and the same digest-only ID. A1MiB exploratory tail bound was incomplete; a16MiB bounded read found the latest turn context. This is measurement of read budget/fidelity, not permission to ignore an exhausted budget in production. No native selector/app-server was executed; cloud eligibility remains unproved, so these are conditional local-provider evidence controls rather than complete runtime-configuration conformance.

Scratch Python measurement scripts used only stdlib/file reads plus owned temporary clone snapshots and read-only Git observations; no builds or suites. Snapshot removed on context exit. The draft plan and all discoveries are published in the issue record while branch preparation remains paused; canonical durable plan will be materialized on the issue branch before change-code.

### 2026-10-15 — final cloud and credential boundary

TL decision bed8618a-f832-428c-9b32-e697bf43e0e0: do not open Codex auth or credential files, even for classification. Updated the active plan to make them forbidden inputs and remove the pending request. Cloud eligibility is unknown unless source-backed non-credential evidence rules out a material cloud layer; unknown invokes native compatibility unchanged, with exactly one not-guarded note and no reservation. Neither an absent cloud cache nor a successful source-derived query establishes guarded support. No auth/credential file was opened during this item's measurements.

Final plan retains separate TUI and exec latest algorithms, NAME.config.toml profiles, configured sqlite_home ahead of CODEX_SQLITE_HOME, default worktree-inclusive TUI scope, isolated physical native/state homes, explicit-ID pinning and namespace checks through #556 admission. Added tests proving forbidden inputs are not touched and cloud-unknown never gains admission authority. Build/refresh/test restriction remains in force; branch preparation/durable-plan materialization waits for space and integration waits for #556 on main.

### 2026-10-15 — TL plan approval; implementation held

TL approved the source-derived plan in receipt34e0ac39-2479-4ee2-aa83-463034e5d3d1 (reviewed publication f3cd4848), including pure configuration precedence, inventory-owned native/state homes, separate TUI/exec collectors, reuse of #556 admission, native compatibility for ties/pickers/names/remote/incomplete evidence, and no invented ordering. The final credential prohibition and cloud-unknown compatibility boundary are published in7399d165 and agree with that approval. Do not build, refresh or run tests until the hub releases the scratch-space restriction AND #556 has landed on main. Approval alone does not release either prerequisite.

### 2026-10-15 — scratch space released; dependency still pending

TL receipt6e084767-8928-44a8-8bdf-2b63b4fe1d00 releases the scratch-space restriction after operator-approved cache cleanup. No #571 recorded suite exists to retry: this item remains planning-only. Fresh fetch and sdlc issue show556 --json confirm #556 is still working on pair:2, landing outcome not-landed. The other approved prerequisite therefore remains: wait for #556 on main before building/integration. No suite or refresh started merely to refill the cache.

### 2026-10-15 — plan-quality refinement before implementation

PQ-1/PQ-2 require concrete production policy provenance and named function-level test strategies. Source tracing independently proves the narrow local API-only bootstrap class for both TUI and exec excludes backend-auth cloud loading without inspecting credentials; absent proof stays native compatibility. The durable plan now names that production reader/projector predicate and positive real config fixture, plus per-function adversarial/mechanical guards. Condensed the active issue Plan to its canonical link/tasks; earlier full proposal remains in published history. No implementation before the revised gate passes.

## Estimate

Produced via brain/data/life/42shots/velocity/estimate-logic-v3.1.md against baseline-v3.1.md. Method A only, after plan-quality CLEAN (PQ-1/PQ-2 addressed). Calibration source is stale; values are provisional. Familiarity1.0: existing Go inventory/SQLite/admission seams, source investigation already bounded. A real Go TOML library (pelletier/go-toml/v2) supplies parsing, halving that greenfield primitive's design before the thorough-spec x0.2 discount. No hand-written TOML grammar.

Primitives: config projection base1.0d/0.8impl =>0.10/0.32 (library x0.5, spec x0.2, impl x0.4); config reader/policy API2.0/1.5=>0.40/0.60; TUI collector1.0/1.0=>0.20/0.40; exec collector2.0/1.5=>0.40/0.60; Git/trust scope well-specced module0.30/0.50=>0.06/0.20; namespace admission API2.0/1.5=>0.40/0.60; parser/createflow cross-cutting0.80/0.50=>0.16/0.20; real config/SQLite/handoff conformance API1.0/1.0=>0.20/0.40; docs0.20/0.20=>0.04/0.08; one close review0.20/0.50=>0.04/0.20. Design2.0+15%buffer0.30, scaled implementation3.60, total5.90h. No additional queue-wait primitive.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: greenfield-go-module design=0.10 impl=0.32
item: api-integration design=0.40 impl=0.60
item: api-integration design=0.20 impl=0.40
item: api-integration design=0.40 impl=0.60
item: smaller-go-module design=0.06 impl=0.20
item: api-integration design=0.40 impl=0.60
item: cross-cutting-refactor design=0.16 impl=0.20
item: api-integration design=0.20 impl=0.40
item: atlas-docs design=0.04 impl=0.08
item: milestone-review design=0.04 impl=0.20
design-buffer: 0.15
total: 5.90
```
