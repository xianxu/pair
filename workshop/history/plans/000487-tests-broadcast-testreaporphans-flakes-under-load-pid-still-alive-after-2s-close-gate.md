---
gate: boundary-review
issue: 487
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-12T13:28:52-07:00"
      dispatched: "2026-10-12T13:26:49-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: waitDead and startOrphan hand-roll t.Deadline() polling that cmd/internal/testwait already provides (ARCH-DRY)
          detail: Instances in window - guard_test.go:107-116 (waitDead) and cloudflared_test.go:285-291 (startOrphan sleep-child loop). Both re-implement testwait.Until with a divergent 1s margin vs testwait's 10s/half-remaining. Replace with testwait.Until(t, what, cond).
          family: reuse-testwait-deadline-helper
          round: 1
        - id: BR-2
          severity: Minor
          title: Sibling fixed wall-clock windows remain beside the swept waitDead (ARCH-PURPOSE)
          detail: 'guard_test.go:92 waitFile 5s; guard_test.go:131 guard-did-not-stop 5s; cloudflared_test.go:218 Close took >3s; live_test.go:65 15s teardown. Same rule as #480; waitFile fits testwait.Until directly.'
          family: wallclock-window-under-load
          round: 1
        - id: BR-3
          severity: Minor
          title: startOrphan relies on /bin/sh not exec-optimizing the final command
          detail: Using `sleep 300 & wait` (as the guard test does) keeps the shell as parent on any shell; current form fails loudly rather than flakes if a shell execs, so low risk.
          family: fixture-shell-portability
          round: 1
        - id: BR-4
          severity: Minor
          title: Done-when 200 runs under load were run at ambient load only, with an uncommitted delay variant standing in
          detail: Disclosed in the Log; the 500ms delayed-reap reproduction is not checked in, so it cannot be re-run.
          family: done-when-evidence-substitution
          round: 1
      recipe: small-diff-review
      isolation: fenced
      blocked: true
    - "n": 2
      timestamp: "2026-10-12T13:40:37-07:00"
      dispatched: "2026-10-12T13:36:09-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: waitDead (guard_test.go:104-107), waitFile (guard_test.go:91-100) and startOrphan's settle loop (cloudflared_test.go:285-288) all now call testwait.Until; no hand-rolled t.Deadline polling remains in the window.
          round: 2
        - id: BR-2
          disposition: not-addressed
          note: 'waitFile is swept, but positive event waits in the same guard/cloudflared family still use fixed windows: guard_test.go:69 (10s), 122/141/162 (5s), cloudflared_test.go:239 (5s), live_test.go:65 (15s). Fix: use select on testwait.After(t). The Log''s out-of-scope list (host/notes/pointer/server/session) does not cover these files. Minor, non-blocking.'
          round: 2
        - id: BR-3
          disposition: not-addressed
          note: The fixture is still a bare `sleep 300` as the script's last command. bash and dash do not exec-optimize the last command of a script file, and the settle check would turn any such shell into a deterministic, named failure, so this stays low-risk Minor. `sleep 300 & wait` would close it.
          round: 2
        - id: BR-4
          disposition: addressed
          note: 'The Log now names the permanent guard: startOrphan''s settled-state check fails deterministically on the causal regression (exec), so the substitute delay variant no longer carries the evidence. The 200/200 -race runs at ambient load up to 25 are disclosed.'
          round: 2
      recipe: small-diff-review
      isolation: fenced
      reviewed: 391d0219603083b5f25cc13781b1b28f46422960
      blocked: false
---

# Gate ledger — pair#487 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-12T13:28:52-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `reuse-testwait-deadline-helper` waitDead and startOrphan hand-roll t.Deadline() polling that cmd/internal/testwait already provides (ARCH-DRY)
  Instances in window - guard_test.go:107-116 (waitDead) and cloudflared_test.go:285-291 (startOrphan sleep-child loop). Both re-implement testwait.Until with a divergent 1s margin vs testwait's 10s/half-remaining. Replace with testwait.Until(t, what, cond).
- **BR-2** [Minor] `wallclock-window-under-load` Sibling fixed wall-clock windows remain beside the swept waitDead (ARCH-PURPOSE)
  guard_test.go:92 waitFile 5s; guard_test.go:131 guard-did-not-stop 5s; cloudflared_test.go:218 Close took >3s; live_test.go:65 15s teardown. Same rule as #480; waitFile fits testwait.Until directly.
- **BR-3** [Minor] `fixture-shell-portability` startOrphan relies on /bin/sh not exec-optimizing the final command
  Using `sleep 300 & wait` (as the guard test does) keeps the shell as parent on any shell; current form fails loudly rather than flakes if a shell execs, so low risk.
- **BR-4** [Minor] `done-when-evidence-substitution` Done-when 200 runs under load were run at ambient load only, with an uncommitted delay variant standing in
  Disclosed in the Log; the 500ms delayed-reap reproduction is not checked in, so it cannot be re-run.

## Round 2 — 2026-10-12T13:40:37-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — waitDead (guard_test.go:104-107), waitFile (guard_test.go:91-100) and startOrphan's settle loop (cloudflared_test.go:285-288) all now call testwait.Until; no hand-rolled t.Deadline polling remains in the window.
- BR-2 — not-addressed — waitFile is swept, but positive event waits in the same guard/cloudflared family still use fixed windows: guard_test.go:69 (10s), 122/141/162 (5s), cloudflared_test.go:239 (5s), live_test.go:65 (15s). Fix: use select on testwait.After(t). The Log's out-of-scope list (host/notes/pointer/server/session) does not cover these files. Minor, non-blocking.
- BR-3 — not-addressed — The fixture is still a bare `sleep 300` as the script's last command. bash and dash do not exec-optimize the last command of a script file, and the settle check would turn any such shell into a deterministic, named failure, so this stays low-risk Minor. `sleep 300 & wait` would close it.
- BR-4 — addressed — The Log now names the permanent guard: startOrphan's settled-state check fails deterministically on the causal regression (exec), so the substitute delay variant no longer carries the evidence. The 200/200 -race runs at ambient load up to 25 are disclosed.

## Open findings

- **BR-2** [Minor] `wallclock-window-under-load` Sibling fixed wall-clock windows remain beside the swept waitDead (ARCH-PURPOSE)
- **BR-3** [Minor] `fixture-shell-portability` startOrphan relies on /bin/sh not exec-optimizing the final command
