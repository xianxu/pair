# Boundary Review — pair-m2#865 (milestone M1)

| field | value |
|-------|-------|
| issue | 865 — Live Couch sustains ~1.3 cores: attribute and bound it |
| repo | pair-m2 |
| issue file | workshop/issues/000865-live-couch-burns-1-2-cores-continuously-in-the-chrome-cast-recorder-png-decode.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 64e0a8452388843b1fd1791059368682eec1236e..5905c8c7dcb4c326932e740b598f8573894c3507 |
| command | sdlc milestone-close --issue 865 --milestone M1 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-24T20:32:31-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned range delivers M1’s attribution and planning scope without introducing M2 production changes. Independent inspection of the retained profiles, executable, metadata and source confirms the principal attribution claims. The documents preserve the limits of startup measurements and invalid synthetic controls. No blocking findings.

```findings
{}
```

1. **Strengths**

   - Executable and all four profile-artifact hashes match [live-profile.md](/Volumes/TestScratch/review-checkouts/98dea3c352b55762/workshop/evidence/000865/live-profile.md:7). Build metadata identifies the documented clean revision.
   - Re-running pprof confirms 44.80 CPU seconds over 30.22 seconds, capture’s 46.11% allocation share, and approximately 9.32 GiB through synchronized publication capture. Webcast/PNG filtering matches no CPU samples.
   - Retained manifests substantiate the rejected N5 qualification: four workers stopped before capture. Historical issue/design contents match their preserved source commit, with only explanatory wrappers added.
   - The plan preserves independent consumer clones and correctly describes resize hold cancellation. M2’s allocation correction and regression proof remain explicitly pending.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Independently passed:

   - Focused Endpoint sync, resize, publication and EndInput tests.
   - Pure Geometry validation and Frame validation/clone tests.
   - Pinned-range `git diff --check`.

   Existing Endpoint tests use the actual emulator and injected ttyio fake. No executable behavior changes or prior findings require a revert-to-red experiment at this boundary. M2’s allocation tests remain required before its correction can be accepted.

6. **Architectural notes for upcoming work**

   | Principle | Assessment |
   |---|---|
   | ARCH-DRY | Pass — reuses the landed profiler and existing terminal seams. |
   | ARCH-PURE | Pass — named pure entities contain deterministic logic; emulator access stays in Endpoint. |
   | ARCH-PURPOSE | Pass — delivers measured attribution while explicitly retaining unresolved aged-CPU and fseventsd questions. |
   | ARCH-MOCK | Pass — no new dependency seam; existing integration uses stateful ttyio doubles and the real emulator. |
   | ARCH-CONSTRAINTS | Pass — measurements distinguish CPU, sampled allocation and retained memory; M2 has counted acceptance budgets. |
   | ARCH-SECURE | Pass — no new parser, credential handling or external-state mutation. |
   | ARCH-ORDER | Pass — documented transitions match current source; no state transitions change in this range. |
   | ARCH-FUNERAL | Pass — retained diagnostic files have an existing investigation-owner removal policy; no runtime artifact family is added. |
   | ARCH-EVOLVE | Pass — no persisted format or rewrite behavior changes. |

   Core-concept locations exist and classifications fit their implementations. The `Endpoint.capture` modification is clearly scoped to upcoming M2. Atlas is updated; this range adds no user-facing command, flag or configuration requiring a README change.

7. **Plan revision recommendations:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"50d58553-4899-4b00-b781-66771bbe80f2","repository":"github.com/xianxu/pair","issue_id":"000865","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/98dea3c352b55762","reviewed_head":"5905c8c7dcb4c326932e740b598f8573894c3507","slot":"pair:9","requested_at":"2026-10-25T03:30:50.884647Z"},"phase":"finished","pid":66530,"started_at":"2026-10-25T03:30:51.048024Z","finished_at":"2026-10-25T03:32:29.078403Z","session_id":"01a1711c-7022-76c0-a096-def9c42fb319","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.520807375s
head: "5905c8c7dcb4c326932e740b598f8573894c3507"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
