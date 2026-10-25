# Native upgrade qualification — 2026-10-24

The reusable `TestDisposableCouchBuildUpgrade` passed a same-main control for
Claude and Codex, then failed at startup adoption against the failed combined
stage. No readiness files were removed or synthesized. The fixture retained
one owner, root, RunID, genuine old agents, registries and native sockets while
replacing the private Pair/Couch/helper copies and `PAIR_HOME`.

| Selected source | Commit | Compiled identity |
|---|---|---|
| Old main | `38fe8ea272d960705134e21ad3fbf61d26a4bb00` | every executable matches; `vcs.modified=false` |
| Failed stage | `93a57a380d290105b497dfd620ffb9e2e5a15528` | every executable matches; `vcs.modified=false` |

Both source checkouts were clean, with their own ignored runtime assets generated
before building. These are fresh coupled builds of the exact revisions, with
hashes below; they are not claimed byte-identical to an operator's prior build.

| Source | Executable | SHA256 |
|---|---|---|
| Old main | pair | `c6e80e8ad971dcdcf0f5bc04d88d0589c3d889cb645bbd17f1cdd4cd5eec1a7d` |
| Old main | couch | `051d2e59aecc0f16ab3cf478a4245be506daee3cf7455e97d5cd553b1efb8e20` |
| Old main | helper | `dfa5506642ec13da84f4a6d2abc94965189e6847f7b493f0b1980e1be78aefd4` |
| Failed stage | pair | `2d1b6537dd0f293a7b81d66ee16a45cd3c928963fa182791676df7d0f9f7ccea` |
| Failed stage | couch | `21fb42c9d0f482a383a36ccffdef5d499c28b5d0372ba9be284e61cec78ec870` |
| Failed stage | helper | `f9b32fe152c26c225825033e41f12f32ca8fa956f8fdc27b2fa1a6b0ca6c9c06` |

## Same-main healthy control

Command (candidate source equals baseline source):

```sh
env PAIR_NATIVE_COUCH_UPGRADE=1 \
PAIR_COUCH_UPGRADE_BASE_SOURCE=/Volumes/TestScratch/pair879-stage.OeEwVU/baseline \
PAIR_COUCH_UPGRADE_CANDIDATE_SOURCE=/Volumes/TestScratch/pair879-stage.OeEwVU/baseline \
go test ./cmd/couch -run '^TestDisposableCouchBuildUpgrade$' -count=1 -v -timeout=20m
```

PASS127.301s; Claude60.22s and Codex58.19s, sequential. Each had one configured
hub and one provisioned worker. Both passed the actual copied-triplet handoff,
fresh startup clients, public actors/list/show, exact old registry identities
and parent/side-document reads, successful public final resume receipts,
physical exact-tag attachment to both old sessions with distinct retained
drafts, positive native clients and unchanged full actor bindings, fixture
PIDs/generations/native IDs and socket device/inode identities.

Full log: `/tmp/pair879-upgrade-healthy7.log`.
SHA256 `724f562ec33723c73ccb64d61db130f8d314d4d5a5a043a6ee40ba3e7f388e75`.
Exact roots `/Volumes/TestScratch/pc-733546003` and
`/Volumes/TestScratch/pc-3755470119` were cleaned: all owned generations absent,
direct child and IO joined. No native owner remains from this run.

## Failed-stage RED

Same command, changing only candidate source to
`/Volumes/TestScratch/pair879-stage.OeEwVU/tree`.

FAIL214.126s, exit1. Old Claude hub and worker were genuinely registered, their
distinct drafts and old parent/side document registrations were created, and
the old console detached and joined. The physical runtime triplet was replaced
with the failed stage. Candidate startup then could not adopt the old actors.
The captured screen reports:

```text
startup: unknown; observe zellij sessions: zellij list-sessions --no-formatting: Zellij initialization is unverified: 📁
fixture ... Relaunch · state could not be checked
fixture:1 ... Relaunch · state could not be checked
```

The existing `RestartSupervisor` helper eventually failed its steward-registration
wait at the runtime deadline. The observed initialization refusal precedes that
timeout; it is not the unrelated old-main native-creation panic below. The
120-column screen clips the remainder of the session/path; this capture does
not claim the complete missing-marker filename or a public resume receipt.
Stage `zellijquery/readiness_os.go:qualifyCandidates` rejects a naturally old
socket without its positive marker before session listing. The test's baseline
snapshot confirmed that `zellij-ready-v1` was absent/empty and never mutated it.

The RED stopped at its first failed personality, before candidate explicit
resume/physical attach and before a Codex stage case. The same-main control
above separately qualified all those later oracles for both personalities.

Full log: `/tmp/pair879-upgrade-red.log`.
SHA256 `c587cbcfb6c1fbf912a385637b08bf0173753bb73a876bb4a8ff743074151c1b`.
Exact root `/Volumes/TestScratch/pc-1323742597` cleaned with all native
generations absent and child/IO joined. No native owner remains from this run.

## Earlier control failures retained

The issue Log records source-preparation and oracle mistakes, their corrections
and their cleanup results; they are not counted as requested stage REDs.
One unchanged old-main control also hit the known native creation failure,
before any replacement: Zellij server main panicked at
`zellij-server/src/lib.rs:1462:26`, `Option::unwrap()` on `None` at17:41:23.522.
No fixture readiness existed. Its full log is
`/tmp/pair879-upgrade-healthy5.log`, SHA256
`b8dea20226a864414a9e430f53fdc351eab1a0b8cf241a72abc3682564da0a3b`.
Exact root `/Volumes/TestScratch/pc-456185548` cleaned positively; the subsequent
healthy pass does not erase that baseline failure.

Pair:5 separately reports a one-off corrected-#866 GREEN. That driver did not
use unlanded #879 code; its result is not presented as a run of this fixture.
