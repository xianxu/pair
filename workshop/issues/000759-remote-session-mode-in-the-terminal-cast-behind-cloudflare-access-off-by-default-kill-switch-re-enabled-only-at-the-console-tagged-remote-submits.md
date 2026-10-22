---
id: 000759
status: working
deps: []
github_issue:
created: 2026-10-20
updated: 2026-10-21
estimate_hours: 9.07
collaboration-mode: collaboration
card_mirror: '21c0d8d674fc4b06a04976c4927f34dffa530d95' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-21T01:17:25-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
---

# remote-session: 👽 mode in the terminal cast behind Cloudflare Access, off by default, kill switch re-enabled only at the console, tagged remote submits

## Problem

remote-session (ops project, operator-approved 10-20): write access to a session file instructs agents that run with
bypass permissions, so remote access is a security surface. Chrome cast and terminal cast assume the operator is at the
machine; 👽 mode assumes they're not.

## Spec

- **👽 mode in the terminal cast:** off by default (`couch --remote on` at the machine), served through the existing
  cloudflared tunnel, with **Cloudflare Access in front** (the operator's email one-time code or a passkey), so
  unauthenticated requests never reach Couch.
- **Least capability:** the remote API reaches only registered session files and side docs (through the allowlisted
  proxy) and submit; no arbitrary files, shell or lifecycle verbs.
- **Kill switch:** one action on the phone stops the tunnel, ends every remote session and writes a "remote disabled"
  mark; while it exists, Couch refuses to start remote access. **Only a keystroke in Couch's own UI at the machine**
  clears it: no CLI verb, no remote verb, and `couch --keys` must never be able to drive Couch's own UI (test it).
- **Remote submits:** header `[remote <Access email> <verified Cf-Connecting-IP>]` before the canonical submit body; console submits untagged. Preserve authoritative Receipt.Message.Origin=remote and RemoteEdited; retain the transcript as a second paper trail plus metadata audit lines per write/submit.
- **Ordinary toggle:** 👽 click starts/shows the link or stops it; couch --remote on|off equivalent. Each re-enable rotates the link passcode and invalidates old links. Off/kill/rotation revoke every session. This is separate from the host-key-reset-only kill latch.
- **Session lifetime:** no15-minute absolute or5-minute idle Couch expiry. Session ends on off/kill/logout or Access JWT expiry, bounded by Cloudflare policy; silent fresh handshake with valid Access. At most four concurrent sessions. Any authenticated allowed remote user may kill (off only).

## Done when

- Tests: no request reaches Couch without Access; the kill switch persists across Couch restarts and refuses
  re-enable from CLI, remote and `--keys`; a console keystroke clears it; remote submits carry the verified email/IP prefix and receipt metadata; console submits are untagged. Off/on invalidates old links and sessions; valid Access permits silent handshake without artificial idle/absolute timers; transcript keeps a second paper trail.
- Operator approved with changes in TL receipt7c5b26ce-a932-4fb1-8d97-21c12f04fa05; record Revisions and pass change-code before implementation. No account/network changes without operator.

## Plan

Approved durable design: [remote security plan](../plans/000759-remote-security-plan.md). Operator approval with changes is recorded below; change-code is next.

- [x] Operator approved threat model and plan with explicit toggle/header/session/transcript changes; #758 stateful-backend sequencing settled.
- [x] M1 — Verified Access identity, pure lifecycle/session policy, durable latch, bounded audit and replay state.
- [x] M2 — Exclusive protected exposure, console-only reset, CLI refusal and kill/late-completion conformance.
- [x] M3 — Trusted remote provenance through submit/accounting, typed adapter integration and private qualification, full code milestone review; operator acceptance and landing stay held afterwards.

- [ ] Tailscale pivot — qualify the installed-Mac private socket and proxy-peer boundary, then implement the approved transport revision and pass operator acceptance before final close.

## Log

### 2026-10-20


### 2026-10-21 — planning only
- 2026-10-21: closed M3 — M3 code/private qualification: BR9 configured cross-slot submits race PASS1.780s, stable-ID rename regression RED0.104s then full sessionsync race PASS23.099s; BR10 retained evidence bounded across recreated owners at N8/16; BR11 held post-handoff completion/revocation and counted cycles, full remotesession race PASS12.536s; repolint/artifactpath PASS11.442s/10.776s and reader inventory PASS. Corrected head pushed, one recorded suite queued; its pass required before landing. No stage/live acceptance claimed: operator Access/iPhone check tomorrow, #758/#760 editor step deferred explicitly.; review verdict: SHIP
- 2026-10-21: closed M2 — BR-8 RED actual killed-opening cleanup0.228s and existing-close ownership0.246s; GREEN three race repetitions1.315s/1.335s. Complete corrected broadcast/remotesession race194.955s/12.093s; remote console/CLI race2.237s/2.186s; catalog18.311s/artifactpath21.482s and staged reader inventory PASS. ErrProtectedCleanup independently retains failed opening teardown; same closing attempt is joined; canceled/failed opening plus listener cleanup controls keep Join error, blocked state and reset refusal. Approved TLbe526c06, no new authority/state; raw errors stay out of status. Prior BR-5/6/7 disposed. M3 and live Access/editor acceptance remain pending.; review verdict: SHIP
- 2026-10-21: closed M1 — Round2 disposed BR-1/2/3 addressed and found no new runtime defect; BR-4 complete delivered-entity audit now maps each declaration/classification/file correctly and future M2/M3 code explicitly planned. No runtime changes since passing1811e47e: package normal/race, signed issuer, generated controlled sessions, bounded crash-stage recovery, independent lifecycle/replay/permit invariants and catalog/artifactpath evidence retained. Foundation only, no exposure or real-nvim acceptance claimed.; review verdict: SHIP

- Authenticated dispatch 942390da-f154-4785-bd03-d6108e1ae119; card unowned, claimed and start-plan completed in pair:6. Read ops remote-session project.md and design.md before design. No project edits, Cloudflare account/configuration/network changes or live tunnel starts.
- Source audit: current production cast links have a numeric-code lock in addition to URL tokens (#495); earlier shorthand saying bearer-only was corrected to TL. View-only, pointer annotations, and Chrome drive have separate tokens/codes and capability checks. Tokens and unlock sessions have no server wall-clock TTL; URL tokens last until cast end. All link kinds share a per-link cross-IP bucket (burst5, then1/sec), rotate the numeric code after8 wrong well-formed attempts, preserve already-unlocked sessions on forced rotation, and keep URL token unchanged. The existing default quick-tunnel exposure question was reported separately to TL, not silently folded into this item's code scope.
- Read-only parallel mapping identified the host-TTY input path versus broker/wrapper keys, shared Host last-mount shutdown, the public200 readiness probe, document ID/registry lifecycle, and SubmitPath's exact first-line parsing. Plan reuses these authorities rather than exposing generic IPC or raw nvim methods (ARCH-DRY/ARCH-SECURE).
- TL793b0f54-b454-4226-a0da-227c22d7a76b confirmed protected named ingress, application verification of the signed Cf-Access JWT against team keys, no quick/local fallback, separate routes and the explicit same-UID threat-model limit. Proposed plan also makes remote exclusive on the named transport, persists armed state before exposure to survive kill-publication failure, and reserves all reset authority for Couch's host-key path.
- #758 currently has no owner/design/code. The plan names a strict document backend/permit seam and asks TL/operator to settle real-nvim acceptance sequencing before implementation; fake-backed security tests alone do not establish the remote editor works.
- Awaiting one advisory fresh-eyes plan read, then send the durable plan to TL for operator review. No implementation approval inferred from this checkpoint.

- Advisory plan review returned four concrete gaps; accepted all in the first revision: settle #759 HTTP versus #758 adapter ownership before M1; atomic kill/handoff ordering including audit wait; durable conservative per-document provenance across restart/handover; validate navigation destinations before nvim opens them. Added Revisions to the durable plan. Operator approval remains pending, including provenance retirement and exclusive-tunnel tradeoffs.

- TL e205ff19-08b2-4909-be86-ea8e4d728d82 put the plan with the operator and requested explicit expiry UX. Added silent fresh Couch sessions within valid Access (idle renews on next activity), no mutation replay and no renewal across a killed epoch. Access failure asks for sign-in; the15-minute Couch bound is not a continuous-authorization limit. TL settled stateful document-backend qualification now and real-nvim drill after #758/parley.nvim#405 integration. Code remains held for the operator answer.

- TL d191c1f3-8552-4c95-8488-16bd73e45ca4: pinned proposed JSON key Receipt.Message.RemoteEdited (server-assigned boolean; true means durable remote-edit evidence), independent of direct-phone Origin=remote. Same RemoteEdited key persists in submit/delivery and owed projections; missing/false does not establish clean local provenance. Added provenance contract and Revisions to plan for ariadne#520 coordination. Code remains held.


### 2026-10-21 09:04 — operator approval received

Authenticated TL7c5b26ce-a932-4fb1-8d97-21c12f04fa05: APPROVED with changes. Updated current Spec/Done when and plan, preserving earlier decisions in Revisions: ordinary 👽 click and CLI on/off toggle with new link passcode per enable and session revocation; any authenticated remote user may kill; exact verified Access-email/client-IP remote header, console untagged, receipt Origin/RemoteEdited unchanged; remove15min/5min Couch limits, use Access JWT lifetime and silent handshake with cap4; keep transcript plus audit. Persistent kill reset remains host-key only. No Cloudflare account/network setup changes. Integrated current main before implementation; go to change-code next.

- Plan-quality round1 blocked on PQ-1 persisted-format preservation/mixed-version contract and PQ-2 named-function test strategy. Addressed the full format/function classes in the plan, reusing strictjson duplicate detection and retained raw-object patterns; no policy/scope change. Rerun change-code before implementation.

- Plan-quality round2 closed PQ-1 but retained PQ-2 for duplicate prose unit-case inventories. Swept all M1–M3 steps to reference one named-function/adversarial-class/mechanical-guard table, named actual transcript functions and added server/lint functions. No policy change or implementation.

### 2026-10-22 — shared retained-number helper coordination with #758

TL6bf10860 approves #758's additive preserve-number duplicate checker, SplitObject and EncodeObject entry points as its producer-codec prerequisite, used only by new retained-raw endpoint state. Existing strict readers and pinned equivalence tests stay unchanged. #759 already carries the sibling preserve-number checker for its new stores. Whichever issue lands second must integrate the first's shared helper, reconcile signatures/semantics and delete the duplicate; no permanent parallel scanner (ARCH-DRY). This note changes no #759 executable source or operator acceptance hold.

## Revisions

### 2026-10-21 09:04 — approved current contract

Operator approval via TL7c5b26ce supersedes original “submitted remotely” display marker and short-lived-session proposal. Exact remote header now carries authenticated email and trusted-proxy client IP; console submits untagged, metadata fields preserved. Access alone supplies session time limits, with silent handshake and four concurrent sessions. Ordinary toggles/rotation are distinct from persistent kill and host-only reset; transcript retained. See durable plan Revisions for full provenance and acceptance changes.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only. Calibration is marked stale by estimate-source; use provisionally, without inventing a replacement scale.

Scope is the complete security foundation, tests, integration and three review boundaries; #758's actual nvim adapter is excluded. The multi-concern authenticated service (policy, sessions, replay, durable authority, audit, document seam and HTTP) uses the greenfield-service primitive at design8 ×0.2=1.6 and implementation10 ×0.40=4.0. This is the high implementation end because the security interleavings and stateful tests are part of the work. Access integration uses an established OIDC verifier plus Go HTTP, not custom crypto: API design3 ×0.2 ×0.5 library shortcut=0.3, implementation1.5 ×0.4=0.6. Existing transport and provenance/codec extensions are two cross-cutting refactors, each design0.5 ×0.2=0.1 and implementation0.5 ×0.4=0.2. Console controls use TUI design1 ×0.2=0.2 and implementation1 ×0.4=0.4. Familiarity1.0 reflects existing Go/Couch patterns; the separate API integration and discovery cover the bounded new Access seam. Three milestone reviews each use design0.04, implementation0.2; docs design0.04, implementation0.08; one external API discovery uses implementation0.4 ×0.4=0.16. No implementation scaling is applied twice.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: greenfield-service design=1.6 impl=4.0
item: api-integration design=0.3 impl=0.6
item: cross-cutting-refactor design=0.1 impl=0.2
item: cross-cutting-refactor design=0.1 impl=0.2
item: tui-screen design=0.2 impl=0.4
item: milestone-review design=0.04 impl=0.2
item: milestone-review design=0.04 impl=0.2
item: milestone-review design=0.04 impl=0.2
item: atlas-docs design=0.04 impl=0.08
item: real-api-discovery design=0 impl=0.16
design-buffer: 0.15
total: 9.069
```

Design subtotal2.46h, buffered to2.829h; already-scaled implementation6.24h; total9.069h (round to9.07h). Range roughly6–12h, most likely9.07h. Operator scheduling and external CI queue waits are not active effort. The full design and test obligations, rather than code volume alone, determine the service primitive.


Estimate-quality INFO accepted: M1 and M2 close through milestone-close; M3 is the final issue-close boundary (three reviews total). The service allocation includes provenance codec/integration tests and active private/live conformance, rather than charging those twice as refactors/API discovery. Refactor rows budget only the existing-reader wiring; API discovery is only external protocol investigation. Design allowance covers effect-door/interface integration choices and operator acceptance iteration, not redoing approved policy. No extra M3 milestone-close is intended.


### 2026-10-21 — change-code passed; M1 implementation begins

Plan-quality CLEAN through round4; estimate-quality INFO accepted, central change-code exited0. Estimate9.07h derived from configured v3.1 calibration. Three boundaries clarified: milestone-close M1/M2, final close M3. Starting pure lifecycle reducer and bounded latch worker in disjoint files; no Cloudflare account/configuration/public exposure or real-nvim drill. Latch restart/failure paths and late-completion/kill ordering are first red/green obligations.


### 2026-10-21 — M1 foundation red/green checkpoint (not a milestone close)

- Added pure lifecycle transitions, exact-origin/client-IP policy, cap-four Access-lifetime sessions and bounded sequence replay. Observed missing-API RED, then GREEN. Independent generated lifecycle sequences check no authority after kill; a separate duplicate-completion RED caught accidental disposal of the already-owned live handle and now passes. Sessions cover concurrent admission, no artificial24-hour idle cutoff, JWT expiry, silent fresh handshake, CSRF/principal isolation and stream-revocation signals.
- Bounded workers implemented durable latch and signed Access verification, then audit metadata/rotation in disjoint files. Root integrated and inspected these APIs. Latch publication failures poison the owner; Arm post-rename failures reopen killed. Authorized reset/confirmed ordinary teardown may have already published off before a directory-sync error; current owner remains blocked, reopening honors the actual record and never automatically enables. No filesystem rollback is claimed. Fault seams exercise publication/append outcomes, not every internal syscall.
- Unknown-number regression found shared strictjson's intentional float64 range check. TL87a8195f-642a-4ab2-a90d-f53a95a75bce approved a purely additive preserve-number duplicate-validation entrypoint for new remote retained-raw codecs only. Existing strict entrypoints and equivalence tests are unchanged. RED missing entrypoint, then GREEN, including raw1e999/precise fractions/negative zero and nested duplicate keys. Fuzz preserve-number scanner PASS109809 executions/6.216s against independent UseNumber token oracle. Atlas documents the boundary.
- Origin evidence survives actual MarkRemote/retire/reopen writes with unknown root/per-marker bytes preserved, refuses invalid/ambiguous records and blocks after uncertain publication. Audit preserves raw unknown metadata through actual append/rotation/retention, counts8/16 intent/result pairs and poisons after IO failures. New Couch namespace paths keep latch/origin outside thread GC; initial full artifactpath run caught the missing GC disposition, now explicitly protected.
- M1 remains open: document/permit seam, supervisor integration and remaining adversarial qualification are next. No CLI/console/tunnel exposure, Cloudflare account/configuration changes, native run or live public test occurred. #758 remains unowned/open; real-nvim drill stays with its later integration.


### 2026-10-21 — M1 ready for boundary review

Added the typed document backend contract and opaque one-use effect permit. Generated/strict request tests reject forged source fields, paths as generations, case aliases, missing/null typed members and non-allowlisted operations. Stateful held backends refuse a rebind, queued work after kill, and navigation to a target forgotten after resolution. They do not open nvim or any public tunnel. Root final package race cohort PASS2.478s (remotesession),1.155s (strictjson); complete artifactpath PASS10.723s; tracked Go catalog PASS2.770s. M1 review follows committed/pushed evidence; M2 runtime owner and protected exposure are not claimed complete.


### 2026-10-21 — M1 review round1 REWORK addressed by class

Accepted BR-1/BR-2/BR-3. The plan/atlas now classify the real Sessions entropy/mutex owner as integration and identify actual pure identity predicates; no fictional SessionState remains. Added a private entropy seam plus24 reproducible128-event session models, independent ownership/revocation oracles, controlled revoke/logout/expiry handshakes and8/16 held capacity waves. Focused race repetition PASS20 runs/6.756s; no sleeps or shortened deadlines.

Every remote durable publication now reuses one exact artifactpath-resolved stage per record. Validated reopeners reclaim bounded regular stages, refuse unsafe/oversized slots, and never adopt uncommitted bytes. Tests seed16 interrupted cycles across all four records, preserve a foreign sibling and refuse symlink/directory/oversized slots through every opener. Published+staged maximum66MiB+8KiB is explicit. No glob cleanup or second filesystem framework was introduced. A first-audit-intent nested-root RED also exposed missing durable parent creation; now uses durablefile.EnsureDirectory/SyncDirectory and passes. Current class tests PASS0.244s; final package race after main fixes PASS4.200s, strictjson PASS1.191s; full artifactpath PASS23.138s. Re-run M1 review after the committed evidence.


### 2026-10-21 — M1 review round2: prior findings closed, one table-path correction

Round2 disposed BR-1/2/3 addressed and found no new runtime defect. BR-4 repeats the concept-map family: DocumentResult was grouped under protocol.go although declared in documents.go. Swept all delivered rows by actual declaration/file/effects, split pure values from integration owners, and marked all future M2/M3 wiring planned. This is a documentation correction; runtime test evidence remains on1811e47e. Re-run M1 on the corrected map, without changing code or test deadlines.


### 2026-10-21 — M1 finalized; M2 starts

M1 SHIP, boundary commit4f9bb9b371b5 pushed; no open findings. Reported to hub via8482b4b5-595b-4170-baec-0c7422404c53. M2 capture/link refinement approved by TL7b706dfa: enable refuses with the named unset-and-restart recovery while COUCH_CAPTURE_DIR is configured; CLI carries no secret link, host console only. Cheap local QR rendering planned with text-only fallback for insufficient space; no network encoder. Protected transport assigned a bounded worker; root retains supervisor/HTTP/console ownership. Project now says ariadne#520 dropped; references will be historical only, no confirmation behavior is built here. No live exposure or setup change.


### 2026-10-21 — M2 integrated checkpoint (not a milestone close)

Protected named transport, lifecycle owner, initial HTTP security surface, local CLI, host console controls and bounded local QR are implemented. All new features began with missing-API or behavior REDs; the browser GET policy RED caught the absence of Origin on normal same-origin navigation. The real Unix-origin HTTP test now passes with signed assertions and8/16 request batches requiring at most one initial JWKS fetch. Handshake refresh reuses its cookie/CSRF pair, the fifth session refuses, invalid assertions revoke presented sessions, and authenticated kill works outside the passcode and full session cap. No document route/backend is exposed yet. The8-request bound includes2 reserved kill lanes; ordinary exchanges have5-second context and socket read/write deadlines.

Transport qualification reproduced an inherited reopen race (2–4 connector opens for one lost protected owner), fixed with a per-reservation generation fence. Full broadcast PASS58.122s; final protected race PASS4.398s. Root lifecycle tests cover default off, per-enable link rotation, held late-open disposal, capture refusal, arm failure without opening, and accepted reset-before-kill persistent ordering. A separate idle-kill RED now makes durable kill publication part of the reducer's join fence. Root remotesession race PASS5.917s before the final exchange-bound adjustment; final rerun follows. Full artifactpath PASS25.907s. Cross-package targeted race PASS couchcmd1.488s, couchtty1.802s, couchmessage13.848s.

Console tests send actual host CSI-u Ctrl+Alt+R through the decoder; paste/reply/release/repeat cannot reset. A real broadcast viewer verifies capability text absent from broadcast/chrome while the operator sees it. QR encoding is cached per link/geometry with1024-byte input/185×93-cell bounds and text fallback. The remote-console-reset AST rule rejects foreign call/alias, key routing from paste and asynchronous/returned closure escape; only a real testing entrypoint receives a test exemption. Tracked Go catalog PASS3.599s after staging the new files. Shipped skill reference retrieval failed before the new command/policy section and succeeds after it; help/skill cost consistency also went RED then PASS0.362s.

M2 is still open: complete integration/ownership review, actual broker-key negative path, broader regression checks and milestone-close remain. M3 document dispatch/provenance/audit integration follows. No public tunnel, Cloudflare account/network changes, phone scan or real-nvim drill.


### 2026-10-21 — portable fixture inherited-writer incident and read-only audit

The initial portable broker/wrapper test incorrectly supplied an empty-base testpair.Environment slice to additive ptychild.Start; this overlays rather than replaces the parent environment. Two helpers reached the live-PID refusal before native-agent launch/peer runtime/ready/PID publication or signals, but earlier retention/diagnostic metadata opens had already used the inherited live identity. This was a fixture isolation defect. No live file was removed, restored or hand-edited. TL581d8392 requested the exact write list and cleanup proposals before any live removal. The read-only report below was sent to the hub and proposes preserving all live files, leaving benign stale helper entries to existing normal identity-checked pruning. No automatic live cleanup is authorized. Private test resume waits on completion of that audit, not a blind passing retry.

The class fix is direct testpair.ApplyEnvironment before the additive launcher, a new additive-pty-environment catalog rule (RED missing matcher, then GREEN including late/conditional clearing), plus an actual child poisoned-identity regression. The latter is still pending. The local rule does not pretend to be whole-program dataflow: it detects the direct Environment+ptychild.Start misuse; runtime assertions defend actual child authority.

```text
#759 inherited writer audit (READ ONLY)
Observed UTC: 2026-10-21T18:50:17.363282+00:00

Scope: first two fixture runs used testpair.Environment with additive ptychild.Start; inherited Pair identity survived. No baseline hashes exist for these runs. A later ApplyEnvironment run finished privately before the stop message; no test is running now.

Inherited authority:
PAIR_DATA_DIR=/Users/xianxu/.local/share/pair/repos/50f39cf60172c338
PAIR_TAG=1-pair-59
PAIR_SCOPE_KEY=50f39cf60172c338
PAIR_RETENTION_START_ID=2a40c2b999f854dfda330076f15bad3f
COUCH_LIFETIME_LEASE=
PAIR_WRAP_LOG=

Completed source path before refusal:
1. wrapcmd.Run -> storagegc.AcquireSelectedProcess: acquire role wrapper under selected live owner; save process registration, optional AcknowledgeStart. Deferred ReleaseProcess removes only exact returned ID on normal return. Outer fixture Child.Close can terminate helper before release; current dead helper registration remains.
2. lifetimelease.HoldFromEnv: no COUCH_LIFETIME_LEASE, so no lease open.
3. selfBuildIdentity reads test executable; resolvePaths computes live tag paths.
4. adapt.OpenBuffered and openWrapEvents -> diagnosticlog.Open: shared diagnostic metadata/registry opens; writer list saved BEFORE later content recovery/append checks. Buffered start/defer-close records are attempted, but stable content below shows no fixture append.
5. No --scrollback-log argument: inherited events pathname alone never opens/truncates raw/events. No PAIR_WRAP_LOG, so no debug writer. Agent is claude, so codex lifecycle journal branch not entered.
6. notifytransport.Start: shared namespace lock may be opened; clearDeadBinding reads live pair-wrap-pid and checks live PID46104, refuses before socket publication, PID replacement, native child spawn, peer runtime, agent ready, or watcher.

Observed exact paths:
/Users/xianxu/.local/share/pair/.retention/owners/b4bbee2925a959ce3f900b7b69204977151e4ed74688b382542a0576c8e3038a.json
  mode=-rw------- size=1011 mtime_ns=1792608343449030049 sha256=bc24b74bc62c1334a3337cf75bf5b3f7874e0dc4f524d88caa6275050961f52e
/Users/xianxu/.local/share/pair/.retention/coordinator.lock
  mode=-rw------- size=0 mtime_ns=1789355372583740462 sha256=e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
/Users/xianxu/.local/share/pair/repos/50f39cf60172c338/pair-wrap-pid-1-pair-59
  mode=-rw------- size=5 mtime_ns=1792519273559846131 sha256=5df79e007f23c9e1898413fd69da82ac47ea647d54356474ddf799e6daeb9d21
/Users/xianxu/.local/share/pair/repos/50f39cf60172c338/adapt-1-pair-59.jsonl
  ABSENT
/Users/xianxu/.local/share/pair/repos/50f39cf60172c338/adapt-1-pair-59.jsonl.pair-diagnostics.lock
  mode=-rw------- size=0 mtime_ns=1792262574590950449 sha256=e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
/Users/xianxu/.local/share/pair/repos/50f39cf60172c338/adapt-1-pair-59.jsonl.pair-diagnostics/state.json
  mode=-rw------- size=647 mtime_ns=1792608343477095325 sha256=105f832cf7ef0356130a10a634e1e54d759192c08ade887a499c2908f960a72d
/Users/xianxu/.local/share/pair/repos/50f39cf60172c338/adapt-1-pair-59.jsonl.pair-diagnostics/.append-reservation
  mode=-rw------- size=428 mtime_ns=1792354738758113171 sha256=5ee194523cfaa6b8fcd341e99eb5951350ff6c8057ad555a8bf1c5dc65261b3f
/Users/xianxu/.local/share/pair/repos/50f39cf60172c338/adapt-1-pair-59.jsonl.pair-diagnostics/.metadata-publication
  ABSENT
/Users/xianxu/.local/share/pair/.retention/diagnostics/c9ae1881130a8876949ac66fdc44890dccbad5e0465d21e73944429f391f94f4.json
  mode=-rw------- size=315 mtime_ns=1792262574603057694 sha256=b18069f615b95e7a2882e6843d5d092a2a57315ad48701ab66cc61826922e627
/Users/xianxu/.local/share/pair/repos/50f39cf60172c338/wrap-events-1-pair-59.jsonl
  mode=-rw------- size=215250 mtime_ns=1792451118573808616 sha256=6ab8eccab32a55afa7386460e2da973eb40b402645680106c1ad86bee5fdcc57
/Users/xianxu/.local/share/pair/repos/50f39cf60172c338/wrap-events-1-pair-59.jsonl.pair-diagnostics.lock
  mode=-rw------- size=0 mtime_ns=1792262574721873665 sha256=e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
/Users/xianxu/.local/share/pair/repos/50f39cf60172c338/wrap-events-1-pair-59.jsonl.pair-diagnostics/state.json
  mode=-rw------- size=564 mtime_ns=1792608343486745432 sha256=2b87fe2206cf49d421f3399908cc9af090f9780265db863705ce2005f6581edb
/Users/xianxu/.local/share/pair/repos/50f39cf60172c338/wrap-events-1-pair-59.jsonl.pair-diagnostics/.append-reservation
  mode=-rw------- size=429 mtime_ns=1792451118573384530 sha256=39fa079ea2649e3a4568973cb5b1d4af2cec1804c75d587656786ca229fdb69b
/Users/xianxu/.local/share/pair/repos/50f39cf60172c338/wrap-events-1-pair-59.jsonl.pair-diagnostics/.metadata-publication
  ABSENT
/Users/xianxu/.local/share/pair/.retention/diagnostics/607da4b23b0451f5f4559cde53bf863141e48f1039c1ed58e99a162236829c37.json
  mode=-rw------- size=333 mtime_ns=1792262574728782876 sha256=0c9697603e17bb53cee585792b0ae27a2da7f039d710c65c711d8c6391840c09
/Users/xianxu/.local/share/pair/.retention/diagnostics/registry.lock
  mode=-rw------- size=0 mtime_ns=1789521610017501600 sha256=e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
/tmp/pair-notify-501/lock
  mode=-rw------- size=0 mtime_ns=1792517306129354172 sha256=e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855

Relevant parsed current metadata (no transcript bodies):
Owner process registrations: [{"id": "d1bbccb1c0eedb485afd17110d447f47", "process": {"birth": "darwin:1792519270.611533", "pid": 44457}, "role": "launcher"}, {"id": "795dd63a5a97cae38ad62d8875c7e18f", "process": {"birth": "darwin:1792519273.410231", "pid": 46120}, "role": "draft-editor"}, {"id": "461d99c359ec1b367b122e9c52564b1f", "process": {"birth": "darwin:1792519273.384564", "pid": 46104}, "role": "wrapper"}, {"id": "2935cc1f3313367c28d5e79863a63339", "process": {"birth": "darwin:1792519274.311823", "pid": 46502}, "role": "session-watch"}, {"id": "9e30ef8507d1ed61448eef67a037fb42", "process": {"birth": "darwin:1792597478.880490", "pid": 12920}, "role": "launcher"}, {"id": "d6806df75e3fec46cb9861ecf9e788f3", "process": {"birth": "darwin:1792608343.433006", "pid": 10438}, "role": "wrapper"}]
Owner start reservations: []
/Users/xianxu/.local/share/pair/repos/50f39cf60172c338/adapt-1-pair-59.jsonl
  writers=[{"birth": "darwin:1792519273.384564", "pid": 46104, "root": "/Users/xianxu/.local/share/pair"}, {"birth": "darwin:1792519274.311823", "pid": 46502, "root": "/Users/xianxu/.local/share/pair"}, {"birth": "darwin:1792608343.433006", "pid": 10438, "root": "/Users/xianxu/.local/share/pair"}]
  current={"identity": {"device": 16777232, "inode": 770640032}, "last_write": "2026-10-18T20:18:58.758028Z", "mod_time": "2026-10-18T13:18:58.758183296-07:00", "name": "", "size": 33962, "start": "2026-10-17T18:42:54.654134Z"}
  pending=null creating=null deleting=null appending=null
/Users/xianxu/.local/share/pair/repos/50f39cf60172c338/wrap-events-1-pair-59.jsonl
  writers=[{"birth": "darwin:1792519273.384564", "pid": 46104, "root": "/Users/xianxu/.local/share/pair"}, {"birth": "darwin:1792608343.433006", "pid": 10438, "root": "/Users/xianxu/.local/share/pair"}]
  current={"identity": {"device": 16777232, "inode": 832108174}, "last_write": "2026-10-19T23:05:18.571856Z", "mod_time": "2026-10-19T16:05:18.573808616-07:00", "name": "", "size": 215250, "start": "2026-10-19T22:56:21.090975Z"}
  pending=null creating=null deleting=null appending=null

Confirmed effects and limits:
CONFIRMED: dead helper PID10438 birth darwin:1792608343.433006 is retained as wrapper registration id d6806df75e3fec46cb9861ecf9e788f3 in owner metadata and as diagnostic writer in both state.json files. ps -p10438 produced no process. These metadata files have current attempt-time mtimes. Existing owner IDs/PIDs are preserved in parsed snapshot above. No baseline means removal of any previously dead writer cannot be ruled out: diagnostic Open prunes definitelyDead entries while adding its own.
NO EVIDENCE of successful content append: adapt stable path absent; wrap-events stable path has old mtime and no TestRemoteKeys record in its complete215250-byte content. Both current last_write times, reservation mtimes, diagnostic registry entries and existing segment mtimes predate attempt. No pending/creating/deleting/appending transactions remain. This does not establish byte-for-byte baseline equality.
GC: AcquireRoleProcessTarget/ReleaseProcess do not invoke Sweep or RecoverUse; they load/save the selected owner. diagnostic Open may recover metadata-publication and pending generation transactions if present; no pending transaction is observed now and no pre-run snapshot exists, so a stale publication-stage cleanup cannot be disproved. Native gcruntime.StartAfterReady is AFTER notification refusal and was not reached. No collection/deletion of live transcript artifacts is evidenced.

Proposed cleanup per changed path (NO cleanup performed):
/Users/xianxu/.local/share/pair/.retention/owners/b4bbee2925a959ce3f900b7b69204977151e4ed74688b382542a0576c8e3038a.json: prefer leaving benign dead helper registration for normal identity-checked storagegc.Coordinator.RecoverUse. Existing exact ReleaseProcess(owner,id) exists, but do not invoke without TL approval and codec/unknown-field preservation check. Never replace raw owner JSON; preserve every real process ID and unknown field.
/Users/xianxu/.local/share/pair/repos/50f39cf60172c338/adapt-1-pair-59.jsonl.pair-diagnostics/state.json and /Users/xianxu/.local/share/pair/repos/50f39cf60172c338/wrap-events-1-pair-59.jsonl.pair-diagnostics/state.json: leave dead helper writer for normal diagnosticlog.Open definitelyDead pruning; no public exact foreign-writer prune API found. Writer.Close deliberately retains process registrations across handles and exec; it does not prune them. Do not hand-edit shared JSON or touch real PID46104/46502.
Stable logs, .append-reservation, .metadata-publication (absent), registry entries, coordinator/diagnostic/notify locks, PID binding: leave unchanged; no evidence justifies cleanup.

Current fixture correction: direct testpair.ApplyEnvironment precedes additive Child.Start, with explicit disposable HOME/workspace/namespace and no inherited PAIR_TAG,PAIR_DATA_DIR,PAIR_RETENTION_START_ID or COUCH_LIFETIME_LEASE. Root is adding a catalog guard; poisoned inherited identity regression still pending. No more fixture run until parent disposition.

Additional writer-authority interpretation:
diagnostic diskState.Writers is a collection, not a single owner. Open removes entries only when definitelyDead, then appends its own Registration if absent; PID46104 remains present in BOTH diagnostic states, PID46502 remains in adapt. The helper appended independent PID10438 writer authority; it did not replace the live PID46104 entry. Writer limit1024 is checked only after building the list; observed counts are3 adapt,2 wrap-events, so the extra entry is not near that ceiling. Unknown/unreadable identities stay (definitelyDead requires positive gone or a confirmed birth mismatch).
Observed generation/capacity fields below describe NOW, not proven unchanged fields:
/Users/xianxu/.local/share/pair/repos/50f39cf60172c338/adapt-1-pair-59.jsonl.pair-diagnostics/state.json
  {"appending": null, "creating": null, "current": {"identity": {"device": 16777232, "inode": 770640032}, "last_write": "2026-10-18T20:18:58.758028Z", "mod_time": "2026-10-18T13:18:58.758183296-07:00", "name": "", "size": 33962, "start": "2026-10-17T18:42:54.654134Z"}, "deleting": null, "limits": {"max_bytes": 67108864, "max_generations": 3}, "pending": null, "version": 2}
/Users/xianxu/.local/share/pair/repos/50f39cf60172c338/wrap-events-1-pair-59.jsonl.pair-diagnostics/state.json
  {"appending": null, "creating": null, "current": {"identity": {"device": 16777232, "inode": 832108174}, "last_write": "2026-10-19T23:05:18.571856Z", "mod_time": "2026-10-19T16:05:18.573808616-07:00", "name": "", "size": 215250, "start": "2026-10-19T22:56:21.090975Z"}, "deleting": null, "limits": {"max_bytes": 67108864, "max_generations": 3}, "pending": null, "version": 2}
No sealed field exists in the parsed current diskState. Current generation names are empty (stable pathname generation); observed current identities/sizes/last_write are historical and match the reported old stable wrap-events metadata, while adapt stable path is absent. No pending transition exists now. Without pre-run snapshots I cannot rule out prior metadata recovery or stale-writer pruning, nor assert the limits/generation fields were untouched. Added stale diagnostic writer can conservatively delay any operation requiring complete writer proof until normal positive-dead pruning; it cannot displace the retained real writer, and no new generation/seal/capacity exhaustion is evidenced.
Existing prune semantics inspected: storagegc.Coordinator.RecoverUse removes registrations only with ProcessDead, keeps unknown/live, and may also recover use/start intents; it is broader than exact helper cleanup. storagegc.Coordinator.ReleaseProcess removes exact ID only; no unknown-field preservation guarantee was established here. diagnosticlog.Open prunes definitelyDead before registering itself; Writer.Close marks the handle closed and joins maintenance, deliberately retaining all process registrations across handles/exec. Therefore leave these benign dead entries for normal supported cleanup unless TL explicitly approves a separately reviewed exact operation.
```

Audit clarification before checkpoint: diagnostic writers are a collection, not an exclusive replacement. The helper added its own entry and retained real PID46104 (and46502 in adapt); no pending generation transaction is observed. No pre-run baseline exists, so unknown earlier dead-writer pruning or stage recovery cannot be ruled out. Recommendation remains no live deletion or raw JSON rewrite. Full report was too large for Couch’s8192-byte message limit; concise exact-path disposition sent separately.


### 2026-10-21 — fixture incident disposition

TL69a142dc-87db-47a9-8fdb-263be6e98a4c (submitted receipt verified) approved leaving all live files untouched: the three dead-PID10438 registrations are retired only by normal identity-checked retention recovery / diagnostic positive-dead pruning. No hand edits and no ReleaseProcess. Preserve the audit file and this Log as evidence. Resume only private fixture work with the actual child poisoned-identity regression, then continue M2.


### 2026-10-21 — private fixture class fix and held M2 checkpoint

Corrected portable boundary test PASS normal0.658s and race1.982s. It uses the actual broker, Console operation queue, executeAgentDrive, Unix RemoteEndpoint, wrapcmd.Run and PTY, with retained Claude usage-panel captures as the only agent behavior. One positively admitted Esc reaches the private agent (exact1-byte receipt); its reset-shaped CSI-u output reaches the display without clearing the real Supervisor's killed latch or invoking the host reset closure. Unsupported Ctrl+Alt+R steps and raw escape text are refused; actual host-key reset remains separately positive in couchtty. The helper validates poisoned inherited Pair identity/retention/diagnostic fields are absent before any production wrapper startup, and the poisoned path stays empty. Direct ApplyEnvironment precedes the additive launcher; owned wrapper SIGTERM/join precedes Console teardown. Initial contaminated attempts remain the incident above, not a passing retry reclassified as isolated.

Added counted8/16 lifecycle cycles: one open/close per epoch, maximum one active connector, no double close. Cleanup error returns uncertainty and blocks reset/on. The first uncertainty-fixture assertion incorrectly counted confirmed closes as attempted closes; replaced it with the fake's explicit CloseStarted rendezvous. Supervisor race PASS1.370s. Added a seeded foreign reset caller through the actual repository scanner, foreign imported constructor rejection, and explicit late/conditional environment-clearing negatives. After staging all new files, catalog PASS4.636s. Broader checks: entire couchtty PASS45.329s; entire couchmessage PASS28.031s; targeted couchcmd compatibility/help/CLI PASS0.586s; wrapper-drive regressions PASS0.493s. Final artifactpath rerun follows the newly tracked fixture.

M2 remains unclosed. Hold the #759 branch and all milestone gates while the TL's separately authorized current-main detach profile runs in private scratch. No detach investigation artifact/code rides this issue branch; the separate profile has its own scratch evidence and hub reporting. No public remote exposure, Cloudflare account/configuration change, live phone scan or real-nvim drill.

Final artifactpath PASS17.079s after staging the boundary fixture. Preserved the exact audit report as a durable plan sidecar at workshop/plans/000759-remote-security-plan.side/inherited-writer-audit.txt as well as the Log snapshot, so a scratch-disk restart cannot erase the evidence. No live cleanup.


### 2026-10-21 19:55 PDT — M2 resumed, locally qualified, DNS hold

Authenticated TL616b174f resumed M2; returned to the preserved issue branch only after confirming the old #776 runner had exited. No code or evidence from #776 enters this branch. Console 👽 wiring remains as approved after operator retraction4d0da020; drawing/annotation is separate. TL04dd04f3 approved the explicit plan boundary: M2 exposure/lifecycle/session controls; document dispatch and audit-handoff stay M3. Revisions preserves the overlapping original row and its disposition.

Fresh verification: complete remotesession race PASS8.833s; complete broadcast race PASS134.238s; targeted remote/protected couchtty PASS0.407s and couchcmd PASS0.762s; race remote couchtty PASS3.757s and couchcmd PASS1.811s, including actual broker/wrapper/PTY negative host-reset path; message remote/drive/request controls PASS12.761s and wrapper controls PASS1.834s; full repolint PASS10.898s and artifactpath PASS10.392s. Strengthened signed HTTP kill qualification to use a second allowed email/subject without cookie or passcode while all four initiating-user sessions are occupied; focused race PASS1.412s. No production behavior changed in this checkpoint. README now documents the security-only surface and exact capture recovery, host reset, four-session cap and pending editor integration.

Fast reader audit initially correctly refused five unregistered new readers. Audited every input before recording: protected_test reads its own generated private config; documents_test contains literal invalid path input; server_test contains literal invalid routes; remote_console_reset_test parses literal snippets and scans a stateful fake repository (none read live Go sources), so local_or_literal_fixture. remote_keys_boundary_test reads retained Claude screen fixtures from ../wrapcmd/testdata/drive/claude/2.1.304, so non_go_full_fallback. Added independent per-reader certificates; inventory --check PASS after the strengthened test was staged. No scanner waiver or blanket hash refresh.

One-line M2 status report de6fcc4e and boundary question3ccb10f9 confirmed submitted. TLa9bb056d directs holding milestone-close (automatic push, no no-push flag) and all pushes until DNS returns; commit this checkpoint locally. No full recorded suite, fresh M2 review, live Access/phone drill or milestone completion is claimed. No Cloudflare config, named tunnel exposure, staging, live-state cleanup or operator-session change. Next: lift of push hold, integrate current main if the gate requires, mandatory M2 review and disposition, then M3 dispatch/provenance integration.


### 2026-10-21 20:00 PDT — DNS restored, freshness integration

Authenticated TLcd9681ea lifted the push hold. First M2 milestone-close refused before review because this preserved branch lacked main's pre-noon code (eaa758a6f); no review or milestone completion occurred. Followed its ordinary integration remedy with git merge origin/main. Resolved three additive conflicts by retaining both remote and landed entries: artifactpath NonArtifactSources includes the remote modules and #757 mockserve modules; pairCatalog includes remote reset/environment and #744 group completion rules; lint atlas retains both rule explanations plus #771. No remote lifecycle/transport policy changed. Verify combined tree before resubmitting M2.


Combined-head qualification after the freshness merge: remote/supervisor/protected race checks PASS (remotesession4.567s, broadcast4.472s, couchtty1.919s, couchcmd2.036s); full repolint20.091s/artifactpath29.699s PASS; reader inventory --check PASS. PR538 opened through sdlc pr after DNS recovery. Queue exactly one recorded affected/full-fallback check on this pushed combined code, then overlap mandatory M2 boundary review per worker rules. No full-suite result or M2 completion claimed yet.


### 2026-10-21 — M2 review round1, fixture portability and help corrections

M2 review found no production security defect. BR-5 Important identified the shared broadcast shortDir helper's false path-budget assumption: os.MkdirTemp with an empty base inherits potentially long TMPDIR, so TestProtectedUsesExistingGuardAndNamedLease refused its socket before exercising guard/lease. BR-6 Minor found the new --remote synopsis split the --notify explanation. The binary returned FIX-THEN-SHIP but did NOT finalize because BR-5 remains open; therefore fixes require another milestone-close review, not a ledger bypass.

Canceled only own queued recorded check lease-1415646751 through sdlc exact-run cancellation and joined it before edits. It never acquired suite capacity or ran tests; no test-failure/flake claim. New long-TMPDIR owned-listener regression RED0.248s (protected Unix socket path too long). New help-order regression RED0.385s. Shared shortDir now delegates to existing testpair.ShortRoot, fixing all broadcast fixture callers rather than only the reported lease test; that owner budgets canonical roots, prefers usable RAM scratch and registers cleanup. The new regression checks protected listener creation under a deliberately long parent and absence of its owned root after cleanup. Production socket-path refusal is unchanged. Moved remote help after the complete notify block. Focused protected/remote/notify race GREEN broadcast4.487s/couchcmd2.174s.

Added the narrow Pair-local broadcast-short-root catalog rule to prevent the mechanical allocator regression; tests cover inherited allocation, t.TempDir, accepted imported helper and unrelated allocators. Atlas maps ownership and syntax limits. Audited changed reader inputs before updating only their per-file certificates; complete broadcast and catalog requalification pending below.


Review corrections qualified: complete broadcast package PASS55.302s after the shared-root change; full repolint PASS17.332s and staged tracked-catalog/new-rule controls PASS4.737s; reader inventory --check PASS after individually re-auditing cloudflared/protected fixture readers and the new literal-AST rule test. Existing production oversized-path refusal controls remain in the passing broadcast set. Commit fixes/ledger/sidecar and queue exactly one replacement recorded check on the pushed corrected head. M2 remains unfinalized until the review disposes BR-5; no waiver or blind full-suite retry.


### 2026-10-21 — M2 review round2, end-to-end recovery visibility

Review disposed BR-5/6 addressed with both independent RED controls; raised BR-7 Important (failure-recovery-visibility): ErrTunnelBusy lost the explicit stop-existing-casts recovery, CLI discarded RemoteStatus.Detail, and console replaced configuration/cleanup detail with generic advice. Enumerated the class across configuration, synchronous admission/capture errors, startup, latch arm/kill/reset/stop publication and cleanup. Existing supervisor details are fixed secret-free strings; connector errors must remain classified, not interpolated.

Canceled/joined only queued lease-2853467057 before code edits; it never started suite tests. New regressions RED: busy supervisor0.198s, CLI0.360s, console0.330s (disabled, blocked and unconfigured all lost details). Preserve busy sentinel as fixed stop-existing-casts/host-reset-after-cleanup recovery; print safe Detail in CLI and keep it in console notices, retaining fallback instructions when absent. No lifecycle or reset-authority change. Broad remote/supervisor race GREEN remotesession2.656s/couchcmd2.078s/couchtty1.925s; final focused race follows the last harmless notice-format cleanup. Update atlas/revisions and resubmit the actual unfinalized milestone review without any ledger waiver.


Final focused recovery regressions on the exact corrected notice code PASS under race (remotesession1.173s, couchcmd1.587s, couchtty1.646s); complete repolint PASS8.399s and staged reader inventory PASS. No remaining local test runner. Queue exactly one new recorded check only after pushing this correction; the two prior queued checks were explicitly canceled/joined and are not result evidence.


### 2026-10-21 — M2 review round3 Critical BR-8 and approved lifecycle correction

Review disposed BR-7 addressed with independent regressions, then returned REWORK for BR-8 (transport-cleanup-confirmation): kill during held opening plus failed connector Close was discarded because no Mount returned. Join reported nil, CanReset became true and reset succeeded despite uncertainty. It also identified cancellation skipping an already-closing Host attempt. Stopped before mechanism changes, canceled only own queued lease-3273912333 through sdlc and joined it; no suite tests ran. Sent plan b8af322a and authenticated TLbe526c06 approval: preserve cleanup outcome independently, always join existing close attempt, keep existing blocked state/reset refusal, add both regressions. TL also renewed DNS push hold.

Regression RED under race: actual supervisor killed-opening sequence lost cleanup and allowed reset0.228s; held-close cancellation owner check returned nil instead of exact existing attempt0.246s. Correction: Host MountProtected returns an explicit ErrProtectedCleanup classification for uncertain cleanup, including later calls on the same uncertain Host; remoteEpoch records that outcome before started closes and includes it in cleanup even without a mount. Shared releaseIfUnwantedLocked returns the already-closing attempt. Initial controls GREEN three race repetitions (remotesession1.315s, broadcast1.335s). Added failed-opening/listener-close-failure control and strengthened existing canceled-opening error classification; this sweeps all opening outcomes, not only the reported race. Complete broadcast/remotesession race and console/CLI boundary requalification running locally; no push, review or milestone completion claimed.


Final BR-8 correction qualification PASS: complete broadcast race194.955s and complete remotesession race12.093s; console/CLI remote boundary race2.237s/2.186s; full catalog18.311s/artifactpath21.482s; staged reader inventory --check PASS after re-auditing protected fixture inputs. Test resources joined; no recorded suite remains active (all superseded queued runs were canceled exactly, without test execution). Authenticated TL345bfc6c lifted the renewed DNS hold; publish this correction and resume the mandatory M2 review/recorded check. No milestone completion claimed.


### 2026-10-21 — M2 finalized SHIP, recorded suite retained

Mandatory M2 review round4 disposed Critical BR-8 addressed, independently reproduced its RED controls, and reported no new findings. Binary finalized M2 SHIP and committed/pushed evidence c1e73a5cbd9c. Prior BR-5/6/7 are disposed. This is the approved protected exposure/lifecycle/session boundary, not whole-issue completion or public editor acceptance.

Retain exactly one recorded suite on corrected production head752d2c38 (attempt4c2a155c0f28ad76da8594b7c7d6afe0, run lease-4257384631), terminal session93845. It remains queued normally; no passing full-suite result claimed. The close and this documentation are code-identical descendants, so do not cancel the queued run for a docs-only head move. Review runner82837 exited successfully. No native fixture remains. M3 document dispatch/audit handoff, broker/receipt/transcript provenance and operator-assisted Access/phone acceptance remain outstanding. The real nvim drill follows #758 integration as approved. No public tunnel/account/network change or live deployment occurred.


### 2026-10-21 — M3 source map, plan refinement and retained suite result

Recovered the current #759 issue branch and M1/M2 evidence; previous terminal capture is unavailable (`couch --peek pair:6 --previous --all` reports source none). Branch, durable issue/plan and recorded result are the evidence, not historical screen instructions. No M3 implementation yet. TL41f4e736 dispatched the M3 plan; bounded source maps inspected actual broker/wrapper/transcript and registry/origin/ledger paths without edits or tests. The execution slice in workshop/plans/000759-remote-security-plan.md now names the exact owners, dispatch/audit ordering, compatibility refusal, retained-raw ledger changes, costs and operator-last acceptance.

TLb16a99b4 confirmed: leave the eight-turn/256 KiB peer recorder as correlation only; the complete paper trail is the native agent transcript carrying the exact delivered remote email/IP header, qualified beyond turn eight. Forget routes through Couch's single origin writer with durable intent in its existing record; retire only the exact absent ID after owned work/receipts settle; offline forget says “start Couch and retry”. No registry tombstone or independent CLI writer. The plan includes intent-only versus remote-edited distinction, lost-reply/restart and path-reuse tests. Existing #758 real-editor and #760 phone-UI boundaries stay intact; no Cloudflare/network/live-state action authorized.

Retained recorded suite completed FAILED, not still queued: attempt4c2a155c0f28ad76da8594b7c7d6afe0, run lease-4257384631, original production head752d2c38, elapsed11m54s, log SHA prefix8a38dff966af, terminal93845 exited1 (suite exit2). Failure: couchcmd.TestSkillDocumentsRecovery at messages_test.go:523 rejects the literal executable skill example couch --remote on|off|status. Other suite phases completed; no full-suite pass claimed. This is our documentation defect, not a flake: first M3 task replaces it with concrete on/off/status examples and reruns the existing regression without weakening parser/scanner. No same-head retry or resume. Report7454984c is confirmed submitted to hub. No own recorded check or review remains running.


### 2026-10-21 — M3 admitted; executable help regression corrected

TLa3a2761b approved the execution slice. Re-ran change-code on the revised durable plan: plan-quality CLEAN with no open findings; estimate-quality INFO (existing derivation accepted, integration budget tight), exit0. No bypass. Independently reproduced TestSkillDocumentsRecovery RED0.346s with the exact literal remote synopsis parse error, then replaced that executable example with concrete on/off/status commands; same existing regression GREEN0.349s. Parser/scanner unchanged. This targeted result fixes the known branch defect; it is not a full-suite pass or milestone close. Bounded origin-store, broker/wrapper and ledger work is delegated under AGENTS section3 with disjoint file ownership; root owns dispatch, HTTP and service integration.


### 2026-10-21 — forget retention proof stopped and split

Source integration found historical owners require all-ledger settlement reads. TL6b2915fd approved a bounded explicit-forget-only scan, but further analysis found receipt/ledger/audit retention can erase old uncertainty. Stopped the mechanism under the third-refinement rule; did not implement a scanner or claim its absence proves settlement. TLd789d04d split proof-based forget to unclaimed pair#795 (creation confirmed published on origin; no checkout import). M3 instead persists conservative submit_seen before remote or marked-local submit and permanently refuses that ID's forget with the #795 recovery link. Unknown or failed publication refuses before handoff; never-submitted origins retain exact-ID forget after active work is fenced. Revisions records the override and the false-positive/conservative block tradeoff. No new pending-operation design inside M3.


### 2026-10-21 — M3 implementation checkpoint, private qualification

Implemented owned dispatch/replay/rate accounting, intent-before-effect and result audit, final Permit kill fence, closed authenticated document routes and bounded session streams. The same origin owner is opened before service socket publication and borrowed by remote configuration; local submits consult it while off, after reopen and rename. A source/test-discovered startup publication race was independently RED (six races) and fixed by initialize-before-sockets plus locked reuse; targeted owner/CLI race GREEN1.634s. Conservative submit_seen blocks forget through pair#795; untouched IDs use exact registry transactions and a bounded active-operation fence, no retained settlement scanner. M3 operator acceptance remains outstanding.

Broker origin/source/RemoteEdited/DocumentID propagate through singleton, batch, receipts, compatibility and recording. Exact unsupported-wrapper refusal is relaunch-classified, never downgraded; a held capability callback cancellation regression RED prevented a late enqueue and is GREEN. Full couchmessage suite PASS26.951s; targeted provenance races PASS13.849s and1.228s. Actual broker/socket/wrapper PTY input tests PASS2.396s; combined wrapper provenance/native tests race PASS5.363s. Native-format shared fixture producers for Claude/Codex, canonical NativeRoots and production recovery reader retain all ten remote headers/bodies through the private PTY, with Pair correlation still exactly eight; independent changed-IP negative control RED then restored GREEN1.506s. This is fixture qualification, not real-agent/iPhone evidence; Qoder native fixture is unsupported. Explicit empty child environment plus poisoned live-identity guard passed; no operator sessions or network configuration touched.

Private signed HTTP -> Supervisor.Dispatch -> stateful typed backend -> actual Broker.SendRemote -> capable endpoint terminal receipt qualification PASS under race1.942s: N8/16 unique submissions plus duplicates produce exactly N deliveries/2N audit rows, exact verified identity/IP/canonical registered path and submit_seen, with forged JSON, legacy capability and held-handoff kill producing zero deliveries. Its initial N16 setup hit the normal eight-message receiver breaker; fixture now uses a real hub recipient's existing allowance64, without production changes or a flake claim.

Root integration regressions caught two real boundary failures: a pre-canceled HTTP observer could still start work (RED0.201s), and result-audit failure could kill the observer before preserving a known receipt. Dispatch now refuses pre-canceled admission, owns completion after departure, and returns uncertainty retaining validated known receipt metadata. Focused combined race PASS2.445s; entire remotesession race PASS9.869s. Local marked submit current/reopened/renamed controls race PASS1.535s; service/forget/owner/recorder/help controls race PASS3.442s. Full sessionsync PASS2.373s; retained-raw/owed race PASS2.734s. A malformed future JSON value before a later provenance key could hide that key from the old recognition helper (RED0.174s); lexical string-key recognition now detects plain/escaped keys without interpreting quoted value text, ledger/owed race GREEN5.179s. Unknown root/nested fields traverse real append/trim/reopen/owed, and new document IDs cannot attach to replacement registrations.

Full repolint PASS13.440s. Artifactpath first correctly refused six new sources plus remote_service's new resolver import; classified the concrete producer/consumers with named origin/audit path members (no violation waiver), then full artifactpath PASS17.614s. Reader inventory first refused two new inputs; audited remote_origin_test's literal ../outside and native producer's literal /bin/sh/generated-private transcript reads as local_or_literal_fixture, recorded only those certificates; inventory --check PASS. Atlas/README now describe delivered surfaces, conservative forget, authority and transcript limits. No M3 close, recorded full-suite pass, stage, public exposure or Access/iPhone acceptance claimed. Checkpoint these changes before recorded checks/review.


### 2026-10-21 — morning staging ordered, M3 code review tonight

TLc4102caf fixes tomorrow's :0 order: #753 smoke, #739 check, then #759 before #737. No staging tonight. Its explicit request for full milestone-close adds an M3 code boundary before operator acceptance, superseding the earlier final-close-only M3 choice; Revisions in the durable plan records the scope/sequencing change. Checklist: workshop/plans/000759-remote-security-plan.side/operator-acceptance.md. It includes official current Cloudflare clicks, exact environment keys, phone enable/login/kill/console reset, expected transcript/audit, and honest deferred edit/submit steps requiring #758/#760. Whole issue remains working and cannot land on code/private fixtures alone.

Broader packages on c3634473 passed: couchcmd71.744s, couchmessage25.136s, remotesession4.634s, sessionsync9.708s. PR538 updated on that exact head. Recorded suite attempt ba061185bb8fc8e7b42a4eeb64fa93bd, lease-4249139414, terminal85112 remains queued normally (last observed position5); no pass claimed and no duplicate runner. Preserve its FIFO position across this docs-only checkpoint.


### 2026-10-21 — M3 review corrections and named-ingress gap

Full M3 milestone review returned REWORK with BR-9/10/11; no close was finalized. Canceled only queued suite lease-4249139414 through sdlc before code corrections, joined its terminal (exit1 cancellation); it never ran tests and supplies no verification. Reports f76e8975 and 2484e505 are confirmed submitted.

BR-9: independent configured-service cross-slot controls were RED before correction; removed caller/owner conflation and kept unmarked submits on the legacy content-read/wire path. Existing rename delivery now initializes the actual origin namespace. Targeted couchcmd races PASS1.780s. A further stable-ID rename/admission-gap test was RED0.104s: matching new ID rows were incorrectly filtered through their old path at the later broker timestamp. Owed and round-clear now use matching stable ID for valid submits; legacy rows retain historical path/time attribution and replacements remain excluded.

BR-10: reserve the encoded next row through the existing trim path before append. Failed validation, scanner read, retention publication, write-open and gap-expiry paths preserve evidence and remain bounded across fresh Ledger owners. N8/16 attempts check byte-identical evidence; no repair/truncation or in-memory poison substitute. BR-11: controlled backend-result, result-audit and settled cuts against observer cancel, logout and kill; failed-publication cuts; N8/16 canceled-observer cycles assert exact handoff/effect/audit counts, cached replay, retained known receipt uncertainty and join/reset exclusion. Full affected sessionsync/remotesession race suites PASS23.099s/12.536s. No live acceptance claimed.

Read-only standard cloudflared config search found no config.yml/config.yaml under ~/.cloudflared, ~/.config/cloudflared, /etc/cloudflared, /usr/local/etc/cloudflared or /opt/homebrew/etc/cloudflared, and no YAML/TOML in ~/.cloudflared. One JSON credential and cert.pem exist; contents not read. Tunnel name and hostname cannot be confirmed from local config and remain operator staging inputs. No account/network/console changes.


### 2026-10-21 — M3 finalized SHIP, corrected-head suite retained

Full milestone review round2 independently mutated all three corrections, reproduced regression failures, restored the pinned head and passed complete remotesession, sessionsync, couchmessage, couchcmd and wrapcmd packages plus focused races. BR-9/10/11 addressed, no new findings; boundary ledger round9 has no open blockers. Binary finalized M3 SHIP and committed/pushed df1c5254e5f02aefbde1cb1cdf5be8c00b18c7b3. M3 code/private qualification is complete; this does not finalize the issue or establish live acceptance.

Retain exactly one recorded suite on corrected code5136229868a8fc0806d934317d8104154e763dee: attempt246fb0ef459d6358aa6ffd4ddc5ffc51, lease-1141578967, terminal73968; detached checkout /Volumes/TestScratch/local-checks/sdlc-local-3885321496. Last observed queued FIFO position5, never started, no pass claimed. Milestone commit and this note are docs-only descendants; do not cancel or duplicate its run for them. No review remains running. Staging/landing stays held for the operator tomorrow in the assigned order; tunnel name/hostname remain missing config inputs.


### 2026-10-22 — retained suite recovered PASS; ready for scheduled staging

Sweep c96bec4e: joined original terminal73968, which exited0. Recorded suite attempt246fb0ef459d6358aa6ffd4ddc5ffc51 on5136229868a8fc0806d934317d8104154e763dee PASSED in12m24s; GitHub sdlc-local/suite is success (posted2026-10-22T06:44:05Z). Saved log SHA-256 a458d6136bb2a5bfadc8f32ff2783dacd8cf50394b8c29761737ac0a2715ae20 verified locally. No retry or replacement run. M3 close df1c5254e remains SHIP; descendants contain only issue/review evidence. No own suite or review remains running.

Next is the operator-assisted scheduled stage/Access/iPhone acceptance, not another close or merge. Request the hub's staging turn after #753/#739, preserving the approved hold until the live check. Named tunnel name/hostname remain unconfirmed from absent local config; no live/account/network change performed. #758/#760 edit integration remains explicitly deferred in the operator checklist.


### 2026-10-22 — approved dedicated remote tunnel revision

TL 795fb6dc and 05e409a2 approve a dedicated tunnel: remote reads only
COUCH_REMOTE_TUNNEL and COUCH_REMOTE_HOSTNAME. The ordinary cast configuration
is passed only to a collision guard. Equal names or canonical UUIDs refuse
before connector spawn, including name/UUID aliases. Names resolve through
read-only cloudflared tunnel list --output json --name NAME; UUID inputs skip IO.
At most two sequential lookups share a five-second deadline; stdout is capped
at 1 MiB and diagnostics are capped by the existing owned ProvisionIO runner.
Unavailable, malformed, ambiguous or expired evidence refuses and tells the
operator to inspect tunnel settings and retry. Access checks, connector guards,
record ownership, capture refusal, kill/reset authority and session policy stay
unchanged. The operator creates couch-remote and changes its DNS route; agents
make no account/network writes. No shared-ingress owner is added.

Implementation uses the existing couchcore.ProvisionIO seam (ARCH-DRY), whose
OS runner joins cancellation and bounded collection; no new process collector.
One optional DistinctFrom tunnel reference on Cloudflared enables the preflight
only for protected remote transport. Pure UUID/metadata validation lives beside
the bounded lookup adapter. Tests cover direct name/UUID collision, aliases,
malformed/ambiguous/unavailable resolution, two-query/deadline bounds, remote
settings independence and simultaneous cast+remote mounts on distinct tunnels.
New temporary configs remain in the existing listener/run-record lifecycle;
no new durable artifact family or cache is introduced (ARCH-FUNERAL).

Current-main integration retained both endpoint and remote protocol additions,
used the single shared preserve-number checker and combined both regression
oracles. Targeted strictjson, couchtty, couchcmd remote/endpoint checks passed;
couchmessage remote/forget/orphan checks passed independently. The first console
check lacked generated embedded runtime assets; generation resolved setup only.


### 2026-10-22 — operator pivot to Tailscale, approved before code

Operator decision a487d0d1 supersedes the entire remote Cloudflare Access path
and the dedicated-tunnel revision. TL f8ad8c19 approved the complete revised
transport plan below. #803 landed as PR544 before this branch resumed. Its code
is integrated; the shared retained-number codec remains reconciled with #758.
Public cloudflared casts remain unchanged. No Tailscale product code or live
Serve mapping has been created. The Homebrew tailscale-app 1.104.2 installation
still needs readiness confirmation from the operator through the TL.

Threat model and authority

- Protect against an unallowed tailnet user, shared/tagged device, LAN or public
  caller, browser script, forged HTTP identity/IP headers, stale session/link,
  another local UID, and late completion after off/kill. Root and malicious
  code running as the operator's UID remain trusted, as in the existing plan;
  this does not protect against a compromised operator account or tailscaled.
- Remote uses HTTPS Serve only, never Funnel, quick tunnels or the public cast
  host. A nonempty COUCH_REMOTE_ALLOWED_LOGINS is the remote allowlist. Remote
  ignores cast/Cloudflare tunnel, hostname, team, audience and email settings.
- Use a private physical 0700 directory and 0600 Unix socket, no TCP listener
  or loopback fallback. Bind the accepted connection to verified proxy-peer
  evidence in server ConnContext; an HTTP header cannot create that evidence.
  Require kernel Unix peer credentials/PID plus identity-bracketed supported
  Tailscale daemon inspection through procutil.Source. Unavailable or mismatched
  inspection refuses. The exact installed Mac daemon/extension identity must
  be measured; do not infer it from an argv substring or assume UID alone proves
  Serve. The first bounded native probe qualifies this before product code.
- Only after that peer check accept one bounded Tailscale-User-Login and one
  canonical IP in X-Forwarded-For. Reject duplicate, malformed, empty, control-
  bearing or unexpected encoded values; compare the decoded login exactly with
  the configured allowlist. Tagged/missing-login and Funnel-marked requests
  refuse. Pin expected forwarded host/protocol and existing Origin/CSRF checks.
  Identity is authenticated anew on every API request, including kill and
  streams; a browser cookie never substitutes for current ingress identity.

Owner and lifecycle

- Keep the existing transport-independent supervisor and its on/opening/on,
  closing, killed and cleanup-blocked states. Keep the 👽 console switch, CLI
  on/off/status, console-only link/QR, capture refusal, four-session cap, replay
  limits, audit-before-effects, dispatch fencing and host-key-only latch reset.
- Preflight the installed CLI/daemon and logged-in Running state, the canonical
  node HTTPS name and HTTPS readiness with bounded read-only CLI commands.
  Missing install/login/HTTPS permission refuses with one concrete recovery;
  no automatic login, policy/certificate enrollment, sudo or machine settings.
- Run one foreground tailscale serve --https=443 unix:<owned-socket>, without
  --bg, --yes, --service or Funnel. The foreground daemon session is the
  lifetime ownership resource. Native admission must atomically refuse an
  occupied port, including a racing foreground owner; preflight inventory is
  diagnostic, not absence authority. Never overwrite another Serve config.
- Reuse the current guarded-child lifetime/EOF collection model and bounded
  metadata runner. Opening has the existing 30-second envelope; each metadata
  exchange has a five-second cap and 1 MiB output cap; closing retains the
  existing five-second join budget. No new periodic process or liveness loop.
  Publish on/link only after the exact foreground mapping and private-origin
  readiness are confirmed. Ordinary remote requests spawn zero children.
- Off/kill first revoke epoch, sessions, passcode and dispatch admission, then
  cancel/join the owned foreground attempt and close the origin. Verify its
  exact mapping is absent; unrelated mappings may remain. Never issue serve
  reset or a generic off that might erase another owner. Unknown teardown keeps
  the existing blocked state and reset unavailable. A child exit, daemon loss
  or profile switch revokes remote mode; no silent connector restart.
- Sessions have no new idle/absolute timeout. They end on off, kill, logout or
  current identity/epoch loss. Same valid authenticated identity can silently
  re-handshake while mode is on. Re-enable rotates passcode/link and revokes
  every prior session. A stream cannot survive identity/transport revocation.
- Provenance stays Origin=remote and independently derived RemoteEdited;
  delivered text is [remote <login> <IP>]. Use an explicitly named Login field
  in the unlanded remote identity contract rather than treating arbitrary
  logins as verified email. Update receipt/audit/ledger/receiver tests together
  and tell the hub the exact JSON key. Preserve unknown fields and conservative
  historical evidence. Native agent transcript remains the second paper trail.

Visible states and recovery

| Failure/state | Operator sees | One recovery | Possible loss |
|---|---|---|---|
| Missing CLI, logged out, HTTPS unavailable or invalid allowlist | unavailable, specific prerequisite | install/sign in/configure as named, then retry on | none; no exposure |
| Occupied Serve endpoint | busy; existing service preserved | stop/reconfigure that service explicitly, then retry | none |
| Socket inaccessible or proxy identity cannot be proved | unavailable; secure origin unsupported | inspect installed Mac variant, then retry | none; no weaker fallback |
| Opening/cancel race | opening or closing, no usable old link | wait for owned attempt | no submit admitted until ready |
| Healthy on | on; private tailnet link on console only | use phone | existing bounded operation uncertainty applies |
| Unallowed/forged/missing identity | request denied | use an allowed personal login | no effect |
| Off or phone logout | off or signed out | enable or re-handshake, respectively | draft saving follows existing explicit result |
| Kill latch set | killed | host physical keypress, then re-enable | revoked sessions; retained unsent work stays local |
| Cleanup outcome uncertain | blocked; reset unavailable | inspect/recover locally and restart | no claim of removed ingress; effects remain revoked |

Implementation and qualification

1. After #803 lands, record this pivot and operator approval. Coordinate one
   bounded private native Serve probe on the actual Mac installation: socket
   access, kernel peer identity, header replacement/IP, foreground owner conflict
   and daemon-session cleanup. No production listener, global reset, public
   exposure or unrelated mapping change. If secure peer proof/socket access is
   unsupported, stop and return the evidence and simplest alternative for plan
   review; do not quietly fall back to localhost.
2. Replace only the remote transport/auth adapter and its configuration. Remove
   remote Access/JWK/JWT and dedicated-tunnel code, retain ordinary cloudflared
   paths/tests. Keep a pure login/IP authorization core separate from IO.
3. Stateful fake daemon/CLI behind the same seam: install/login errors,
   malformed/capped output, held opening/closing, late success, mapping conflict,
   losing the owner, uncertain teardown, port races and unaffected other Serve
   mappings. Count one foreground child/epoch and bounded metadata operations;
   N/2N requests/connections have linear validation and zero per-request spawns.
4. Negative HTTP/Unix tests: forged direct TCP and unrelated Unix peer denied;
   unavailable peer inspection denied; duplicate login/IP headers, public/Funnel,
   tagged identity, wrong login/host/origin/CSRF and stale cookies/links denied
   with zero dispatch. Identity changes revoke existing sessions and streams.
5. Re-run lifecycle/capture/four-session/kill-reset/replay/audit/provenance and
   native-transcript regressions. Include browser logout/re-handshake, off/kill
   while a request or stream is held, daemon exit and profile change. Full
   packages/race, source and reader inventories, full recorded suite, mandatory
   fresh review. No live-network test is represented by a fake result.
6. Update README, atlas and acceptance steps. Operator installs/logs in on Mac
   and iPhone and enables HTTPS if required. Stage through sdlc on TL go; phone
   opens private link, allowed login edits/submits, transcript/audit show login
   and verified IP, phone kills, only host key resets. Check an unallowed login
   and direct-origin spoof control. Public cast still works independently.

Evidence informing the proposal (installed version still must be qualified)

- https://tailscale.com/docs/features/tailscale-serve — identity header handling
  and the direct-origin spoofing caveat.
- https://github.com/tailscale/tailscale/blob/main/cmd/tailscale/cli/serve_v2.go —
  Unix targets and foreground ownership through a daemon session.
- https://github.com/tailscale/tailscale/blob/main/ipn/ipnlocal/serve.go — source-IP
  forwarding, removal of client identity headers, atomic foreground port
  conflict checks and foreground session removal. This is upstream evidence,
  not proof of the operator's installed Mac binary.


### 2026-10-22 — native probe waiting on operator readiness

TL 79b46dec requested exact prerequisites. Read-only inspection finds no
tailscale command on PATH and no /Applications/Tailscale.app or user Applications
copy; Caskroom holds only tailscale-app/1.104.2/Tailscale-1.104.2-macos.pkg.
Reported this observation without claiming the installer ran. Sent the hub the
Mac install/open/network-extension approval/login, admin DNS MagicDNS and HTTPS
Certificates, same-tailnet untagged iPhone identity, exact allowed login and free
Serve HTTPS/443 prerequisites. No manual certificate generation, Funnel,
configuration reset, installation, app opening or Serve mapping was performed.
Probe remains gated on the hub's readiness confirmation. No remote product code
has changed for either transport revision; no stage or code acceptance claimed.

During #803's live merge, the auxiliary sibling worktree failed change-code's
workspace identity guard. Returned to the supported checkout, then stopped the
new plan-quality call with Ctrl-C when the TL's hold arrived and restored #803
immediately. #803 subsequently landed with its retained runner. The independent
worktree was removed cleanly; its committed #759 integration/plan survived. The
aborted plan-quality call grants no new implementation acceptance. The current
Tailscale revision supersedes that interrupted dedicated-tunnel review input.


### 2026-10-22 — authenticated drain lift, saved-state recovery

Authenticated TL lift d64292dd; checkout clean on #759, sdlc state confirms
pair:6 ownership and working status. No check/merge/probe was paused for this
drain. Readiness resend 28a7d09a is confirmed submitted. The installed CLI now
appears at /opt/homebrew/bin/tailscale, but read-only status still exits 1
with no JSON: "failed to connect to local Tailscale service; is Tailscale
running?" Running/login/HTTPS readiness remains unproved. No Serve mapping
or configuration change. Added a pending pivot row to Issue Plan so the prior
closed M1–M3 code boundaries do not imply the new transport or acceptance passed.


### 2026-10-22 — installed Tailscale native gate: scratch socket denied

TL readiness 202ff650 confirmed Mac/iPhone login, MagicDNS and HTTPS, with
COUCH_REMOTE_ALLOWED_LOGINS corrected by b5dc44eb. Actual installed app CLI
reports 1.104.1 (not the previously observed installer package version). Scripted
CLI requires TAILSCALE_BE_CLI=1; without it the clean-environment control emitted
a GUI startup error and did not produce JSON. No app settings were changed.

Evidence: workshop/evidence/000759/tailscale-scratch-probe-20261022/ contains
the probe source, events, capped Serve output, exact daemon error, entitlements
and control result. Explicit empty-base child environment inherited no Couch or
Pair identities. Physical0700 scratch directory and0600 Unix socket; direct
forged-header control captured its own Python PID/UID through kernel credentials.
One foreground Serve epoch advertised HTTPS443, no Funnel or background mode.
The Mac HTTPS control returned502, with zero proxied origin requests. Native log
12:47:16.303: IPNExtension[17726] dial unix to the exact scratch socket failed
with operation not permitted. Installed process is the UID501 IPNExtension app
plugin; signed entitlements include app-sandbox=true and Downloads read/write.
No iPhone request reached the origin; the first notice was submitted near the
end of its180s window, so this is not an iPhone acceptance failure. Cleanup joined
the owned foreground child exit0, removed only its socket and confirmed original
empty Serve configuration restored. No remote transport code or fallback.

Official macOS-variants documentation says both GUI extension variants are
sandboxed and the open-source tailscaled variant is not. Serve documentation
limits GUI file serving but does not promise Unix proxy access. The observed
scratch location is denied; this does not prove every possible location denied.
Reported options and their trade-offs before choosing: bounded Downloads probe;
operator-managed unsandboxed root tailscaled with service/update/migration burden;
or a separately reviewed TCP authentication boundary (not approved/implemented).
TL bb09d562 approved exactly one0700/0600 Downloads probe, armed only until the
operator explicitly says ready, then a600s window. Even success requires operator
review of a user-visible Downloads product socket and entitlement-update risk
versus the root-daemon option. No product design acceptance is inferred.


### 2026-10-22 — Downloads probe failed; exact cleanup complete

Authenticated operator-ready57e75d63 opened the single approved probe at12:55:53
PDT, maximum600s. URL readiness receipt a06c0eb7 was submitted. Physical0700
Downloads directory and0600 socket were created only for this attempt; empty
Serve baseline confirmed, one foregroundHTTPS epoch, noFunnel/TCP fallback.
Operator reported blank iPhone page through TL3dab8619. Expected successful
response was visible text Pair private probe received. OK. Zero proxied requests
reached the socket, hence no proxy kernel identity or login/IP evidence. Native
IPNExtension log at12:56:49/57/58 and12:57:25/28/29 explicitly denied connect
to the exact Downloads Unix socket: operation not permitted. The direct local
control still captured its own kernel PID/UID. Downloads entitlement therefore
did not admit the tested private socket; native gate failed, no product code.

Stopped the exact owned probe early after matching its PID/start/argv, with
SIGINT. Its finally joined foregroundServe exit0, reader joined, socket and
Downloads directory absent, original Serve configuration{} restored at12:58:04.
Top-level probe exit130 and raw result not-run reflect KeyboardInterrupt, not
an inconclusive socket outcome: repeated daemon denial is the measured failure.
Retained source/events/output/error and explicit interpretation under
workshop/evidence/000759/tailscale-downloads-probe-20261022/. No live cleanup
outside owned paths, no daemon restart/reset, no further location experiments.
TL f361e073 requested a read-only tsnet/root-daemon/authenticated-loopback
comparison before an operator transport decision. Implementation remains held.


### 2026-10-22 — transport options requested, decision pending

TL f361e073 requested tsnet versus root tailscaled versus authenticated loopback
before choosing. Recommendation sent as receipt08f12732: tsnet, subject to
operator choice, a revised security/dependency plan and explicit node enrollment
authorization. No code, dependency addition, auth key or node creation.

| Option | Security boundary | Operator and alpha setup | Maintenance |
| --- | --- | --- | --- |
| tsnet | Couch owns a userspace tailnet node; HTTPS tailnet listener and its own LocalClient.WhoIs identify the accepted connection, not forwarded headers. Keep exact login allowlist; missing/tagged user identity denies. Persistent node keys and same-UID/root remain local trust limits. No Funnel or exposed LocalAPI. | One Couch node enrollment via browser login, possible device approval, HTTPS enabled, explicit0700 state and protected key files. Phone Tailscale login remains. Existing desktop app can coexist; no VPN replacement. Display actual assigned hostname, including collision handling. | Pair takes userspace network stack and transitive dependencies, pinned API/toolchain, security updates, lifecycle/state/expiry/revocation and measured build-size/startup/RSS costs. None of these costs have been measured yet. |
| Open-source tailscaled | HTTPS Serve plus private Unix/kernel-peer boundary, still requiring a fresh native qualification. Unsandboxed root service expands host privilege. | Deliberate operator-managed install, login, existing-app transition/coexistence policy, service and upgrades; repeated by each alpha tester. | Avoids embedding the stack in Pair but adds privileged service administration and supported CLI/daemon version/mapping ownership. |
| Loopback with independent authentication | Local processes can connect and forge identity headers. Requires independent cryptographic user authentication; a port or shortPIN cannot attest Tailscale identity. Verified tailnet IP would need an additional authenticated binding, not header trust. | Another credential/enrollment or identity-provider configuration for operator and alpha. Current app can remain. | Pair owns authentication/recovery/replay/CSRF/session controls plus proxy hardening. Potentially smaller dependencies, larger authentication design. |

Primary references: https://pkg.go.dev/tailscale.com@v1.104.1/tsnet and
https://tailscale.com/docs/reference/tsnet-server-api document per-process node,
TLS tailnet listener and native WhoIs; https://tailscale.com/docs/concepts/macos-variants
documents daemon packaging/privilege trade-offs. Recommendation is an engineering
judgment, not live tsnet evidence. Existing transport-independent controls stay.
No fallback is authorized by this comparison. The tested GUI Unix path remains
blocked; any alternative needs an explicit revised plan before product code.


### 2026-10-22 — tsnet revision proposed after operator choice A

Operator choice relayed by TL2e7f4b13 supersedes the Serve transport revision.
The failed native socket probes remain evidence, not an implementation route.
This is the plan sent for approval; no tsnet dependency, node or auth key exists.
Public cloudflared cast behavior is unchanged. All remote Cloudflare verification
and Serve/socket/proxy authentication paths are removed from the remote feature.

Core concepts (ARCH-PURE, ARCH-DRY, ARCH-SECURE)

| Entity | Location | Change |
| --- | --- | --- |
| TailnetPrincipal (canonical login, node/user IDs, accepted peer IP, epoch) | cmd/internal/remotesession/tailnet_identity.go | new pure identity validation; replaces AccessIdentity semantics |
| Exact login policy | cmd/internal/remotesession/policy.go | modified; COUCH_REMOTE_ALLOWED_LOGINS only, nonempty explicit entries, no suffix/domain/case-fold match |
| Remote epoch/reducer | cmd/internal/remotesession/state.go and supervisor.go | modified for enrollment/health events; retains existing kill/reset and cleanup authority |
| Remote receipt/audit identity | cmd/internal/remotesession/protocol.go, audit_codec.go, session.go | additive Login; immutable old Email data remains historical, never reinterpreted as newly verified Login |

| Integration | Location | Change / external boundary |
| --- | --- | --- |
| TailnetNode owner + stateful fake | cmd/internal/remotesession/tailnet.go, tailnet_fake_test.go | new narrow adapter over pinned tsnet Start/Up, LocalClient status/WhoIs/watch, ListenTLS, Close |
| Accepted connection authentication | cmd/internal/remotesession/server.go, tailnet_identity.go | modified; trusted ConnContext inserted only by owned tsnet listener |
| Namespace state and console publication | cmd/internal/couchcmd/remote.go, remote_service.go; cmd/internal/artifactpath/remote.go | replaces cast tunnel settings with owned node state/config and console-only enrollment/link |
| Host UI and docs | cmd/internal/couchtty remote controls; README.md; atlas existing remote page | preserve controls, show enrollment/assigned name and recovery |

1. Dependency gate, before integrating the adapter (ARCH-CONSTRAINTS)

- Candidate exact pin is tailscale.com v1.104.2, go.sum verified; no latest/main
  dependency. Source go.mod declares Go1.27.3; Pair currently declares Go1.26.3.
  This is a material compatibility decision, not permission to upgrade Pair's
  toolchain. Report it to TL with the dependency review before accepting the pin;
  if a toolchain change is required, stop for a decision rather than silently
  selecting an obsolete version or editing go directives. v1.104.1 also requires
  Go1.27.1 and is not an assumed compatible escape hatch.
- After plan approval, inspect/download the pinned source in isolated scratch
  without starting a node, changing product go.mod, or running enrollment. Record
  module provenance/checksum, licenses, govulncheck results, actual compiled
  Darwin/arm64 package/module closure (not all monorepo go.mod entries), changed
  shared dependency versions, CGo/toolchain and supported-platform effects.
- Same compiler, architecture, tags and release flags for matched baseline and
  proposed linked Couch binary. Record byte sizes, delta, cold/warm build times
  and retained module/package lists. Use a compile-only measurement fixture;
  importing/building tsnet is never consent to call Start/Listen/Dial/LocalClient.
- Measure baseline idle Couch and candidate remote-OFF RSS; once the operator
  authorizes enrollment, measure remote-ON idle for60s and four-session edit/stream
  load, peak and steady RSS, CPU, goroutines and node-state bytes. Include disabled
  startup and off cleanup. Same workload/host controls, raw timestamped samples;
  no guessed footprint or subtraction presented without a comparable baseline.
- Report actual costs and any toolchain conflict for acceptance before final
  integration/landing. No claim of zero overhead; zero tsnet node instances and
  zero enrollment/network work while remote is off is an enforced lifecycle rule.

2. Connection identity and authority (ARCH-SECURE)

- Only tsnet.Server.ListenTLS on tailnet TCP443 can feed remote HTTP. Never
  ListenFunnel, Serve, host TCP/loopback listener, exposed LocalAPI/Loopback helper,
  generic reverse proxy, automatic route advertisement or exit-node selection.
  Existing public cast transport is separate. RunWebClient=false; no extra
  tailnet management listener. No forwarded header establishes login, host or IP.
- At accept, bind the actual net.Conn.RemoteAddr to unexported ConnContext with
  owner/epoch. HTTP RemoteAddr/header rewrites cannot replace this evidence.
  Use this node's LocalClient.WhoIs with that captured address only. Require
  canonical tailnet peer IP and complete untagged human user/node identity;
  refuse missing login, tags/service identities, malformed address or lookup
  failure. Exact login membership, e.g. xianxu@gmail.com, comes from explicit
  COUCH_REMOTE_ALLOWED_LOGINS. No email-domain or suffix inference.
- Revalidate WhoIs and current node/epoch health on each request, immediately
  before protected handoff, and before each stream delivery; bind sessions to
  verified user/node/login/IP plus epoch. Keep Origin/Host/CSRF/replay/passcode
  checks, four-session cap and capability checking. HTTP/2 connections retain
  the same captured peer; any identity change ends their sessions and streams.
- One owned event watch observes node/netmap changes; membership/expiry/auth
  loss revokes admissions and streams immediately on observation. Re-run WhoIs
  for affected active sessions; no persistent identity cache can override newer
  evidence. Watch loss or uncertain health fails closed and requires explicit
  local recovery. An expiry deadline timer is not an idle-session timer.
- Revocation is bounded by Tailscale's control-plane propagation and what this
  node observes; do not claim instantaneous global revocation during partitions.
  No positive WhoIs from a stale map is advertised as fresh control-plane proof.
  Native acceptance must measure removal/revocation behavior; if the pinned API
  cannot establish the planned expiry/health policy, stop and report the gap.
- Delivered header remains [remote <login> <IP>]; receipt Origin=remote and
  RemoteEdited stay. Login is explicit end-to-end, including broker, ledger,
  audit and native transcript. Console submits remain untagged. Native transcript
  is the paper trail; bounded correlation recorder is not complete delivery proof.

3. Enrollment, state and operator recovery (ARCH-FUNERAL, ARCH-EVOLVE)

- One stable node per leased Couch namespace, one tsnet instance per active
  epoch. Reuse explicit namespace-owned state under the existing remote state
  root, in a new physical0700 tailnet directory; reject symlinks, foreign owner
  or unsafe existing permissions instead of traversing or broad chmod repair.
  Key/state files0600. Namespace lease plus exclusive directory ownership keeps
  two Couch owners from using the same keys. No implicit user config fallback.
- State is tsnet-owned opaque storage, never decoded/re-written by our codecs;
  audit its actual writer permissions/atomicity at the pin. New Pair metadata
  uses the shared retained-raw strict codec, preserving unknown nested fields.
  No secrets in audit/transcript/CLI/status/test fixtures. Enrollment login URL
  is console-only; capture-configured refusal still applies before starting.
- Explicit initial enrollment is a local console action with the operator
  present. Show tsnet's login URL on the console; operator opens it, signs into
  their intended tailnet and approves the new Couch device (including admin
  approval if required). No automatic auth-key/OAuth creation, copied desktop
  credentials or account/policy changes. No node Start before this authorized
  enrollment test. Reject inherited TS_AUTHKEY/TS_AUTH_KEY, OAuth/federation,
  TSNET_FORCE_LOGIN and control-server override inputs rather than consuming
  ambient credentials or mutating process-global environment.
- Use the backend-confirmed assigned FQDN and own node/tailnet identity after
  enrollment; never assume requested couch is available. Verify HTTPS readiness
  and publish the actual https://FQDN link/QR only on Couch's console once ready.
  Record enrolled identity and refuse unexpected tailnet/node replacement until
  a fresh explicit local enrollment decision. Default-off never auto-enrolls.
- Initial local enrollment may wait up to10min with visible status and cancel;
  no remote route/dispatch is live meanwhile. Ordinary enrolled enable retains
  the30s opening budget. WhoIs/status operations cap at2s and active request
  concurrency is bounded by the existing server; overflow refuses. Cancellation
  revokes the epoch before joining the node, with existing5s cleanup budget.
- Off/kill revoke link/passcode/sessions/dispatch FIRST, close HTTP connections,
  cancel the same opening attempt, join watch and tsnet.Close. Late readiness
  cannot publish. Unproved cleanup stays blocked, host reset unavailable. Off
  preserves enrolled keys for explicit re-enable without creating a new device.
  Re-enable rotates capability/passcode; never rotates node identity silently.
- Node key expiry, needs-login, device removal, missing policy access, certificate
  failure, or observed caller revocation denies requests and stops dispatch.
  Recovery names the action: reconnect/check tailnet policy, or local re-login
  and operator approval. No automatic new node or disabled key expiry. Policy
  outages are not repaired by deleting state. Same account reauthentication
  never restores killed latch authority; reset remains physical-host-key only.
- Durable node state has one identity rather than per-epoch/per-session growth;
  inventory/cap upstream caches and logs during dependency review. Keep it across
  off/restarts, remove only through explicit operator retirement after device
  removal and proven Close, never automatically reset/delete to repair errors.
  Enrollment cancelled after possible registration may leave a device: preserve
  state and report exact identity/unknown outcome for operator inspection.

4. Reducer and uncertainty table (ARCH-STATE)

| State/event | Outcome and owned effects |
| --- | --- |
| off + local enrollment approved | opening/enrolling; one owner, console URL; no remote admission |
| opening + enrolled and HTTPS/identity ready | on; publish actual origin under current epoch only |
| opening + timeout/cancel/error | revoke then closing; join same attempt, retain possibly enrolled state |
| on + off or authenticated kill | revoke before close; kill persists existing latch |
| opening/on + expiry, identity loss or watch uncertainty | revoke; close; recovery status identifies login/policy/health cause |
| closing + late open completion | never publish; join exact resource and preserve its cleanup outcome |
| closing + confirmed completion | off or killed; persistent node identity retained |
| closing + uncertain completion | existing cleanup-blocked; reset unavailable; no second owner |
| killed + remote/CLI reset or re-handshake | refuse; only actual host key can clear after cleanup proven |

5. Implementation and qualification sequence (tests first, no extra milestones)

- After TL plan approval, run change-code plan-quality on this revision. Finish
  dependency/compatibility report first; seek the explicit toolchain decision
  if needed. No node or auth key until operator enrollment window.
- Red/green pure login/principal and reducer tests; then stateful TailnetNode
  fake behind the production seam. Model enrollment pending/approved/rejected,
  actual hostname collision, held startup/close, late readiness, expiry, device
  removal, watch loss, WhoIs errors/change/tagged identities and state contention.
  Count N/2N requests and stream events: bounded linear identity operations,
  zero spawned CLI children; one node/watch per epoch and zero while off.
- Add root/header spoof tests proving WhoIs input is captured connection address,
  never HTTP headers. Missing context, forged login/IP/RemoteAddr, unallowed login,
  wrong origin, stale cookie/capability and expired/revoked identity yield zero
  document effects. Deterministic off/kill during held WhoIs/dispatch and both
  existing failed-cleanup opening/closing regressions retain the same authority.
- Replace Cloudflare-only remote transport/verifier and tests, reconcile shared
  host coupling narrowly, retain public cast tests. Sweep Login/provenance across
  server, sessions, broker, IPC receipt, audit, ledger and transcript. Preserve
  old audit unknown fields and historical Email bytes. Forget split remains.
- Operator-approved enrollment test: isolated explicit0700 state, no auth key;
  operator sees console URL and approves one identified device. Confirm actual
  HTTPS name, phone WhoIs login/IP, disallowed identity and header-forgery cases,
  off/re-enable identity reuse and link rotation, kill from phone, host reset,
  expiry/revocation response, joined cleanup and unchanged desktop app/public cast.
  Native test node is retired only with operator authorization; no silent delete.
- Measure full footprint above, update README/atlas/operator acceptance, targeted
  packages/race plus recorded suite, then required fresh-context boundary review.
  Stage only on TL go. Operator iPhone edit/submit checks native transcript and
  audit provenance before landing; fakes never replace live acceptance.

References checked 2026-10-22: tailscale.com/docs/reference/tsnet-server-api;
pkg.go.dev/tailscale.com@v1.104.1/tsnet (API/enrollment/ambient credentials);
raw.githubusercontent.com/tailscale/tailscale/v1.104.2/go.mod (candidate pin and
Go1.27.3 requirement). API/Go requirements must be rechecked at selected pin.


### 2026-10-22 — approved dependency ordering and security delta

TL9027b5a2 approved the tsnet plan with exact candidate pin tailscale.com
v1.104.1, using installed Go1.27.1. Pair go directive1.26.3 ->1.27.1 must be
its own commit; record a full-suite pass on that bump alone before dependency
addition. Compile-only measurements precede integration; no node/auth key before
operator enrollment test. No automatic toolchain download is authorized.

Pre-code release review found a security-relevant .2 delta. Official changelog
https://tailscale.com/changelog says v1.104.2 updates Go to1.27.3 to distrust
compromised TLS intermediates (TS-2026-013). Bulletin
https://tailscale.com/security-bulletins#ts-2026-013 recommends1.104.2 or later.
Inference: relevant to embedded tsnet outbound TLS/control-plane trust; this
is not evidence of a direct WhoIs/ListenTLS authorization defect. Other listed
.2 fixes affect Windows peer-update disconnects/DNS and macOS GUI menu timing
and services filtering, outside the embedded feature. Reported to TL before
dependency addition for the operator Go-upgrade decision. Independent authorized
Go1.27.1 bump can proceed; dependency and enrollment remain gated.


### 2026-10-22 — operator approves Go1.27.3 installation later; pure work proceeds

TL ca5f4e9e relayed operator authorization for Go1.27.3 installation during
the planned full shutdown for#739; TL will install and confirm. Target returns
to exact tailscale.com v1.104.2 and Go1.27.3. Do not change the directive to
1.27.3 or allow a toolchain download until that confirmation. Directive-only
Go1.27.1 commit729246582127 is pushed onPR538 and its recorded suite remains
lease-779340230/attempt40c81bb875dedee0ff366c6fbe41a503 (FIFO position15 at
13:08). No dependency added before the directive-only verification.

Change-code plan-quality CLEAN round6, estimate-quality INFO, exited0 with
accepted-design commitd346f62d3. The old estimate's Cloudflare allocation needs
a tsnet qualification update before dependency integration; no measurement or
active-time value is invented here. TL a47a5ebb explicitly permits targeted
pure-package RED/GREEN with installedGo1.27.1 and GOTOOLCHAIN=local: compile-free
means no tsnet import or toolchain download. Preparing pure principal/reducer
and stateful fake through this compiler. Operator enrollment remains separate
and unstarted; no account, node or auth-key creation.


### 2026-10-22 — tsnet-independent principal/reducer/fake checkpoint

Pure principal and exact-login policy now validate canonical tailnet peer IP,
user/node/epoch binding, explicit expiry versus positive non-expiring evidence,
and unknown/revoked/tagged denial. No forwarded header supplies authority; these
are values only, not IO authentication. Unicode Cf format controls are denied
for login/node attribution as a class, preserving ordinary Unicode logins.
Observed RED for missing behavior and format-control class, then GREEN; bounded
fuzz exercised both login and peer-address inputs. No artificial session TTL.

Added eventTailnetUnavailable to the existing reducer's failure branch, preserving
revocation before kill publication/cleanup, stale-attempt immunity, late-ready
disposal and the joined-host-reset fence. RED missing event, then GREEN.
The partial tailnetNode seam and stateful fake model enrollment, assigned FQDN,
lookup identity, single owner/watch, cancellation/join, uncertain cleanup, watch
loss and overflow without networking. Native ListenTLS and accepted-connection
provenance remain deliberately absent until dependency/native qualification.

Fresh ad-hoc review found a real fake ordering issue: status mutation and watch
publication were separate, allowing an older Running event after Revoked. A
controlled publication hook reproduced the reversed order RED; one owner lock
now covers mutation and publication. Watch.Close releases its watch mutex before
acquiring the owner, so the ordering introduces no lock inversion. Reviewer
confirmed addressed; deterministic race regression passed50 runs. Added the
class lesson to workshop/lessons.md. This is not a milestone-close review.

Final root verification with installedGo1.27.1 and GOTOOLCHAIN=local:
- Complete remotesession package race PASS21.145s.
- Complete internal/repolint PASS18.330s.
- Earlier combined Tailnet/RemoteState cohort PASS0.118s; diff check clean.

No tsnet import, dependency addition, toolchain download, node or auth-key
creation. Current go.mod remains1.27.1. Directive-only recorded suite stays on
its exact pushed72924658 head with original lease-779340230/attempt40c81bb8;
local pure checkpoint does not change that detached suite or serve as its result.
Next: TL confirms installedGo1.27.3; then amend the isolated directive commit
and preserve/reapply the local pure checkpoint, qualify the bump alone on an
exact recorded head before dependency integration. Finish compile-only pinned
v1.104.2 closure/security/license/binary measurements and estimate revision;
node/enrolled RSS tests still require the operator's explicit enrollment window.
