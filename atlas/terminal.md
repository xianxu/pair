# Terminal ownership

The shared terminal implementation for #255 lives in `cmd/internal/terminal`.
Couch and Pair use the same endpoint/presenter adapters. Live symptom acceptance
is held for the operator after M4.

An `Endpoint` owns one child's emulator and negotiated modes. Output updates that
state before a consumer can see the batch. Queries are answered from that endpoint
and enter the same bounded FIFO as encoded operator input. Immutable `Publication` values combine `Frame` and bounded typed history. Frame
publications carry endpoint identity, generation and acknowledged geometry epoch.
Synchronized output retains the last complete frame, with a 150ms recovery bound.
Clipboard writes, notifications, title/directory changes and bells are ordered,
origin-bearing effects; snapshots cannot replay them. Clipboard reads are explicitly
unsupported and receive an empty response.

`Frame`, `View`, `Compose` and `Render` express screen composition and admission
without IO. `Presenter` owns the physical parent writer and commits input admission
only after presentation completes. A failed partial write invalidates that
connection's presentation state. Product code retains shortcut and notification
policy, while terminal negotiation and encoding belong to the shared connection.
Parent mouse capture differs by boundary: Couch needs motion for its chrome;
Pair's terminal pane must leave native Zellij selection available when its child
requests no mouse reporting. Child requests still determine forwarded events.

`ttyio.File` acquires input/output descriptor flags before pumps start. Its read and
write adapters share that lifetime, use nonblocking syscalls and bounded polling,
and restore flags after operations join. Duplicating a descriptor would not isolate
its nonblocking flag. `InputWriter` preserves packet order and accepted-prefix
failures, with 128 packets / 1MiB including the packet currently being written.

The child contract is `pair-vt-256color`, derived from the explicit capability table
in `terminal/profile_query.go`. `terminfo/` contains its source and audit.
Capabilities are not inferred from the outer terminal. The checked-in local module
fork in `third_party/vt` preserves upstream provenance and records its narrow
repairs and bounds in `PAIR_PATCHES.md`. Root Go tests do not traverse that nested
module; its tests must also run from `third_party/vt`.

Verification has three independent layers: literal terminal/frame fixtures,
`terminalqualify` backend and integration cases, and actual renderer wire interpreted
by pinned `@xterm/headless` under `tests/terminal-oracle`. Production composition and
sustained native terminal acceptance add evidence beyond those library checks.

Production adapters live in `ptychild/terminal.go`, `couchtty/terminal.go` and
`termcmd/presentation.go`. Each PTY receives an endpoint before its reader starts.
Output delivery has bounded backpressure; a blocked UI callback does not block
query/input serialization. EOF ends input immediately, preserves the final
publication, and only announces drained child exit after callbacks complete.
Consumer failure cancels the read pump and reaps a silent child. Disposal cancels
and joins both output delivery and input workers. Snapshot ingestion precedes
output enqueue; `FlushOutput` drains queued batches, so screen readiness is
proved through the corresponding parent presentation. Couch retains ownership after a
pane leaves its map: ordinary exit drains, deselects/retires, then disposes; the
last visible final frame remains owned through parent release. Rejected startup
and attachment paths dispose their unaccepted client, and teardown disposes all
remaining accepted children after release. Final Console exit classification waits
for Presenter release/join: shutdown-only cancellation is expected, while
previously latched live failures, mixed host errors and cleanup failures remain
errors regardless of which stop/failure notification was selected first.

Normal history carries monotonic row IDs, clear epochs, blank provenance and soft
wrap metadata. `RenderWithHistory` serializes owned cells in bounded chunks; the
presenter commits its installed cursor only after all writes succeed. Continuous
appends retain already delivered parent history even when endpoint retention
evicts its prefix; missing coverage, clear epochs, owner or geometry changes
rebuild the bounded retained suffix. Older physical scrollback follows the parent
terminal's own policy. Erased backgrounds remain paint, without becoming printed
spaces in native copied text. Child alternate-screen transitions become presenter-owned parent transitions; panels
retain the current physical buffer. Release leaves only an alternate buffer the
presenter actually entered, then restores parent controls.

The runtime bundle includes `tic` output compiled during generation through
`runtimebundlegen.TerminfoCompiler`. A filesystem-producing fake tests staging,
normalization and failure preservation; native compiler conformance is separate. Both launch
roots install the versioned profile and pass `TERM=pair-vt-256color` plus its
`TERMINFO` directory. Runtime launch does not execute `tic`. `pair wrap` preserves
Codex synchronized output, focus and keyboard negotiation; notification
normalization and Return adaptation remain product behavior. Its terminal
observer consumes the normalized queued visual stream, while raw capture remains
separate; observation does not claim physical-write acknowledgment.

## No destination is an answer, not a failure (#265)

`ErrNoDestination` (`destination.go`) is the presenter's typed answer when it
holds no endpoint to deliver an input event to. `Presenter.Input`,
`Presenter.mouseInput` and `Presenter.UpdateChrome` all wrap it through the one
constructor `noDestination`, which carries the `View` so a caller that surfaces
it can say which endpoint in which state.

It is deliberately separate from physical failure: `Presenter.fail` latches the
view into `Failed` and closes `Failed()`, and nothing about this sentinel touches
that channel. The distinction exists because a consumer that cannot tell the two
apart tears down terminal ownership over a question it merely asked at the wrong
moment -- which is what exited couch on a keystroke before #265. Consumers
classify with `errors.Is`.

## Notification output ownership

Automatic wrapper attention and explicit `pair notify` hooks share one output
owner. `notifytransport` uses the existing exact wrapper PID binding for a private
bounded Unix datagram broker. The wrapper frames insertion into its normal pane
stream; no producer writes Zellij's outer TTY. Pinned Zellij converts OSC777 to
OSC9, and Endpoint's registered adapter restores the Pair title and full4096-byte
message before Presenter delivery. Generic backend metadata limits stay unchanged.
Native qualification covers delayed Unicode chunks, maximum message size,
focused/hidden attention and persistent-client reattachment.

The broker namespace owns both socket addresses and its persistent lock.
`PAIR_NOTIFY_SOCKET_DIR` selects an absolute private root; conformance tests and
the smoke launcher always provide one. Each broker captures its admitted root
for teardown. Normal close removes owned inodes; bounded admission sweeps reclaim
provably dead same-UID socket owners independently of PID-binding survival,
covering crash followed by launcher or artifact-GC cleanup. Live, unknown and
foreign entries survive; the1024-entry namespace capacity refuses admission
explicitly rather than permitting unbounded crash residue.

## Native CI qualification

`make -f Makefile.local test-native-terminal-ci` builds a fresh temporary Pair
candidate and runs native Couch/Zellij direct and wrapped paths, nvim, scrolling
and broker-PTY checks under the race detector. Its test-event gate rejects missing
or skipped required tests. `.github/workflows/couch-zellij-conformance.yml` runs
this alongside existing lifecycle conformance, installing the pinned native and
independent-oracle dependencies. Terminal, backend, consumer, runtime and oracle
source paths trigger it. Temporary state is removed after the invocation.
