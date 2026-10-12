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
- `Close()` unmounts every share. The last unmount closes the Host's tunnel
  (the Host already does that).

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
  broker, it asks for the share, polls until it's ready (bounded by 45s), prints the
  public link and exits. A DIR must be a recording in Couch's store; otherwise
  it refuses and names `--local`. When no Couch answers, it serves on loopback as
  today, and says so on stderr. A Couch that predates the op gets the same
  fallback, with a different message.
  `--local` forces loopback. With no target it still serves the local listing on
  loopback: the listing isn't shared.
- `couch --broadcast-web-query DIR link [--at MS]` prints the same public link for an
  agent (the moment to point a reviewer at). `query` stays Couch-free: `Main`
  takes a `Linker` that couchcmd supplies, and the same function backs both CLIs.

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

- `ReplayShares` against a `FakeTunnel` Host: Share → opening → ready; the
  link serves the viewer and records; a second id gets a different token; the
  first share's token can't open the second recording; the listing isn't
  served; tunnel exit → the next Share re-mounts with a new token; Close
  closes the tunnel (`Opens == Closes`).
- The scoped `ReplayServer`: another id is not found, and so is the listing.
- The broker op validation: `replay-share` takes only Target, a valid id.
- CLI: a fake `messageCall` gives opening, opening, ready → the link printed with
  `#t=`; no Couch answers → loopback fallback; a DIR outside the store → refused;
  `query link` uses the Linker.
- Advisory: a recording whose last line is torn lists `Ended=false`, and the
  listing HTML says unknown.
- Live (existing `replay_live_test` style): a real Chrome opens the share's link
  served through a LocalOnly-tunnel Host and seeks with `#t=`. The tunnel smoke
  test is the operator's.

## Milestones

- M1: replay shares on the Host, the broker op, both CLIs, the advisory, atlas
  and skill text.
- M2: merge main once #470 lands; the replay viewer draws notes and strokes with
  #470's modules; a live check that a note shows at its moment.
