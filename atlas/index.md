# pair atlas

`pair` is a small launcher that gives any TUI coding agent (Claude Code, Codex, Antigravity, Muse, Qoder, Grok) a Neovim-backed input field, decoupling the input scroll from the output scroll inside a zellij session.

## Map

- [VM runner](vm.md) — machine-wide train/ad-hoc positions, private tracked snapshots, bounded execution/cleanup and retained startup uncertainty (#743).
- [Project mock server](mock-server.md) — standalone loopback serving of versioned clickable mocks, shared fixtures, authoritative page identity and content-driven live reload (#755).
- [Mock-to-real development](mock-to-real.md) — any-framework development front, catalogue/report and production convention (#757).

- [Architecture](architecture.md) — what the pieces are and how they fit together.
- [Zellij reply admission](zellij-reply-safety.md) — strict native observations, bounded confirmation, lifecycle authority, the staged caller safety map (#647), and the standalone lookup-race reproduction (#775).
- [Session identity and storage](session-identity.md) — Pair's scoped address claims, deterministic native-session forests, durable resume targets, launch-specific probation and optional parsed projections, Couch's independent ThreadStore authority, Couch C/N/M allocation and live ownership proof, the standalone `📁` naming scheme, and legacy flat-data recovery.
- [Codex resume](codex-resume.md) — native indirect selection and exact-ID transcript-home routing.
- [Slot continuity target](../workshop/targets/slot-continuity.md) — the operator-approved commitment: Continue, Resume from disk or Start fresh, with one action per outcome and recovery available to the person or hub.
- [Slot recovery target](../workshop/targets/slot-recovery.md) — confirmed disk resume, always reachable Fresh and the crash-only recovery contract; [confirmed recovery](couch.md#confirmed-disk-recovery-556-m1) maps the person/hub execution path.
- [Go migration inventory](go-migration-inventory.md) — artifact/caller/runtime contract for the staged primary-Go-binary migration.
- [Workflow](workflow/index.md) — issue-based development loop inherited from the ariadne base layer.
- [How-to-bring-up-a-new-harness-cli](how-to-bring-up-a-new-harness-cli.md) — guide on integrating a new agent harness CLI.
- [couch](couch.md) — the session supervisor (`cmd/couch`): one leased namespace, Couch-owned composite durable threads coordinated with Pair-owned address claims, recoverable pre-exec starts, stale-thread/checkpoint recovery, explicit archive, tty routing, the after-restart recover plan with resume/reboot over the message socket (#367), confirmed remote agent switching, exact previous tty reads (#517), and listing-based terminal admission, observation-only startup and capture producer witnesses (#528), plus bounded failed-launch evidence with one-line recovery guidance (#531), typed raw native-query failure evidence and bounded false-miss recovery (#761), shared hosted title ownership on the existing activity schedule (#649), sampled per-slot CPU/RSS/disk writes, orphan attribution and hub alerts (#587), and capped FIFO hub-message batches with per-message receipts (#648), shared positive geometry for tall Codex composer confirmation and Return (#652), and bounded server-origin lifecycle notices with restart summaries (#574), the ephemeral attention list with its top-right panel, clear-all control, and status-row badge (#673, #733), and the lifecycle primitives: additive preserving records, the transcript binding resumes open by one stat, the slot lifecycle address with its per-thread lifetime lease, proven teardown, typed retire authorities and the left-running orphan registry (#695).
- [Workspace provisioning](workspace-provisioning.md) — repeatable numbered worktree setup, local Couch state, Weave readiness and durable recovery (#305–306); the slot resource table, observation and reconcile plan (#387); repository-family admission and subdirectory propagation are mapped in [couch](couch.md).
- [Workbench performance](workbench-performance.md) — normal-suite counted UI/message/attach invariants, scan ratchets, extension guide and separate live CPU acceptance.
- [Couch slot scratch](couch-scratch.md) — opt-in TestScratch temp environments, current-selection diagnostics, age cleanup, socket pinning and reboot-only Zellij metadata routing (#708, #720).
- [Supervision fact health](couch.md#supervision-fact-log-543) — request-time memory-only logger state, bounded backend errors, dropped/acknowledged counts and the coordinated CLI/supervisor upgrade (#736 M1).
- [Agent scratch volume](agent-scratch-volume.md) — Go build cache and agent/test temp roots on an APFS volume fseventsd doesn't journal (#496): setup, verify, lifecycle, teardown.
- [Review workbench](review-workbench.md) — embedded nvim document-review pane (#66): agent proposes edit records, nvim applies them undo-ably + journals rounds via docflow.
- [Agentic document caller contract](couch-agentic-documents.md) — durable registration, read-only discovery, rename and forget for submitted markdown files; shared with Parley.

## See also

- `.sdlc/landing.json` — pair’s 60-minute landing-train pilot (#709), read from main by `sdlc merge`; inspect the line and recent trains with `sdlc landing status`. The operator verifies changes through the local-gates label; the TL owns the first multi-car conformance check.

- [Go lint catalog](go-lint-catalog.md) — pinned upstream rules, repository coverage, reasoned exceptions, fixture ownership and terminal-aware producers, and shrink-only sleep debt.

- `.sdlc/affected.json`, `scripts/test-affected.py` — affected branch checks
  consume ariadne#473's main-authoritative selection through `make test-affected`.
  `make test-affected-inventory` reuses the reader audit before close and before
  affected execution; explicit per-file repair commands retain scanner coverage.
  Selected Go argv stays exact; fixed named source scanners run first for Go
  selections. Non-Go file-reader populations and unknown paths force full.
  `tests/affected-consumers.json` audits recipe inputs against the real planner;
  the Go-reader inventory requires re-audit for new or changed source readers.
  Its per-reader certificates live in `tests/affected-readers/<reader-path>.json`;
  `make affected-readers READER=<path> DISPOSITION=<audited-kind>` refreshes just
  that audited reader, preserving independent changes and unknown JSON fields.
  `--remove` retires certificates after reader removal; no shared index is generated.
  The environment guard runs once; groups share a private Pair executable.
  The operator owns the local-gates label; the hub owns first recorded
  affected/docs-train evidence after landing.

- `make test` — the full gate. It runs every shell/Lua suite, then `go test ./...`
  with test-result caching off, so it covers every package. Make adds `-trimpath`
  to existing `GOFLAGS` to share package compilation across worktrees; the recorded
  suite pins `PAIR_GO_TRIMPATH=-trimpath`. `internal/reporoot` finds test assets by
  cwd and validated module identity, and `probes/zellijprobe.RepoRoot` accepts
  `PAIR_PROBE_REPO` for probes launched outside a checkout. Inside a Couch shell it drops
  the session's environment-origin names first (pair#436); a raw `go test ./...`
  keeps them, so a test that reads one pins it. The environment probe
  (`tests/make-test-env-test.sh`) exercises both bare and command-line-assigned
  outer Make invocations; its synthetic fixture clears inherited Make controls
  so caller assignments cannot override the values under test (#577). Explicit
  and recursive Go-flag precedence remain covered inside that fixture. Its first step,
  `tests/host-preflight.sh`, refuses a host that cannot bind a unix socket, open
  a PTY or exec `ps` (an agent sandbox), because the suite then fails in ~150
  places that look like code bugs (pair#450). A refusal it does not probe still
  fails as an ordinary test; add the probe there.
- `cmd/internal/testpair` — real Pair test executables. `make test` owns one
  fresh plain binary through `tests/lib/with-pair-test-binary.sh` and exports
  `PAIR_TEST_BINARY`; the development binary has different linker defaults.
  `Binary` shares it, while `At` hard-links a private executable name/directory
  (symlinks across filesystems, preserving the shared executable image). Targeted Go runs build lazily once per package;
  their `TestMain` calls `testpair.Run` after helper dispatch to clean up only
  the owned fallback. Context variants preserve caller build deadlines. Shell
  consumers use `tests/lib/pair-test-binary.sh`, which owns the sole full-Pair
  test build; the source guard and lifetime contracts run in `testpair`.
  Subprocess environments use `Environment` and the shared
  `tests/lib/pair-test-env.sh` empty-environment boundary (PATH, disposable HOME,
  explicit fixture values). A source guard covers Pair/wrapper spawn callers;
  nested fixtures outside a checkout use `EnvironmentFrom` with the parent’s
  explicit helper path.
- `tests/without-ancestors.py` — reparented real-process fixtures retain the
  existing private-group cancellation and intentional-success-sidecar contract.
  `make test-without-ancestors` checks that lifetime. Its launcher temporarily
  normalizes SIGINT before exec and restores the caller's disposition: background
  shells can otherwise make pre-handler cancellation disappear through inherited
  SIG_IGN (#598). It also launches the helper with SIGINT and SIGTERM blocked
  (restoring its own mask right after spawning), and the helper unblocks them
  only after installing its cancel handlers. Before those handlers exist,
  interpreter startup can swallow SIGINT's KeyboardInterrupt (an importlib
  weakref callback, zipimport's argv[0] probe); blocked, the signal waits in the
  kernel instead (#674). A pipe-gated bootstrap pins both orderings by sending
  the signal while the child is held before the helper. Failures retain bounded
  captured output, test phase, helper PID, descendant group, and the caller's
  disposition and blocked mask, which tell the two classes apart.
- `cmd/internal/testdisk` — normal-run own-process physical write budgets for
  seven measured packages. Darwin+cgo uses the identity-fenced native process
  reader; unsupported platforms retain portable controls and report no native
  evidence. See [fixture writes and budgets](workbench-performance.md#fixture-writes-and-budgets).
- `cmd/internal/testgit` — scoped empty Git templates and completed fixture-tree
  file/directory/logical-byte budgets; constructor audits precede test mutations.
- `cmd/internal/testwait` — how a Go test waits for an event: bounded by the
  test's own deadline minus a margin (`After` for a select guard, `Until`/`Wait`
  to poll, `Budget` for a hand-rolled loop or a helper's timeout), never by a guess at how fast the event comes, so a loaded `-race`
  run stays green and a lost event still fails naming what it waited for
  (pair#479). Where a test's subject is a bound, assert the outcome that proves
  it (the product's timeout error, a child's ending signal), not elapsed time.
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

  **A third kind: the operator-driven instrument** (`probes/zellijcalls`, `#215`;
  also `probes/zellijbirthrace`, `#287`, which counts new sessions that die at
  birth, and since `#288` whether Pair's launcher then exits or hangs. It must
  not run unattended: its modes kill zellij servers and start real Pair
  sessions; and `probes/messageidle`, `#365`, a `zellij`+`ps` shim that records
  only couch-parented calls and measures an idle Couch window).
  It measures a workload only a human can start — a real couch thread — so it
  takes a verb (`arm` / `report` / `disarm` / `overhead`) and needs an
  interactive shell whose `PATH` it modifies. It cannot be auto-run, so neither
  branch above fits: `cmd/probes/` is for things a target CAN run, and this has
  nothing to run unattended. It stays under `probes/` because it is a probe, and
  it is safe under the wholesale loop **by construction, not by exemption** —
  the shim refuses to act unless `argv[0]` is `zellij`, so `go run ./probes/…`
  prints what it is and exits 0. That matters more than it sounds: a bare
  `zellij` STARTS A SESSION, so a shim that ran when smoke-tested would leak a
  server on every `make test-smoke`. Such a probe ships a `SKILL.md` instead of
  a target, since the runbook is the interface.

- `doctor/README.md` — `pair-doctor`: read the adaptation flight recorder to diagnose harness integration drift (see the bring-up guide §3 for the signal registry). Primary entry is the agent-agnostic `:PairDoctor` nvim command (`nvim/doctor.lua`); the procedure is single-sourced in `doctor/SKILL.md`, optionally registerable as a Claude skill.
- `doctor/terminal-pressure.md` — isolated Couch pressure experiments: full matrix
  and focused runs, evidence capture, metric interpretation and limits (#373).
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
  Its `hoprttcmd.sampleSpawns` loop isolates command preparation from execution;
  virtual-time tests assert exact command-only samples, while the shipped-binary
  smoke checks valid output without imposing a host-speed ceiling (#655).

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

- [Storage retention](storage-retention.md) — Pair/Couch clocks, protected users, collection, migration and bounded 24-hour resource-monitor snapshots.

- [History backup](history-backup.md) — Explicit encrypted bundles, verified local publication, private sharded catalog, bounded live SQLite snapshots and per-database freshness, pinned Drive month uploads, atomic dual-layout index discovery and recorded-location restore, retry guarantees and operator key recovery.
- [Managed retention I/O](storage-retention-io.md) — Checked entrypoints, protected artifacts, guard call chains, meaningful-use rules and behavioral evidence.

- [Terminal ownership](terminal.md) — Shared endpoint, presenter, profile, transport and qualification boundaries.
- [Couch broadcast](broadcast.md) — view-only streaming of the composed Couch screen: Presenter tap, LIVE-gated hub, switcher privacy (#395); the code lock on every cast link, the Host gate, the operator-only codes UI and the threat model (#495).
- [Couch broadcast-web](webcast.md) — the cast Chrome (#454): CDP over the debugging pipe, the `Browser` seam and its fake, active-tab inference, the isolated-world in-page control and operator annotation (#494), shared viewer widgets, document-bound Board requests and capture/input suppression; persistent pressure sketches (#752), held source scenes behind an opaque origin, acknowledged recovery and indexed rendering; the recorder (#458) on the CDP side of the seam; the asset store and proxy (#459); the fidelity check (#478): each recording's frames vs the viewer's replay, a report per recording, `couch --broadcast-web-fidelity` and at the end of a cast.
- [Send cast notes to the agent](webcast-submission.md) — the draft chord, recorded context and current PNGs, portable evidence bundles, and delivery/acknowledgment recovery (#483).
- [Broadcast-web recording format](webcast-format.md) — the recording (#457): one record stream live and on disk, stable node ids, the reference reducer and validator, the seams, fakes and fixtures each wave-1 component builds against, and the notes model (#461): anchors (with their `kind`/`path`), focused context, and crops that are the note's box plus a margin (#488). Agent queries (#462): `couch --broadcast-web-query`; replay with a scrubber (#471): `couch --broadcast-web-replay`, shared through a running Couch's tunnel for remote viewers (#477), in [webcast.md](webcast.md#replay-471). Remote driving (#466): `ctl` control records, `by` attribution, and `drive`'s input wire, confinement, `Injector` and `Controller` seams.

- [Opt-in Couch live capture](couch-live-capture.md) — opt-in display-boundary recording and replay evidence for #379.
- [Disposable native Couch fleets](couch-testing.md) — shared testcouch root ownership, real provisioning/actors, fixture personalities and exact teardown (#744).
