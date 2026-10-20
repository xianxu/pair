# Couch slot scratch on TestScratch

At startup, `cmd/couch` pins the existing native socket directory in
`ZELLIJ_SOCKET_DIR`. Slot temp settings cannot change that namespace, including
commands run inside a slot. Couch keeps its own inherited TMPDIR.

`couchscratch.ReadProbe` admits `/Volumes/TestScratch` only on a separate device
from `/Volumes`, with at least 2 GiB available. `Manager.PrepareSlot` creates a
0700 `tmp/<repo>-slot<N>` and supplies TMPDIR, GOTMPDIR and
CLAUDE_CODE_TMPDIR. Missing/full/unwritable scratch preserves inherited temp
settings. Durable Pair state and Go caches keep their existing destinations.

Before preparing a slot, `couchscratch.ScratchAllowed` checks the selected
agent's sandbox policy (#720). The host write probe alone is insufficient:
Couch runs outside the sandbox it is configuring. Claude's supported local
user/project/local settings combine filesystem allowances and denies; an
enabled sandbox must allow the volume or a parent, with denies taking
precedence. Disabled sandboxing permits the optional root. Codex normally adds
the inherited TMPDIR to workspace-write roots, so its normal configuration
keeps RAM scratch; read-only or `exclude_tmpdir_env_var` declines it. CODEX_HOME
and CLAUDE_CONFIG_DIR route only the non-credential settings reads.

The projection deliberately declines unsupported policy layers rather than
implementing native configuration resolvers: detected Claude managed settings,
CLI settings-source overrides, Codex profiles/config overrides, malformed or
unreadable files, unsupported path forms and unknown agents use normal temp.
Each settings read is capped at 1 MiB; configuration is never rewritten. The
decision runs at launch or explicit show, not UI ticks, and creates no cache,
worker or journal. Existing inherited temp values are preserved on fallback.

`couch --show` includes `scratch: RAM disk` or a normal-temp reason, explicitly
labelled **current launch selection**. This read-only view recomputes admission;
it does not inspect an existing child's environment or create scratch. A launch
that cannot use the volume emits one bounded note. Add the missing allowance
and relaunch the slot to adopt it. Fresh/switch launches supply the selected
invocation through a call-local record projection, while durable last-registered
profiles retain their meaning. Tests include a successful host write alongside
a denying consumer policy, the operator's real allowance shape, and unchanged
Codex TMPDIR-default admission. Live acceptance rides a natural relaunch.

Admission checks availability before identity lookup and falls back immediately
if maintenance holds the manager. Owner-index locking is also nonblocking. The
launch context bounds retained lookup and the first-workspace fallback together.

Launches use the retained thread's repository/slot authority, including warm
reattach from a neutral working directory. An initial primary checkout without
retained authority uses the existing verified workspace resolver; unknown or
conflicting ownership keeps inherited temp. Immutable owner declarations live
outside the swept interiors, reject conflicting identities, and preserve
unknown members. Equal repository basenames cannot silently share ownership.

At startup and hourly, one serial worker walks declared slot interiors. It
removes ordinary top-level entries only when their own and every descendant's
mtime are older than 24 hours. Slot directories and bookkeeping remain until
RAM reboot. Sockets, FIFOs, symlinks and trees containing specials are retained.
A job leaving ordinary temp files untouched for a day accepts the normal
age-cleaned `/tmp` loss risk, including open regular files. No process census or
holder proof participates. Nofollow directory descriptors and revalidation
protect traversal; concurrent filesystem changes can leave an old tree partly
pruned, and there is no atomic compare-and-unlink guarantee against replacement
at the final unlink. Fresh observed entries and special trees are kept.

`ConfigureMetadata` runs once before owner operations. With any native Zellij
server running, it changes nothing and reports
`Zellij metadata stays on disk until the next reboot`. Unknown server evidence
also leaves paths alone. With no server and usable RAM, it retains the original
native `session_info` directory as `session_info.disk` and links the route to
`TestScratch/zellij-session-info`. It never overwrites that backup. Without
usable RAM, its own exact link becomes an empty real directory; the backup is
kept, without restoration. Foreign routes or backup conflicts remain untouched.
There is no live migration, layout linking, or cache pruning.

Existing panes need relaunch to inherit new temp settings. The operator's next
machine reboot supplies the safe metadata switch; a Couch restart with live
servers deliberately defers it. Isolated Couch fixtures disable the global
scratch volume and use private injected roots. Tests cover policy boundaries,
all three metadata branches, guarded age cleanup, counted linear inspection
budgets, slot identity, child environment inheritance and unchanged sockets.
