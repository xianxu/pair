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
