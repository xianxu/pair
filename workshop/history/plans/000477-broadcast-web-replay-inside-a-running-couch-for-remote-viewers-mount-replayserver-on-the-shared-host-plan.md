# #477 plan: replay inside a running Couch, for remote viewers

## Goal

A replay link (`<base>/<token>/r/<id>/#t=MS`) produced on this machine opens on
another device through the shared broadcast Host's tunnel (#454), plays the
recording, seeks, and (after #470 lands) draws its notes and strokes.

## Architecture

**A replay share** is one recording mounted on the shared `broadcast.Host` under
its own random token: a `webcast.ReplayServer` scoped to that one recording
(no listing, every other id not found). One token per recording follows the cast token
model: a link grants exactly what it names. A colleague holding a link to one
recording can't list or open the others.

**`webcast.ReplayShares`** owns a running Couch's shares, keyed by recording id:

- `Share(id) ShareStatus` is idempotent: the first call starts the mount in the
  background and answers `opening`; later calls answer the share's current
  state (`opening`, `ready` with its base, or `failed` with the reason). A
  share whose mount was lost (the tunnel died) or failed is started again by
  the next `Share`, with a new token. This is a reconcile: the request states
  the desired state, and the poll reads the current one.
- `Close()` ends them all and returns once nothing is left mounted.

A share's states, under the shares' mutex:

    opening --mounted-->        ready (base = mount.Base()+"/"+token)
    opening --Mount failed-->   failed (reason)
    ready   --mount.Exited-->   lost (mount.Err())
    failed | lost --Share-->    opening (a new token)
    any     --Close-->          closed (Share after Close answers closed)

Each share's mount runs in one goroutine under a context the shares own.
`Close` marks the shares closed, cancels that context (`Host.Mount` abandons on
a cancelled context and closes a tunnel nobody wants, host.go:85), unmounts
every ready share, and waits for the goroutines. A mount that completes after
`Close` sees `closed` and unmounts at once. Two `Share` calls for one id while
it's opening find the opening share and start nothing, so there's one mount and one token.
The goroutine that watches `mount.Exited()` for a lost share ends with the
shares' context too.

**Why async:** the broker socket gives each exchange 2s (`TransportTimeout`),
but opening a tunnel and probing the link takes up to `ProbeTimeout` (30s).
So the CLI polls, the way slot operations poll `operation-status`.

**Broker op `replay-share`** (`Request.Target` = recording id): no conversation
identity, like `broadcast-status` and `tail`. The broker socket lives in the
operator-only store, the same trust as reading the recordings on disk.
The answer is `Response.Replay *ReplayShare{State, Base, Error}`. Unlike
`broadcast-status`, it carries the token. A replay link is what the operator
asked for, and the live broadcast's link is still never listed.

**Wiring:** `consoleRunnerFor` makes the shares beside `castHost` (rooted at
`recordingsDir(store)`, only when there's a store). The console holds them:
`teardown` closes them beside the web cast, and `Console.ReplayShare(id)` is
installed into the message service like `BroadcastStatus`.

**CLIs:**

- `couch --broadcast-web-replay ID|DIR [--at MS]`: when a Couch answers the
  broker, it asks for the share, polls every 300ms until it's ready, and
  prints the public link and exits. The poll gives up at 45s, past the Host's 30s probe:
  it says the link is still opening in Couch and that a re-run will print it
  (the share keeps opening, and the re-run is idempotent), and exits 1. A DIR must be a recording in Couch's store; otherwise
  it refuses and names `--local`. When no Couch answers, it serves on loopback as
  today, and says so on stderr. A Couch that predates the op gets the same
  fallback, with a different message.
  `--local` forces loopback. With no target it still serves the local listing on
  loopback: the listing isn't shared.
- `couch --broadcast-web-query DIR link [--at MS]` prints the same public link for an
  agent (the moment to point a reviewer at). `query` stays Couch-free: `Main`
  takes a `Linker func(dir string, at int64) (string, error)` that couchcmd
  supplies, and the same share-and-poll function backs both CLIs. With no Couch
  answering, it refuses (exit 1: "no running Couch can share it; `couch
  --broadcast-web-replay DIR` serves it on this machine"). An agent's command
  must return, so it never serves on loopback itself.

**#471's advisory:** `headAndEnd` leaves `end` at 0 when the last line's
envelope doesn't parse (a torn write), and the listing says `0s`. `Listed`
gains `Ended bool`; the listing prints `unknown`.

**Drawing (M2, after #470 lands):** the replay viewer draws notes and strokes
with #470's modules. It waits on #470 (pair:6). Merge main, then wire it.

## Lifecycle (ARCH-FUNERAL)

- A share is created by a `replay-share` request. It's removed by Couch's
  teardown, or replaced after its mount is lost. It's in memory, bounded by the
  number of recordings asked for in this Couch run (one mount and one token each),
  and its token dies with Couch. Links are valid for this Couch run. That's the
  same lifetime a cast link has.
- The Host's tunnel stays open while any share is mounted, that is, until Couch
  exits. Each request is a deliberate operator or agent act, and each one prints
  the link. A share expiry, or an "unshare" verb, is a follow-up if the operator
  wants one. This is flagged in the report to the TL.
- A loopback replay (fallback or `--local`) is the same as today: it ends at Ctrl-C.

## Tests

- `ReplayShares.Share` / `Close`, the state machine: sequence tests on a
  `FakeTunnel` Host with the ordering controlled (`OpenDelay`, `IgnoreCancel`,
  `Exit`). Cover each transition above, two Shares while opening (one mount,
  one token), and Close while opening (`Opens == Closes`, Share answers closed,
  and the goroutines have joined). The ready link serves the viewer and the records.
- The scoped `ReplayServer`: a table of paths (another id, the listing,
  traversal, an encoded slash) all 404, and its own recording serves.
- `ValidateRequest` for `replay-share`: only Target, a valid recording id.
- The CLI share-and-poll: a fake `messageCall` scripted with opening, then
  ready (the link carries `#t=`), the deadline, no Couch (loopback fallback for
  replay, a refusal for `query link`), an old Couch, and a DIR outside the store.
- `headAndEnd` / `ListRecordings`: a torn or garbage last line gives
  `Ended=false`, and the listing says unknown.
- Live: a real Chrome opens a share's link through a LocalOnly-tunnel Host and
  seeks with `#t=`. The tunnel smoke test is the operator's, scheduled by the TL.

## Milestones

- M1: replay shares on the Host, the broker op, both CLIs, the advisory, atlas
  and skill text.
- M2: merge main once #470 lands; the replay viewer draws notes and strokes with
  #470's modules; a live check that a note shows at its moment.

## Revisions

### 2026-10-12, during M1

- **`Share` / `Status` split.** Reason: with one idempotent `Share`, the poll restarts a failed share on every call,
  so the CLI could never see the failure. The test caught it: a Host with no tunnel stayed `opening`. Change: `Share`
  starts a share unless it's opening or ready, and `Status` only reads. That's two broker ops, `replay-share` and
  `replay-status`, mirroring slot operations and `operation-status`.
- **A DIR outside Couch's store serves on loopback, with a note, instead of refusing.** Reason: refusing gave the
  operator nothing they wouldn't get with `--local`. The query CLI's `link` still refuses, because it can't serve.
- **`ShareStatus.Local`.** When the Host's tunnel is off (`COUCH_BROADCAST_TUNNEL=off`), the link works on Couch's
  machine only. Both CLIs say so, rather than print a loopback link as if other devices could open it.
- **Torn last line.** The `{"t":N` match now needs the digits to end at `,` or `}`, so a line cut off inside its `t`
  doesn't read as an end.
