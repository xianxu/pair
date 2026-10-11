# couch --session-sync Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `couch --session-sync <diff|write|release|status> FILE` is the one guarded mutation path for a `type: session-sync` file, and Couch keeps a durable ledger of deliveries to each slot, so `status` can list the events the owner still owes the file.

**Architecture:** A new pure-leaning package `cmd/internal/sessionsync` owns the file rules (lock, shadow, sidecar, sections, restamp), the delivery ledger (append/read/trim of one JSONL per slot under the Couch store) and the owed-events computation. `couchcmd` adds a CLI verb that runs in-process (no daemon RPC; it reads the store directly), and the daemon's `messageEndpoint.Deliver` appends a ledger row after each landed delivery. Nothing in `couchmessage` changes, which keeps clear of pair#433/#435.

**Tech Stack:** Go stdlib; `diff -u` (POSIX) for the human-readable diff.

---

## Decisions (settled here, per the Spec's "settle in the plan")

1. **The ledger lives in Couch's store; the per-file fold state lives beside the file.**
   - The broker knows the target *slot* at delivery, not the file. So deliveries are recorded per slot:
     `<store>/session-sync/<repo>_<N>.jsonl` (the `:` of the slot replaced by `_`).
   - What a given file has folded, and when its ledger horizon starts, belong to that file:
     `.<base>.sync` (JSON) beside it, next to the existing `.<base>.shadow` and `<base>.lock`. It dies with the file,
     like the shadow. ops must gitignore it (`teams/*/.tl-status-*.sync`); the TL owns that edit, so I ask rather
     than touch it.
2. **A submit is a delivery whose body's first line is `submitted: <abs path of FILE>`.**
   - parley.nvim#313 sends exactly that through `couch --send-to <owner>`.
   - Recognizing it by body means no wire change. Submits naming *another* file are not owed by this file.
     Paths compare after `filepath.EvalSymlinks`, so the macOS `/tmp` vs `/private/tmp` split can't hide a submit.
3. **What is owed:** ledger rows for the file's `owner:` slot with `At >= Since`, not in `Folded`, minus submits for
   other files.
   - `Since` is the ledger horizon: it comes from the sidecar when one exists. Otherwise it is the shadow's mtime,
     which carries over from `tl-status.py`, the last agent write. With neither file it is the zero time.
   - Deliberately *not* "since the last write": an implicit clear on every write is exactly the liveness hole the
     issue exists to close. Every delivery must be folded by ID.
4. **Clearing:** `write --fold ID[,ID]` and `release [--fold ID[,ID]]` fold the given IDs. `release` also folds every
   owed *submit*, because handing the turn back answers the submit. Folding an ID that is not owed refuses, which
   catches typos.
5. **Landed means `Submitted` or `Indeterminate`.** An indeterminate paste may have landed. A spurious owed row costs
   one fold; a missed one is the bug. The row records `Uncertain: true` so the text view can say so.
6. **Lock semantics (parley#313):**
   - Absent lock = free; `holder: agent` = the agent's turn; any other holder (incl. `operator`, unknown) blocks.
   - The agent never writes the lock; `release` deletes it only when the holder is `agent`.
7. **Exit codes:** 0 ok, 2 usage, 3 operator holds the lock, 4 unread operator edits, 1 any other refusal.
   These match `tl-status.py`, so the xx-tl skill's handling carries over unchanged.
8. **Writes are in place, not temp+rename,** to match the prototype that passed parley#313's live check. A rename
   swaps the inode under parley's file watch. The file is one screen, so a torn write is not a realistic risk.
   The shadow and sidecar are agent-private, so they use temp+rename.
9. **`status --json` shape** (agreed with parley.nvim#314 via ops:0, 2026-10-10):
   ```json
   {"Code":"ok","Error":"","Status":{"File":"/abs","Owner":"ops:0","Holder":"free|operator|agent",
    "LastWrite":"RFC3339|null","Owed":[{"ID":"…","Kind":"submit|delivery","From":"slot","At":"RFC3339"}]}}
   ```
   - `Owed` is always an array, sorted oldest first.
   - A refusal prints `{"Code":"<code>","Error":"…","Status":null}` and exits non-zero.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `Paths` | `cmd/internal/sessionsync/file.go` | new |
| `ParseOwner` | `cmd/internal/sessionsync/file.go` | new |
| `ParseHolder` | `cmd/internal/sessionsync/file.go` | new |
| `ReplaceSection` | `cmd/internal/sessionsync/file.go` | new |
| `Restamp` | `cmd/internal/sessionsync/file.go` | new |
| `Entry` | `cmd/internal/sessionsync/ledger.go` | new |
| `State` | `cmd/internal/sessionsync/state.go` | new |
| `Owed` / `OwedEvent` | `cmd/internal/sessionsync/owed.go` | new |
| `Refusal` | `cmd/internal/sessionsync/sync.go` | new |

- **Paths**: lock, shadow and sidecar paths derived from the file path. It is the one place the sidecar naming lives.
- **ReplaceSection**: matches whole heading lines only (`line == "## "+heading` after trimming trailing space). It
  refuses unless exactly one match exists, and the section ends at the next `## ` line. Pins the Done-when "a heading
  quoted in prose never matches".
- **Owed**: `Owed(entries, state, owner, absFile) []OwedEvent`. Pure decision 3, colocated tests without IO.
  - **DRY:** `status`, the `--fold` validation and `release`'s submit-folding all call it.
- **Refusal**: a typed error with an exit code and a JSON `Code` (`operator-lock`, `unread-edits`, `refused`), so the
  CLI never re-derives codes from strings.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `Ledger` | `cmd/internal/sessionsync/ledger.go` | new | per-slot JSONL files in the Couch store |
| `Sync` (Diff/Write/Release/Status) | `cmd/internal/sessionsync/sync.go` | new | the session-sync file, its lock, shadow, sidecar; `time.Now` injected |
| `runSessionSyncCLI` | `cmd/internal/couchcmd/session_sync.go` | new | argv → `Sync`, output, exit code |
| `messageEndpoint.Deliver` | `cmd/internal/couchcmd/message_service.go` | modified | appends a ledger row after a landed delivery |
| couch skill text | `cmd/internal/couchcmd/skills/…` | modified | documents the verb for agents |

- **Ledger**: `Append(slot, Entry)` is mutex-guarded (deliveries to different slots run concurrently in the daemon),
  `O_APPEND` one line per row. `Read(slot)` skips a torn trailing line.
  - **Bound (ARCH-FUNERAL):** on append, when the file passes 1024 lines, rewrite it (temp+rename) keeping rows
    younger than 14 days, at most 512 of them. So one slot's file holds at most 1024 rows (~200 KB), and the number
    of files is bounded by slot addresses (repo:N). Folded IDs in a sidecar are pruned to IDs still in the ledger on
    every write, so the sidecar is bounded by the ledger.
  - The body is not stored, only the first line truncated to 256 bytes, which submit recognition needs. So the
    store holds no message content beyond a header line.
  - **Injected into:** `Sync` (reads), the message service (appends). Tests use a temp dir; no fake is needed for a
    plain filesystem.
- **Sync**: `Sync{Ledger, Now func() time.Time, Diff func(a, b string) (string, error)}`. The real `Diff` execs
  `diff -u --label shadow --label file` (exit 1 = differences). A test passes a trivial differ, and one test runs the
  real one.
- **messageEndpoint.Deliver**: after `e.endpoint.Deliver` returns a `Submitted` or `Indeterminate` receipt, call
  `e.service.recordDelivery(m, receipt)`. A ledger append failure is logged and never fails the delivery
  (ARCH: messaging must not degrade because bookkeeping did). The ledger is installed on the service by
  `startMessageService` from `c.Namespace.Dir()`. A nil ledger records nothing, which keeps existing tests unchanged.

## Tasks

### Task 1: File rules (`file.go`)
- [ ] Failing tests in `cmd/internal/sessionsync/file_test.go`:
  - `ReplaceSection` with `## Needs you` present once, plus a prose line `` see `## Needs you` `` and a fenced line
    `  ## Needs you`: only the real heading's body is replaced.
  - Zero or two heading matches refuse.
  - The section ends at the next `## `; the last section runs to EOF.
  - `Restamp` replaces only the first `Updated YYYY-MM-DD HH:MM`.
  - `ParseOwner` reads `owner:` from frontmatter only (an `owner:` in the body is ignored), and returns `""` when
    absent.
  - `ParseHolder` maps absent→free, `agent`→agent, anything else→operator.
- [ ] Implement; `go test ./cmd/internal/sessionsync/` green.

### Task 2: Ledger (`ledger.go`)
- [ ] Failing tests:
  - Append then Read round-trips.
  - Two slots stay separate, and `ops:0` maps to `ops_0.jsonl`.
  - A torn last line is skipped.
  - The trim at >1024 rows keeps the newest 512 under 14 days.
  - Concurrent appends from 8 goroutines lose no row (`-race`).
  - A slot address with a `/` or `..` refuses, so the slot can't escape the directory.
- [ ] Implement; green.

### Task 3: Owed computation (`owed.go`, `state.go`)
- [ ] Failing tests:
  - Rows before `Since` are excluded, and folded IDs are excluded.
  - A submit for this file is `Kind: submit`; one for another file is excluded.
  - A symlinked path still matches.
  - Rows for other slots are excluded.
  - The result is sorted by `At`, and is an empty non-nil slice when nothing is owed.
  - `State` load: missing sidecar → `Since` = shadow mtime or zero. Save is temp+rename.
- [ ] Implement; green.

### Task 4: `Sync` operations (`sync.go`)
- [ ] Failing tests, using a temp dir, a fixed clock and the ledger. These pin the Done-when rows:
  - **Operator lock:** `Write` under an operator lock returns a `Refusal` with exit 3, and the file, shadow and
    sidecar bytes are unchanged.
  - **Unread edits:** with file != shadow, `Write` refuses with exit 4. With `AcceptEdits` it writes, and the shadow
    then equals the file.
  - **Owed until folded:** a ledger row for the owner appears in `Status().Owed`, a `Write` without `Fold` leaves it
    owed, and `Write{Fold: [id]}` clears it.
  - **Fold validation:** folding an unknown ID refuses and writes nothing.
  - **Release:** deletes an agent-held lock and folds owed submits but not deliveries. It refuses (exit 3) under an
    operator lock and leaves no lock behind otherwise.
  - **Whole-file write:** `Write{All}` replaces the whole file, keeps the guards, and restamps.
  - **Empty body:** an empty body refuses.
  - **Diff:** returns the holder plus the diff text, using the real `diff -u` once.
  - **Status:** `LastWrite` is null before the first write.
- [ ] Implement; green.

### Task 5: CLI (`couchcmd/session_sync.go`, `cli.go`, `run.go`)
- [ ] Failing tests in `couchcmd/session_sync_test.go`.
  - `ParseCLI` forms:
    ```
    --session-sync diff FILE
    --session-sync status [--json] FILE
    --session-sync write --section HEADING BODYFILE FILE [--accept-edits] [--fold IDS]
    --session-sync write --all BODYFILE FILE [--accept-edits] [--fold IDS]
    --session-sync release FILE [--fold IDS]
    ```
    Parsed before the layout scan. A HEADING that starts with `--` is still a value.
  - Bad shapes are usage errors (exit 2).
  - Running against a `testRT` store dir: the JSON envelope matches decision 9 byte-for-byte on keys; a refusal under
    `--json` prints `Code` != ok and exits non-zero; the text `status` lists owed rows.
- [ ] Implement:
  - a `cliSessionSync` kind dispatched in `run.go` to `runSessionSyncCLI`;
  - the ledger dir from `rt.StoreDir()`, as `--broadcast-list` does;
  - no supervisor lease, because this verb never constructs a supervisor.

### Task 6: Daemon records deliveries (`message_service.go`)
- [ ] Failing test in `couchcmd/message_service_test.go`, extending `serviceRig`:
  - Configure a ledger dir, send a message to a connected fake endpoint that answers `Submitted`, and assert one
    ledger row for the target slot (ID, From slot, first body line).
  - A `Cancelled` delivery records nothing.
  - Then `Sync.Status` on a session-sync file with `owner:` = that slot lists the ID as owed. This is the
    end-to-end Done-when row.
- [ ] Implement `recordDelivery` and install the ledger in `startMessageService`.

### Task 7: Docs and handoff
- [ ] Couch skill text (`couchcmd/skills/`): the verbs, the exit codes, the fold discipline.
- [ ] `atlas/couch.md`: a "Session-sync (#437)" section covering the ledger location and bound, the sidecar,
  submit recognition and the JSON shape. Plus an `atlas/index.md` link if the section becomes its own file.
- [ ] Message ops:0:
  - the verb is ready for xx-tl;
  - the gitignore line `teams/*/.tl-status-*.sync`;
  - `tl-status.py` → `couch --session-sync` mapping: `section F H B` → `write --section H B F`;
    `replace F B` → `write --all B F`.

  The skill switch itself is ops' edit; the Done-when row closes on the TL's confirmation.

## Verification
- Targeted while iterating: `go test -race ./cmd/internal/sessionsync/ ./cmd/internal/couchcmd/ -run 'SessionSync|Ledger|Owed|Deliver'`.
- Before close: full `make -k test` + `go test ./...` (per memory: scratchpad TMPDIR, log to file).
- Live: rebuild, run `couch --session-sync status --json` on ops' `tl-status-xian-xu.md` (read-only). Then send a
  message to a scratch slot that owns a scratch session-sync file and watch it appear as owed, then fold it.
