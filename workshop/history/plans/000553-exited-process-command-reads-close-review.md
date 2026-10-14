# Boundary Review — pair#553 (whole-issue close)

| field | value |
|-------|-------|
| issue | 553 — procutil: skip exited processes during native command reads |
| repo | pair |
| issue file | workshop/issues/000553-exited-process-command-reads.md |
| boundary | whole-issue close |
| milestone | — |
| window | bc761e7f8db87f65ad4df89542aeeeed25572096..b3340038a14cb6f2462d50297df345e7f1b6cbf1 |
| command | sdlc close --issue 553 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-14T11:55:33-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The implementation matches the Spec: exited processes are omitted, genuine inspection failures remain visible, and identity fences remain intact. No blocking code findings. Operator detach/restart acceptance remains pending under the documented landing hold; this review does not establish that acceptance.

Done-when: current — covers exit races, zombies, valid servers, uncertainty, identity fencing, and pre-landing acceptance.

```findings
{}
```

1. **Strengths**
   - One shared classifier handles wrapped gone errors across all changed callers (`cmd/internal/procutil/command.go:18`).
   - Snapshot enrichment drops replaced generations while retaining unavailable identity evidence (`command.go:79`).
   - Diagnostics handles disappearance during its later environment read (`cmd/internal/diagnosticlog/proof.go:234`).
   - The atlas documents the changed inspection contract (`atlas/architecture.md:1317`).

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage**
   - All three affected packages passed `go test -race`.
   - Restoring base behavior through a disposable Go overlay made regression tests fail for snapshot exit races, PID reuse, real Darwin zombies, launcher discovery, and diagnostic environment disappearance.
   - The full recorded suite and staged operator repro were not independently repeated.

6. **Architecture**
   - **ARCH-DRY: pass** — callers share the error classifier.
   - **ARCH-PURE: pass** — classification remains pure; process reads use existing injected seams.
   - **ARCH-PURPOSE: pass** — all scoped consumers use the shared classification, including the second diagnostic read.
   - README gate: no new commands, flags, configuration, or usage surface.

7. **Plan revisions:** None required. Retain the explicit landing hold until operator acceptance is confirmed.

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.56150675s
head: "b3340038a14cb6f2462d50297df345e7f1b6cbf1"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000411
