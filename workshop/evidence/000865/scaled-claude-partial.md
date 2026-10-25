# First scaled Claude collection: partial and unqualified at scale

Source1eb2135667e8b608cd369529ffdff19353f8d486, compiled vcs.modified=false; retained candidate SHA256a44dd8dfa3d6161fe3d7981ac12c649372496bc71f474a96346b322eddd001ff. TestDisposableCouchProfileScale/claude failed514.825s. The matrix stopped on N10; N20, N20_history2000 and N20_age did not run. No retry.

| Requested actors | External CPU / one core | Physical-write delta /30s | Qualification |
|---|---|---|---|
| N1 | 3.576% | 0 | valid single-actor control |
| N5 | 4.708% | 20,480 | INVALID as five steady actors: four stopped before capture |
| N10 | unavailable | unavailable | failed worker5 reboot receipt polling during setup; no profile |

N1 completed1176 ticks over117.67s. N5's steward completed1144 ticks, but workers1–4 stopped at96–97 ticks after9.66–9.75s (about08:18:39), BEFORE capture08:19:23–08:19:53 and the later external window08:19:54–08:20:24. N5's Go subtest PASS does not qualify the requested workload. All actors acknowledged start and retained identities, but the consumer lacked a duration/tick qualification. The native receipts expose the invalid control. Do not infer scale behavior from this CPU contrast.

N10 failed worker5 CLI after a slot-catalog warming attempt: reboot request c7f11c5f-10b2-472f-822d-ced172133631 returned unavailable because caller steward thread9223742121301770243-fixture-1 was not positively listed. Outcome was uncertain; no request was resent. Failure occurred before supervisor reattach/profiling, so missing profile outputs in deferred retention are secondary evidence, not a profiler failure. Native startup diagnostics and bounded server log tails are retained in collection.log. No claim that this shares #866's panic root cause.

All three attempted fleets proved every exact owned generation absent and direct child/IO joined. Their manifests contain cleanup_proved=true with no cleanup_error. Native roots were removed by their owner; retained evidence remains at /Volumes/TestScratch/tmp/pair865-scale.HgmR9S/evidence, including exact executable, complete N1/N5 profiles/top reports, partial N10 manifest, and collection.log. Explicit retention/removal ownership remains with the requesting operator. Cast off, zero viewers, synthetic Claude fixtures only. External CPU windows follow the runtime profiles and must not be substituted for the earlier sample window. Whole-process writes do not establish fseventsd causality.

TL's subsequent direction: repair the workload control and prove tick progression before capture; classify the captured startup failure and file a separate defect if warranted; use the largest reliably starting cohort. Independently extract/land the opt-in profiler from main so TL can request an operator live restart later. #865 retains measurement and optimization; M2 remains unapproved.
