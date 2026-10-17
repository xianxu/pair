---
id: 000640
status: codecomplete
deps: []
github_issue:
created: 2026-10-17
updated: 2026-10-17
estimate_hours:
collaboration-mode: collaboration
card_mirror: '5616e249d1c12a4fc64af426bc8cee1b425e442e' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-17T09:08:23-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
actual_hours: 2.25
---

# Native fullscreen chord fixture loses its disposable session during conformance

## Problem

Native TestFullscreenChordZellijLive intermittently loses its disposable Zellij session: list-panes --json --all returns exit 1, "There is no active session!". Seen while qualifying #639, before any toggle on the candidate attempt; reproduced later during the original three-state scenario with the original layoutcmd code and test body. Root cause is unknown. This is a session-loss symptom, distinct from #636's ErrZellijNoReply.

## Spec

Investigate the native fixture/server exit and make the actual-chord conformance stable without sleeps, longer deadlines or hiding failed queries. Preserve all input, geometry and focus assertions.

## Done when

**Diagnosis closeout (operator decision, 2026-10-17; see Revisions):** the original fixture/product-fix criteria below are transferred to #647. This issue closes on the native diagnosis, retained reproduction, caller-impact audit and upstream/follow-up references, without claiming the native defect is fixed.

- Native crossed replies and false absence/empty-success behavior are reproduced and documented with explicit limits on the inferred internal schedule.
- One-shot and periodic caller effects are audited, and the operator's selected Pair-side hardening is tracked as #647; #636 follows that hardening.
- Existing upstream reports/fix are linked, with no new upstream publication or local Zellij build.

Historical implementation criteria (superseded for this issue's close):

- A deterministic reproduction identifies why the disposable session disappears.
- The fixture/product cause is fixed with a regression and native conformance proof.

## Plan

- [ ] Capture an isolated native failure and server/client diagnostics; identify a deterministic trigger.
- [ ] Add a deterministic regression and minimal cause-level fix while preserving every chord/input/geometry/focus assertion.
- [ ] Verify native conformance and the recorded pre-close suite, then close and let the landing turn integrate main.

## Log

### 2026-10-17 — #639 qualification controls
- 2026-10-17: closed — Diagnosis-only close per operator transfer to #647: retained native first-response controls reproduce crossed replies (65/2000 parallel, 57/2000 sequential), protocol controls connect false absence and empty success, and source/caller/upstream audits state residual limits. No production or test change; original implementation checklist retained as historical and transferred to #647. TL owns project updates. Exact-head da26fa0a8900192c0c95f0174e9fa34f57116163 recorded suite passed (11m59s); native defect is not claimed fixed.; review verdict: SHIP

Candidate first run: PAIR_LIVE_ZELLIJ=1 go test ./cmd/internal/layoutcmd -run '^(TestFullscreenZellijLive|TestFullscreenChordZellijLive)$' -count=1 -v (the first named test does not exist; only chord ran). Failed after 11.18s at fullscreen_chord_live_test.go:149, before toggle input, list-panes reports no active session.

Original control source: 1ac101a17404a392927a4dac83676b43e6030b9a. All tracked layoutcmd Go files restored via go test/go build overlay from this Git commit. Build original-pair with go build -overlay ORIGINAL.json -o original-pair ./cmd/pair-go, then PAIR_TEST_BINARY=original-pair PAIR_LIVE_ZELLIJ=1 go test -overlay ORIGINAL.json ./cmd/internal/layoutcmd -run '^TestFullscreenChordZellijLive$' -count=1 -v. This passed draft-nvim and agent-wrapper scenarios, then failed after 4.22s during terminal-shell-bottom at list-panes (line308), same no-active-session diagnostic; restore-tiling current-tab-info exit1 also logged. The precise failing phase differs, so this establishes only the pre-existing session-loss family. Workbench help text on disk was newer; original planner, runtime, test body and supplied binary's layout code came from the overlay. No live tests overlapped these controls.

Local evidence root /tmp/pair639-7xyqf5rg; original.json and original-*.go preserve the exact source; original-build.log, environment.json and original-chord-correct-binary.log retained. Original binary SHA256 6b99f66417c5204867841b112751d920e7754b21d24a482d17dda0dc7bb736bf; control log SHA256 225feb439e35694e73558be8dc20488af695c9437f8a63fac3473229b8b214f4.

An earlier original-test-only run used a candidate Pair binary, so it is excluded as a baseline. A separate direct-conformance invocation inherited Couch's long ZELLIJ_SOCKET_DIR and was refused before render (session name must be less than 0 characters); short private socket roots are required for that fixture and this is not the session-loss reproduction. Candidate Focus-return errors (already-focused exit2 and skipped same-pane return-focus) are #639's defects and are being corrected there, not assigned to this issue.

### 2026-10-17 — resumed diagnosis

Fresh dispatch claimed on main 30e3f624ccf13df378f2dac5c5c036ebdc0015ce; current native Zellij is 0.45.1. The retained original-pair control log still contains both no-active-session and restore-tiling/current-tab-info failure. Root cause remains unknown. A bounded read-only upstream audit confirms ordinary CLI actions send ClientExited after their response; query completion does not prove server-side disconnect cleanup completed. Cached screen.rs also has a last-selectable-tiled-pane close path, but neither that path nor the startup RemoveClient race is yet established as the cause. Server routing/lib.rs and native fatal diagnostics are missing from the existing evidence.

The chord fixture clears every inherited ZELLIJ-prefixed variable (including an externally supplied socket root), then resets TMPDIR. Diagnostic controls will exec the unchanged real binary through a private-environment wrapper, giving native processes short owned socket/log roots without masking exit codes or failed queries. Preserve native logs outside testing.TempDir cleanup. ARCH-ORDER: observe the exact failure ordering before choosing a readiness change. ARCH-SECURE/FUNERAL: diagnostic sessions and files belong to the private fixture; no operator session or :0 binary is touched.

### 2026-10-17 — isolated diagnostic controls

Evidence root /tmp/pair640-mi9l7uyn holds current Pair/test binaries, a sterile native-env.json produced by tests/lib/pair-test-env.sh, and an exec-only zellij wrapper fixing short private native TMPDIR/socket roots. Tests run from cmd/internal/layoutcmd. An initial invocation from the repo root failed locating zellij/config.kdl before session startup; harness-wrong-cwd.log is excluded.

baseline-1.log: unchanged chord test fails after 16.03 s restoring the terminal-shell-top geometry; current-tab-info exit1 was logged, but the private native log shows a live initialized server until fixture cleanup, without a native panic. This is distinct from proven session disappearance. A logging-only overlay of execZellijContext preserves each invocation/output/error and records argv, effective native executable/session/socket root and ExitError stderr. baseline-trace-1.log fails after 7.57 s in terminal-shell-top on unexpected end of JSON input. query-trace.log identifies a list-panes --json --command --state --geometry command returning success with empty stdout; all traced calls resolve the intended wrapper and private socket/session, excluding wrapper bypass. No traced command has an exit error in this run. Neither run yet reproduces the original no-active-session diagnostic; root cause remains open. No production/test implementation changed and no failed query was retried or hidden.

Exact native v0.45.1 source audit (copies retained here: server-lib.rs, server-route.rs, commands.rs, sessions.rs) distinguishes discovery failure from server exit: no-active-session is emitted before attachment when the session inventory is empty. Inventory filters sockets through ConnStatus probes; refused connections unlink their socket paths. Startup RemoveClient can unwrap absent metadata, but neither that race nor a socket unlink has been observed here. A possible ConnStatus duplicate-removal/client-ID-reuse ordering is an unproven hypothesis, not a fix justification. ARCH-ORDER: preserve separate failure observations until a deterministic mechanism connects them.

### 2026-10-17 — false absence and protocol controls

baseline-trace-2.log reproduces Pair current-tab-info exit1 with exact stderr "There is no active session!" at 09:25:44.494; the observer subsequently succeeds on list-panes through the same session/socket root until teardown. This disproves treating the message as server death for this run. Its final failure is restored geometry (13.11 s). The observer additionally received two successful empty list-panes replies and an empty list-clients reply. All failures remain retained; nothing was retried within an observation.

A transparent frame-logging socket proxy (wire-proxy.py, diagnostic test overlay wire-overlay.json) changes scheduling: wire-run-1.log and wire-run-2.log each pass, followed by all ten in wire-count10.log (3.30–3.68 s). These are diagnostic perturbation controls, not fixes or acceptance evidence. wire-trace.jsonl retains native frames; none of those passing runs captured the bad probe ordering. Proxy/client/server resources were private and joined on fixture cleanup.

protocol-control.py drives the installed, unchanged Zellij 0.45.1 CLI against an owned live Unix-socket protocol fake, using separate length-prefixed protobuf frames. Connected then a JSON Log yields exit0 / [] JSON. UnblockInputThread before Connected makes the real CLI return exit1 / "There is no active session!" despite the listener remaining alive. Connected followed by query UnblockInputThread before the JSON Log yields exit0 / empty stdout. Exact requests/results are in protocol-control.jsonl. An initial fake incorrectly put two oneof variants inside one frame, then timed out; that setup failure is excluded and the corrected fake uses distinct frames. This deterministically establishes both native client failure modes, but does not prove the real server's triggering schedule.

Version-pinned server source has a concrete stale-ID candidate: ConnStatus both removes the client in its handler and queues a second RemoveClient on router exit; IDs reuse the lowest free integer. ServerOsInputOutput clones share one numeric-ID-to-sender map; send_to_client resolves the current sender, and remove_client unconditionally deletes by number. Thus delayed cleanup/response can target a replacement connection. This remains a candidate until the real bad frame/interleaving is captured. No sleeps, deadlines, native installation, Pair production code or conformance assertions changed. The present evidence exceeds a simple fixture-readiness diagnosis; request hub scope guidance before any native dependency change or new transport/serialization mechanism.

### 2026-10-17 — native response misrouting reproduced without Pair or a proxy

The bounded read-only investigation established native cross-connection response misrouting. No Pair implementation or fixture fix is proposed, and this issue is not ready for close/landing. Its original “session disappears” premise is not valid for the fresh observed failure: the native server stays alive while discovery rejects its response.

The minimal diagnostic starts one owned quiet native session with the existing ControlledZellij helper, then sends only the real CLI's ConnStatus and ListPanes protobuf messages over independent Unix sockets. Each query sends ClientExited on completion. No Pair child, chord, proxy, sleep, failed-operation retry, native rebuild, source edit or installed-setting change is involved. Every request records its own first response. Independent requests continue for the declared sample budget; a failure is never retried as a passing observation.

| Control | Independent requests | Bad first replies | Exact crossed replies | Other premature replies | Duration |
| --- | ---: | ---: | --- | --- | --- |
| 8 workers × 250 | 2,000 | 65 | 8 pane queries receive Connected; 1 probe receives pane JSON Log | 39 probes + 17 queries receive Exit(Disconnect) | 1.04 s |
| 1 worker × 2,000, requests sequential | 2,000 | 57 | 1 pane query receives Connected; 1 probe receives pane JSON Log | 54 probes + 1 query receive Exit(Disconnect) | 1.21 s |

Both controls fail on their first recorded attempt. The sequential result rules out merely serializing callers as an adequate fixture fix: native cleanup outlives a successful response. Crossed request-specific replies are stronger evidence than disconnects alone; neither a probe-only connection legitimately receiving pane JSON nor a query-only connection receiving Connected has a valid dispatch path in the audited implementation.

Native source-supported mechanism (zellij-server/src paths, release-tag source core files byte-identical to the existing checkout):

1. lib.rs:655–672 allocates the lowest unused numeric ClientId. Listener acceptance inserts the current sender for that number (lib.rs:949–960; os_input_output.rs:464–475).
2. Both ordinary ClientExited and ConnStatus cause duplicate cleanup. ClientExited queues RemoveClient at route.rs:2676–2678 and queues it again after breaking the router at :2941. ConnStatus queues its instruction at :2685–2687 and the router's final RemoveClient at :2941; the server's ConnStatus handler itself already sends Connected and removes that client (lib.rs:1617–1619).
3. ServerOsInputOutput clones share Arc<Mutex<HashMap<ClientId, ClientSender>>> (os_input_output.rs:273–276). send_to_client resolves the current sender solely by number (:448–460), while remove_client deletes that number unconditionally (:495–503). Neither operation carries a connection generation.
4. A valid interleaving is: old O's first cleanup frees n; A reuses n; O's delayed second cleanup removes A; B reuses n; A's pending response addressed to n reaches B. This directly explains both crossed reply kinds. Removing A's sender drops its channel; its sender thread sends Exit(Disconnect) after draining (os_input_output.rs:233–243).

The exact internal schedule of each native sample was not instrumented. Observed native cross-connection misrouting is proven; duplicate cleanup plus reusable unversioned IDs is the source-supported cause, not a claim that a native debugger captured every instruction in that interleaving. A skeptical independent source audit found no legitimate alternative routing for those crossed replies.

The fresh deterministic CLI protocol controls now replay the actual native first frame Exit(Disconnect), hex 1a020805, as well as the earlier Unblock control. A live listener sending Disconnect before Connected causes the unchanged installed CLI to emit exit1 / “There is no active session!”; Connected then Disconnect before the query's JSON causes exit0 / empty stdout. Normal Connected + JSON returns exit0 / valid []. This links the captured native frames to both #640's false absence and #636's empty reply family. It does not prove every historical #636/#640 failure shares this cause.

Installed Zellij is 0.45.1, binary SHA256 604c684330fe097bc1363ada020d1b9b0ad17caa48d633e838869dfe9fc49e9c. Source release tag v0.45.1 dereferences via Git to efd8fd5a89a20c07a111d248ad7fce53848d2c18; existing source checkout HEAD is a162323384dff3fbd9c68a4c0971f4f6efca424a. This is source-version provenance, not proof of the Homebrew binary's build commit. The checkout remains clean.

Retained artifacts under /tmp/pair640-mi9l7uyn: native_raw_probe_test.go and raw-overlay.json are the exact tested parallel diagnostic; native_raw_sequential_test.go/raw-sequential-overlay.json change only worker/sample counts and output path. raw-run.log/raw-probe-parallel.jsonl and raw-sequential-run.log/raw-sequential.jsonl retain every bad first reply. protocol-control-expanded.jsonl retains all five fake controls. native-trigger-manifest.json pins artifact SHA256s. Parallel result log SHA256 f2f61a6262577b04b3ffb109b0ae7c31aeb1bbfbb9ec2bdf20edd4e7fa9782e5; sequential result log SHA256 5f8336dc8012dfeab4bf1ef64d8f647ae56c88e5ff2fe712c0b7a283dbd0cc82.

To reproduce locally from this branch: replace fullscreen_chord_live_test.go through the retained Go overlay, compile ./cmd/internal/layoutcmd with go test -trimpath -overlay /tmp/pair640-mi9l7uyn/raw-overlay.json -c, then run only TestNativeRawProbeDiagnostic from cmd/internal/layoutcmd under retained native-env.json. That sterile environment supplies the unchanged exec-only native wrapper and owned short socket/log roots. This is a diagnostic overlay, never a proposed test transport or assertion relaxation. The exact small diagnostic source is preserved below so the proof is not only an ephemeral path reference; change its evidence root when moving machines. Existing helper owns session shutdown, and both runs completed cleanup. No owned native process remains. The operator's 10:48 Couch restart lost no background work.

```go
package layoutcmd

import (
 "context"
 "encoding/binary"
 "encoding/hex"
 "encoding/json"
 "fmt"
 "io"
 "net"
 "os"
 "path/filepath"
 "sync"
 "testing"
 "time"

 "github.com/xianxu/pair/cmd/internal/pairlifecycletest"
)

func TestNativeRawProbeDiagnostic(t *testing.T) {
 ctx,cancel:=context.WithTimeout(context.Background(),2*time.Minute);defer cancel()
 t.Setenv("SHELL","/bin/sh")
 for _,k:=range []string{"HOME","XDG_CONFIG_HOME","XDG_DATA_HOME","XDG_CACHE_HOME","XDG_RUNTIME_DIR"} {t.Setenv(k,t.TempDir())}
 config:=filepath.Join(t.TempDir(),"config.kdl")
 if err:=os.WriteFile(config,[]byte("pane_frames false\nshow_startup_tips false\nshow_release_notes false\n"),0600);err!=nil {t.Fatal(err)}
 session:=fmt.Sprintf("raw-%d",os.Getpid())
 fixture,err:=pairlifecycletest.StartControlledZellijWithOptions(ctx,session,pairlifecycletest.ZellijOptions{ConfigFile:config});if err!=nil {t.Fatal(err)}
 defer func(){if err:=fixture.Close();err!=nil {t.Log(err)}}()
 path:=filepath.Join("/tmp/pair640-mi9l7uyn/sock/contract_version_1",session)
 log,err:=os.Create("/tmp/pair640-mi9l7uyn/raw-probe.jsonl");if err!=nil {t.Fatal(err)};defer log.Close()
 var mu sync.Mutex;var wg sync.WaitGroup;bad:=0;total:=0
 query,_:=hex.DecodeString("4a0b0a07920604280130012001")
 send:=func(c net.Conn,p []byte)error {frame:=make([]byte,4+len(p));binary.LittleEndian.PutUint32(frame,uint32(len(p)));copy(frame[4:],p);_,err:=c.Write(frame);return err}
 for worker:=0;worker<8;worker++ {wg.Add(1);go func(worker int){defer wg.Done()
  for sample:=0;sample<250;sample++ {
   probe:=(sample+worker)%2==0;payload:=query;if probe {payload=[]byte{0x6a,0}}
   c,err:=net.DialTimeout("unix",path,time.Second)
   var reply []byte
   if err==nil { _=c.SetDeadline(time.Now().Add(time.Second));err=send(c,payload)
    if err==nil {var header [4]byte;_,err=io.ReadFull(c,header[:]);if err==nil {n:=binary.LittleEndian.Uint32(header[:]);if n>1048576 {err=fmt.Errorf("oversize %d",n)} else {reply=make([]byte,n);_,err=io.ReadFull(c,reply)}}}
    if !probe { _=send(c,[]byte{0x5a,0}) };_ =c.Close()
   }
   ok:=err==nil&&len(reply)>0&&((probe&&reply[0]==0x22)||(!probe&&reply[0]==0x2a))
   mu.Lock();total++;if !ok {bad++;shown:=reply;if len(shown)>200 {shown=shown[:200]};_ =json.NewEncoder(log).Encode(map[string]any{"worker":worker,"sample":sample,"probe":probe,"error":fmt.Sprint(err),"reply":hex.EncodeToString(shown),"time":time.Now().Format(time.RFC3339Nano)})};mu.Unlock()
   if ctx.Err()!=nil {return}
  }
 }(worker)}
 wg.Wait();t.Logf("independent requests=%d bad-first-responses=%d",total,bad)
 if bad>0 {t.Fatalf("native protocol misroutes: see raw-probe.jsonl")}
}
```

ARCH-ORDER/FUNERAL: native connection lifetime must cover its pending cleanup and responses; freeing/reusing a numeric identity before that work finishes breaks ownership. No sleeps, longer waits, blind query retries, Pair transport changes or known-cause exemptions were introduced. Stop at this native finding and bring any dependency fix/upstream publication to the TL/operator, as required by the approved scope. Pair:6 receives the evidence and the released native-run window; #636 remains independently owned there.

### 2026-10-17 — upstream search and operator disposition

The native source mechanism is already reported in [zellij-org/zellij#5270](https://github.com/zellij-org/zellij/issues/5270): ConnStatus double cleanup, immediate client-ID reuse and stale instructions targeting a replacement client. [#5382](https://github.com/zellij-org/zellij/issues/5382) reports concurrent list-panes exit0 with empty output; [#5546](https://github.com/zellij-org/zellij/issues/5546) reports stale cleanup evicting a replacement attach. [PR #5272](https://github.com/zellij-org/zellij/pull/5272) closed without merge; [PR #5630](https://github.com/zellij-org/zellij/pull/5630) merged September24 with retirement of client IDs. The [main changelog](https://github.com/zellij-org/zellij/blob/main/CHANGELOG.md) lists that fix as unreleased; the latest released/installed binary remains0.45.1. Our reproduction has not qualified patched main. No upstream issue, comment, PR or local native build was produced.

Read-only caller audit distinguishes periodic observations from one-shot effects. The message listing poller refreshes both listing forms nominally each second and retains last good for5 seconds on error. Inventory refreshes on lifecycle/UI events. Title liveness misses debounce across five60-second ticks. Wait-watcher is a15-second registry/self-report/tail observer, not a pane-reply authentication fence. Process PID/start checks protect lifetime and quiescence but cannot undo a wrong focus, paste, layout, note anchor or restart decision. Valid same-kind replies can differ by tab, flags, focus and time; global discovery can silently omit a live server when a probe rejects a response. The observed2.85–3.25% is a synthetic raw-request rate, not a production or user-action rate. The race is per server, not evidence of cross-daemon/session response transfer.

Authenticated operator dispatch delegates Pair-side hardening to #647: classify every audited caller as re-checked/hardened/fail-closed; add request-compatible reply validation and independent confirmation before one-shot mutations; clipboard fails closed; fake-Zellij tests per hardened caller plus native refusal proof. Full-flow plan must be posted before code. #636 remains separately owned by pair:6 and follows #647. #640 now closes solely as diagnosis, with no production changes and no claim of fixing Zellij.

## Revisions

### 2026-10-17 09:30 — approved read-only native-trigger investigation

TL approved bounded investigation in the existing Zellij checkout, read-only, time-boxed to approximately two hours (stop around 11:30 local and report). No upstream issue/PR, dependency edits/installation, retries/sleeps/transport machinery in Pair tests or known-cause exemption is authorized. If stale-ID race is confirmed, retain minimal reproduction in this issue and bring any upstream publication to the TL first. This replaces the assumption that the next step is necessarily a quick fixture-only implementation. Production and tests remain unchanged. Pair:6 now owns #636 and has been sent the shared empty-reply findings; native runs are serialized between slots.

### 2026-10-17 — diagnosis complete; implementation transferred to #647

Operator chose option(c): Pair-side hardening, with no local Zellij build and adoption of upstream PR#5630 when released. Delta: supersede #640's original implementation acceptance criteria/unchecked fix tasks with the diagnosis closeout bullets above; preserve the historical plan unchanged. Follow-up #647 owns all hardening implementation, caller atlas classifications, fake tests and native refusal acceptance. #640's verification is the retained native/protocol controls and source/caller/upstream audit; closing it does not assert native conformance passes. The narrow close plan-check acknowledgment applies only to those transferred implementation tasks.
