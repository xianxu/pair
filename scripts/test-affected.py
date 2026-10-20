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
SOURCE_GUARDS = {'cmd/internal/artifactpath': ('TestArtifactConstructorAuthorityRejectsProductionCommandMutations',
                               'TestArtifactDocsUseBindingsForCurrentConsumers',
                               'TestDeadSymbolGuardDeclarationRules',
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
 'cmd/internal/checkpoint': ('TestPhaseListIsWrittenOnlyInAllPhases',),
 'cmd/internal/couchcmd': ('TestAtlasDocumentsEveryTypedOperation',
                           'TestM3DocsMatchActionableSwitcherInventoryProvider',
                           'TestNoCurrentSourcesAdvertiseObsoleteCouchArgv',
                           'TestOperationPresentationDocs',
                           'TestREADMEDocumentsEveryCouchChord',
                           'TestREADMEDocumentsEveryPanelControl',
                           'TestREADMEDocumentsM3ThreadSemantics',
                           'TestREADMEDocumentsOnlyThePublicProjection',
                           'TestREADMEDocumentsTheOperatorFacingSurface',
                           'TestREADMERootEscapeMatchesReducerWhenNoActorIsLive'),
 'cmd/internal/couchcore': ('TestEverySlotStateSiteIsAResource',
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
                            'TestOpaqueIdentityCommentDoesNotReintroducePathDerivedContract',
                            'TestRetiredClaimsAreNotRestatedAsCurrent',
                            'TestSlotTokenAllowlistIsCurrent'),
 'cmd/internal/couchtty': ('TestConceptInventoryRejectsWholeRowDeletion', 'TestCoreConceptsContract'),
 'cmd/internal/dispatcher': ('TestEveryImplementedFamilyIsRoutable',
                             'TestLuaBuiltPairSubcommandsAreDeclaredAndRoutable'),
 'cmd/internal/launcher': ('TestPSGuardDetectsDirectContextAbsoluteAndWrappedCalls',
                           'TestProductionDoesNotExecPS'),
 'cmd/internal/mouseinput': ('TestFindLocatesAReportAfterOtherBytes',
                             'TestIsPrefixCoversEveryGenuinePrefixAndNothingElse',
                             'TestNoConsumerComparesARawButtonAgainstAWheelConstant',
                             'TestParseDecodesBothTerminators',
                             'TestParsePrefixReturnsTheWireBytes',
                             'TestParseRefusesWhatItCannotDecode'),
 'cmd/internal/notifytransport': ('TestNamespaceLockCloseOnExec', 'TestRawOpensAreAtomicCloseOnExec'),
 'cmd/internal/sessioninventory': ('TestNativeAuthorityShadowSweepRejectsSyntheticConsumer',
                                   'TestNoNativeAuthorityShadowInInteractiveConsumers'),
 'cmd/internal/storagegc': ('TestManagedIOInventoryAnchors',),
 'cmd/internal/termcmd': ('TestNoDeclarationCarriesTwoStackedGodocs',),
 'cmd/internal/terminal': ('TestConsumersOfNoDestinationMethodsClassifyIt',
                           'TestNoDestinationProducersAreEnumerated'),
 'cmd/internal/testpair': ('TestNoPerTestBuildsOfFullPair',
                           'TestPairEnvironmentBootstrapIntegration',
                           'TestPairEnvironmentGuardFixtures',
                           'TestPairEnvironmentGuardIndirectTerminalFixture',
                           'TestPairEnvironmentGuardInstalledCouchFullSource',
                           'TestPairEnvironmentGuardScriptFixtures',
                           'TestPairSubprocessesUseIsolatedEnvironment',
                           'TestPerTestBuildGuardFixtures',
                           'TestPerTestBuildGuardOriginalGoRecipes'),
 'cmd/internal/zellijreply': ('TestEffectCoverageRejectsMissingAndWeakenedGuards',
                              'TestEffectCoverageRejectsUnclassifiedSinkReferences',
                              'TestM2EffectsUseExistingAdmissionToken'),
 'cmd/internal/zellijtest': ('TestEveryZellijFixtureReplyUsesProductionDecoder',
                             'TestFixtureReplyGeneratorUnitContracts'),
 'internal/repolint': ('TestNativeFixtureRootReader',
                       'TestNativeFixtureRootRealSources',
                       'TestNativeFixtureRootRule',
                       'TestTrackedGoFilesPassCatalog'),
 'probes/zellijprobe': ('TestNoProbeExitsPastItsOwnCleanup',)}


def source_guard_argv() -> list[str]:
    names = sorted({name for tests in SOURCE_GUARDS.values() for name in tests})
    return ["go", "test", *("github.com/xianxu/pair/" + package for package in SOURCE_GUARDS),
            "-run", "^(" + "|".join(names) + ")$", "-count=1", "-timeout", "20m"]


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
                subprocess.run(source_guard_argv(), check=True)
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
