# Boundary Review — pair#735 (whole-issue close)

| field | value |
|-------|-------|
| issue | 735 — couch: messages to an idle Claude pane are held while a background shell runs; held messages don't say why |
| repo | pair |
| issue file | workshop/issues/000735-couch-messages-to-an-idle-claude-pane-are-held-while-a-background-shell-runs-held-messages-don-t-say-why.md |
| boundary | whole-issue close |
| milestone | — |
| window | 1c5ee7a2f6e1c8d4dda9f35e07ea0c81b6f4c0d2..08a5d49ba46ff7cf71310be3929c8207662b9405 |
| command | sdlc close --issue 735 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-20T12:36:42-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned diff satisfies the revised Spec and Plan. Duplicate resizes preserve valid composer evidence, actual resize failures remain conservative, and held receipts explain missing evidence without exposing screen text. No blocking findings.

```findings
{}
```

1. **Strengths**

   - `terminal_model.go:147` requires synchronized geometry before treating a resize as unchanged, preserving recovery after aborted resizes.
   - `peer_resize_test.go:21` exercises terminal painting and actual peer delivery for singular and plural shell footers.
   - `status_prefix_test.go:76` verifies that status retains the receiver’s reason for both the queue head and followers.
   - README and atlas describe the changed behavior and diagnostic detail.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Passed `go test ./cmd/internal/wrapcmd ./cmd/internal/couchmessage` (62.335s and 27.014s), plus `git diff --check`. Coverage includes hidden cursors, incomplete frames, real resizes, aborted retries, drafts, pickers, batch preparation, and status projection. Live operator qualification remains deferred until the next Couch restart under the documented revision.

6. **Architectural notes**

   - **ARCH-DRY — pass:** resize behavior stays in the existing serialized owner; receipt diagnostics use one helper.
   - **ARCH-PURE — pass:** diagnostic selection is deterministic; integration tests exercise the terminal and delivery boundaries through existing fixtures.
   - **ARCH-PURPOSE — pass:** the implementation addresses evidence loss without weakening composer admission; the logging split is explicitly documented.
   - **ARCH-EVOLVE — pass:** no persisted schema or lifecycle changes; diagnostics reuse existing receipt Detail.

7. **Plan revision recommendations:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"c0b7fe98-754b-4aa8-a01e-996651e7e789","repository":"github.com/xianxu/pair","issue_id":"000735","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/1b471ac2f0d18e16","reviewed_head":"08a5d49ba46ff7cf71310be3929c8207662b9405","slot":"pair:3","requested_at":"2026-10-20T19:35:00.958526Z"},"phase":"finished","pid":28911,"started_at":"2026-10-20T19:35:01.07315Z","finished_at":"2026-10-20T19:36:42.003043Z","session_id":"01a15acf-5bef-7710-a915-3e19500d4807","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.848323375s
head: "08a5d49ba46ff7cf71310be3929c8207662b9405"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
