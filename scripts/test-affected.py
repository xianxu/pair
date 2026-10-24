#!/usr/bin/env python3
"""Execute sdlc's already-planned selection without widening Go coverage."""
from dataclasses import dataclass
import os
import re
import shlex
import subprocess
import sys
import time

GROUPS = frozenset({"lua", "review", "doctor", "build", "shell"})
PACKAGE = re.compile(r"github\.com/xianxu/pair(?:/[A-Za-z0-9_][A-Za-z0-9_.-]*)+")
RUNTIME = "github.com/xianxu/pair/cmd/internal/runtimebundle"


# Source readers bypass the import graph. Run this fixed audited set before
# selected Go packages; never widen the provider's selected-package argv.
SOURCE_GUARDS = {'cmd/internal/adapt': ('TestDetailTruncatedToValidUTF8',
                        'TestGoldenMatchesFixture',
                        'TestLogLineFormat',
                        'TestNilLoggerIsNoOp',
                        'TestOpenAppendsAcrossOpens',
                        'TestOpenBufferedPreservesEventsAndDrainsOnClose',
                        'TestOpenNoTagReturnsNilNoOp',
                        'TestOpenRejectsUnsafeTagBeforeArtifactIO',
                        'TestTruncateEdges'),
 'cmd/internal/agentdrive': ('TestAdmitsIsTheOneAdmissionRule',
                             'TestDismissOnlyIsExactlyOneEsc',
                             'TestDismissablePanelDeclinesEverythingElse',
                             'TestDismissablePanelRecognizesTheUsagePanel',
                             'TestDismissablePanelRefusesEveryWorkingTurn',
                             'TestDismissalToleratesThePanelsTurnFlag',
                             'TestDriveDismissalEndsDoneOrUnconfirmed',
                             'TestNativePrefixSelectionRetainsDriveFreshnessGuard',
                             'TestPanelRefusalNamesEsc',
                             'TestReadModelScreenCaptured',
                             'TestReadModelScreenClaudeCaptured',
                             'TestReadModelScreenCodexAdvancedAndDefaultCaptured',
                             'TestReadModelScreenCodexExplicitNativeSelection',
                             'TestReadModelScreenCodexSelectionRequiresUniqueExplicitPrefix',
                             'TestReadModelScreenCurrentStatusCountsHistoricalConfirmations',
                             'TestReadModelScreenDeclinesAmbiguity',
                             'TestStepsTableIsTheParsersAcceptedSet'),
 'cmd/internal/artifactpath': ('TestArtifactConstructorAuthorityRejectsProductionCommandMutations',
                               'TestPaneReadyDiagnosticsHaveExactVocabularyAllowances',
                               'TestZellijReadinessConstructorsUseCanonicalNamespace',
                               'TestZellijReadinessKeysRemainBoundedAndCurrent',
                               'TestZellijReadinessManifestProtectsLifecycleArtifacts',
                               'TestZellijReadinessSourcesArePreciselyClassified',
                               'TestZellijReadinessStoreBindingNegativeControls',
                               'TestZellijReadinessStoreRetainsCanonicalResolver',
                               'TestArtifactDocsUseBindingsForCurrentConsumers',
                               'TestDeadSymbolGuardDeclarationRules',
                               'TestDeletedVocabularyIsNotDescribedAsLive',
                               'TestDiffKnownViolationsCountsDuplicates',
                               'TestEveryMainPackageIsIgnoredAtTheRepoRoot',
                               'TestEveryMakefileTestSelectorMatchesALiveTest',
                               'TestLifecycleArtifactFamiliesAreClassified',
                               'TestManifestCoversRequiredArtifactFamilies',
                               'TestNoCompanionSuffixConstructionOutsideArtifactpath',
                               'TestNoProductionSymbolIsReferencedOnlyByTests',
                               'TestNonArtifactSourceCannotImportArtifactpath',
                               'TestNonGoConsumersUseExactPathBindings',
                               'TestNormalizeArtifactViolationStripsLineNumbers',
                               'TestProductionArtifactReferencesAreExactlyClassified',
                               'TestProductionRootsCoverTopLevelCommandPackages',
                               'TestProductionSourceFileRecognizesExtensionlessShebang',
                               'TestProductionSourceInventoryIsIndependentOfArtifactTokens',
                               'TestResolvedBindingCatalogIsUniqueAndReferencesKnownFamilies',
                               'TestResolvedConsumerRejectsDirectArtifactConstruction',
                               'TestResolvedConsumerRequiresFamilyCorrelatedBinding',
                               'TestVocabularyConsumerContract'),
 'cmd/internal/broadcast': ('TestDrawNotes',
                            'TestDrawNotesBlankKeepsStyle',
                            'TestDrawNotesCutsWideCharacters',
                            'TestDrawNotesOutlinesTheBox',
                            'TestDrawNotesProperty',
                            'TestDrawNotesWidensUsedColumns',
                            'TestHasPinStopsAtTheFirstSpot',
                            'TestNoteBoardIDsAndCopies',
                            'TestNoteBoardLimits',
                            'TestNoteBoardOverlap',
                            'TestNoteBoardPlacesThePin',
                            'TestNoteBoardResolveIdempotent',
                            'TestNoteBoxEdgeAndWire',
                            'TestNoteCrop',
                            'TestNoteVisible',
                            'TestParseNote',
                            'TestParseNoteAuthor',
                            'TestParseNoteRefusals',
                            'TestParseNoteRefusesQuickly',
                            'TestParseResolve',
                            'TestPinSpotsCaseTable'),
 'cmd/internal/checkpoint': ('TestPhaseListIsWrittenOnlyInAllPhases',),
 'cmd/internal/contextcmd': ('TestAgentPaneJSONRoundTripsThroughKDL',),
 'cmd/internal/continuationcmd': ('TestNewContinueRestartCmd_FakesInZellij',
                                  'TestReadBodyRejectsOversizedInput',
                                  'TestRunRestartFailurePreservesCommittedCheckpoint',
                                  'TestRun_FoldsDraftWhenInCompaction',
                                  'TestRun_NoFoldOrRestartStandalone',
                                  'TestRun_NoRestartFlagSuppressesInCompaction',
                                  'TestRun_TriggersRestartInCompaction'),
 'cmd/internal/couchcmd': ('TestARefusalsNamedCommandsActuallyWork',
                           'TestAutomaticSystemSendHasOnePolicyOwner',
                           'TestAnUnreadableRecordCanBeRebootedThroughTheRuntime',
                           'TestAtlasDocumentsEveryTypedOperation',
                           'TestBeginConsoleArmsWarmPassOnlyAfterSuccessfulAttach',
                           'TestBeginConsoleArmsWarmPassWithoutInitialHandle',
                           'TestBindArgsAcceptsPrepareStartFlagsAndPositionals',
                           'TestBindArgsRejectsMissingOrEmptyValueBearingFlag',
                           'TestCLICompositeReferencesDeriveCurrentRepositoryScope',
                           'TestCLIRejectsMissingOrEmptyExplicitAgentBeforeSpawn',
                           'TestCandidateResumeRetainsQueuedHubGuard',
                           'TestClassifyWait',
                           'TestClassifyWaitCompletedModelControl',
                           'TestClassifyWaitCompletedModelControlPending',
                           'TestClassifyWaitModelConfirmationDoesNotHideLaterInput',
                           'TestConformanceWorkflowProvisionsModuleToolchain',
                           'TestConformanceWorkflowRunsWithoutSiblingMakefile',
                           'TestConsoleExitForgetsThroughCouchRegistry',
                           'TestConsoleGetsAnActionDispatcher',
                           'TestConsoleGetsCouchsActionableProvider',
                           'TestConsoleRunnerDeclinesWithoutATerminal',
                           'TestConsoleRunnerDeclinesWithoutATerminalWiring',
                           'TestConsoleRunnerDetectsARealPTY',
                           'TestConsoleRunnerWiresThePtyRunnerWhenATerminalExists',
                           'TestConsoleWiringPropagatesAuthoritativeThreadStoreFailures',
                           'TestContinuationPublicationProcessHelper',
                           'TestContinuationWriterPublishesExactCheckpointAcrossWorktrees',
                           'TestCurrentRepoScopeUsesGitRootFromSubdirectory',
                           'TestDialogTitleIgnoresNumberedProse',
                           'TestDirectStoreOperationDoesNotAcquireSupervisorLease',
                           'TestDispatchTableIsIdenticalToTheDeclaredOperationSet',
                           'TestEveryOperationHasASummaryAndDescribedArgs',
                           'TestExternalStopRefusesUntilOwnerRoutingExists',
                           'TestHeldSupervisorRefusesBeforeStartingActor',
                           'TestHelpAndSkillCarryEveryVerbCost',
                           'TestInitialConsoleAttachDispatchesDeclaredOperation',
                           'TestInitialHubBorrowedAttachmentNoticeAppearsOnFirstPaint',
                           'TestInteractiveHubLaunchResumesDetachedSession',
                           'TestInteractiveHubLaunchLeavesProvenParkedThreadIdle',
                           'TestInteractiveHubLaunchLeavesMissingSessionIdle',
                           'TestIsQuestionToleratesTrailingMarkdown',
                           'TestLastProseLineSkipsChrome',
                           'TestListOnEmptyThreadStore',
                           'TestListShowsATreeWithNoAgent',
                           'TestM3DocsMatchActionableSwitcherInventoryProvider',
                           'TestMissingRequiredArgumentIsRejectedBeforeAnyWork',
                           'TestNoCurrentSourcesAdvertiseObsoleteCouchArgv',
                           'TestNotifyHelpAndSkillExplainHubAuthority',
                           'TestOSCompositionChecksSwitchExecutablesAndWiresExactStatus',
                           'TestOperationArityMatchesExpectation',
                           'TestOperationPresentationDocs',
                           'TestParseWaitOptOut',
                           'TestPublicHelpListsOnlyPublicSurface',
                           'TestPublicInternalPublishDescriptionIsTheOnlyHiddenOperation',
                           'TestPublicListAndShowUseDiagnosticFlags',
                           'TestPublicNonTerminalLaunchRefusesBeforeEffects',
                           'TestPublishDescriptionEmptyFlagClearsTheDisplayedSummary',
                           'TestPublishDescriptionUsesCompositeThreadEnvironment',
                           'TestREADMEDocumentsEveryCouchChord',
                           'TestREADMEDocumentsEveryPanelControl',
                           'TestREADMEDocumentsM3ThreadSemantics',
                           'TestREADMEDocumentsOnlyThePublicProjection',
                           'TestREADMEDocumentsTheOperatorFacingSurface',
                           'TestREADMERootEscapeMatchesReducerWhenNoActorIsLive',
                           'TestReadmeDoesNotAdvertiseOwnerRequiredStopAsExternalCommand',
                           'TestReadmeDoesNotAdvertiseRemovedAdmissionFlags',
                           'TestRebootEntrypointAcquiresOwnerScope',
                           'TestRecoverPlanCLIEmitsTheReport',
                           'TestRenderThreadRowsDistinguishesParkedFromDetached',
                           'TestRenderThreadsIsLabelFirstAndKeepsSamePathThreadsDistinct',
                           'TestRenderThreadsQualifiesCollidingDirectoryLabels',
                           'TestRenderedOutputHasNoANSIWhenNotATerminal',
                           'TestResumeAcquiresAndReleasesSupervisorLease',
                           'TestResumeRunsAsTheNewLiveOwner',
                           'TestShortcutConformanceWorkflowSourceTriggers',
                           'TestStartAcquiresAndReleasesSupervisorLease',
                           'TestStartComposesRootAgentAndMatchingRepoDefaultThroughSharedLauncherProfile',
                           'TestStartDefaultsItsPathToCwd',
                           'TestSwitchAgentProducesReadinessWithoutDelivery',
                           'TestSwitcherDialectReachesReboot',
                           'TestTheConsoleRunnerOpensTheTimingTraceFromTheEnvironment',
                           'TestThreadStateTextUsesSharedContinuityOutcome',
                           'TestTypedRegistryResolvesExactlyDeclaredOperations',
                           'TestUnknownPublicOptionIsNonZero',
                           'TestVerbLineRefusesAnUntaggedVerb',
                           'TestWaitNoticeBodyPutsSlotFirstAndBoundsTheSummary',
                           'TestWaitNoticeDialogNotifiesTheHubOncePerEpisode',
                           'TestWaitNoticeHeldByABusyHubArrivesOnce',
                           'TestWaitNoticeIdleOffRestingBranchNotifies',
                           'TestWaitNoticeMutedSlotAndHubAreSilent',
                           'TestWaitNoticeOutputWithoutATurnKeepsTheEpisode',
                           'TestWaitNoticeQuestionNotifiesNormalEndDoesNot',
                           'TestWaitNoticeQuestionSetStartsAndEndsWithTheEpisode',
                           'TestWaitNoticeRetriesAfterARefusedAdmission',
                           'TestWaitNoticeSaysWhenTheMuteListIsUnreadable',
                           'TestWaitNoticeThreshold',
                           'TestWantsConsole',
                           'TestWarmOnlyIsUnreachableFromTheCommandLine',
                           'TestWireAttachAbortCleansStartedActorAfterConsoleRefusal',
                           'TestWireResolverPropagatesContextIntoActionableInventory',
                           'TestWireResolverReportsUnboundParkInBothViews'),
 'cmd/internal/couchcore': ('TestCouchPreferencesNeverReadPairDefaults',
                            'TestArchiveRefusesOpenPark',
                            'TestArchiveRefusesProofForMovedRevision',
                            'TestEveryRetirementTakesAnAuthority',
                            'TestEverySlotStateSiteIsAResource',
                            'TestFreshLeavesRemnantsNeedsTheRetainedSelection',
                            'TestIssue149BlockedRunnersDelegateToOneHandshakeAuthority',
                            'TestIssue149CurrentCoreConceptKinds',
                            'TestIssue149M5CoreConceptInventoryContract',
                            'TestIssue149M5CoreConceptInventoryRejectsRowMutation',
                            'TestIssue149M5DeclarationDispositionSetIsClosed',
                            'TestIssue149M5DeclarationDispositionSourceSetMatchesMilestoneDiff',
                            'TestIssue149M5UnmarkedExportedAuthorityFailsClosed',
                            'TestIssue149PureCoreTestsStayAtPureBoundary',
                            'TestIssue151CoreConceptsMatchCurrentBoundary',
                            'TestIssue151M3ArchitecturalLedgerIsClosed',
                            'TestIssue151M3ChecklistMatchesCurrentBoundary',
                            'TestIssue151M3DeclarationDispositionMutationsFailClosed',
                            'TestIssue151M3DeclarationDispositionSetIsClosed',
                            'TestIssue151M3DeclarationDispositionSourceSetMatchesMilestoneDiff',
                            'TestIssue151M3HistoricalOracleRejectsMovingHeadAndWorktreeBytes',
                            'TestIssue151M3ImplementationDependencyRemovalFails',
                            'TestIssue151M3IntegrationDependenciesMatchPinnedSource',
                            'TestIssue151M3SourceConceptsAppearAtExactPlanPaths',
                            'TestIssue152DeliveredCoreConceptsResolveToGoDeclarations',
                            'TestIssue256CoreConceptsProseCitesTestsNotCoordinates',
                            'TestIssue256PlanTablesMatchTheTree',
                            'TestOnlyTeardownProofHasNoProductionCaller',
                            'TestOpaqueIdentityCommentDoesNotReintroducePathDerivedContract',
                            'TestProjectMilestoneStateProblemsRejectsPrematureClose',
                            'TestProvenTeardownHidesEarlierUnprovenFailure',
                            'TestRetiredClaimsAreNotRestatedAsCurrent',
                            'TestSlotTokenAllowlistIsCurrent',
                            'TestTeardownFoldsCompletedPark',
                            'TestUncheckedProjectMilestoneHasNoClosedMetadata',
                            'TestUnreadableEscapeRefusesADecodableRecord'),
 'cmd/internal/couchtty': ('TestAPassAttemptEndsAsOkItsDiagnosticOrError',
                           'TestATraceEventIsOneLineOfFourTabSeparatedFields',
                           'TestAnUnarmedConsoleTracesNoSeeding',
                           'TestAtlasNamesEveryTraceEvent',
                           'TestConceptInventoryRejectsWholeRowDeletion',
                           'TestCoreConceptsContract',
                           'TestOnlyAPassAttemptsEndIsTraced',
                           'TestTheConsoleTracesStartupThePassAndEachAttempt',
                           'TestTheFirstFrameIsTracedOnce',
                           'TestTheTimingTraceIsSilentWhenOffAndSaysSoWhenItCannotStart'),
 'cmd/internal/dispatcher': ('TestEveryImplementedFamilyIsRoutable',
                             'TestLuaBuiltPairSubcommandsAreDeclaredAndRoutable',
                             'TestOperatorFamiliesHaveREADMEUsage'),
 'cmd/internal/keyhelp': ('TestParseNvimKeymapsHandlesSingleModeAndTrailingSpaceDesc',
                          'TestParseNvimKeymapsInvariantsHoldAcrossForms',
                          'TestParseNvimKeymapsMultiLineCall',
                          'TestParseNvimKeymapsReconcilesAgainstRealFile',
                          'TestParseNvimKeymapsReportsDynamicLhs',
                          'TestParseNvimKeymapsSkipsNonPairDescs',
                          'TestParseNvimKeymapsUnquotedLhsIsUnresolvedNotMisassigned',
                          'TestParseZellijRunBindsAgainstRealFile',
                          'TestParseZellijRunBindsAttributesRunToItsOwnBind',
                          'TestParseZellijRunBindsIgnoresPassThroughBinds',
                          'TestZellijShortcutConfigHasNoImplicitConsumers'),
 'cmd/internal/launcher': ('TestCouchLaunchPromptTransportHelper',
                           'TestCouchLiveCleanupHelper',
                           'TestCreateLayoutWrapperPreservesAgentCommand',
                           'TestLaunchPromptCaptureChild',
                           'TestPSGuardDetectsDirectContextAbsoluteAndWrappedCalls',
                           'TestProductionDoesNotExecPS'),
 'cmd/internal/layoutcmd': ('TestRightPaneRungsZellijConformance',),
 'cmd/internal/mouseinput': ('TestFindLocatesAReportAfterOtherBytes',
                             'TestIsPrefixCoversEveryGenuinePrefixAndNothingElse',
                             'TestNoConsumerComparesARawButtonAgainstAWheelConstant',
                             'TestParseDecodesBothTerminators',
                             'TestParsePrefixReturnsTheWireBytes',
                             'TestParseRefusesWhatItCannotDecode'),
 'cmd/internal/notifytransport': ('TestNamespaceLockCloseOnExec', 'TestRawOpensAreAtomicCloseOnExec'),
 'cmd/internal/reviewcmd': ('TestCleanBesidesLockIgnoresOnlyTheReviewLock',
                            'TestDefinitionCLIContextFlags',
                            'TestDefinitionContextValidationAndEcho',
                            'TestIdentityCollectorRefusesIncompleteObservation',
                            'TestIdentityHistoryLimitAndFraming',
                            'TestIdentityIndexAdmissionKeepsUnstagedEdits',
                            'TestIdentityRejectsIndexMovementDuringCollection',
                            'TestIdentityRejectsPausedCheckout',
                            'TestIdentityRejectsPublishedIndexBeforeHEAD',
                            'TestIdentityRejectsUnsafeAndMissing',
                            'TestIdentitySnapshotBindsWorkingBytes',
                            'TestIdentitySnapshotRejectsMovementAndSize',
                            'TestIdentitySnapshotRejectsNonTextAndEncodedOverflow',
                            'TestIdentityUsesWorktreeLocalIndex',
                            'TestOSGitContextOutputCapAndCancellation',
                            'TestOSGitContextToleratesBrieflyHeldStdout',
                            'TestPrepareRealFirstOpenReceipt',
                            'TestPrepareRejectsAmbiguousBeforeTracking',
                            'TestReadinessTreatsTheReviewLockAsClean',
                            'TestResolveIdentityHistoryAndReceipt',
                            'TestRunDefinitionValidation',
                            'TestRunDefinitionWriteError',
                            'TestRunDefinitionWritesResult',
                            'TestRunOpenMissingFile',
                            'TestRunOpenRefusesLiveWithoutEffects',
                            'TestRunOpenSpawnsWhenNoLivePane',
                            'TestRunReadinessJSON',
                            'TestRunReadinessPrepareNew',
                            'TestRunReadinessPrepareResume',
                            'TestRunReadinessPrepareStopAndInteract',
                            'TestRunReadinessPrepareTrack',
                            'TestRunTargetInvalidStatus',
                            'TestRunTargetSessionPriority'),
 'cmd/internal/sessioninventory': ('TestEveryCoreConceptIntroductionMatchesDeclarations',
                                   'TestEveryIssueOwnedTypeHasConceptDisposition',
                                   'TestIssueOwnedSourceCandidatesIncludeNewPackagesAndPlanSources',
                                   'TestNativeAuthorityShadowSweepRejectsSyntheticConsumer',
                                   'TestNoNativeAuthorityShadowInInteractiveConsumers',
                                   'TestREADMEDocumentsSessionInventoryContract',
                                   'TestRecoverPairBindingsClassifiesEveryRecognizedEvidenceRejection',
                                   'TestRecoverPairBindingsDiagnosesRejectedAndUnknownEvidence',
                                   'TestRunCLIResultMatrix',
                                   'TestTypeDispositionRejectsPrivateUnknownAndInlineBypassClasses'),
 'cmd/internal/storagegc': ('TestManagedIOInventoryAnchors',),
 'cmd/internal/termcmd': ('TestChordRoleResolvesOwnPaneUnderAmbiguousFocus',
                          'TestCurrentRightTerminalPaneFastPathAnswersWhatTheSlowPathWould',
                          'TestEveryHandledTerminalChordIsDocumented',
                          'TestFocusedWorkbenchPanesFallsBackWhenTheGateFails',
                          'TestFocusedWorkbenchPanesFastPathAnswersWhatTheSlowPathWould',
                          'TestNoDeclarationCarriesTwoStackedGodocs',
                          'TestParseSGRMousePress',
                          'TestPumpStdinDecodesSplitAltChord',
                          'TestPumpStdinDoesNotRouteDraftAltNFromTheRightPane',
                          'TestPumpStdinHandlesTerminalTabActions',
                          'TestPumpStdinOffersLeftPressToStrip',
                          'TestPumpStdinPassesDraftAltNWhenDraftMissing',
                          'TestPumpStdinPassesDraftAltNWithoutPaneInventory',
                          'TestPumpStdinRenameBareEscapeCancelsOnTimer',
                          'TestPumpStdinRenameCancelsOnEOF',
                          'TestPumpStdinRenameCmdDeleteDeletesToStart',
                          'TestPumpStdinRenameCommitsInFrameWithoutChildPrompt',
                          'TestPumpStdinRenameConsumesSameReadSuffix',
                          'TestPumpStdinRenameConsumesShortcutMouseAndPaste',
                          'TestPumpStdinRenameEntryFailureConsumesInput',
                          'TestPumpStdinRenameEscapeContinuationBeatsTimer',
                          'TestPumpStdinRenameEscapeTimeoutThenNextReadForwards',
                          'TestPumpStdinRenameFinishFailurePreservesOutcome',
                          'TestRightTerminalClassifierClassifiesAsARightTerminal',
                          'TestRightTerminalPaneShellMatchesLayout3',
                          'TestRunTestShortcutGlobalLayoutToggle',
                          'TestRunTestShortcutIgnoresNonTerminalPane',
                          'TestRunTestShortcutRecordsLeftPane',
                          'TestRunTestShortcutRightTerminalActions',
                          'TestSplitHalfChordsWorkViaRegistry',
                          'TestSplitTerminalDownIsNativeTiledSplit',
                          'TestSplitTerminalDownRefusesWithoutRightTerminal',
                          'TestTerminalAltKRecordsLeavingSplitHalf',
                          'TestTerminalHelpAndChangelogRouteWithoutFocusChange',
                          'TestTerminalMuxChildStopsOneRowShortOfThePane',
                          'TestTerminalMuxPaneTitleShowsTabs',
                          'TestTerminalMuxSetPaneTitleTargetsOwnPane',
                          'TestTheRenameFieldPlacesTheCaretAtTheCursor'),
 'cmd/internal/terminal': ('TestConsumersOfNoDestinationMethodsClassifyIt',
                           'TestNoDestinationProducersAreEnumerated',
                           'TestProfileQueriesUseEndpointReplyTransport',
                           'TestProfileQueryLiteralReplies',
                           'TestProfileRejectsUnsupportedAdvertising',
                           'TestProfileTerminfoCompiledContract'),
 'cmd/internal/testdisk': ('TestGoCacheWriteBudgetConformance',),
 'cmd/internal/testpair': ('TestNoPerTestBuildsOfFullPair',
                           'TestPairEnvironmentBootstrapIntegration',
                           'TestPairEnvironmentGuardFixtures',
                           'TestPairEnvironmentGuardIndirectTerminalFixture',
                           'TestPairEnvironmentGuardInstalledCouchFullSource',
                           'TestPairEnvironmentGuardScriptFixtures',
                           'TestPairSubprocessesUseIsolatedEnvironment',
                           'TestPerTestBuildGuardFixtures',
                           'TestPerTestBuildGuardOriginalGoRecipes'),
 'cmd/internal/webcast/assets': ('TestRewriteCSSAbsolutizes',
                                 'TestRewriteCSSCorpus',
                                 'TestRewriteCSSExactWhenUnchanged',
                                 'TestRewriteCSSToHash',
                                 'TestRewriteCSSUnterminated'),
 'cmd/internal/workbenchshortcut': ('TestTerminalRoleRequiresLiveRegistration',
                                  'TestAgentReservationMetadataIsExact',
                                    'TestAgentReservationPolicyMatrix',
                                    'TestAltNIsDraftScoped',
                                    'TestAltXCanonicalEncodingContract',
                                    'TestChordAltShiftTDecodesAndNames',
                                    'TestChordMaxFollowsEveryEncodedChord',
                                    'TestDecodeAltArrowChords',
                                    'TestDecodeChord',
                                    'TestDecodeGlobalChord',
                                    'TestDeliverChordArgsEmitsTheCanonicalEncoding',
                                    'TestDraftScopeWinsOverAgentReservation',
                                    'TestEveryArrowChordRegistersBothModifierFamilies',
                                    'TestEveryGlobalBindingHasHelp',
                                    'TestFullscreenAndDraftResizeScope',
                                    'TestGlobalDecisionMatrix',
                                    'TestHostedHelpCoversExactlyTheChordsHostingChanges',
                                    'TestIsGlobalChordMatchesTheGlobalTable',
                                    'TestLastLeftPaneStore',
                                    'TestLastTerminalPaneStoreIsDistinctFromLastLeft',
                                    'TestNoChordSequenceIsAProperPrefixOfAnother',
                                    'TestPaneRole',
                                    'TestRenderedLuaGlobalMapsMatchCommittedFile',
                                    'TestRightTerminalChordPassesThrough',
                                    'TestRightTerminalTUIChordPassesThroughOnlyExplicitPairControls',
                                    'TestRoleBindingsCoverTerminalSwitch',
                                    'TestRoleForPaneWithRegisteredTerminals',
                                    'TestRoleLocalHelpAndChangelog',
                                    'TestShortcutDecision',
                                    'TestTabChordForDeliversGlobalChords',
                                    'TestTabChordsAreHandledInPaneFromEveryRole',
                                    'TestTerminalPaneRegistry'),
 'cmd/internal/zellijreply': ('TestAdmissionContext',
                              'TestConfirmOrdering',
                              'TestConfirmReturnsSecondProjectedSnapshot',
                              'TestEffectCoverageRejectsMissingAndWeakenedGuards',
                              'TestEffectCoverageRejectsUnclassifiedSinkReferences',
                              'TestM2EffectsUseExistingAdmissionToken',
                              'TestParsedFacts',
                              'TestSharedContracts',
                              'TestTolerantPaneTabFieldsRemainAdditive'),
 'cmd/internal/zellijtest': ('TestEveryZellijFixtureReplyUsesProductionDecoder',
                             'TestFixtureReplyGeneratorUnitContracts'),
 'internal/repolint': ('TestNativeShortcutRootRule',
                       'TestTrackedZellijNativeSources',
                       'TestZellijAdmissionProductionMutationControls',
                       'TestZellijNativeAdmissionCatalog',
                       'TestZellijNativeSourceControls',
                       'TestNativeFixtureRootReader',
                       'TestNativeFixtureRootRealSources',
                       'TestNativeFixtureRootRule',
                       'TestTrackedGoFilesPassCatalog'),
 'probes/zellijprobe': ('TestNoProbeExitsPastItsOwnCleanup',)}


# Only audited nested modules participate; never discover packages dynamically or
# widen the provider's selected-package argv. Reader certificates still name the
# original repository-relative package, while Go runs within its module.
SOURCE_MODULES = ("examples/mock-to-real",)
SOURCE_GUARDS["cmd/internal/editorial"] = ("TestContractPinnedSourceBytes",)
SOURCE_GUARDS["examples/mock-to-real/internal/app"] = (
    "TestRealAppAndContract", "TestMissingRealHandler", "TestPreparedRoutesAndNavigationOwnSnapshot")
SOURCE_GUARDS["examples/mock-to-real/internal/buildsample"] = (
    "TestProductionPartialAndCompletedPrivateCopy", "TestCanceledBuildPublishesNothing",
    "TestActiveChildCancellationJoins", "TestReportFailureDoesNotLeaveArtifact",
    "TestExecutorFailureRetiresPartialCandidate", "TestExecutorSuccessValidatesThenPublishesOnce",
    "TestExecutorCancellationJoinsBeforeRetirement")


def _guard_argv(guards, prefix):
    names = sorted({name for tests in guards.values() for name in tests})
    return ["go", "test", *(prefix + package for package in guards),
            "-run", "^(" + "|".join(names) + ")$", "-count=1", "-timeout", "20m"]


def source_guard_argv() -> list[str]:
    root = {package: tests for package, tests in SOURCE_GUARDS.items()
            if not any(package.startswith(module + "/") for module in SOURCE_MODULES)}
    return _guard_argv(root, "github.com/xianxu/pair/")


def source_guard_commands():
    commands = [(source_guard_argv(), None)]
    for module in SOURCE_MODULES:
        guards = {package[len(module) + 1:]: tests for package, tests in SOURCE_GUARDS.items()
                  if package.startswith(module + "/")}
        if guards:
            commands.append((_guard_argv(guards, "./"), module))
    return commands


@dataclass(frozen=True)
class TestSelection:
    packages: tuple[str, ...]
    groups: tuple[str, ...]


def parse_selection(package_text: str, group_text: str) -> TestSelection:
    packages, groups = tuple(package_text.split()), tuple(group_text.split())
    if not packages and not groups:
        raise ValueError("selection is empty (sdlc handles docs-only changes)")
    if len(set(packages)) != len(packages) or len(set(groups)) != len(groups):
        raise ValueError("duplicate package or group")
    if any(not PACKAGE.fullmatch(package) or "..." in package for package in packages):
        raise ValueError("expected exact Pair Go import paths")
    if any(group not in GROUPS for group in groups):
        raise ValueError("unknown test group")
    return TestSelection(packages, groups)


def go_test_argv(selection: TestSelection) -> list[str]:
    packages = list(selection.packages)
    if "lua" in selection.groups and RUNTIME not in packages:
        packages.append(RUNTIME)
    return ["go", "test", *packages, "-count=1", "-timeout", "20m"] if packages else []


def run_selection(selection: TestSelection, make_argv: list[str]) -> None:
    subprocess.run([*make_argv, "test-affected-inventory"], check=True)
    # This guard scans source text across import boundaries. Build owns the
    # same recipe; every other affected selection needs it once up front.
    if "build" not in selection.groups:
        subprocess.run([*make_argv, "test-make-env"], check=True)
    argv = go_test_argv(selection)
    if argv:
        subprocess.run([*make_argv, "runtimebundle-generate"], check=True)
        if selection.packages:
            started = time.monotonic()
            try:
                for guard, directory in source_guard_commands():
                    subprocess.run(guard, cwd=directory, check=True)
            finally:
                print(f"affected source preflight: {time.monotonic() - started:.3f}s", flush=True)
        subprocess.run(argv, check=True)
    if selection.groups:
        subprocess.run(["bash", "tests/lib/with-pair-test-binary.sh", *make_argv,
                        *("test-affected-" + group for group in selection.groups)], check=True)


def main() -> int:
    try:
        selection = parse_selection(os.environ.get("SDLC_AFFECTED_GO_PKGS", ""),
                                    os.environ.get("SDLC_AFFECTED_GROUPS", ""))
    except ValueError as error:
        print(f"invalid affected selection: {error}", file=sys.stderr)
        return 2
    try:
        run_selection(selection, shlex.split(os.environ.get("MAKE", "make")))
    except subprocess.CalledProcessError as error:
        return error.returncode if error.returncode > 0 else 128 - error.returncode
    return 0


if __name__ == "__main__":
    sys.exit(main())
