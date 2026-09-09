# pair atlas

`pair` is a small launcher that gives any TUI coding agent (Claude Code, Codex, Antigravity, Muse) a Neovim-backed input field, decoupling the input scroll from the output scroll inside a zellij session.

## Map

- [Architecture](architecture.md) — what the pieces are and how they fit together.
- [Session identity and storage](session-identity.md) — Pair's scoped address claims, deterministic native-session forests, established-only consumer projections, exact artifact bindings, Couch's independent ThreadStore authority, the `📁` public zellij name scheme, and legacy flat-data recovery.
- [Go migration inventory](go-migration-inventory.md) — artifact/caller/runtime contract for the staged primary-Go-binary migration.
- [Workflow](workflow/index.md) — issue-based development loop inherited from the ariadne base layer.
- [How-to-bring-up-a-new-harness-cli](how-to-bring-up-a-new-harness-cli.md) — guide on integrating a new agent harness CLI.
- [couch](couch.md) — the session supervisor (`cmd/couch`): one leased namespace, Couch-owned composite durable threads coordinated with Pair-owned address claims, recoverable pre-exec starts, and tty routing.
- [Review workbench](review-workbench.md) — embedded nvim document-review pane (#66): agent proposes edit records, nvim applies them undo-ably + journals rounds via docflow.

## See also

- `probes/` — committed probe drivers that exercise real binaries end to end.
  `make test-smoke` runs **every** directory under `probes/`, so a new probe is
  covered by existing it rather than by remembering to add a line. A probe earns
  a place here when its output is quoted as close evidence; each explains in its
  own header what question it answers, which is where to look rather than in a
  list here that would drift.

  **Shared harness:** `probes/zellijprobe` is the one implementation of "start a
  zellij session under a pty, read what it renders, tear it down". It exists
  because the two copies diverged in a way that mattered: one discovered its
  session by DIFFING `zellij list-sessions` and force-deleted an arbitrary new
  name, from `make test-smoke` — so any session that appeared in that window,
  including an operator's workbench, was a candidate. `Start` names the session
  and `Close` deletes that name, which makes "a probe can only destroy a session
  it created" structural rather than remembered.

  **There are exceptions, and they are the whole reason to state the rule.**
  `cmd/probes/` is a second probe home, reachable only through a per-probe
  target, and a probe belongs there for one of two reasons. It needs an
  ARGUMENT the wholesale loop cannot supply — `couchstartrecovery` takes
  `bin/pair-launch-helper`; `termctrlc` and `termrows` take `bin/pair`. Or it
  must import a `cmd/internal/…` package, which Go forbids from `probes/`
  outright: `couchnestedrows` reserves a row with the real
  `hostty.Reservation`, and a probe that reimplemented the escape it is
  measuring would be measuring itself.

  So "probes live in `probes/`" is true of every probe that runs with no
  arguments and needs nothing internal; anything else goes under `cmd/probes/`
  **with its own target in the same commit**, since nothing will run it
  otherwise. Recorded because the rule without its exceptions is how `#199`'s
  first probe landed in the wrong home and was then hand-added to two lists to
  compensate — the exact remembering `test-smoke` exists to abolish.

- `doctor/README.md` — `pair-doctor`: read the adaptation flight recorder to diagnose harness integration drift (see the bring-up guide §3 for the signal registry). Primary entry is the agent-agnostic `:PairDoctor` nvim command (`nvim/doctor.lua`); the procedure is single-sourced in `doctor/SKILL.md`, optionally registerable as a Claude skill.
- `doctor/perf.sh` — the performance half of the same entry (`#208`): a snapshot
  of load, per-process resource **rates**, the render path (WindowServer), and
  three latency probes, taken at the moment slowness is felt. Two rules give it
  its shape: it emits **raw two-sample output and does no arithmetic** (the pid
  join is error-prone, so it lives in tested Lua, not shell), and it never uses
  `ps %cpu`, which is a lifetime average that hid a 42.6% spinner during the
  investigation that motivated the issue. `pair hoprtt` is its probe —
  pipe round-trip and fork+exec timing, in-process because a hop is ~7 µs and
  timing that from shell measures the clock process instead. It is a SUBCOMMAND
  rather than a binary because `pair` is the only thing that always ships: the
  Homebrew formula builds just `./cmd/pair-go`, and `PAIR_HOME` at runtime is
  the extracted bundle root, which carries no helper binaries.

  **The capture writes a file and sends a pointer.** The complete report —
  compact half, joined per-process rates, and both raw `ps` samples — lands in
  `$PAIR_DATA_DIR/perf-capture-<epoch>.txt` — named per capture, so re-running
  never overwrites the file an earlier prompt points at; the prompt carries a ~12-line
  headline plus that path, and the path is placed in the **first ~60 bytes,
  ahead of the operator's note**. That placement is load-bearing, not
  cosmetic: the draft-editor→agent send path drops a contiguous chunk from the
  MIDDLE of a payload (`#211`, measured — 1,025 bytes gone from a 2,447-byte
  send with head and tail intact), so the head is the only region that can be
  relied on. The invariant to preserve is that **nothing of value exists only
  in the prompt**; an earlier shape kept the rates there alone, where a
  truncated send destroyed them with no copy on disk. Each capture also appends
  a row to `perf-captures.jsonl`, which is what makes the next reading
  comparative rather than absolute.
- `README.md` (repo root) — install and usage.
- Design pensive (sibling repo): `~/workspace/brain/docs/vision/2026-05-02-01-pensive-nvim-as-input-field-for-tui-coding-agents.md`
