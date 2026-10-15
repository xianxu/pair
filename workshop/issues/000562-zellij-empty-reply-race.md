---
id: 000562
status: open
deps: []
github_issue:
created: 2026-10-14
updated: 2026-10-14
estimate_hours:
card_mirror: 'ba16ce1b51ffb8e832e24b3fd3efb72ebbd0c90a' # card fields mirrored from issue-cards; edit via sdlc
---

# Zellij CLI queries can finish before their reply

## Problem

Zellij0.45.1 has a source-level race in which a global UnblockInputThread can arrive on a temporary CLI/probe connection before the response that connection expects. A CLI query may finish successfully without its Log; a ConnStatus probe may silently omit a live session. #218 retained inconsistent successful inventories, but did not capture IPC ordering, so the mechanism is not yet established as the cause of those observed failures.

### Captured evidence

- In #218 MATRIX5, two successful commands about50ms apart returned20 then21 session names. The --short reply omitted the selected owned session; the immediate --no-formatting reply included it. Both had empty stderr and exit0. Exact stdout/times/status are retained locally in /tmp/pair218-MATRIX5.vHTGIP/listing-disagreement.json; the issue remains understandable without this temporary artifact. No automatic rerun or concurrency conclusion followed.
- [Main conformance run38488929394](https://github.com/xianxu/pair/actions/runs/38488929394/job/115609240748), source29dce157654ab7850f6a654738d94547bd841ded, macos-26-arm64, pinned Zellij0.45.1: TestRecoveryRealHelperAndSessionConformanceLive/warm failed at2026-10-14T00:15:50Z with `strconv.Atoi: parsing "": invalid syntax`. The failing make target was test-couch-zellij-live.
- CI evidence boundary: the Atoi input came from source-agent.pid, not CLI stdout. At that revision the source script published the PID via shell redirection, while waitFile accepted os.Stat success before reading it. A zero-length publication interval is a separate plausible cause. The CI log alone does not establish an empty CLI reply or the protocol race; classify this independently before using it as upstream evidence. Source pointers: [PID read](https://github.com/xianxu/pair/blob/29dce157654ab7850f6a654738d94547bd841ded/cmd/internal/couchcore/recovery_conformance_live_test.go#L182-L195), [existence-only wait](https://github.com/xianxu/pair/blob/29dce157654ab7850f6a654738d94547bd841ded/cmd/internal/couchcore/conformance_live_test.go#L35-L45).
- MATRIX4 separately stopped on row7 Leave refusing invalid live pane JSON. The failing authority query was not captured in that version; do not label its bytes empty or infer the same cause.

## Spec

Diagnose the reply/discovery race without retries, sleeps or treating unknown ownership as absence. Preserve exact generation and command authority. A native live server and successful CLI discovery are separate evidence.

### Upstream report draft — internal only, not submitted

**Title:** v0.45.1: global input-unblock broadcasts can finish CLI queries and exclude live sessions from discovery.

The server registers temporary CLI and session-probe sockets alongside interactive clients, and broadcasts UnblockInputThread to every registered client. Pane/tab close and plugin reload can emit these broadcasts. The CLI receive loop accepts an unblock as successful completion before an output-bearing Log may arrive. Discovery sends ConnStatus and reads exactly one response; anything other than Connected excludes the socket.

Primary source pointers: [registration](https://github.com/zellij-org/zellij/blob/v0.45.1/zellij-server/src/lib.rs#L949), [broadcast](https://github.com/zellij-org/zellij/blob/v0.45.1/zellij-server/src/lib.rs#L1287), [pane/tab emitters](https://github.com/zellij-org/zellij/blob/v0.45.1/zellij-server/src/pty.rs#L580), [plugin emitter](https://github.com/zellij-org/zellij/blob/v0.45.1/zellij-server/src/plugins/mod.rs#L440), [CLI receive loop](https://github.com/zellij-org/zellij/blob/v0.45.1/zellij-client/src/cli_client.rs#L206-L252), [discovery probe](https://github.com/zellij-org/zellij/blob/v0.45.1/zellij-utils/src/sessions.rs#L145-L170).

Expected regressions: inject an unrelated unblock before query Log and before probe Connected; the query must preserve its output and discovery must retain the live session. These are source-supported mechanisms, not captured runtime ordering for our failures. Explicit --session actions still enumerate globally before connecting ([dispatch](https://github.com/zellij-org/zellij/blob/v0.45.1/src/commands.rs#L404-L451)), so changing our listing format or naming a target cannot eliminate exposure.

External posting, upstream installation and operator environment changes require their own authorization. Do not publish local fixture paths or session identifiers upstream. Follow-up #563 owns passive IPC-frame observation and the renewed parallelism experiment. Historical cold attribution remains #560.

## Done when

- A deterministic protocol regression establishes whether unrelated unblock can finish a query before Log and exclude a live ConnStatus probe; preserve raw command outcomes and source version.
- The CI empty-PID sidecar is classified separately, with a regression for complete PID publication if that boundary is responsible; no claim that Atoi proved empty CLI stdout.
- A reviewed correction or explicitly assigned upstream dependency prevents unknown/empty replies from becoming positive absence or detached authority, without retry/sleep workarounds or weakened ownership fences.
- The upstream draft reflects measured findings and clearly distinguishes hypotheses; external submission occurs only with authorization.

## Plan

- [ ] Reproduce the protocol interleavings deterministically and classify the CI PID publication boundary independently.
- [ ] Design the smallest correction or upstream dependency with the TL; preserve uncertainty and exact-generation effect guards.
- [ ] Verify stateful regressions and applicable live conformance, then hand the stable observation contract to #563.

## Log

### 2026-10-14

Split from #218 during warm-fleet measurement. Filed as an unassigned follow-up; no implementation or upstream post performed. Full evidence and incomplete matrices remain recorded in #218. CI log retrieved through gh; source at the exact failing commit inspected rather than attributing the Atoi message to CLI output.
