# Couch slot scratch on TestScratch

At startup, `cmd/couch` pins the existing native socket directory in
`ZELLIJ_SOCKET_DIR`. Slot temp settings cannot change that namespace, including
commands run inside a slot. Couch keeps its own inherited TMPDIR.

`couchscratch.ReadProbe` admits `/Volumes/TestScratch` only on a separate device
from `/Volumes`, with at least 2 GiB available. `Manager.PrepareSlot` creates a
0700 `tmp/<repo>-slot<N>` and supplies TMPDIR, GOTMPDIR and
CLAUDE_CODE_TMPDIR. Missing/full/unwritable scratch preserves inherited temp
settings. Durable Pair state and Go caches keep their existing destinations.

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
