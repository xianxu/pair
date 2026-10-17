---
id: 000624
status: working
deps: []
github_issue:
created: 2026-10-16
updated: 2026-10-16
estimate_hours:
card_mirror: 'ecb3f43df4a6c8b9fcf920b477891760c3e7d958' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-16T20:02:10-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "e9385e2f", done: "e8c56ece"}
---

# Something rebuilds pair:0's bin/couch during other slots' work, deleting it for seconds (sends fail)

## Problem

On 2026-10-16, `~/workspace/pair/bin/couch` (pair:0, the machine's installed Couch CLI that every slot and the TL use)
**disappeared for several seconds, twice**, and came back rebuilt at the same revision (9fc035e7, vcs.modified=false):
around 15:40 and at 16:04–16:06. During the gap every `couch --send-to` failed with "no such file or directory".
Nobody asked for a pair:0 rebuild. At 16:04:17 pair:4 (Qoder) started `sdlc milestone-close --issue 583 --milestone M1`
from its slot; the 16:06 rebuild followed. Hypothesis: a review prepare step (pair#549-era "prepare runtime assets
before boundary reviews") or a dev alias builds into pair:0's `bin/` instead of the worktree's own. The same mechanism
would clobber a staged branch build on pair:0.

## Spec

- Find what writes pair:0's `bin/` outside an explicit `make build` / `sdlc stage` in pair:0 (fs_usage or a guard).
- A slot's build, test or review preparation writes only inside its own worktree. pair:0's `bin/` changes only via an
  explicit build or stage there.
- Builds that replace a shared binary replace it atomically (write to a temp file, then rename), never delete first.

## Done when

- A test (or guard) shows a milestone-close / review prepare in a slot worktree leaves pair:0's `bin/` untouched.
- `make build` in pair:0 replaces binaries atomically (no window where `bin/couch` is missing).

## Plan

- [x] Red first in private fixture: paused compiler exposes delete/partial publication; cover direct binary targets and full make build, failure/cancellation cleanup, concurrent publishers and foreign PAIR_HOME isolation.
- [x] Use one same-directory temporary-build/rename publisher for pair, couch and pair-launch-helper; opt owned binaries out of the generic direct-output scanner (ARCH-DRY). Keep old executable on failed builds and clean owned temporaries.
- [x] Verify review asset preparation and worker build stay in their own fixture tree under foreign live-environment values; document observed writers versus unproven historical caller, run recorded full suite.

## Log

### 2026-10-16 — writer investigation and reproduction
- 2026-10-16: closed — Complete recorded suite PASS f769895f18db7deb52e4e28e0bd3b8632c3724ad, 11m0s, log SHA256 3afac57cd20f; no production/test changes since. BR-1 measured actual woven scanner b816fcb7c31d3426775ea583e802835c756756a97c241f4610be51d7e1b345ec in retained private fixture pair624-woven-proof-4bb6qatn: 3 staged writes passes, removal of each new sentinel independently fails with corresponding direct output (4 writes), both absent fails (5 writes). Source, exact command and log paths in issue. Eight atomic tests pass; real Go before missing1/partial418, after missing0/partial0. Two introduced fixture failures fixed and recorded. TL approved private/headless acceptance, unknown historical caller; primary untouched. README and atlas updated. (done-when: contract moved since change-code; judged by the close review); review verdict: SHIP

Read-only scan found two direct Couch writers: Makefile.local's explicit bin/couch target and the shared Makefile.workflow build scanner. The latter is an ariadne symlink; Pair can opt its owned binaries out through the existing .skip-make-build mechanism rather than editing the peer. Direct Pair and launch-helper targets share the same unsafe output publication. Current installed couch is a symlink to primary bin/couch, not a rebuilding wrapper. The current interactive alias generator already builds privately and renames, and skips existing binaries in COUCH_TREE. runtimebundle-generate invokes only its generator (whose subprocess is tic), not a Pair build.

The installed Go source (cmd/go/internal/work/shell.go moveOrCopyFile/CopyFile) falls back to removing/truncating its final destination before copying when rename is unavailable or a cached artifact is copied. First private probe was inconclusive because the output was already current. Alternating two valid builds forced replacement: exit 0, 2753 observations, one missing destination and 418 partial-size observations, with Go -x explicitly showing cp to the final path. Evidence retained in /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair624-probe.r7aBha/result.json and cached-replacement.trace. No build or clipboard/GUI action ran in pair:0.

The exact historical initiator at 15:40/16:06 remains unproven. PAIR_DEV plus primary PAIR_HOME can request a primary rebuild through dev_rebuild, but current hosted markers suppress it; do not label review preparation as the culprit without evidence. The proposed fixture guard covers current review preparation/build routing despite foreign environment values. Asking the hub whether this bounded diagnosis satisfies acceptance, rather than silently dropping the historical-caller clause.

Start-plan refreshed dependencies and required two reruns before creating this branch; brew reported existing go/cue/uv, no new machine packages chosen.

### 2026-10-16 — deterministic red/green and real Go evidence

Before production edits, the new fixture suite failed five assertions: each direct binary exposed partial bytes during a paused compiler write; an unsuccessful build left partial output; root make build invoked five compiler writes instead of three. Review asset preparation stayed local. The shared publisher and two additional skip sentinels make all six tests pass, including cancellation and concurrent publishers. Canonicalized the fixture root to compare macOS /var versus /private/var by actual path; no production workaround was needed. Install-link regression passes; the complete install-layout test also passed with actual Go builds in this worker and an isolated HOME. No primary-checkout build occurred.

The real Go alternating-cache fixture, now using scripts/build-go-atomic.sh, completed exit 0 with 1696 observations, zero missing and zero partial-size observations, correct executable output 33554432 and zero staging directories remaining. Logs atomic-result.json and atomic-replacement.trace are retained beside the initial failing probe evidence. This is headless temporary-copy acceptance, not a pair:0 rebuild.

### 2026-10-16 — recorded archive fixture failure, fixed in this item

First recorded suite on 5645fb1ddbc79dbd3b6de61706fe6d145ebe99a6 waited 2421s, then failed in 3s: all six atomic-build fixtures raised FileNotFoundError copying Makefile.workflow. The local-check archive intentionally lacks woven dependency symlinks. Failed log /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/9b4bf99e3d93959063fb8b4d43c1a98e.log, SHA256 prefix 2644910bcffd. This is a reproducible test defect introduced here, not an unrelated flake.

The fixture now mirrors root Makefile's optional workflow include: use the actual base scanner when available, remain valid in a source archive where no scanner exists. An explicit sentinel contract check remains mandatory in both layouts. Both complete fixture layouts pass (woven checkout and temporary source archive). Added a durable nested archive regression; all eight tests pass. The next recorded run completes the full suite on the corrected test head; production code is unchanged by this correction.

### 2026-10-16 — derived environment-knob fixture interaction, fixed

The corrected recorded run on 4b618c0c5e7b8694b50f1544c06754eebd3a14b9 acquired after 3923s. Atomic-build's eight tests passed in the archive, and the shell/editor checks passed through review-producer-context. At 4m33s the suite stopped on test-make-env: the new fixture's command-line PAIR_GO_SRCS override was discovered as a test-only environment knob, but Make's internal source-list assignment does not accept environment precedence. Failed log /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/7f097566671506a33b7149191165d962.log, SHA256 prefix b6c720d68077. This interaction is introduced here, not an unrelated flake; Go's full package sweep had not run yet.

Removed the fixture's internal-variable override. Its private Makefile now supplies a no-op default recipe for omitted fake-compiler source prerequisites, leaving the real production source-list contract intact. All eight atomic tests pass, and both real outer-Make variants of make-test-env pass (247 derived knobs preserved). No production change or environment-scan exemption was needed. A fresh recorded full run remains required before close.

### 2026-10-16 — fresh 22:27 occurrence, bounded read-only follow-up

TL receipt c8db71a8-f340-436b-ac01-cfde208001e2 reports another primary Couch replacement after the 22:22 restart, with no TL build: bin/couch mtime 22:27:23, inode 749783777, unchanged vcs.revision 712a6478; pair:5 observed that the loaded Couch inode differs. Read-only stat confirms Couch's mtime/inode, pair-launch-helper mtime 22:22:47 and pair mtime 20:01:58. A timestamp-filtered shell-history search found couch-dev at 22:22:40 and no make/stage command in 22:20–22:30. This dev command is another candidate: couch-dev calls the proven make-build writer, but process birth at 22:22:40 does not establish when its build or exec finished (PID birth survives exec). Uneven binary mtimes and the generic scanner's duplicate Couch build do not prove a causal timeline.

Bounded active-slot scrollback tails and timestamp-selected wrap-event chunks around 22:25–22:29 showed no identified primary writer. No primary build, process mutation or filesystem tracing installation was performed. The exact fresh caller remains unknown; TL explicitly keeps this follow-up nonblocking for the atomic publisher.

Further read-only evidence resolves the exec-time ambiguity: lsof shows PID 99807's loaded Couch inode 749561659, while both of its listening message sockets were created at 22:22:49 (birth and mtime). Thus the supervisor was already listening by 22:22:49, before disk inode 749783777's 22:27:23 replacement. This excludes the initial couch-dev build merely finishing late; a subsequent writer remains unidentified. Socket paths: /tmp/pair-message-501/debdf0fb268a662a10ea064abcec7751132c8a42.sock and /tmp/pair-message-501/597a1d5494d215ea6158c1344955eb96677e7a75.sock. These are local ephemeral evidence, not an assumption about process birth.

### 2026-10-16 — full recorded verification passed

Corrected head f769895f18db7deb52e4e28e0bd3b8632c3724ad passed the complete declared suite (shell/editor checks plus go test ./... -count=1 -timeout 20m), exit 0 in 11m0s. Log /Users/xianxu/workspace/pair/.git/sdlc/local-checks/f769895f18db7deb52e4e28e0bd3b8632c3724ad/suite.log, SHA256 prefix 3afac57cd20f. Both introduced fixture defects are fixed; no unrelated failures were waived. The runner survived the 22:22 Couch restart. Subsequent commits only record the requested read-only writer investigation; production and tests remain exactly this passing head's versions.

### 2026-10-16 — BR-1 woven generic-scanner measurement

First close returned REWORK/BR-1, kind unverifiable: the disposable review checkout lacks Makefile.workflow, so its archive-only run could not exercise generic scanner admission. The reviewer independently confirmed the direct atomic publisher red/green and real Go build. Re-ran the missing dependency path in a private woven copy with the actual file resolved from this worker: /Users/xianxu/workspace/worktree/pair-slot7/ariadne/Makefile.workflow, SHA256 b816fcb7c31d3426775ea583e802835c756756a97c241f4610be51d7e1b345ec. No peer source was changed or scanner code reimplemented.

Retained reproducible fixture: /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair624-woven-proof-4bb6qatn. Run python3 -B tests/build-go-atomic-test.py AtomicBuildTest.test_root_build_has_one_publisher_per_binary_and_no_foreign_writes from that copy. With all three sentinels, exit 0, exactly three staged writes. Removing only Couch's new sentinel gives exit 1, four writes with direct bin/couch; removing only launch-helper's gives exit 1, four writes with direct bin/pair-launch-helper; removing both gives exit 1, five writes with both direct outputs. Each mutation was in the private copy, then both sentinels restored. report.json and present.log/couch-absent.log/helper-absent.log/both-absent.log retain source identity and actual outputs for review inspection and reproduction. This supplies the measured host evidence the archive checkout lacked; no production/test change or additional full suite is needed for this evidence-only answer.

## Revisions

### 2026-10-16 — bounded writer diagnosis approved

TL approval fdfe17db-5144-4cc2-8ad9-902345e90249 supersedes the original requirement to identify a non-explicit historical writer: acceptance is the proven direct-output writer class, one temp-sibling-plus-rename publisher, skip sentinels eliminating duplicate generic writers, deterministic atomicity tests and the foreign-PAIR_HOME isolation guard. The exact historical caller remains unknown. The TL reports running make build in pair:0 around that period while rebuilding for #618; this is a likely candidate, not an observed process-to-failure correlation. The TL owns filing the ariadne generic-build half; this issue edits Pair-owned files only.

### 2026-10-16 — verification checkbox boundary

The final plan checkbox now ends at its completed verification work; close review and landing remain the subsequent binary-owned workflow gates, rather than being ticked before they occur. No implementation or acceptance scope changed.
