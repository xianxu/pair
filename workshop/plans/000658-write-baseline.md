# Pair Go-package write baseline

Baseline code: b08a927f6fed942fdbb4aef84b09cf07b2a32f47; issue branch has documentation-only measurement checkpoints.

Owned diagnostic run on 2026-10-17. GOFLAGS=-p=3 -trimpath; each binary uses -test.count=1 -test.timeout=20m -test.parallel=1 -test.v. The machine full-suite lock is held. Fresh compiled Pair is supplied via PAIR_TEST_BINARY; ambient Pair/Couch/Zellij session keys are removed. Package cwd is the actual package directory.

The byte counter is macOS proc_pid_rusage RUSAGE_INFO_V2 ri_diskio_byteswritten. Calibration: a flushed4MiB write increments the own-process counter by exactly4MiB. Counters exclude child processes, compiler/build output, shell/Lua test phases and sampler output. These are observed own-process physical-byte lower bounds, not whole-machine SSD writes. A final process-exit race may miss the last increment; the accepted regression check will read inside TestMain before exit.

Per-test attribution samples verbose boundaries with serial test execution. Values are approximate around boundaries. Writable-FD paths are sampled high-water evidence, not exact per-path syscall totals; short file opens can be missed. No privileged tracing was used or needed to identify the dominant fixture writer.

Completed package binaries: 116. Sum of observed own-process bytes: 1,640,353,792. Nonzero exits: 0.

## Per package

| Package (below repository root) | Observed physical bytes | Samples | Exit |
|---|---:|---:|---:|
| cmd/internal/couchcmd | 625,799,168 | 15239 | 0 |
| cmd/internal/couchtty | 609,632,256 | 13929 | 0 |
| cmd/internal/couchcore | 95,576,064 | 67060 | 0 |
| cmd/internal/workbenchperftest | 90,030,080 | 36 | 0 |
| cmd/internal/sessionsync | 71,299,072 | 1901 | 0 |
| cmd/internal/threadactivity | 30,089,216 | 117 | 0 |
| cmd/internal/launcher | 19,476,480 | 6625 | 0 |
| cmd/internal/webcast | 12,791,808 | 1676 | 0 |
| cmd/internal/reviewcmd | 10,682,368 | 1049 | 0 |
| cmd/internal/titlepoller | 10,592,256 | 311 | 0 |
| cmd/internal/diagnosticlog | 10,211,328 | 6879 | 0 |
| cmd/internal/historyarchive | 10,149,888 | 2450 | 0 |
| cmd/internal/couchidentity | 9,166,848 | 817 | 0 |
| cmd/internal/sessionwatch | 6,221,824 | 3313 | 0 |
| cmd/internal/archivecmd | 4,268,032 | 63 | 0 |
| cmd/internal/storagegc | 4,042,752 | 3394 | 0 |
| cmd/pair-go | 2,867,200 | 959 | 0 |
| cmd/internal/wrapcmd | 2,433,024 | 8646 | 0 |
| cmd/internal/webcast/notedraft | 1,777,664 | 887 | 0 |
| cmd/internal/sessionledger | 1,617,920 | 457 | 0 |
| cmd/internal/gcruntime | 1,249,280 | 2581 | 0 |
| cmd/internal/workbenchshortcut | 1,183,744 | 212 | 0 |
| cmd/internal/runtimebundle | 1,052,672 | 131 | 0 |
| cmd/internal/sessioninventory | 1,044,480 | 2560 | 0 |
| cmd/internal/artifactpath | 987,136 | 791 | 0 |
| cmd/internal/runtimebundlegen | 876,544 | 361 | 0 |
| cmd/internal/couchsingleton | 749,568 | 1300 | 0 |
| cmd/internal/submission | 733,184 | 906 | 0 |
| cmd/internal/scrollbackcmd | 679,936 | 341 | 0 |
| cmd/internal/changelogcmd | 516,096 | 304 | 0 |
| cmd/internal/pairlog | 294,912 | 434 | 0 |
| cmd/internal/checkpoint | 270,336 | 99 | 0 |
| cmd/internal/gccmd | 241,664 | 482 | 0 |
| cmd/internal/broadcast | 184,320 | 5308 | 0 |
| cmd/internal/terminalcapture | 176,128 | 225 | 0 |
| cmd/internal/webcast/query | 167,936 | 393 | 0 |
| cmd/internal/notifytransport | 139,264 | 406 | 0 |
| cmd/internal/adapt | 106,496 | 63 | 0 |
| cmd/internal/clipcmd | 98,304 | 295 | 0 |
| cmd/internal/testpair | 81,920 | 913 | 0 |
| cmd/internal/retentioncmd | 69,632 | 189 | 0 |
| cmd/internal/pairlifecycletest | 61,440 | 301 | 0 |
| cmd/internal/diagnosticcmd | 57,344 | 59 | 0 |
| cmd/internal/webcast/recording | 53,248 | 309 | 0 |
| cmd/internal/nativelock | 49,152 | 108 | 0 |
| internal/reporoot | 40,960 | 36 | 0 |
| cmd/internal/crashreport | 40,960 | 908 | 0 |
| cmd/internal/osfs | 36,864 | 58 | 0 |
| probes/zellijprobe | 36,864 | 43 | 0 |
| cmd/internal/webcast/fidelity | 32,768 | 29 | 0 |
| cmd/internal/slugcmd | 32,768 | 114 | 0 |
| cmd/internal/webcast/assets | 32,768 | 244 | 0 |
| cmd/internal/continuationcmd | 32,768 | 168 | 0 |
| cmd/internal/model | 28,672 | 96 | 0 |
| cmd/internal/dispatcher | 28,672 | 95 | 0 |
| cmd/internal/couchmessage | 20,480 | 3816 | 0 |
| cmd/internal/webcast/recorder | 16,384 | 929 | 0 |
| cmd/internal/keyscmd | 16,384 | 78 | 0 |
| cmd/internal/contextcmd | 16,384 | 65 | 0 |
| cmd/internal/layoutcmd | 12,288 | 195 | 0 |
| cmd/internal/durablefile | 12,288 | 38 | 0 |
| cmd/couch | 12,288 | 683 | 0 |
| cmd/internal/procutil | 12,288 | 391 | 0 |
| probes/zellijcalls | 8,192 | 138 | 0 |
| cmd/internal/opener | 8,192 | 84 | 0 |
| cmd/internal/webcast/submission | 8,192 | 38 | 0 |
| cmd/internal/scribecmd | 4,096 | 33 | 0 |
| cmd/internal/notifycmd | 4,096 | 34 | 0 |
| cmd/internal/termcmd | 4,096 | 1238 | 0 |
| cmd/internal/draftroute | 4,096 | 39 | 0 |
| cmd/internal/testwait | 0 | 41 | 0 |
| cmd/internal/zellijsession | 0 | 16 | 0 |
| cmd/internal/webcast/board | 0 | 276 | 0 |
| cmd/internal/slugline | 0 | 21 | 0 |
| cmd/internal/castlock | 0 | 88 | 0 |
| cmd/internal/webcast/drive | 0 | 125 | 0 |
| cmd/internal/terminal | 0 | 7264 | 0 |
| probes/messageidle | 0 | 17 | 0 |
| cmd/internal/rowtext | 0 | 43 | 0 |
| cmd/internal/ctxmeter | 0 | 16 | 0 |
| cmd/internal/hostty | 0 | 73 | 0 |
| cmd/internal/agentdrive | 0 | 273 | 0 |
| cmd/internal/mouseinput | 0 | 96 | 0 |
| cmd/internal/webcast/capability | 0 | 47 | 0 |
| cmd/internal/cdp | 0 | 83 | 0 |
| cmd/internal/commitoutcome | 0 | 32 | 0 |
| cmd/internal/threadrecord | 0 | 290 | 0 |
| cmd/internal/perfbudget | 0 | 17 | 0 |
| cmd/internal/couchkeys | 0 | 40 | 0 |
| probes/zellijwrapmargin | 0 | 34 | 0 |
| cmd/internal/orientation | 0 | 39 | 0 |
| cmd/internal/panebirth | 0 | 31 | 0 |
| probes/zellijwritechars | 0 | 17 | 0 |
| probes/claudedraftsend | 0 | 17 | 0 |
| cmd/internal/historyarchive/archivetest | 0 | 27 | 0 |
| cmd/internal/hoprttcmd | 0 | 111 | 0 |
| cmd/internal/ansi | 0 | 121 | 0 |
| cmd/internal/strictjson | 0 | 72 | 0 |
| cmd/internal/sessioninventorytest | 0 | 30 | 0 |
| cmd/internal/textwidth | 0 | 19 | 0 |
| cmd/internal/zellijpane | 0 | 72 | 0 |
| cmd/internal/notifyosc | 0 | 41 | 0 |
| cmd/internal/tailmarkup | 0 | 42 | 0 |
| cmd/internal/pairlifecycle | 0 | 303 | 0 |
| cmd/internal/readiness | 0 | 38 | 0 |
| cmd/internal/terminalqualify | 0 | 170 | 0 |
| cmd/internal/resumeform | 0 | 39 | 0 |
| cmd/internal/nativeid | 0 | 17 | 0 |
| cmd/internal/ptychild | 0 | 546 | 0 |
| cmd/internal/webcast/notes | 0 | 176 | 0 |
| cmd/internal/keyhelp | 0 | 115 | 0 |
| cmd/internal/ttyio | 0 | 619 | 0 |
| cmd/probes/terminalqualify | 0 | 23 | 0 |
| cmd/internal/entrypoint | 0 | 68 | 0 |
| cmd/internal/agentcmd | 0 | 17 | 0 |
| probes/zellijbirthrace | 0 | 19 | 0 |

## Leading tests in packages with at least1MiB observed writes

| Package | Test / boundary | Sample-attributed bytes |
|---|---|---:|
| cmd/internal/couchcmd | TestWorkbenchPerformanceMessageUnchangedEvidence | 600,489,984 |
| cmd/internal/couchcmd | TestCapturePTYBothBoundaries/regular | 8,720,384 |
| cmd/internal/couchcmd | TestCapturePTYBothBoundaries/isolated | 7,753,728 |
| cmd/internal/couchtty | TestWorkbenchPerformanceIdleUITick | 600,244,224 |
| cmd/internal/couchtty | TestCaptureBadgeFirstPaintAndIdleFailure/actor | 4,202,496 |
| cmd/internal/couchtty | TestCaptureBadgeFirstPaintAndIdleFailure/switcher | 4,202,496 |
| cmd/internal/couchcore | TestLocalJournalRecoversImagesLargerThanPayloadReadLimit | 14,684,160 |
| cmd/internal/couchcore | TestResourceHistoryStoreFixedStageAndByteBound | 8,392,704 |
| cmd/internal/couchcore | TestLaunchFailureStoreEncodedLimitPreservesPreviousPublication | 4,460,544 |
| cmd/internal/workbenchperftest | TestFixtureStreamsValidDistinctOwnerArtifacts | 90,021,888 |
| cmd/internal/workbenchperftest | TestFixtureSmallLedgerRetainsRequiredRecords | 8,192 |
| cmd/internal/workbenchperftest | __startup__ | 0 |
| cmd/internal/sessionsync | TestAReaderOutsideTheFlockNeverSeesATornWrite | 19,849,216 |
| cmd/internal/sessionsync | TestLedgerTrimKeepsNewestYoungRows | 9,449,472 |
| cmd/internal/sessionsync | TestRegisteredGrowthReservesHandoffBeforePublication/rename | 5,292,032 |
| cmd/internal/threadactivity | TestIdleActivityTickWarmAllocationIndependentOfLedgerHistory | 30,023,680 |
| cmd/internal/threadactivity | TestIdleActivityTickUnchangedFilesDecodeZeroBytes | 49,152 |
| cmd/internal/threadactivity | TestIdleActivityUsesCurrentLaunchNotResumeFallback | 16,384 |
| cmd/internal/launcher | TestNativeSnapshotFindsArgvServer | 10,514,432 |
| cmd/internal/launcher | TestOSRuntimeInferAgentBoundsLocalPresentationMetadata/ledger_oversize | 4,202,496 |
| cmd/internal/launcher | TestOSRuntimeLifecycleMetadataRejectsOversizeAndKeepsLegacyQuit | 1,908,736 |
| cmd/internal/webcast | TestReplayServerRoutes | 1,941,504 |
| cmd/internal/webcast | TestRecordingServerServesOnlyItsRecording | 1,937,408 |
| cmd/internal/webcast | TestReplaySharesShareOneRecordingPerToken | 1,937,408 |
| cmd/internal/reviewcmd | TestIdentitySnapshotRejectsMovementAndSize | 8,396,800 |
| cmd/internal/reviewcmd | TestIdentitySnapshotRejectsNonTextAndEncodedOverflow | 2,109,440 |
| cmd/internal/reviewcmd | TestIdentityRejectsPausedCheckout | 28,672 |
| cmd/internal/titlepoller | TestTitleProjectionRepeatedPollWorkIndependentOfLedgerHistory | 10,498,048 |
| cmd/internal/titlepoller | TestTitleCLILeaseEOFRecoversInterruptedPublications/publication-pid | 16,384 |
| cmd/internal/titlepoller | TestTitleCLILeaseEOFRecoversInterruptedPublications/publication-workspace | 16,384 |
| cmd/internal/diagnosticlog | TestBackupFenceRotatesBeyondFormerCountAndInventoryBounds | 2,289,664 |
| cmd/internal/diagnosticlog | TestConcurrentWritersReopenCurrent | 815,104 |
| cmd/internal/diagnosticlog | TestDiagnosticLargeInventoryDoesNotBlockRotation | 593,920 |
| cmd/internal/historyarchive | TestSameContentStreams | 9,449,472 |
| cmd/internal/historyarchive | TestContinuousPassArchivesQuietFilesOnceAndNewVersions | 147,456 |
| cmd/internal/historyarchive | TestCiphertextOverflowIsTheSentinelAndASnapshotSkip | 65,536 |
| cmd/internal/couchidentity | TestInspectRejectsRegressedAndMalformedAuthority | 4,255,744 |
| cmd/internal/couchidentity | TestStoreAuthorityValidation/oversized-host | 4,214,784 |
| cmd/internal/couchidentity | TestStoreCapacityAndOverflow/true | 253,952 |
| cmd/internal/sessionwatch | TestRunConfirmsCodexBindingFromARecordedPeerTurn/the_rollout_holds_an_unrelated_turn | 5,992,448 |
| cmd/internal/sessionwatch | TestMigrateProoflessBindingsUpgradesPersistedLegacyOwner | 16,384 |
| cmd/internal/sessionwatch | TestRepairPreviewAndApplyPreserveConfig | 16,384 |
| cmd/internal/archivecmd | TestArchiveCLIRejectsRequestFiles/oversized | 4,202,496 |
| cmd/internal/archivecmd | TestArchiveCLIRejectsRequestFiles/unknown_field | 8,192 |
| cmd/internal/archivecmd | TestArchiveCLIRejectsRequestFiles/invalid | 8,192 |
| cmd/internal/storagegc | TestPlaintextMetadataPublishesMissingTargetAtLimit | 1,048,576 |
| cmd/internal/storagegc | TestCollectorLegacyPendingCancellationAndSharedBudget | 446,464 |
| cmd/internal/storagegc | TestReserveStartReclaimsDeadPreSpawnReservationsBeforeCap | 172,032 |
| cmd/pair-go | TestEmbeddedAssetsHonorSelectedGlobalRootBeforeLauncher/selected-global | 802,816 |
| cmd/pair-go | TestEmbeddedAssetsHonorSelectedGlobalRootBeforeLauncher/selected-global-over-scoped | 802,816 |
| cmd/pair-go | TestPublicPairCommandFamiliesIgnoreCouchStore/launch-resume-attach/valid-forward | 45,056 |
| cmd/internal/wrapcmd | TestPeerBatchTurnRecorderExactCombinedTurnAndSharedBudget | 933,888 |
| cmd/internal/wrapcmd | TestDetachBeforeFirstTurnAuthorizesRelaunch/codex | 114,688 |
| cmd/internal/wrapcmd | TestDetachBeforeFirstTurnAuthorizesRelaunch/claude | 114,688 |
| cmd/internal/webcast/notedraft | TestDiskStoreIgnoresForeignEntriesAndBoundsRecords | 528,384 |
| cmd/internal/webcast/notedraft | TestDiskStoreByteQuotaAccountsActualFiles | 131,072 |
| cmd/internal/webcast/notedraft | TestRecoveryGlobalReservationsAreAtomic | 126,976 |
| cmd/internal/sessionledger | TestLedgerStoreEveryIncompleteWriteRemainsNonAuthoritative/binding | 995,328 |
| cmd/internal/sessionledger | TestLedgerStoreEveryIncompleteWriteRemainsNonAuthoritative/launch | 458,752 |
| cmd/internal/sessionledger | __between__ | 12,288 |
| cmd/internal/gcruntime | TestRuntimeRegistryEnumeratesPastFilteredResidues | 425,984 |
| cmd/internal/gcruntime | TestApplyRetainsAndCollectsOwnersWithoutPairNamespace/legacy-archive | 94,208 |
| cmd/internal/gcruntime | TestApplyRetainsAndCollectsOwnersWithoutPairNamespace/tracked-archive | 86,016 |
| cmd/internal/workbenchshortcut | TestFullscreenStoreBoundsPayloadAndRejectsNonregularFiles | 1,089,536 |
| cmd/internal/workbenchshortcut | TestFullscreenStoreDiagnosticsManagedAndRetained | 32,768 |
| cmd/internal/workbenchshortcut | TestFullscreenStoreRejectsMalformedAndOversizedRecords | 20,480 |
| cmd/internal/runtimebundle | TestTerminalEnvironmentProvidesCompiledProfile | 802,816 |
| cmd/internal/runtimebundle | TestStoreConcurrentExtractSameDigestSucceeds | 200,704 |
| cmd/internal/runtimebundle | TestStoreExtractsAssetsAndWritesMarker | 12,288 |

## Path attribution and retained scenarios

- couchcmd and couchtty: sampled writable paths under their TestWorkbenchPerformance* private temp tree, pair/repos/<scope>/ledger-work-NN.jsonl. Source constructs twenty separate30,000,000-byte ASCII-space-padded JSONL files in each test. Their attributed writes are600,489,984 and600,244,224 bytes respectively.
- workbenchperftest: three private30MB ledgers in TestFixtureStreamsValidDistinctOwnerArtifacts; all remain decoded and checked for exact size and owner-specific launch/binding content.
- threadactivity and titlepoller: private large ledger fixtures in the warm allocation / title projection invariants; retain logical sizes and existing cold/warm read/decode assertions.
- sessionsync: sampled atomic publication paths include private .tl.md.sync.tmp-* and store/documents/state/<id>/.state.json.tmp-* around1MiB. Keep the large replacement, ledger trim and registered-state envelope tests; add a package budget without shrinking them.
- Other large retained scenarios include couchcmd real PTY capture; couchtty oversized capture metadata; launcher oversized metadata/native snapshots; historyarchive streaming content; reviewcmd identity bounds; diagnosticlog rotation/concurrency; couchidentity oversized authority.
- Where an open was shorter than the sampler interval, attribute by the named test and source; do not claim an uncaught path as a traced write.

## Raw local evidence

Temporary diagnostic directory: /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair658-profile.yqKPI1. Each package JSON retains per-test deltas, sampled path high waters, sample counts, errors and explicit limitations, with its verbose log alongside. all-baseline-go.log retains Go outcomes. The initial two rejected harness runs used the wrong cwd (one also retained ambient keys); they are not acceptance evidence. The separate corrected standalone couchcmd run passed with627,539,968 observed bytes; the whole-Go-phase row above is the matched before/after baseline.

## Revisions

- 2026-10-18T07:21:07+00:00 — appended the complete matched before/after comparison after the first after attempt exposed an artifact inventory/auxiliary-name defect. Preserved the original baseline and first failed attempt; six affected package rows were refreshed after the correction. This is a composite measurement, not a claim that one corrected 118-package run passed.

## After comparison

The same 116 original packages measured **1,640,353,792 bytes before** and **445,779,968 bytes after**, a reduction of **1,194,573,824 bytes (72.824%)** in sampled own-process physical writes. Every selected after row below exited 0. Two new packages are listed separately and excluded from this matched total.

The after column deliberately combines two retained attempts:

- **A — 110 unchanged rows:** first complete after attempt from production/test code at `d9c9bab5bb60d3cf68b753bbe079606e8139ac0f`, in `/var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair658-profile.yqKPI1/after-packages.39qinwud`. The 118 package binaries finished in 571.551 seconds after normal one-token admission: 117 passed and one failed. `artifactpath/TestProductionArtifactReferencesAreExactlyClassified` rejected nine absent exhaustive inventory entries and the auxiliary seed’s reserved `ledger-` prefix. The original JSON and verbose failure remain intact; its failing artifactpath row is superseded below.
- **C — six refreshed rows:** corrected commit `b9f427f2602bb3046371e45f1171f17b2bce20df`, in `/var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair658-profile.yqKPI1/correction-packages.4RJduh`. The correction added exact `NonArtifactSources` entries and renamed only the auxiliary seed from `ledger-padding` to `padding-seed`. Exactly artifactpath, workbenchperftest, couchcmd, couchtty, threadactivity and titlepoller were profiled again with a fresh Pair; all six passed in 78.926 seconds under the targeted Go cap. No full-suite token or retry was used for this targeted correction.

Both attempts retain `GOFLAGS=-p=3 -trimpath`, actual package cwd, sanitized Pair/Couch/Zellij environment, supplied freshly built Pair, and `-test.v -test.count=1 -test.timeout=20m -test.parallel=1`. The original sampler/dylib were reused; the original baseline tools and Pair were not overwritten. A machine suite token covers the first full after attempt; package/test concurrency and sampling behavior match the original baseline.

| Package (below repository root) | Before bytes | After bytes | Reduction | Samples before / after | After exit | After source |
|---|---:|---:|---:|---:|---:|:---:|
| cmd/internal/couchcmd | 625,799,168 | 56,639,488 | 569,159,680 | 15239 / 8727 | 0 | C |
| cmd/internal/couchtty | 609,632,256 | 39,669,760 | 569,962,496 | 13929 / 12683 | 0 | C |
| cmd/internal/couchcore | 95,576,064 | 96,018,432 | -442,368 | 67060 / 60932 | 0 | A |
| cmd/internal/workbenchperftest | 90,030,080 | 33,538,048 | 56,492,032 | 36 / 91 | 0 | C |
| cmd/internal/sessionsync | 71,299,072 | 71,299,072 | 0 | 1901 / 1684 | 0 | A |
| cmd/internal/threadactivity | 30,089,216 | 30,093,312 | -4,096 | 117 / 81 | 0 | C |
| cmd/internal/launcher | 19,476,480 | 19,476,480 | 0 | 6625 / 6442 | 0 | A |
| cmd/internal/webcast | 12,791,808 | 12,787,712 | 4,096 | 1676 / 1635 | 0 | A |
| cmd/internal/reviewcmd | 10,682,368 | 10,682,368 | 0 | 1049 / 1487 | 0 | A |
| cmd/internal/titlepoller | 10,592,256 | 10,592,256 | 0 | 311 / 358 | 0 | C |
| cmd/internal/diagnosticlog | 10,211,328 | 10,211,328 | 0 | 6879 / 5323 | 0 | A |
| cmd/internal/historyarchive | 10,149,888 | 10,149,888 | 0 | 2450 / 2321 | 0 | A |
| cmd/internal/couchidentity | 9,166,848 | 9,166,848 | 0 | 817 / 323 | 0 | A |
| cmd/internal/sessionwatch | 6,221,824 | 6,221,824 | 0 | 3313 / 3097 | 0 | A |
| cmd/internal/archivecmd | 4,268,032 | 4,268,032 | 0 | 63 / 64 | 0 | A |
| cmd/internal/storagegc | 4,042,752 | 4,042,752 | 0 | 3394 / 3398 | 0 | A |
| cmd/pair-go | 2,867,200 | 2,867,200 | 0 | 959 / 990 | 0 | A |
| cmd/internal/wrapcmd | 2,433,024 | 2,433,024 | 0 | 8646 / 8600 | 0 | A |
| cmd/internal/webcast/notedraft | 1,777,664 | 1,777,664 | 0 | 887 / 825 | 0 | A |
| cmd/internal/sessionledger | 1,617,920 | 1,617,920 | 0 | 457 / 584 | 0 | A |
| cmd/internal/gcruntime | 1,249,280 | 1,249,280 | 0 | 2581 / 1232 | 0 | A |
| cmd/internal/workbenchshortcut | 1,183,744 | 1,183,744 | 0 | 212 / 211 | 0 | A |
| cmd/internal/runtimebundle | 1,052,672 | 1,638,400 | -585,728 | 131 / 121 | 0 | A |
| cmd/internal/sessioninventory | 1,044,480 | 1,044,480 | 0 | 2560 / 2620 | 0 | A |
| cmd/internal/artifactpath | 987,136 | 987,136 | 0 | 791 / 1232 | 0 | C |
| cmd/internal/runtimebundlegen | 876,544 | 876,544 | 0 | 361 / 756 | 0 | A |
| cmd/internal/couchsingleton | 749,568 | 749,568 | 0 | 1300 / 556 | 0 | A |
| cmd/internal/submission | 733,184 | 733,184 | 0 | 906 / 857 | 0 | A |
| cmd/internal/scrollbackcmd | 679,936 | 679,936 | 0 | 341 / 414 | 0 | A |
| cmd/internal/changelogcmd | 516,096 | 516,096 | 0 | 304 / 370 | 0 | A |
| cmd/internal/pairlog | 294,912 | 294,912 | 0 | 434 / 298 | 0 | A |
| cmd/internal/checkpoint | 270,336 | 270,336 | 0 | 99 / 102 | 0 | A |
| cmd/internal/gccmd | 241,664 | 241,664 | 0 | 482 / 341 | 0 | A |
| cmd/internal/broadcast | 184,320 | 184,320 | 0 | 5308 / 5690 | 0 | A |
| cmd/internal/terminalcapture | 176,128 | 176,128 | 0 | 225 / 210 | 0 | A |
| cmd/internal/webcast/query | 167,936 | 167,936 | 0 | 393 / 395 | 0 | A |
| cmd/internal/notifytransport | 139,264 | 139,264 | 0 | 406 / 214 | 0 | A |
| cmd/internal/adapt | 106,496 | 106,496 | 0 | 63 / 63 | 0 | A |
| cmd/internal/clipcmd | 98,304 | 98,304 | 0 | 295 / 214 | 0 | A |
| cmd/internal/testpair | 81,920 | 94,208 | -12,288 | 913 / 964 | 0 | A |
| cmd/internal/retentioncmd | 69,632 | 69,632 | 0 | 189 / 163 | 0 | A |
| cmd/internal/pairlifecycletest | 61,440 | 61,440 | 0 | 301 / 249 | 0 | A |
| cmd/internal/diagnosticcmd | 57,344 | 57,344 | 0 | 59 / 44 | 0 | A |
| cmd/internal/webcast/recording | 53,248 | 53,248 | 0 | 309 / 315 | 0 | A |
| cmd/internal/nativelock | 49,152 | 49,152 | 0 | 108 / 96 | 0 | A |
| cmd/internal/crashreport | 40,960 | 40,960 | 0 | 908 / 143 | 0 | A |
| internal/reporoot | 40,960 | 40,960 | 0 | 36 / 34 | 0 | A |
| cmd/internal/osfs | 36,864 | 36,864 | 0 | 58 / 51 | 0 | A |
| probes/zellijprobe | 36,864 | 36,864 | 0 | 43 / 42 | 0 | A |
| cmd/internal/continuationcmd | 32,768 | 32,768 | 0 | 168 / 179 | 0 | A |
| cmd/internal/slugcmd | 32,768 | 32,768 | 0 | 114 / 109 | 0 | A |
| cmd/internal/webcast/assets | 32,768 | 32,768 | 0 | 244 / 245 | 0 | A |
| cmd/internal/webcast/fidelity | 32,768 | 32,768 | 0 | 29 / 26 | 0 | A |
| cmd/internal/dispatcher | 28,672 | 28,672 | 0 | 95 / 98 | 0 | A |
| cmd/internal/model | 28,672 | 28,672 | 0 | 96 / 95 | 0 | A |
| cmd/internal/couchmessage | 20,480 | 20,480 | 0 | 3816 / 3919 | 0 | A |
| cmd/internal/contextcmd | 16,384 | 16,384 | 0 | 65 / 72 | 0 | A |
| cmd/internal/keyscmd | 16,384 | 16,384 | 0 | 78 / 83 | 0 | A |
| cmd/internal/webcast/recorder | 16,384 | 16,384 | 0 | 929 / 956 | 0 | A |
| cmd/couch | 12,288 | 12,288 | 0 | 683 / 637 | 0 | A |
| cmd/internal/durablefile | 12,288 | 12,288 | 0 | 38 / 29 | 0 | A |
| cmd/internal/layoutcmd | 12,288 | 12,288 | 0 | 195 / 204 | 0 | A |
| cmd/internal/procutil | 12,288 | 12,288 | 0 | 391 / 393 | 0 | A |
| cmd/internal/opener | 8,192 | 8,192 | 0 | 84 / 70 | 0 | A |
| cmd/internal/webcast/submission | 8,192 | 8,192 | 0 | 38 / 37 | 0 | A |
| probes/zellijcalls | 8,192 | 8,192 | 0 | 138 / 154 | 0 | A |
| cmd/internal/draftroute | 4,096 | 4,096 | 0 | 39 / 38 | 0 | A |
| cmd/internal/notifycmd | 4,096 | 4,096 | 0 | 34 / 31 | 0 | A |
| cmd/internal/scribecmd | 4,096 | 4,096 | 0 | 33 / 37 | 0 | A |
| cmd/internal/termcmd | 4,096 | 4,096 | 0 | 1238 / 1210 | 0 | A |
| cmd/internal/agentcmd | 0 | 0 | 0 | 17 / 16 | 0 | A |
| cmd/internal/agentdrive | 0 | 0 | 0 | 273 / 270 | 0 | A |
| cmd/internal/ansi | 0 | 0 | 0 | 121 / 118 | 0 | A |
| cmd/internal/castlock | 0 | 0 | 0 | 88 / 83 | 0 | A |
| cmd/internal/cdp | 0 | 0 | 0 | 83 / 85 | 0 | A |
| cmd/internal/commitoutcome | 0 | 0 | 0 | 32 / 20 | 0 | A |
| cmd/internal/couchkeys | 0 | 0 | 0 | 40 / 42 | 0 | A |
| cmd/internal/ctxmeter | 0 | 0 | 0 | 16 / 20 | 0 | A |
| cmd/internal/entrypoint | 0 | 0 | 0 | 68 / 66 | 0 | A |
| cmd/internal/historyarchive/archivetest | 0 | 0 | 0 | 27 / 30 | 0 | A |
| cmd/internal/hoprttcmd | 0 | 0 | 0 | 111 / 91 | 0 | A |
| cmd/internal/hostty | 0 | 0 | 0 | 73 / 79 | 0 | A |
| cmd/internal/keyhelp | 0 | 0 | 0 | 115 / 111 | 0 | A |
| cmd/internal/mouseinput | 0 | 0 | 0 | 96 / 83 | 0 | A |
| cmd/internal/nativeid | 0 | 0 | 0 | 17 / 12 | 0 | A |
| cmd/internal/notifyosc | 0 | 0 | 0 | 41 / 37 | 0 | A |
| cmd/internal/orientation | 0 | 0 | 0 | 39 / 37 | 0 | A |
| cmd/internal/pairlifecycle | 0 | 0 | 0 | 303 / 304 | 0 | A |
| cmd/internal/panebirth | 0 | 0 | 0 | 31 / 30 | 0 | A |
| cmd/internal/perfbudget | 0 | 0 | 0 | 17 / 17 | 0 | A |
| cmd/internal/ptychild | 0 | 0 | 0 | 546 / 539 | 0 | A |
| cmd/internal/readiness | 0 | 0 | 0 | 38 / 41 | 0 | A |
| cmd/internal/resumeform | 0 | 0 | 0 | 39 / 39 | 0 | A |
| cmd/internal/rowtext | 0 | 0 | 0 | 43 / 43 | 0 | A |
| cmd/internal/sessioninventorytest | 0 | 0 | 0 | 30 / 29 | 0 | A |
| cmd/internal/slugline | 0 | 0 | 0 | 21 / 21 | 0 | A |
| cmd/internal/strictjson | 0 | 0 | 0 | 72 / 72 | 0 | A |
| cmd/internal/tailmarkup | 0 | 0 | 0 | 42 / 18 | 0 | A |
| cmd/internal/terminal | 0 | 0 | 0 | 7264 / 7569 | 0 | A |
| cmd/internal/terminalqualify | 0 | 0 | 0 | 170 / 162 | 0 | A |
| cmd/internal/testwait | 0 | 0 | 0 | 41 / 31 | 0 | A |
| cmd/internal/textwidth | 0 | 0 | 0 | 19 / 16 | 0 | A |
| cmd/internal/threadrecord | 0 | 0 | 0 | 290 / 287 | 0 | A |
| cmd/internal/ttyio | 0 | 0 | 0 | 619 / 555 | 0 | A |
| cmd/internal/webcast/board | 0 | 0 | 0 | 276 / 276 | 0 | A |
| cmd/internal/webcast/capability | 0 | 0 | 0 | 47 / 47 | 0 | A |
| cmd/internal/webcast/drive | 0 | 0 | 0 | 125 / 127 | 0 | A |
| cmd/internal/webcast/notes | 0 | 0 | 0 | 176 / 178 | 0 | A |
| cmd/internal/zellijpane | 0 | 0 | 0 | 72 / 71 | 0 | A |
| cmd/internal/zellijsession | 0 | 0 | 0 | 16 / 17 | 0 | A |
| cmd/probes/terminalqualify | 0 | 0 | 0 | 23 / 23 | 0 | A |
| probes/claudedraftsend | 0 | 0 | 0 | 17 / 16 | 0 | A |
| probes/messageidle | 0 | 0 | 0 | 17 / 17 | 0 | A |
| probes/zellijbirthrace | 0 | 0 | 0 | 19 / 19 | 0 | A |
| probes/zellijwrapmargin | 0 | 0 | 0 | 34 / 33 | 0 | A |
| probes/zellijwritechars | 0 | 0 | 0 | 17 / 17 | 0 | A |

### New packages outside the matched 116

| Package | After bytes | Samples | Exit | Source |
|---|---:|---:|---:|:---:|
| cmd/internal/testdisk | 0 | 84 | 0 | A |
| cmd/internal/testgit | 12,288 | 41 | 0 | A |

### Source and counter limits

The first Pair was built at A; later commits during its queue/run changed only the issue and plan prose. Its retained `source.json` and `source-files.json` identify the build head and hash 2,603 source files. Fresh Pair SHA-256 values are `8d9af2526ea51842370a1e173035de93412206e95e2116dc14f9f5ab5ca18e08` (A) and `6608e632664d8dc67298d46f258bb38561d38a3477c67fd567449bdc94952d7b` (C). Original baseline Pair SHA-256 remains `6ccf6cb559b91d33c415368590a21c02d52fcf2b032cb6a3471c423b5276e0a2`. First-attempt `run-result.json` retains admission/output; `run-summary.json` is its clean machine-readable result. Corrected `summary.json` and `composite-comparison.json` preserve the selected rows and totals.

These are macOS own-process physical-byte samples, not whole-device write totals. Descendants, compiler/cache writes, shell/Lua phases and sampler output are excluded. In particular, couchcore measured 95,576,064 → 96,018,432 own-process bytes and passed in 557.124 seconds; Git child-process savings cannot be inferred from that root-only row. Shared cache savings have a separate direct inventory experiment and are not added here.

Each package JSON retains sample/gone-sample counts, sampled per-test attribution and writable-path high waters. A last sample can race process exit, short open files can evade path sampling, and verbose-boundary attribution is approximate. Zero observed bytes is a lower-bound observation, not proof of no writes. Run-to-run changes of a few pages remain visible in the table rather than being clamped to zero. The mixed-source table does not replace the required recorded full suite on the corrected landing head.
