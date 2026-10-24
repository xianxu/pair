# First native terminal controls

Source88c6b3288e1ac5dadfdf707b36b776e9621c5c13, compiled vcs.modified=false. Exact retained candidate SHA256fcf38decfd8d61415e9eedfd939b908f2bbedad422bd01dbe16f7ea43a5801f8. Native TestDisposableCouchProfile PASS415.469s (test415.27s); all four cleanup proofs reported every owned generation absent and direct child/IO joined.

| Fixture | Workload | CPU sampled / window | CPU share of one core | Completed commands during capture |
|---|---|---|---|---|
| Claude | quiet |380ms /30.10s |1.26% |0 |
| Claude | busy |640ms /30.02s |2.13% |196 |
| Codex | quiet |360ms /30.09s |1.20% |0 |
| Codex | busy |670ms /30.16s |2.22% |200 |

Each case has four genuine fixture actors at120x32, cast off/no viewers. Busy alternates Working/Idle through the existing mailbox; it is not continuous native spinner output. Both personalities parsed all three files with go tool pprof using the retained exact binary. Profile times are60–90s after Couch launch, after actor readiness. CPU samples are coarse at these small totals; they clearly do not reproduce the original135.69% external live measurement.

Claude quiet alloc_space snapshot difference134.39MB, busy275.19MB; terminal capture/clone/compose dominate. These are sampled cumulative counter deltas, not RSS or exact time-sliced allocation rates: runtime allocation accounting can lag GC, and the profiling operation itself allocates. They are component evidence, not grounds to optimize before a reproduced workload.

Raw artifacts remain local (repo text-only asset policy): /Volumes/TestScratch/tmp/pair-slot9/pair865-native.f2WNZr/evidence. The fixed files include one candidate, per-case profiles/top reports/metadata and manifests containing accepted configuration, actor identities and actual command times. Retain until this investigation ends, then requesting owner removes them. Native log /tmp/pair865-native-collection.log.

TL01a28fc1-4c1c-41ea-bed0-0ac980b58d11 identifies scale/age as the next contrast: around25 live slots, steady spinners/Working timers,9h uptime and long scrollbacks. Approved scaled revision9f0916d7-6ff1-4cc4-abde-6e1745f3a52a is recorded in the plan. This is a negative result, not acceptance of a live-cost fix.
