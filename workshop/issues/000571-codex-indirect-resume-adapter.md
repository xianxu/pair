---
id: 000571
status: codecomplete
deps: [pair#556]
github_issue:
target: slot-recovery
created: 2026-10-15
updated: 2026-10-15
estimate_hours: 5.9
card_mirror: '6f5440633b9c3e83add1a8120c87397fd49ec491' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-15T10:22:13-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
actual_hours: 15.65
---

# Keep Codex indirect resumes native; route exact-ID transcript homes

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
- [x] Validate native transcript record eligibility, replace incomplete positive fixtures, and rerun targeted/full verification for close (BR-2).

- [x] Preserve native trust/discovery distinction and fail closed on unproven worktree scope (BR-3).
- [x] Observe Codex SQLite through bounded coherent owned private snapshots with no native mutation (BR-4).
- [x] Resolve the native-selection mechanism re-plan: TL chose native compatibility, superseding the BR-5 classifier.
- [x] Delete Codex prediction/config/snapshot machinery, including the old main collector; prove indirect zero-work native pass-through.
- [x] Retain thin exact-ID home routing and baseline admission, with transcript-only tests and other-agent controls.
- [x] Update final docs/line accounting and pass targeted/race and recorded full suite; prepare fresh close and landing.

## Log

Operator decision relayed by ops:0, verified receipt b927c539-6b36-4710-b39e-8b9142e1721b. Split unassigned from #556. Existing evidence is on #556's branch in workshop/plans/000556-codex-config-probe.md and its plan revisions. Private config/read experiments establish that API is merged configuration, omits defaults/env and rejects runtime profiles; this follow-up explicitly uses read-only files instead.

### 2026-10-15 — read-only planning and machine measurement
- 2026-10-15: closed — Approved native indirect pass-through/exact-ID home scope. BR6 now checks every represented agent across typed-only, compatibility-only and mixed ledgers, preserving config/refusing Codex restart on unavailable homes while others work; measured regression RED then normal/race GREEN. BR7 shared discovery/validation routing accepted. make build and source/resource guards pass. Final recorded full suite on 9c4fe5c4ffeed3bd6c75079309735e45894c4f8a exit0 in11m56s, all recipes/packages, no failures, SHA256prefixf35132f334a1; includes main #587 and manifest conflict resolution. CI conformance green. Earlier unrelated telemetry flake is unassigned #596; final suites pass it. Only docs since suite; read-only/headless acceptance approved, no pair:0 changes or credentials/native-state writes. Admission protocol unchanged.; review verdict: SHIP

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

### 2026-10-15 — reduced scope approved; implementation resumed

TL submitted receiptd27b887d-2cdd-427e-bcb9-d0287ec22e9a approves fafc33b8 and the latest reduced-scope contract. Proceed with native indirect pass-through before probing, thin CODEX_HOME transcript routing for exact IDs, removal of all Codex selection policies/snapshots/TOML and the old main SQL collector. Keep baseline global admission untouched so #595 can replace it separately. The code/test net+150 to+400 projection is a target range, not an exact measured final result; report actual counts after implementation. No staging or native resolver process is authorized or needed.

### 2026-10-15 — zero-drift scope decision; reduced proposal awaiting review

TL submitted receipt3b2dcde8 chooses native-compatible indirect Codex resumes and existing exact-ID guarding. This supersedes the original indirect-selection Spec/Done-when and the preceding BR-5 classification proposal. The canonical plan's latest Revision contains the reduced keep/delete map, active acceptance criteria and measured-versus-projected line accounting. Production remains unchanged until that reduced scope is reviewed.

Keep only home-relative transcript inventory support needed by exact IDs and policy-free ledger/recovery identity; restore main's unchanged admission/wrapper protocol. Delete config/Git/provider/cloud/SQLite/latest/schema policy, snapshots, supporting tests/dependencies/exemptions, and main's older Codex SQL collector so it cannot reappear. Other agents remain unchanged. Relative CODEX_HOME uses proven startup cwd, never -C or remembered session cwd; unavailable provenance is not a guessed default.

Measured current Go/dependency/test patch is net+6324 lines; projected reduced patch is roughly net+150 to+400 (about5900-6200 removed), with exact signed final counts due after approved implementation. Workshop history is retained separately. The condition for revisiting is an upstream native read-only "resolve resume target" command/API returning the actual selected ID for the real invocation without opening/writing native state. Our own queries over a safe DB snapshot do not satisfy it. #595 separately owns locking after the actual ID becomes known.

### 2026-10-15 — native mechanism re-plan requested before BR-5

TL receiptab449251 requires stepping back after five fidelity defects, with no code until the Revision is seen. The canonical plan now records the native-surface survey first: no exposed read-only TUI/exec resume-to-UUID resolver was found in rust-v0.162.1. Resume enters live lifecycle; thread/list is a primitive requiring client-side configuration/CWD policy; native DB snapshot reads do not supply that policy; native StateRuntime listing initialization is writable. No Codex/app-server or credential access was used for the survey.

The plan compares native surfaces, names the missing contract, distinguishes the existing source-derived adapter from a future native resolver, and asks the TL to choose between continuing the minimal source-filter correction and retaining native compatibility until delegation exists. It enumerates every source family and persisted form, the proposed pure eligibility rule and selected-ID tests. Four RED measurements were completed atomically before the mechanism stop; production remains unchanged. Implementation and another full suite/close wait for the TL's decision; no automatic fifth patch.

### 2026-10-15 — close round3: prior fixes verified; native source filter fidelity

Close against2084334f6e1ecef502f755df75507d0443373189 returned REWORK with one new Critical BR-5 (native-selector-filter-fidelity), while disposing BR-1 through BR-4 as addressed. Removal overlays independently reproduced both the wrong-ID Git alias defect and native SHM mutation; the pinned fixes pass. Both recorded full suites were green.

BR-5 reproduces a separate native selection rule: exec currently filters provider but admits Custom/Internal sources excluded by native all_thread_source_kinds/source_kind_matches. Four private production-selector overlays selected newer custom atlas/chatgpt and internal guardian/memory_consolidation conversations instead of an older eligible cli conversation. Begin read-only tagged-source enumeration of every SessionSource variant, its persisted DB representation and TUI/exec eligibility, plus selected-ID REDs. Schema validity is separate from selection eligibility. Production changes wait for the measured rule/revision; unknown representations must not establish guarded selection. No gate bypass; close is not finalized and the issue remains working.

### 2026-10-15 — final integrated full suite passed

Recorded suite on a669a090c3b886ecf96853f3bb84e6c1e840aaeb (including main's #579) passed every shell/editor recipe and Go package, exit0 in13m7s after101s queue. Log: /Users/xianxu/workspace/pair/.git/sdlc/local-checks/a669a090c3b886ecf96853f3bb84e6c1e840aaeb/suite.log; sha256prefix3cfb5c2d56d7. Artifactpath6.860s, broadcast33.963s, couchcore489.548s, diagnostics92.226s, launcher83.033s, inventory130.357s, wrapper55.489s and workbench performance0.235s all pass. No failures or flakes. The earlier981cfb28 full suite also passed; two runs were required because main gained code, not to retry unchanged failures. Ready for fresh binary-owned close review; no approval or validation gate bypassed.

### 2026-10-15 — full suite passed; new main code integrated

Recorded suite on981cfb28fc504441a6799b8f93f6424b756c07f1 passed every shell/editor recipe and Go package, exit0 in12m29s after231s queue. Log: /Users/xianxu/workspace/pair/.git/sdlc/local-checks/981cfb28fc504441a6799b8f93f6424b756c07f1/suite.log; sha256prefix0aee61766901. Includes artifactpath7.007s, broadcast33.025s, couchcore471.007s, launcher55.906s, inventory114.576s and wrapper72.450s. No flakes or failures.

Main then contained #579 (PR355), so integrated it without conflicts as23460512ddbb8f47af8685cbf683d9a88771fd98. This new code integration requires a fresh recorded suite; start it through the ordinary machine gate before close. No custom retry, no bypass and no duplicate concurrent suite owned by this slot. The previous run completed and released its slot first. These changes are validation of the integration, not another attempt to pass an unchanged flaky head.

### 2026-10-15 — BR-4 final targeted proof and recorded-suite queue

Production/tests checkpoint9728b62d9, docs-only main integrationcbf6cf2e and final child-temp ownership981cfb28fc504441a6799b8f93f6424b756c07f1. Full inventory config/Git/schema/selector normal96.744s and race107.195s pass (go test with Test.*(CodexConfig|CodexGit|CodexLatest|NativeSelector), count1). Source audits, launcher/wrapper normal/race and final make build pass. Read-only machine config conformance passes12files51418bytes, compatibility/observation-incomplete, no selection/admission; no credentials or native process startup.

Deterministic snapshot controls cover DB/WAL replacement/truncation, checkpoint, journal creation, coarse-metadata content changes, generation reset after second DB verification, malformed WAL checksums, no SHM observation, aggregate read budgets with positive controls, cleanup failure refusing authority, and native scratch-parent aliases refused before any creation. Actual child-process env control was RED under inherited temporary directories, then GREEN with SQLITE_TMPDIR/TMPDIR overridden to the owned snapshot; parent environment unchanged. Rc/view external effects were already blocked by this host, so retain the defensive-coverage caveat.

Fresh recorded suite started on981cfb28 and is queued by the normal weighted machine gate (run75c52f80573be0b9288ea118102da3b1). No custom load polling, repeated suite or bypass. Full-suite result and fresh close review remain pending; prior green suite does not substitute for them.

### 2026-10-15 — BR-3 integration and BR-4 headline verification

BR-3 committed at bc6d485de: unknown observed current/primary/linked identities propagate whole-projection compatibility, while strict trust alias rejection and positively excluded candidates remain. Added dangling current marker regression after its separate RED. Git/config/selected-ID tests pass normally0.439s, race3.853s, no-CGO1.893s (logs /tmp/pair571-br3-git-{normal,race,nocgo}.log). The native worktrees resource joins remain exactly two counted sites.

BR-4 headline real OS launcher regressions now pass for TUI and exec: correct ID, unchanged whole native directory including file bytes/identities/mtimes and no SHM; private SQLite target and hermetic flags; scratch cleanup after success/error/cancel. Absent-WAL/non-WAL controls preserve original argv and create no reservation. Broader launcher/wrapper normal17.914s/12.682s and race23.943s/15.901s pass. Full artifactpath passes normal38.766s/race34.090s; resource audit0.650s and make build pass. The new source is non-artifact native metadata IO, matching existing native-selector classification; no Pair artifact constructor exemption was added. Snapshot mutation/budget/cleanup-failure controls and final integration/full suite remain pending; prior9164e3cd suite does not verify these new changes.

### 2026-10-15 — TL approval of BR-3/BR-4 narrowed implementation

TL receipt3303cfe5-0e36-4214-8600-2a73a507022a approves Revision0aa26c95, including stable-present-WAL only. Implement the proposed scope uncertainty policy and private snapshot boundary; absent/non-WAL/hot/changing inputs retain native compatibility. No native SHM observation, source SQLite open or new locking mechanism. Keep the whole-native-directory unchanged test as the headline regression.

Implementation ownership: the configuration agent handles Git scope; the collector agent handles private snapshot construction, budgets and inventory lifecycle; the namespace agent handles real OS launcher WAL fixtures and non-mutation/cleanup assertions. Root integrates documentation, source classification and SDLC verification. Existing Git REDs are committed at ae9ec28a1; launcher WAL RED resolves the expected ID then exposes native SHM creation and missing hermetic invocation flags. Host rc/schema protections already reject the tested external effects: those controls are defensive coverage, not an additional demonstrated unsafe baseline. No success claim until production and regression verification finish.

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

### 2026-10-15 — approved scope Revision: exact map-key vocabulary

The changed-code recorded suite on8d79c161 found a deterministic artifact inventory failure: codex_rollout_eligibility.go contains protocol map key session_id but was registered NonArtifactSources. The existing vocabulary grammar lacks direct map-key declarations. TL approved folding in one exact counted Go map-key context (function + literal + count), negative moved/duplicate/path-use controls, and reclassifying this helper as native-session VocabularyConsumer. No broad allowlist or ownership exemption. Approval delivery9521adf7. Full suite continues through every package; this is a source inventory correction, not a behavioral flake. Run the complete artifactpath package after the change; the earlier name-filtered guard selection failed to include TestProductionArtifactReferencesAreExactlyClassified.

### 2026-10-15 — exact vocabulary classification verified

A positive direct map-key vocabulary test failed before support; implemented one direct literal map-key context restricted to an explicit map composite in the named function, with the existing literal/count fence. Negative moved-function, duplicate, constructed-key and path-call cases pass. The rollout eligibility helper is now native-session VocabularyConsumer with exactly one session_id key allowed in codexNativeSessionMetaEligible; removed its incorrect NonArtifactSources entry. Full artifactpath normal15.705s/race29.096s passed, not a filtered subset (logs /tmp/pair571-artifact-vocab-{green,race}.log). No adapter runtime behavior changed. The8d79c161 full run is still finishing all Go packages, with only the pre-correction inventory failure observed so far; do not stop it or count it as green. Required recorded suite must pass on the corrected source-inventory head before publishing.

### 2026-10-15 — completed changed-code suite; final bounded attempt

The8d79c161 recorded run completed every recipe/package in12m57s; failures were TestProductionArtifactReferencesAreExactlyClassified (corrected exact map-key vocabulary) and independent TestLaunchFailureStoreReopenConcurrent (diagnostics lock timeout, then retention assertion). Log /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/fe1008b1860377ef1d9a72d918548b57.log; sha256 prefix73f371f8e9a9. Launcher54.917s, sessioninventory46.282s and wrapper81.477s passed. No package was skipped or stopped early.

Reported the new flake to hub (fallback exact ops:0 because hub alias registration refused; receipt932423e0 submitted). TL approved filing it independently and letting the already queued9164e3cd7a2150f2f8c8a59cc481bd1939dcf2ab recorded suite proceed. Filed unassigned #590 with exact head/log/first-seen evidence and unproven serialized-IO/scheduling/inherited-lock hypotheses. One unchanged targeted count10 passed5.041s; /tmp/pair571-launchfailure-lock-check.log. No diagnostics runtime change in this issue. If the queued final suite passes, close/land; if it fails on another unrelated flake, report and STOP with no further rerun until the TL's suite-wide flake review. Current final run f7e328dbf2ee719c4503f9ac7e3ab62a is queued, session46227, log under test-stats/inflight; final code9164e3cd is pushed and unchanged.

### 2026-10-15 — final recorded suite green

Final recorded suite on9164e3cd7a2150f2f8c8a59cc481bd1939dcf2ab completed every recipe and Go package exit0 in12m55s, after611s in the machine queue. Log /Users/xianxu/workspace/pair/.git/sdlc/local-checks/9164e3cd7a2150f2f8c8a59cc481bd1939dcf2ab/suite.log; sha256 prefixd0dabb1482db. Artifactpath6.860s, broadcast39.731s, couchcore466.887s, launcher53.713s, sessioninventory32.700s, wrapper77.016s. All three independent flakes passed unchanged; no further repeat needed. All CI checks on this head passed, including zellij conformance2m49s (run38827720463), merge-check1m23s and local-gates guard11s. Every code change through both Critical review fixes and the approved source-vocabulary correction is covered. Remaining changes are issue evidence only. Ready for the second binary-owned close review; no staging or native configuration mutation.

### 2026-10-15 — BR-3/BR-4 proposed Revision: discovery uncertainty and non-mutating WAL snapshots

Close round2 verified BR-1/BR-2 addressed, then raised two new Critical findings. Production edits for this revision are paused pending TL approval; private regressions already reproduce both defects.

**Git rule separation (BR-3).** Tagged [worktree repository identity and discovery](https://github.com/openai/codex/blob/rust-v0.162.1/codex-rs/git-utils/src/worktree.rs#L34-L177) canonicalizes admin pointer targets; tagged [trust discovery](https://github.com/openai/codex/blob/rust-v0.162.1/codex-rs/git-utils/src/trust.rs#L139-L151) rejects the original git-dir symlink before canonicalizing. Keep strict trust rejection, but never treat its inability to prove a discovery candidate as authoritative exclusion. Any observed current/primary/linked candidate with unsupported alias, permission/IO failure, incomplete metadata or exhausted observation becomes whole-projection compatibility. Only positively native-proven exclusions may skip: no eligible Git root in fully observed ancestry, non-directory registration, a nonexistent/non-directory corresponding cwd, a canonical corresponding cwd outside its checkout root, or positively different repository/common/relative identity. No new UID policy. The private production-selector RED selects an older ID after an admin alias hides the newer linked match; the fix must select the native newer ID or retain compatibility, never the older one. Matrix includes current/primary/linked aliases, FIFO, missing metadata, permissions and positive native exclusions; project-only API proof must still refuse a trust alias.

**Read-only SQLite boundary (BR-4; proposed narrowed proof).** Native read-only SQLite can create or update native WAL-index sidecars; [SQLite WAL documentation](https://www.sqlite.org/wal.html#read_only_databases) explicitly permits this. Never open the source database through SQLite. Add one lazy selector-owned private DB+WAL snapshot per Codex resolution attempt, reused for pagination and independent selected-content inventory checks. Generic idle/tick inventory and other agents retain existing behavior; this boundary covers all Codex adapter query paths. Use existing context/deadline and aggregate256MiB source-read ceiling, including copy AND verification bytes and metadata counts. No retry on uncertainty.

The supported snapshot requires a present regular WAL, a WAL-mode DB header, secure no-follow source handles and unchanged file identity/size/generation fences. Copy WAL first, then DB into a private0700 temporary directory with0600 files. Re-read both complete sources and compare hashes/bytes to copies, with pre/during/post identity/size/header/absence checks. A same-generation checkpoint only copies committed WAL pages into DB: replaying the stable committed WAL overlay remains the same logical image. Reset/recycle changes WAL salts/generation and full content even if inode/size/mtime coincide; reject it, replacement, truncation or any changed observation. [SQLite file format WAL sections4.3–4.5](https://www.sqlite.org/fileformat.html#the_write_ahead_log) define committed-frame/checkpoint/reset semantics; the WAL-index contains no authoritative DB content and is rebuilt privately ([WAL format](https://www.sqlite.org/walformat.html#the_wal_index_or_shm_file)). Run private recovery and bounded quick_check before selection: this verifies the recovered image, while source fences establish coherence (quick_check alone is not coherence proof).

Do NOT stat/read/copy native SHM. SQLite executes only against the private copy, with hermetic safe CLI options (-batch, explicit empty init, -safe, -readonly) so a user rc or schema file functions cannot perform external effects; unsupported required CLI behavior yields compatibility. The selector runtime owns lazy snapshot construction and Close, with deferred cleanup in every Codex collector/validation exit; partial creation, query failure, timeout and cancellation remove the private DB/WAL and any private SQLite sidecars. Cleanup failure cannot finalize guarded authority. No new journal, source lock, app-server or credential reads.

**Explicit compatibility branch requiring TL decision.** An absent WAL can appear, checkpoint and disappear between probes; timestamp/absence brackets do not prove no ABA. Correct native byte-range locking would need pinned descriptor/process ownership and introduces a second mechanism. Proposed simple boundary: absent WAL, non-WAL mode, hot journal, unsafe/oversize/changing input => native compatibility, including ordinary closed DBs with no WAL. Preserve original argv and the existing not-guarded notice, no reservation. Upgrade positive SQLite fixtures to native WAL snapshots rather than pretending DELETE fixtures prove native observation. This intentionally narrows guarded evidence while keeping native UX; no blanket config or launch refusal.

**Verification.** Private TUI/exec WAL regression already resolves expected ID then exposes a new32768-byte native SHM before the fix. After the fix the entire native directory file set, bytes/hashes, identities, sizes and mtimes must be unchanged (including absent SHM), and all SQLite query paths must target isolated scratch. Test checkpoint/reset/replacement/truncation/create-delete during copy as conservative compatibility, same-generation stable WAL overlay, unsupported/absent WAL no-reservation fallback, source byte-budget exhaustion, rc/schema external-effect rejection and cleanup after success/error/cancel. Preserve all prior CLI/schema/namespace/ownership tests, then full suite and fresh close before landing.

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

### 2026-10-15 — reduced implementation checkpoint

Change-code accepted the reduced design (CLEAN). The historical 5.90h estimate
above describes the original selector mechanism, not remaining work. Remaining
work is deletion, the home-only route, focused zero-probe controls, documentation
and fresh full verification/review; no original selector completion is planned.
The config/Git/provider/schema/snapshot machinery and TOML dependency are being
removed. The home helper regression first failed because explicit CODEX_HOME
queried HOME; the extracted helper now passes focused normal/race tests.
Original admission/wrapper protocols are restored; exact-ID routing stays thin
for #595. Root owns docs/guards; three bounded agents own home extraction,
inventory collector deletion and launcher routing. No native credential reads
or pair:0 changes.

### 2026-10-15 — reduced patch integrated and measured

Reduction committed as4cd9a91d5; merged origin/main without conflicts at
756e6233b261e51882aee68a192d1d9973e7351e. Final signed diff against integrated
main: production/dependencies +133/-114 (net+19), tests +458/-110 (net+348),
combined code/dependencies/tests net+367, within the approved +150–400 range
(previous mechanism net+6324). Product docs net+51; retained workshop evidence
net+1507 is separate. No TOML dependency, new admission field, SQL selector,
Git/resource exemption or schema policy remains.

Measured regressions first failed on physical-CWD probing, supplied Codex
metadata selecting a UUID, explicit-home HOME lookup, indirect launcher
resolution/reservation, and relative historical homes inferred from process cwd.
Final inventory selector controls pass normal0.155s/race1.192s; home controls
normal0.214s/race1.379s. Earlier whole inventory normal12.031s/race122.041s
passed before the final one-line noncanonical fence, covered by those final
selector controls. Post-main make build, complete artifactpath package4.802s
and exact slot-resource guard0.410s pass. The first artifactpath run preceded
main integration and failed only on main/branch manifest skew; it passes after
integration with no guard waiver. Fresh recorded suite is running on756e6233.

Post-main affected launcher/wrapper checks also pass: normal5.239s/3.897s,
race6.121s/5.182s (Test.*(Native|Recovery)). Protocol/parser/createflow/wrapper
baseline equality check against origin/main exits0. A race build launched
during the merge had incomplete file visibility; this stable-tree rerun passes.

### 2026-10-15 — fresh reduced-scope full suite passed

Recorded suite on756e6233b261e51882aee68a192d1d9973e7351e completed exit0
in11m29s, all shell/editor/Go recipes passed; no flakes or failures. Log:
/Users/xianxu/workspace/pair/.git/sdlc/local-checks/756e6233b261e51882aee68a192d1d9973e7351e/suite.log,
SHA256prefixf864162a0be1. Couchcore378.229s, launcher41.236s, wrapper51.147s.
Only documentation/evidence changed after the pinned suite head. PR351 CI
(conformance, merge-check, local-gates-guard) is green. Read-only/headless
acceptance is approved by the TL; no pair:0 staging is required. Proceed to
fresh full close review of the reduced scope, not a narrow fix review.

### 2026-10-15 — reduced-scope close corrections (BR-6 / BR-7)

Fresh close on1e3129bc returned REWORK: BR1–5 disposed by mechanism deletion;
BR6 reproduces whole-ledger failure leaking an unavailable Codex home into
unrelated Claude/Qoder launch authority, and BR7 reproduces recovery discovery
using the default transcript home while validation uses CODEX_HOME. Fix the
existing home-routing contract; do not revive selection policy or change grants.

Per-agent ledger projection must retain usable other-agent entries, mark the
affected Codex entry unavailable/blocked even when a prior binding exists, and
never fall back to the default Codex home. Mixed-agent saved-config/restart
regressions cover invalid and relative homes with unavailable historical cwd.

Share the small transcript-runtime construction across exact launcher, ledger,
identity, recovery discovery/validation and legacy recovery held-file checks.
Preserve each reader's explicit default home; explicit CODEX_HOME still uses
canonical directory semantics. A recovery working path is not proof of original
startup cwd: historical consumers pass unknown provenance, so relative custom
homes cannot gain recovery authority. Add discovery-to-validation custom-home
coverage, default/invalid/relative controls and other-agent isolation. Sweep
remaining recovery constructors; no config/database/credential probing.

Run affected normal/race checks and full source audits, then fresh full suite
and fresh full close. The prior green run remains recorded, not claimed as
covering these corrections. TL report a53827c7 was confirmed submitted.

### 2026-10-15 — deterministic recovery handoff verification

The actual recovery authority test reaches the unrelated final native-owner
inspection, where real live wrappers make a private fixture machine-dependent.
Use a tiny private inner validator taking that final check as an argument; the
public production entry passes CheckRecoveryNativeOwner unchanged. Launcher
fixtures exercise collection → persisted candidate claim → the same validator
with a deterministic final owner observation, asserting its exact parameters
and that unknown homes fail before it. Couchcore tests exercise actual
Read/Preserve and candidate revalidation. No exported seam, runtime ownership
policy, real-wrapper mutation or production bypass is added (ARCH-MOCK).

### 2026-10-15 — BR-6 / BR-7 corrected and integrated

BR6 behavioral RED reproduced whole-ledger failure for Claude/Qoder under both
invalid absolute and valid relative/unknown-startup Codex homes (0.193s). The
per-agent projection fix passes mixed-ledger saved-config/stateful restart,
bound/compat Codex refusal and real storage-error controls (focused0.213s;
broader launcher normal8.425s/race7.056s).

BR7 behavioral RED reproduced custom-home discovery/legacy checks missing their
root and invalid/relative homes exposing default candidates. One shared runtime
constructor now covers discovery, exact/ledger/identity, validation and legacy
held-file checks. Historical consumers pass unknown startup. Temporarily
restoring selected cwd in recovery validation made the negative control fail
(authority=nil, final owner check called); the fixed route refuses it. Actual
couchcore Read→Preserve→Revalidate passes; launcher collection→persisted choice→
authority validation passes with only the final owner observation injected.

Expanded recovery/home tests pass normal(core4.696s/launcher4.421s/inventory0.404s)
and race(5.551s/4.708s/1.530s); final Read→Preserve test normal0.916s/race2.114s.
After committing5a8c2bba7 and merging latest main at
03495ef700652896ff63f7e93e0ea3b239587b0a, make build, full artifactpath10.690s
and resource guard0.417s pass. No code conflicts or protocol changes. A fresh
recorded full suite is running on03495ef7; prior full pass remains historical.

Updated actual line accounting versus integrated main: production/dependencies
+167/-117 (net+50), tests+878/-110 (net+768), combined net+818. This supersedes
the earlier net+367 measurement: the additional code/tests close the two
reproduced home-consistency findings; production remains only net+50. Product
docs net+55 and retained workshop history net+1985 are separate. The original
selector/config/Git/schema/snapshot policy remains deleted.

### 2026-10-15 — independent telemetry timing flake, bounded retry

Full suite on03495ef700652896ff63f7e93e0ea3b239587b0a finished all recipes and
Go packages in14m15s. Only TestSessionSyncWriteTelemetryBestEffort/timeout
failed:531.972583ms against500ms atsession_sync_test.go335. All other packages
passed (couchcore541.946s, launcher42.600s, wrapper73.027s); no early stop.
Failed log /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/6664352ee3e1ee93acd32ffd5f9c3766.log,
SHA256prefixa8592e44188b. Filed independent unassigned #596 with head/log/first
seen/hypothesis, no fix or timeout change in #571. One unchanged targeted repeat
passed1.567s. TL report70f0451a confirmed submitted. Per the approved flake
rule, one recorded rerun now targets the same exact remote head03495ef7; no
code changed. Stop/report rather than an open-ended series if it fails again.

### 2026-10-15 — integrate #594 before the bounded retry

The exact03495ef7 retry waited in the machine queue and ran no recipes. While
queued, #594 landed messaging code on main. Cancelled only owned sdlc pid30019
(exit130 before lock acquisition), merged main cleanly at
7e383735e01672e3aebcfeae7a385becfc955f11, and started the single final recorded
retry there. The adapter code is unchanged; this avoids a later integration
suite. Hub report55f6456a confirmed submitted. Inflight log:
/Users/xianxu/workspace/pair/.git/sdlc/test-stats/inflight/b30420b01312c5b7d54bb341e95b5c09/run.log.
#596 now carries the retained failure path and targeted pass on main.

### 2026-10-15 — final integrated full suite passed

Final recorded full suite on7e383735e01672e3aebcfeae7a385becfc955f11 passes
exit0 in11m52s after519s queue wait. All recipes/packages passed, including
TestSessionSyncWriteTelemetryBestEffort; no failures in this retry. Log:
/Users/xianxu/workspace/pair/.git/sdlc/local-checks/7e383735e01672e3aebcfeae7a385becfc955f11/suite.log,
SHA256prefix8e81b7749442. Couchcore385.945s, couchcmd56.208s. CI conformance,
merge-check and guard are green. Only docs/evidence changed after this suite
head. Earlier telemetry failure remains filed independently as #596. All
BR6/BR7 corrections now have normal/race and final integrated full-suite
evidence; proceed to fresh full close, with no gate bypass or staging.

### 2026-10-15 — BR-6 ledger-shape enumeration

Round5 leaves BR6 open: mixed typed ledgers pass, but compatibility-only rows
never invoke parseLedger's typed callback, so unavailable Codex storage becomes
Fresh and can remove saved configuration. BR7 is addressed. Enumerate the
same rule across typed-only, compatibility-only and mixed rows, each Codex-only
or alongside Claude/Qoder, under invalid absolute and unproven relative homes.
Unavailable represented agents must retain configuration and refuse restart;
other agents and genuine storage errors retain their behavior. Valid custom/
default compatibility rows retain baseline presentation without gaining typed
resume authority.

Keep one parse and cache transcript-runtime construction once per represented
agent for this ReadLedger call. Typed projection uses that cache; the post-parse
pass checks every represented agent through the same cache before clearing and
blocking unavailable entries. Empty or unrelated ledgers do not probe Codex,
and no synthetic authority row or new admission/config policy is introduced.
Add failing saved-config/restart coverage for compatibility-only rows before
production edits, then normal/race and a new full suite before full close.

### 2026-10-15 — complete represented-agent availability rule

Expanded BR6 matrix first failed1.529s in all four compatibility-only variants
(config removal/Fresh and restart with legacy ID); typed/mixed controls passed.
One per-call native-runtime cache now feeds both typed projection and every
represented post-parse row. Invalid homes clear/block Codex compatibility rows
without probing Codex for empty or unrelated ledgers. Valid custom/default
legacy presentation and non-authority controls remain unchanged. Exact matrix
GREEN1.668s; all TestNativeInventory normal1.158s/race1.626s.

Committed35c042089 and merged main/#587 at
9c4fe5c4ffeed3bd6c75079309735e45894c4f8a. The one manifest conflict joined
our two source entries with #587's entries; no semantic code conflict. Exact
production-source classification3.589s and resource guard0.760s pass. Fresh
full suite queued on9c4fe5c4 (inflight cc4a96c418c8c120039d09d12d658c9b).
Hub reportd915a88e is indeterminate after submit, so it was not blindly resent;
the final landed/blocked outcome will get its own confirmed report.

### 2026-10-15 — complete ledger-shape full suite passed

Recorded full suite9c4fe5c4ffeed3bd6c75079309735e45894c4f8a passes exit0
in11m56s after177s queue wait. All recipes/packages passed, no failures. Log:
/Users/xianxu/workspace/pair/.git/sdlc/local-checks/9c4fe5c4ffeed3bd6c75079309735e45894c4f8a/suite.log,
SHA256prefixf35132f334a1. Couchcore350.885s, launcher42.844s, wrapper55.534s.
The integrated make build and CI conformance/merge/guard pass. Only docs since
the pinned suite head. Final production/dependencies diff vs integrated main
+180/-118 (net+62), tests+985/-110 (net+875), combined net+937; product docs
net+55 are separate from retained workshop evidence. Added regression coverage
accounts for the increase after the original reduced projection. No selector
policy, TOML, snapshot or admission-protocol changes remain. Ready for full
close review of the complete represented-agent availability rule.

### 2026-10-15 — SHIP; wait for prioritized #597 landing

Close round6 returned SHIP with BR1–7 addressed, no blocking findings;
codecomplete is bound to evidencee55a7535ceaa330f041079cd248b1089544d67eb
and reviewed head8190469bb00c2ae6b676042414434d17e8f6f18d. Integrated
main/#569 without conflicts at60ec394708f05b4c7dec2f7af7032a99b5df74da;
PR351 is updated, CI passes and publish validation accepts reviewed code.
Its required integration suite was waiting in the machine queue, with no
recipes started.

TL messagef63f8587-3f0c-4fcd-9cb2-7635fcf69351 prioritizes #597 first,
then #571 while other pair code landings wait. Stopped only owned merge
pid46926 (exit130) before suite acquisition/publication; no remaining process
in owned group46926. PR351 remains OPEN, this issue codecomplete, tree clean
before this checkpoint. Do not land before #597. When the hub confirms #597
landed: fetch/merge main, resolve and verify any conflict, sdlc pr, then
sdlc merge --yes (no bypass); it owns the required fresh integration suite.
If code conflicts change this patch, follow the close revalidation the binary
names. No pair:0 action. Final full pass before the waiting integration is
9c4fe5c4 (11m56s, f35132f334a1); the pending integration is not claimed passed.

### 2026-10-15 — prioritized #597 landed; preserve its recovery gate

TL submitted receiptf26d5382 releases the landing hold after #597
(9b5598bb2785a540e13a2b545d438b5e43f06235). Merging current main conflicts
in the source list and final old recovery-owner check. Join both source lists.
Preserve main's deliberate owner-check deletion: Couch now checks the conversation
registry from recovery_conversation_in_use.go through recovery_prepared.go and
resume.go; createflow/wrapper skip old admission only on the corresponding
Couch-hosted paths. #571 must not resurrect the old scan.

Collapse the now-obsolete private injected-owner validation wrapper into the
actual public authority method. Home/capture/record validation stays unchanged;
the handoff regression can call the public method with private files directly
now that this method no longer inspects unrelated live owners. Preserve custom/
default positive and invalid/relative negatives. This records an integration
fix, not a new selector, ownership policy or scope. Run focused normal/race,
source guards and a fresh full suite, then narrow post-close review with the
full-suite evidence before landing. No staging or operator look is required.

### 2026-10-15 — post-#597 conflict verification passed

Merged headb74920356ba24425da8c21347f7880cca1597910 passes the complete
recorded suite exit0 in11m19s, all recipes/packages, no failures. Log:
/Users/xianxu/workspace/pair/.git/sdlc/local-checks/b74920356ba24425da8c21347f7880cca1597910/suite.log,
SHA256prefix46a491fa074d. Couchcore364.167s, launcher34.590s, wrapper65.762s.
Focused public recovery authority home controls pass normal1.040s/race2.227s;
make build, full artifactpath13.100s and resource guard0.513s pass. The public
validator retains #571 home/capture checks and main's deletion of the old owner
scan; the obsolete private fake-owner seam is gone. Source-list resolution
retains both sides. No code changed after this full-suite head. Proceed to
post-close conflict review with explicit full-suite evidence, then land through
sdlc merge --yes. Read-only/headless acceptance remains sufficient; no pair:0
action, credential read or selector/admission-policy expansion.
