package couchcore

import (
	"bufio"
	"crypto/sha256"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

type issue149ConceptRequirement struct {
	name string
	path string
	kind string
}

const issue149M5DeclarationDigest = "748b33313dc565abbbfd5db00e690892441c4fb408bd42a8ed194d06a75c8b1f"

const (
	issue149M5Base = "6a714336"
	issue149M5Head = "c434016"
	issue151M3Base = "0c40a8d1"
	issue151M3Head = "d3ee08d548aebb38eb8a6f15bea78cf71c2dafc8"
)

const issue151M3DeclarationDigest = "8a193b51373583eec2f7b9c25e4a8df6fdc2bb6b8113adf0005d6f1cc878344f"
const issue151M3ArchitecturalLedgerDigest = "632927f3012589a88f61b150c82dcafbe01a7882b2d2bf76aeb1690614ddf5bd"

// issue151M3GoSources is the exhaustive net set of Go sources changed by M3.
// The declaration digest gives every declaration a closed-set disposition from
// issue151M3ArchitecturalDeclarations; every remaining declaration is detail.
// A declaration cannot be added under either label without deliberately
// updating the immutable snapshot ledger and its plan contract.
var issue151M3GoSources = []string{
	"cmd/internal/artifactpath/coverage_test.go",
	"cmd/internal/artifactpath/manifest.go",
	"cmd/internal/couchcmd/readme_test.go",
	"cmd/internal/couchcmd/run.go",
	"cmd/internal/couchcmd/run_test.go",
	"cmd/internal/couchcore/abort_started_test.go",
	"cmd/internal/couchcore/actionableinventory.go",
	"cmd/internal/couchcore/actionableinventory_test.go",
	"cmd/internal/couchcore/artifactcollision.go",
	"cmd/internal/couchcore/artifactcollision_fake.go",
	"cmd/internal/couchcore/couch.go",
	"cmd/internal/couchcore/couch_test.go",
	"cmd/internal/couchcore/launch_existing.go",
	"cmd/internal/couchcore/launchhelper.go",
	"cmd/internal/couchcore/operationdispatch.go",
	"cmd/internal/couchcore/operationdispatch_test.go",
	"cmd/internal/couchcore/plan_contract_test.go",
	"cmd/internal/couchcore/ptyrunner.go",
	"cmd/internal/couchcore/ptyrunner_test.go",
	"cmd/internal/couchcore/resume.go",
	"cmd/internal/couchcore/runner.go",
	"cmd/internal/couchcore/runner_contract_test.go",
	"cmd/internal/couchcore/runner_fake.go",
	"cmd/internal/couchcore/runner_test.go",
	"cmd/internal/couchcore/starttransaction_integration_test.go",
	"cmd/internal/couchcore/tracked_launch_cancellation_test.go",
	"cmd/internal/couchtty/console.go",
	"cmd/internal/couchtty/console_attach_transaction_test.go",
	"cmd/internal/couchtty/console_menu.go",
	"cmd/internal/couchtty/console_menu_operation_test.go",
	"cmd/internal/couchtty/console_menu_test.go",
	"cmd/internal/couchtty/console_panel_regression_test.go",
	"cmd/internal/couchtty/console_preview_test.go",
	"cmd/internal/couchtty/console_run_menu_test.go",
	"cmd/internal/couchtty/console_test.go",
	"cmd/internal/couchtty/core_concepts_contract_test.go",
	"cmd/internal/couchtty/menu.go",
	"cmd/internal/couchtty/menu_async_test.go",
	"cmd/internal/couchtty/menu_perf_test.go",
	"cmd/internal/couchtty/menu_refresh.go",
	"cmd/internal/couchtty/menu_refresh_test.go",
	"cmd/internal/couchtty/menu_render.go",
	"cmd/internal/couchtty/menu_render_test.go",
	"cmd/internal/couchtty/menu_test.go",
	"cmd/internal/couchtty/operation_queue.go",
	"cmd/internal/couchtty/operation_queue_test.go",
	"cmd/internal/couchtty/park_latency_test.go",
	"cmd/internal/hostty/control.go",
	"cmd/internal/hostty/fake.go",
	"cmd/internal/hostty/hostty_test.go",
	"cmd/internal/sessioninventory/catalog.go",
	"cmd/internal/sessioninventory/event.go",
	"cmd/internal/sessioninventory/event_test.go",
	"cmd/internal/sessioninventory/query.go",
	"cmd/internal/sessioninventory/query_test.go",
	"cmd/internal/sessioninventory/round.go",
	"cmd/internal/sessioninventory/round_test.go",
	"cmd/internal/sessionwatch/run_test.go",
	"cmd/internal/textwidth/textwidth.go",
	"cmd/internal/textwidth/textwidth_test.go",
	"cmd/probes/couchstartrecovery/main.go",
}

var issue151M3DeletedGoSources = []string{
	"cmd/internal/couchtty/panel.go",
	"cmd/internal/couchtty/panel_test.go",
}

type issue151ArchitecturalDeclaration struct {
	kind, delivery, current, source string
	present, absent                 []string
}

var issue151M3ArchitecturalDeclarations = map[string]issue151ArchitecturalDeclaration{
	"ActionableThreadSummary":               {"pure", "M1/M3", "present", "cmd/internal/couchcore/actionableinventory.go", []string{"cmd/internal/couchcore/actionableinventory.go"}, nil},
	"LiveTTYObservation":                    {"pure", "M1", "present", "cmd/internal/couchcore/actionableinventory.go", []string{"cmd/internal/couchcore/actionableinventory.go"}, nil},
	"ParkedResumeObservation":               {"pure", "M3", "present", "cmd/internal/couchcore/actionableinventory.go", []string{"cmd/internal/couchcore/actionableinventory.go"}, nil},
	"ProjectActionableThreads":              {"pure", "M1/M3", "present", "cmd/internal/couchcore/actionableinventory.go", []string{"cmd/internal/couchcore/actionableinventory.go"}, nil},
	"StartResolution":                       {"pure", "M1", "present", "cmd/internal/couchcore/startresolution.go", []string{"cmd/internal/couchcore/startresolution.go"}, nil},
	"StartResolutionFingerprint":            {"pure", "M1", "present", "cmd/internal/couchcore/startresolution.go", []string{"cmd/internal/couchcore/startresolution.go"}, nil},
	"ResolveStartResolution":                {"pure", "M1", "present", "cmd/internal/couchcore/startresolution.go", []string{"cmd/internal/couchcore/startresolution.go"}, nil},
	"MenuState":                             {"pure", "M2", "present", "cmd/internal/couchtty/menu.go", []string{"cmd/internal/couchtty/menu.go"}, nil},
	"MenuFrame":                             {"pure", "M2", "present", "cmd/internal/couchtty/menu.go", []string{"cmd/internal/couchtty/menu.go"}, nil},
	"MenuEvent":                             {"pure", "M2", "present", "cmd/internal/couchtty/menu.go", []string{"cmd/internal/couchtty/menu.go"}, nil},
	"ReduceMenu":                            {"pure", "M2", "present", "cmd/internal/couchtty/menu.go", []string{"cmd/internal/couchtty/menu.go"}, nil},
	"MenuLayout":                            {"pure", "M2", "present", "cmd/internal/couchtty/menu_render.go", []string{"cmd/internal/couchtty/menu_render.go"}, nil},
	"AgeBand":                               {"pure", "M2", "present", "cmd/internal/couchtty/menu_render.go", []string{"cmd/internal/couchtty/menu_render.go"}, nil},
	"RenderMenu":                            {"pure", "M2", "present", "cmd/internal/couchtty/menu_render.go", []string{"cmd/internal/couchtty/menu_render.go"}, nil},
	"PreviewSchedule":                       {"pure", "M2", "present", "cmd/internal/couchtty/menu_async.go", []string{"cmd/internal/couchtty/menu_async.go"}, nil},
	"AdvancePreviewSchedule":                {"pure", "M2", "present", "cmd/internal/couchtty/menu_async.go", []string{"cmd/internal/couchtty/menu_async.go"}, nil},
	"RefreshSchedule":                       {"pure", "M3", "present", "cmd/internal/couchtty/menu_refresh.go", []string{"cmd/internal/couchtty/menu_refresh.go"}, nil},
	"AdvanceRefreshSchedule":                {"pure", "M3", "present", "cmd/internal/couchtty/menu_refresh.go", []string{"cmd/internal/couchtty/menu_refresh.go"}, nil},
	"PanelKey":                              {"pure", "M2", "modified, present", "cmd/internal/couchtty/panelkeys.go", []string{"cmd/internal/couchtty/panelkeys.go"}, nil},
	"DecodePanelKeys":                       {"pure", "M2", "modified, present", "cmd/internal/couchtty/panelkeys.go", []string{"cmd/internal/couchtty/panelkeys.go"}, nil},
	"PanelModel":                            {"pure", "M3", "deleted", "cmd/internal/couchtty/panel.go", nil, []string{"cmd/internal/couchtty/panel.go"}},
	"PanelModel.Filter":                     {"pure", "M3", "deleted", "cmd/internal/couchtty/panel.go", nil, []string{"cmd/internal/couchtty/panel.go"}},
	"Couch.ActionableThreadInventory":       {"integration", "M1/M3", "present", "cmd/internal/couchcore/actionableinventory.go", []string{"cmd/internal/couchcore/actionableinventory.go", "cmd/internal/couchcore/artifactcollision.go", "cmd/internal/couchcore/couch.go", "cmd/internal/couchcore/parktransaction.go", "cmd/internal/couchcore/pathops.go", "cmd/internal/couchcore/resume.go", "cmd/internal/couchcore/threadstore.go", "cmd/internal/launcher/agent_defaults.go"}, nil},
	"NativeBindingResolver":                 {"integration", "M3", "context-bearing exact parked-resume binding resolution present", "cmd/internal/couchcore/resume.go", []string{"cmd/internal/couchcore/resume.go"}, nil},
	"SessionInventoryNativeBindingResolver": {"integration", "M3", "context-bearing exact parked-resume binding resolution present", "cmd/internal/couchcore/resume.go", []string{"cmd/internal/couchcore/resume.go", "cmd/internal/couchcore/couch.go", "cmd/internal/sessioninventory/model.go", "cmd/internal/sessioninventory/query.go", "cmd/internal/sessioninventory/runtime.go"}, nil},
	"Couch.PrepareStart":                    {"integration", "M1", "present", "cmd/internal/couchcore/couch.go", []string{"cmd/internal/couchcore/couch.go", "cmd/internal/couchcore/startargs.go", "cmd/internal/couchcore/startgrant.go"}, nil},
	"Couch.SpawnPrepared":                   {"integration", "M1", "present", "cmd/internal/couchcore/couch.go", []string{"cmd/internal/couchcore/couch.go", "cmd/internal/couchcore/registry.go", "cmd/internal/couchcore/runner.go", "cmd/internal/couchcore/startargs.go", "cmd/internal/couchcore/startgrant.go", "cmd/internal/couchcore/worktree.go"}, nil},
	"StartGrantStore":                       {"integration", "M1", "present", "cmd/internal/couchcore/startgrant.go", []string{"cmd/internal/couchcore/startgrant.go", "cmd/internal/couchcore/clock.go"}, nil},
	"OperationCall":                         {"integration", "M1", "context dispatch present", "cmd/internal/couchcore/operationdispatch.go", []string{"cmd/internal/couchcore/operationdispatch.go", "cmd/internal/couchcore/ops.go"}, nil},
	"DispatchOperation":                     {"integration", "M1", "context dispatch present", "cmd/internal/couchcore/operationdispatch.go", []string{"cmd/internal/couchcore/operationdispatch.go", "cmd/internal/couchcore/ops.go"}, nil},
	"Couch.AbortStarted":                    {"integration", "M1", "exact started-actor abort present", "cmd/internal/couchcore/couch.go", []string{"cmd/internal/couchcore/couch.go", "cmd/internal/couchcore/naming.go", "cmd/internal/couchcore/ops.go", "cmd/internal/couchcore/registry.go", "cmd/internal/couchcore/runner.go", "cmd/internal/couchcore/store.go", "cmd/internal/couchcore/worktree.go"}, nil},
	"Console":                               {"integration", "M3", "hierarchical render, refresh, preview, action, and transactional attach controllers present", "cmd/internal/couchtty/console.go", []string{"cmd/internal/couchtty/console.go", "cmd/internal/couchcore/actorid.go", "cmd/internal/couchcore/operationdispatch.go", "cmd/internal/couchcore/ops.go", "cmd/internal/couchcore/park.go", "cmd/internal/couchcore/ptyrunner.go", "cmd/internal/couchcore/starttransaction.go", "cmd/internal/couchcore/thread.go", "cmd/internal/couchcore/worktree.go", "cmd/internal/couchtty/console_menu.go", "cmd/internal/couchtty/focus.go", "cmd/internal/couchtty/keys.go", "cmd/internal/couchtty/menu.go", "cmd/internal/couchtty/menu_async.go", "cmd/internal/couchtty/menu_refresh.go", "cmd/internal/couchtty/notice.go", "cmd/internal/couchtty/panelkeys.go", "cmd/internal/couchtty/reserve.go", "cmd/internal/hostty/host.go", "cmd/internal/ptychild/child.go", "cmd/internal/ptychild/screen.go"}, nil},
	"wireResolver":                          {"integration", "M3", "actionable refresh, shared action, attach-abort, and hierarchical render wiring present", "cmd/internal/couchcmd/run.go", []string{"cmd/internal/couchcmd/run.go", "cmd/internal/couchcore/actionableinventory.go", "cmd/internal/couchcore/couch.go", "cmd/internal/couchcore/operationdispatch.go", "cmd/internal/couchcore/ops.go", "cmd/internal/couchtty/console.go"}, nil},
	"Runner":                                {"integration", "M1", "context-bearing child lifecycle present", "cmd/internal/couchcore/runner.go", []string{"cmd/internal/couchcore/runner.go"}, nil},
	"FakeRunner":                            {"integration", "M1", "stateful context-bearing double present", "cmd/internal/couchcore/runner_fake.go", []string{"cmd/internal/couchcore/runner_fake.go", "cmd/internal/couchcore/runner.go", "cmd/internal/ptychild/child.go", "cmd/internal/ptychild/fake.go"}, nil},
	"hostty.FakeHost":                       {"integration", "M3", "observable terminal double present", "cmd/internal/hostty/fake.go", []string{"cmd/internal/hostty/fake.go", "cmd/internal/ptychild/child.go"}, nil},
	"TestMenuTargetPerformance":             {"integration", "M3", "present", "cmd/internal/couchtty/menu_perf_test.go", []string{"cmd/internal/couchtty/menu_perf_test.go", "cmd/internal/couchcore/actionableinventory.go", "cmd/internal/couchtty/focus.go", "cmd/internal/couchtty/menu.go", "cmd/internal/couchtty/menu_refresh.go", "cmd/internal/ptychild/child.go"}, nil},
}

// issue149M5GoSources is the exhaustive set of Go sources touched by M5. Every
// declaration in these files receives a disposition: a pair:m5-concept marker
// makes it architectural, and an absent marker explicitly means implementation
// detail. The plan inventory is derived only from the marked declarations.
var issue149M5GoSources = []string{
	"cmd/internal/adapt/adapt.go", "cmd/internal/adapt/adapt_test.go",
	"cmd/internal/agentcmd/restart.go",
	"cmd/internal/artifactpath/coverage_test.go", "cmd/internal/artifactpath/cross_scope_integration_test.go",
	"cmd/internal/artifactpath/manifest.go", "cmd/internal/artifactpath/paths.go", "cmd/internal/artifactpath/paths_test.go",
	"cmd/internal/changelogcmd/changelogcmd.go", "cmd/internal/changelogcmd/run_test.go",
	"cmd/internal/clipcmd/clipcmd.go", "cmd/internal/clipcmd/clipcmd_test.go", "cmd/internal/clipcmd/run.go",
	"cmd/internal/contextcmd/contextcmd.go", "cmd/internal/contextcmd/contextcmd_test.go", "cmd/internal/contextcmd/panejson_kdl_test.go",
	"cmd/internal/continuationcmd/continuationcmd.go",
	"cmd/internal/continuationcmd/draft.go",
	"cmd/internal/couchcmd/readme_test.go",
	"cmd/internal/couchcore/artifactcollision_test.go", "cmd/internal/couchcore/couch.go", "cmd/internal/couchcore/couch_test.go", "cmd/internal/couchcore/launchhelper_test.go", "cmd/internal/couchcore/migration.go",
	"cmd/internal/couchcore/migration_test.go", "cmd/internal/couchcore/plan_contract_test.go",
	"cmd/internal/couchcore/storejournal.go", "cmd/internal/couchcore/threadmetadata.go",
	"cmd/internal/couchcore/threadmetadata_test.go", "cmd/internal/couchcore/threadstore.go",
	"cmd/internal/draftroute/route.go",
	"cmd/internal/dispatcher/dispatcher.go", "cmd/internal/dispatcher/dispatcher_test.go",
	"cmd/internal/launcher/agent_defaults.go", "cmd/internal/launcher/agentargs.go", "cmd/internal/launcher/args.go", "cmd/internal/launcher/args_test.go",
	"cmd/internal/ctxmeter/ctxmeter.go", "cmd/internal/ctxmeter/ctxmeter_test.go",
	"cmd/internal/launcher/config.go", "cmd/internal/launcher/config_test.go", "cmd/internal/launcher/createflow.go",
	"cmd/internal/launcher/createflow_test.go", "cmd/internal/launcher/history.go", "cmd/internal/launcher/layoutflow.go",
	"cmd/internal/launcher/ledger.go", "cmd/internal/launcher/ledger_test.go",
	"cmd/internal/launcher/legacy_live.go", "cmd/internal/launcher/lifecycle.go", "cmd/internal/launcher/lifecycle_test.go",
	"cmd/internal/launcher/markers.go", "cmd/internal/launcher/markers_test.go",
	"cmd/internal/launcher/migrate.go", "cmd/internal/launcher/osruntime.go", "cmd/internal/launcher/osruntime_test.go",
	"cmd/internal/launcher/pick.go", "cmd/internal/launcher/pick_test.go", "cmd/internal/launcher/readiness.go",
	"cmd/internal/launcher/rename.go", "cmd/internal/launcher/rename_test.go",
	"cmd/internal/launcher/restart.go", "cmd/internal/launcher/restart_test.go", "cmd/internal/launcher/runcli.go",
	"cmd/internal/launcher/runtime.go", "cmd/internal/launcher/scoped_paths.go", "cmd/internal/launcher/session.go",
	"cmd/internal/launcher/session_index.go",
	"cmd/internal/launcher/thread_claim.go", "cmd/internal/launcher/thread_claim_test.go",
	"cmd/internal/opener/opener.go", "cmd/internal/opener/opener_test.go", "cmd/internal/opener/run.go",
	"cmd/internal/opener/run_test.go", "cmd/internal/opener/runcli.go", "cmd/internal/opener/runtime.go",
	"cmd/internal/reviewcmd/reviewcmd_test.go", "cmd/internal/reviewcmd/run.go", "cmd/internal/reviewcmd/run_test.go", "cmd/internal/reviewcmd/runcli.go", "cmd/internal/reviewcmd/runtime.go",
	"cmd/internal/pairlog/runcli.go", "cmd/internal/pairlog/runcli_test.go", "cmd/internal/pairlog/store.go", "cmd/internal/pairlog/store_test.go",
	"cmd/internal/runtimebundle/embed_test.go", "cmd/internal/runtimebundlegen/clean_source_test.go",
	"cmd/internal/scrollbackcmd/render_test.go", "cmd/internal/scrollbackcmd/scrollbackcmd.go",
	"cmd/internal/scrollbackcmd/scrollbackcmd_test.go", "cmd/internal/scrollbackcmd/timestamps_test.go",
	"cmd/internal/sessioninventory/activity.go", "cmd/internal/sessioninventory/activity_test.go", "cmd/internal/sessioninventory/activitycli.go", "cmd/internal/sessioninventory/activitycli_test.go",
	"cmd/internal/sessioninventory/conformance.go", "cmd/internal/sessioninventory/conformance_live_test.go", "cmd/internal/sessioninventory/conformance_test.go",
	"cmd/internal/sessioninventory/concept_contract_test.go",
	"cmd/internal/sessioninventory/binding.go", "cmd/internal/sessioninventory/binding_test.go",
	"cmd/internal/sessioninventory/diagnostic.go", "cmd/internal/sessioninventory/diagnostic_test.go",
	"cmd/internal/sessioninventory/event.go", "cmd/internal/sessioninventory/event_test.go",
	"cmd/internal/sessioninventory/events.go", "cmd/internal/sessioninventory/events_test.go",
	"cmd/internal/sessioninventory/forest_projection.go", "cmd/internal/sessioninventory/forest_projection_test.go",
	"cmd/internal/sessioninventory/inventory.go",
	"cmd/internal/sessioninventory/model.go", "cmd/internal/sessioninventory/model_test.go",
	"cmd/internal/sessioninventory/offline.go", "cmd/internal/sessioninventory/offline_test.go",
	"cmd/internal/sessioninventory/order.go", "cmd/internal/sessioninventory/order_test.go",
	"cmd/internal/sessioninventory/pair_inventory.go", "cmd/internal/sessioninventory/pair_inventory_test.go",
	"cmd/internal/sessioninventory/pairfacts.go", "cmd/internal/sessioninventory/pairfacts_test.go",
	"cmd/internal/sessioninventory/query.go", "cmd/internal/sessioninventory/query_test.go",
	"cmd/internal/sessioninventory/render.go", "cmd/internal/sessioninventory/render_test.go",
	"cmd/internal/sessioninventory/round.go", "cmd/internal/sessioninventory/round_test.go",
	"cmd/internal/sessioninventory/runcli.go", "cmd/internal/sessioninventory/runcli_failure_test.go", "cmd/internal/sessioninventory/runcli_test.go",
	"cmd/internal/sessioninventory/runtime.go", "cmd/internal/sessioninventory/runtime_os.go", "cmd/internal/sessioninventory/runtime_os_test.go",
	"cmd/internal/sessioninventory/scan.go", "cmd/internal/sessioninventory/scan_agy.go", "cmd/internal/sessioninventory/scan_agy_test.go",
	"cmd/internal/sessioninventory/scan_claude.go", "cmd/internal/sessioninventory/scan_claude_test.go",
	"cmd/internal/sessioninventory/scan_codex.go", "cmd/internal/sessioninventory/scan_codex_test.go",
	"cmd/internal/sessioninventory/scan_fuzz_test.go", "cmd/internal/sessioninventory/scan_helpers.go",
	"cmd/internal/sessioninventory/scan_muse.go", "cmd/internal/sessioninventory/scan_muse_test.go",
	"cmd/internal/sessioninventory/scan_test.go", "cmd/internal/sessioninventory/scanner_fixture_test.go",
	"cmd/internal/sessioninventory/shadow_test.go", "cmd/internal/sessioninventory/usage.go", "cmd/internal/sessioninventory/usage_test.go",
	"cmd/internal/sessioninventorytest/fake_runtime.go", "cmd/internal/sessioninventorytest/fake_runtime_test.go",
	"cmd/internal/sessionledger/record.go", "cmd/internal/sessionledger/record_test.go",
	"cmd/internal/sessionledger/store.go", "cmd/internal/sessionledger/store_subprocess_test.go", "cmd/internal/sessionledger/store_test.go", "cmd/internal/sessionledger/store_unix.go",
	"cmd/internal/sessionwatch/lifecycle.go", "cmd/internal/sessionwatch/lifecycle_test.go",
	"cmd/internal/sessionwatch/run.go", "cmd/internal/sessionwatch/run_test.go", "cmd/internal/sessionwatch/runcli.go", "cmd/internal/sessionwatch/runcli_test.go",
	"cmd/internal/sessionwatch/runtime.go", "cmd/internal/sessionwatch/sessionwatch.go", "cmd/internal/sessionwatch/sessionwatch_test.go",
	"cmd/internal/slugcmd/slug.go", "cmd/internal/slugcmd/slugcmd.go", "cmd/internal/slugcmd/slugcmd_test.go", "cmd/internal/titlepoller/run.go", "cmd/internal/titlepoller/run_test.go",
	"cmd/internal/strictjson/decode.go", "cmd/internal/threadrecord/record.go",
	"cmd/internal/titlepoller/runtime.go",
	"cmd/internal/workbenchshortcut/shortcut.go", "cmd/internal/wrapcmd/agent_restart_test.go", "cmd/internal/wrapcmd/wrap.go",
	"cmd/pair-go/changelog_seam_test.go", "cmd/pair-go/helper_equivalence_test.go", "cmd/pair-go/main.go", "cmd/pair-go/main_test.go",
}

// issue149M5DeletedGoSources records files in the milestone diff whose deletion
// is their complete declaration disposition.
var issue149M5DeletedGoSources = []string{
	"cmd/internal/codexsid/codexsid.go",
	"cmd/internal/codexsid/codexsid_test.go",
	"cmd/internal/transcript/transcript.go",
	"cmd/internal/transcript/transcript_test.go",
	"cmd/internal/launcher/thread_index.go",
	"cmd/internal/launcher/thread_index_conformance_test.go",
	"cmd/internal/launcher/thread_index_test.go",
}

// issue149M5RetiredGoSources records M5 sources introduced after the diff
// baseline and retired later. They have a historical declaration disposition,
// but their create-then-delete lifecycle is correctly absent from the net diff.
var issue149M5RetiredGoSources = []string{
	"cmd/internal/couchcore/standalone.go",
	"cmd/internal/couchcore/standalone_test.go",
}

// issue149M5RevertedGoSources records M5 sources whose later edits restored
// their baseline content. Their declarations still belong to the historical
// concept inventory, while the files are absent from the current net diff.
var issue149M5RevertedGoSources = []string{}

// issue149M5RetiredConceptRequirements preserves the historical M5 concept
// disposition for declarations removed after that milestone. The plan remains
// the M5 record of truth; retired concepts are not active production symbols.
var issue149M5RetiredConceptRequirements = []issue149ConceptRequirement{
	{name: "LaunchNativeWithStandaloneRegistrar", path: "cmd/internal/launcher/runcli.go", kind: "integration"},
	{name: "RegisterStandalonePair", path: "cmd/internal/couchcore/standalone.go", kind: "integration"},
	{name: "StandaloneThreadRegistrar", path: "cmd/internal/launcher/runtime.go", kind: "pure"},
	{name: "StandaloneThreadRegistration", path: "cmd/internal/launcher/runtime.go", kind: "pure"},
	{name: "ThreadStore.UpsertStandalonePair", path: "cmd/internal/couchcore/standalone.go", kind: "integration"},
}

func TestIssue149M5DeclarationDispositionSourceSetMatchesMilestoneDiff(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	if _, err := os.Stat(filepath.Join(root, ".git")); err != nil {
		t.Skip("source archive has no Git metadata; the checked-in disposition set remains the oracle")
	}
	catalog := map[string]bool{}
	for _, sources := range [][]string{issue149M5GoSources, issue149M5DeletedGoSources, issue149M5RetiredGoSources, issue149M5RevertedGoSources} {
		for _, rel := range sources {
			catalog[rel] = true
		}
	}
	for _, rel := range issue149M5BoundaryGoSources(t, root) {
		if !catalog[rel] {
			t.Errorf("M5 boundary source lacks a declaration disposition: %s", rel)
		}
	}
}

func TestIssue149M5DeclarationDispositionSetIsClosed(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	if got := issue149M5SourceDeclarationDigest(t, root); got != issue149M5DeclarationDigest {
		t.Fatalf("M5 declaration set changed without an explicit concept/detail disposition: got %s, want %s", got, issue149M5DeclarationDigest)
	}
}

func TestIssue149M5CoreConceptInventoryContract(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	raw, err := os.ReadFile(findPlanArtifact(t, root, "000149-couch-opaque-tags-and-a-human-naming-layer-plan.md"))
	if err != nil {
		t.Fatal(err)
	}
	requirements := issue149M5ConceptRequirements(t, root)
	for _, problem := range issue149M5ConceptProblems(string(raw), requirements) {
		t.Error(problem)
	}
}

func TestIssue149M5UnmarkedExportedAuthorityFailsClosed(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "migration.go", "package couchcore\ntype ReviewAddedAuthority struct{}\n", parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	requirements := issue149M5ConceptsForDecl(t, file.Name.Name, "cmd/internal/couchcore/migration.go", file.Decls[0], false)
	if len(requirements) != 1 || requirements[0].name != "ReviewAddedAuthority" || requirements[0].kind != "pure" {
		t.Fatalf("unmarked exported authority disposition = %+v, want one fail-closed pure concept", requirements)
	}
}

func TestIssue149M5CoreConceptInventoryRejectsRowMutation(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	raw, err := os.ReadFile(findPlanArtifact(t, root, "000149-couch-opaque-tags-and-a-human-naming-layer-plan.md"))
	if err != nil {
		t.Fatal(err)
	}
	requirements := issue149M5ConceptRequirements(t, root)
	plan := string(raw)
	for _, requirement := range requirements {
		row := ""
		needle := ""
		for _, line := range issue149ConceptRows(plan) {
			for _, candidate := range issue149RowConceptNames(line) {
				if candidate != requirement.name {
					continue
				}
				row = line
				short := strings.TrimPrefix(candidate, "artifactpath.")
				needle = "`" + short + "`"
				if strings.Contains(line, "`"+candidate+"`") {
					needle = "`" + candidate + "`"
				}
				break
			}
			if row != "" {
				break
			}
		}
		mutated := strings.Replace(plan, needle, "`removed`", 1)
		if problems := issue149M5ConceptProblems(mutated, requirements); len(problems) == 0 {
			t.Fatalf("entity deletion escaped derived contract: %s", requirement.name)
		}
		kindMutation := strings.Replace(plan, row, strings.Replace(row, "| pure", "| integration", 1), 1)
		if requirement.kind == "integration" {
			kindMutation = strings.Replace(plan, "| Integration | Lives in | Status | Wraps |", "| Entity | Kind | Lives in | Status |", 1)
		}
		for label, replacement := range map[string]string{
			"kind":   kindMutation,
			"path":   strings.Replace(plan, row, strings.Replace(row, "`"+requirement.path+"`", "`wrong/path.go`", 1), 1),
			"status": strings.Replace(plan, row, strings.Replace(row, "M5", "M4", 1), 1),
		} {
			if problems := issue149M5ConceptProblems(replacement, requirements); len(problems) == 0 {
				t.Fatalf("%s mutation escaped derived contract for %s", label, requirement.name)
			}
		}
	}
}

func issue149M5ConceptProblems(plan string, requirements []issue149ConceptRequirement) []string {
	var problems []string
	lines := issue149ConceptRows(plan)
	required := map[string]bool{}
	for _, requirement := range requirements {
		required[requirement.name] = true
		var matches []string
		for _, line := range lines {
			for _, name := range issue149RowConceptNames(line) {
				if name == requirement.name {
					matches = append(matches, line)
					break
				}
			}
		}
		if len(matches) != 1 {
			problems = append(problems, "M5 artifact concept must appear in exactly one row: "+requirement.name)
			continue
		}
		row := matches[0]
		if issue149ConceptRowKind(plan, row) != requirement.kind || !strings.Contains(row, "`"+requirement.path+"`") || !strings.Contains(row, "M5") {
			problems = append(problems, "M5 artifact concept has wrong kind/path/status: "+requirement.name)
		}
	}
	for _, name := range issue149M5PlanConceptNames(lines) {
		if !required[name] {
			problems = append(problems, "M5 plan concept has no source declaration disposition: "+name)
		}
	}
	return problems
}

func issue149ConceptRows(plan string) []string {
	var rows []string
	inTable := false
	for _, line := range strings.Split(plan, "\n") {
		if line == "| Entity | Kind | Lives in | Status |" || line == "| Integration | Lives in | Status | Wraps |" {
			inTable = true
			continue
		}
		if inTable && line == "" {
			inTable = false
			continue
		}
		if inTable && strings.HasPrefix(line, "| `") {
			rows = append(rows, line)
		}
	}
	return rows
}

func issue149ConceptRowKind(plan, row string) string {
	position := strings.Index(plan, row)
	if position < 0 {
		return ""
	}
	before := plan[:position]
	core := strings.LastIndex(before, "| Entity | Kind | Lives in | Status |")
	integration := strings.LastIndex(before, "| Integration | Lives in | Status | Wraps |")
	if integration > core {
		return "integration"
	}
	fields := strings.Split(row, "|")
	if len(fields) > 2 && strings.HasPrefix(strings.ToLower(strings.TrimSpace(fields[2])), "pure") {
		return "pure"
	}
	return ""
}

func issue149M5PlanConceptNames(lines []string) []string {
	var names []string
	for _, line := range lines {
		if !strings.Contains(line, "M5") {
			continue
		}
		names = append(names, issue149RowConceptNames(line)...)
	}
	sort.Strings(names)
	return names
}

func issue149RowConceptNames(line string) []string {
	fields := strings.Split(line, "|")
	if len(fields) < 3 {
		return nil
	}
	entity := fields[1]
	var names []string
	for {
		start := strings.Index(entity, "`")
		if start < 0 {
			break
		}
		entity = entity[start+1:]
		end := strings.Index(entity, "`")
		if end < 0 {
			break
		}
		names = append(names, entity[:end])
		entity = entity[end+1:]
	}
	prefix := ""
	if len(names) > 0 && strings.HasPrefix(names[0], "artifactpath.") {
		prefix = "artifactpath."
	}
	for i := 1; i < len(names); i++ {
		if prefix != "" && !strings.Contains(names[i], ".") {
			names[i] = prefix + names[i]
		}
	}
	return names
}

func issue149M5ConceptRequirements(t *testing.T, root string) []issue149ConceptRequirement {
	t.Helper()
	closedSet := issue149M5SourceDeclarationDigest(t, root) == issue149M5DeclarationDigest
	var requirements []issue149ConceptRequirement
	for _, rel := range issue149M5BoundaryGoSources(t, root) {
		raw, exists := issue149M5SourceAtHead(t, root, rel)
		if !exists {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), rel, raw, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range file.Decls {
			requirements = append(requirements, issue149M5ConceptsForDecl(t, file.Name.Name, rel, decl, closedSet)...)
		}
	}
	requirements = append(requirements, issue149M5RetiredConceptRequirements...)
	sort.Slice(requirements, func(i, j int) bool { return requirements[i].name < requirements[j].name })
	return requirements
}

func issue149M5ConceptsForDecl(t *testing.T, packageName, rel string, decl ast.Decl, closedSet bool) []issue149ConceptRequirement {
	t.Helper()
	marker := func(doc *ast.CommentGroup) string {
		if doc == nil {
			return ""
		}
		for _, line := range doc.List {
			text := strings.TrimSpace(strings.TrimPrefix(line.Text, "//"))
			if strings.HasPrefix(text, "pair:m5-concept ") {
				return strings.TrimSpace(strings.TrimPrefix(text, "pair:m5-concept "))
			}
		}
		return ""
	}
	qualified := func(name string) string {
		if packageName == "artifactpath" {
			return "artifactpath." + name
		}
		return name
	}
	switch typed := decl.(type) {
	case *ast.FuncDecl:
		kind := marker(typed.Doc)
		if kind == "" && !closedSet && typed.Recv == nil && ast.IsExported(typed.Name.Name) {
			kind = "pure"
		}
		if kind == "" {
			return nil
		}
		name := typed.Name.Name
		if typed.Recv != nil && len(typed.Recv.List) == 1 {
			receiver := typed.Recv.List[0].Type
			if pointer, ok := receiver.(*ast.StarExpr); ok {
				receiver = pointer.X
			}
			if ident, ok := receiver.(*ast.Ident); ok {
				name = ident.Name + "." + name
			}
		}
		return []issue149ConceptRequirement{{name: qualified(name), path: rel, kind: kind}}
	case *ast.GenDecl:
		var out []issue149ConceptRequirement
		for _, spec := range typed.Specs {
			switch item := spec.(type) {
			case *ast.TypeSpec:
				kind := marker(item.Doc)
				if kind == "" {
					kind = marker(typed.Doc)
				}
				if kind == "" && !closedSet && ast.IsExported(item.Name.Name) {
					kind = "pure"
				}
				if kind != "" {
					out = append(out, issue149ConceptRequirement{name: qualified(item.Name.Name), path: rel, kind: kind})
				}
			case *ast.ValueSpec:
				kind := marker(item.Doc)
				if kind == "" {
					kind = marker(typed.Doc)
				}
				for _, name := range item.Names {
					resolvedKind := kind
					if resolvedKind == "" && typed.Tok == token.VAR && !closedSet && ast.IsExported(name.Name) {
						resolvedKind = "pure"
					}
					if resolvedKind != "" {
						out = append(out, issue149ConceptRequirement{name: qualified(name.Name), path: rel, kind: resolvedKind})
					}
				}
			}
		}
		return out
	default:
		t.Fatalf("unclassified declaration kind %T in %s", decl, rel)
		return nil
	}
}

func issue149M5SourceDeclarationDigest(t *testing.T, root string) string {
	t.Helper()
	var keys []string
	retired := map[string]bool{}
	for _, rel := range issue149M5RetiredGoSources {
		retired[rel] = true
	}
	for _, rel := range issue149M5BoundaryGoSources(t, root) {
		raw, exists := issue149M5SourceAtHead(t, root, rel)
		if !exists {
			status := "deleted"
			if retired[rel] {
				status = "retired"
			}
			keys = append(keys, rel+"|"+status)
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), rel, raw, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range file.Decls {
			switch typed := decl.(type) {
			case *ast.FuncDecl:
				receiver := ""
				if typed.Recv != nil && len(typed.Recv.List) == 1 {
					receiver = issue149ReceiverName(typed.Recv.List[0].Type)
				}
				keys = append(keys, rel+"|func|"+receiver+"|"+typed.Name.Name)
			case *ast.GenDecl:
				for _, spec := range typed.Specs {
					switch item := spec.(type) {
					case *ast.TypeSpec:
						keys = append(keys, rel+"|"+typed.Tok.String()+"|"+item.Name.Name)
					case *ast.ValueSpec:
						for _, name := range item.Names {
							keys = append(keys, rel+"|"+typed.Tok.String()+"|"+name.Name)
						}
					}
				}
			}
		}
	}
	sort.Strings(keys)
	digest := sha256.Sum256([]byte(strings.Join(keys, "\n")))
	return fmt.Sprintf("%x", digest)
}

func issue149M5SourceAtHead(t *testing.T, root, rel string) ([]byte, bool) {
	t.Helper()
	command := exec.Command("git", "-C", root, "show", issue149M5Head+":"+rel)
	raw, err := command.Output()
	if err == nil {
		return raw, true
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return nil, false
	}
	t.Fatal(err)
	return nil, false
}

func issue149M5BoundaryGoSources(t *testing.T, root string) []string {
	t.Helper()
	if _, err := os.Stat(filepath.Join(root, ".git")); err != nil {
		t.Skip("M5 boundary source derivation requires Git metadata")
	}
	command := exec.Command("git", "-C", root, "diff", "--name-only", issue149M5Base+".."+issue149M5Head, "--", "*.go")
	raw, err := command.Output()
	if err != nil {
		t.Fatal(err)
	}
	sources := strings.Fields(string(raw))
	sort.Strings(sources)
	return sources
}

func issue149ReceiverName(expr ast.Expr) string {
	switch typed := expr.(type) {
	case *ast.Ident:
		return typed.Name
	case *ast.StarExpr:
		return "*" + issue149ReceiverName(typed.X)
	case *ast.IndexExpr:
		return issue149ReceiverName(typed.X) + "[]"
	case *ast.IndexListExpr:
		return issue149ReceiverName(typed.X) + "[]"
	default:
		return fmt.Sprintf("%T", expr)
	}
}

func TestIssue149CurrentCoreConceptKinds(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	path := findPlanArtifact(t, root, "000149-couch-opaque-tags-and-a-human-naming-layer-plan.md")
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	want := map[string]string{
		"CouchNamespace":      "integration",
		"PolicyResult":        "pure",
		"AdmissionDecision":   "pure",
		"ThreadAddress":       "pure",
		"StartTransaction":    "pure",
		"ThreadMetadataPatch": "pure",
		"ThreadSummary":       "pure",
		"Operation":           "pure",
		"threadrecord.Record": "pure",
		"strictjson.Decode":   "pure",
	}
	seen := map[string]bool{}
	inTable := false
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "| Entity | Kind | Lives in | Status |" {
			inTable = true
			continue
		}
		if inTable && line == "" {
			break
		}
		if !inTable {
			continue
		}
		if !strings.HasPrefix(line, "| `") || strings.Contains(line, "| Entity |") {
			continue
		}
		fields := strings.Split(line, "|")
		if len(fields) < 5 {
			continue
		}
		name, kind := strings.TrimSpace(fields[1]), strings.ToLower(strings.TrimSpace(fields[2]))
		for symbol, expected := range want {
			if strings.Contains(name, "`"+symbol+"`") {
				seen[symbol] = true
				if kind != expected {
					t.Errorf("%s kind = %q, want %q", symbol, kind, expected)
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	for symbol := range want {
		if !seen[symbol] {
			t.Errorf("core concept row missing current %s", symbol)
		}
	}
}

func findPlanArtifact(t *testing.T, root, name string) string {
	t.Helper()
	active := filepath.Join(root, "workshop", "plans", name)
	if _, err := os.Stat(active); err == nil {
		return active
	}
	var found string
	_ = filepath.WalkDir(filepath.Join(root, "workshop", "history"), func(path string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && d.Name() == name {
			found = path
		}
		return nil
	})
	if found == "" {
		t.Fatalf("find %s in active or archived plans", name)
	}
	return found
}

func TestOpaqueIdentityCommentDoesNotReintroducePathDerivedContract(t *testing.T) {
	raw, err := os.ReadFile("couch.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, obsolete := range []string{"derives from the TREE", "same tree always resumes"} {
		if strings.Contains(string(raw), obsolete) {
			t.Errorf("obsolete path-derived identity comment returned: %q", obsolete)
		}
	}
}

func TestIssue149PureCoreTestsStayAtPureBoundary(t *testing.T) {
	for _, name := range []string{
		"thread_test.go", "starttransaction_test.go", "admission_test.go",
		"threadmetadata_model_test.go", "ops_declarations_test.go",
		filepath.Join("..", "threadrecord", "record_test.go"),
		filepath.Join("..", "strictjson", "decode_test.go"),
	} {
		raw, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		for _, forbidden := range []string{"testCouchNamespace(", "t.TempDir(", "NewFake", "NewThreadStore(", "newTestThreadStore(", "os.", "exec."} {
			if strings.Contains(string(raw), forbidden) {
				t.Errorf("PURE direct test %s crosses integration boundary with %q", name, forbidden)
			}
		}
	}
}

func TestIssue149BlockedRunnersDelegateToOneHandshakeAuthority(t *testing.T) {
	want := map[string]string{
		"runner.go":    "return startBlockedChild(ctx, startExecChild, r.LaunchHelper, dir, argv, env, timeout)",
		"ptyrunner.go": "return startBlockedChild(ctx, r.start, r.LaunchHelper, dir, argv, env, timeout)",
	}
	for name, delegation := range want {
		raw, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		text := string(raw)
		if !strings.Contains(text, delegation) {
			t.Errorf("%s no longer delegates StartBlocked to shared authority", name)
		}
		for _, parallelAuthority := range []string{"os.Pipe(", "newAcknowledgedHandle("} {
			if strings.Contains(text, parallelAuthority) {
				t.Errorf("%s reintroduced blocked-start protocol %q", name, parallelAuthority)
			}
		}
	}
}

func TestIssue152DeliveredCoreConceptsResolveToGoDeclarations(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", ".."))
	planPath := findPlanArtifact(t, root, "000152-couch-verified-park-resume-plan.md")
	raw, err := os.ReadFile(planPath)
	if err != nil {
		t.Fatal(err)
	}
	const heading = "#### Delivered Core Concepts (authoritative)"
	section := strings.SplitN(string(raw), heading, 2)
	if len(section) != 2 {
		t.Fatalf("plan has no %q section", heading)
	}
	body := strings.SplitN(section[1], "\n#### ", 2)[0]
	resolved := 0
	for _, line := range strings.Split(body, "\n") {
		if !strings.HasPrefix(line, "| `") {
			continue
		}
		cells := strings.Split(line, "|")
		if len(cells) < 6 {
			t.Fatalf("malformed delivered concept row %q", line)
		}
		name := strings.Trim(strings.TrimSpace(cells[1]), "`")
		kind := strings.TrimSpace(cells[2])
		path := strings.Trim(strings.TrimSpace(cells[3]), "`")
		status := strings.TrimSpace(cells[4])
		if kind != "PURE" && kind != "INTEGRATION" {
			t.Errorf("%s has invalid kind %q", name, kind)
		}
		if status != "new" && status != "modified" && status != "deleted" {
			t.Errorf("%s has invalid status %q", name, status)
		}
		if err := requireGoDeclaration(filepath.Join(root, path), name); err != nil {
			t.Errorf("%s at %s: %v", name, path, err)
		}
		resolved++
	}
	if resolved != 21 {
		t.Fatalf("resolved %d delivered concepts, want 21", resolved)
	}
}

func TestIssue151CoreConceptsMatchCurrentBoundary(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", ".."))
	planPath := findPlanArtifact(t, root, "000151-hierarchical-thread-menu-plan.md")
	raw, err := os.ReadFile(planPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateIssue151CurrentConcepts(root, string(raw)); err != nil {
		t.Fatal(err)
	}

	mutated := strings.Replace(string(raw),
		"| `RefreshSchedule` | `cmd/internal/couchtty/menu_refresh.go` | new | M3 | present |",
		"| `RefreshSchedule` | `cmd/internal/couchtty/menu_refresh.go` | new | M3 | absent |", 1)
	if mutated == string(raw) {
		t.Fatal("failed to construct current-as-absent mutation")
	}
	if err := validateIssue151CurrentConcepts(root, mutated); err == nil {
		t.Fatal("landed M3 surface presented as absent passed the current-boundary contract")
	}

	missingDependency := strings.Replace(string(raw), ", `cmd/internal/sessioninventory/query.go`", "", 1)
	if missingDependency == string(raw) {
		t.Fatal("failed to construct omitted-dependency-path mutation")
	}
	if err := validateIssue151CurrentConcepts(root, missingDependency); err == nil {
		t.Fatal("Core concepts row with an omitted dependency path passed the boundary contract")
	}
}

func TestIssue151M3ChecklistMatchesCurrentBoundary(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", ".."))
	planPath := findPlanArtifact(t, root, "000151-hierarchical-thread-menu-plan.md")
	raw, err := os.ReadFile(planPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateIssue151M3Checklist(string(raw)); err != nil {
		t.Fatal(err)
	}

	mutated := strings.Replace(string(raw),
		"- [x] **Step 1: Write failing pure refresh-schedule traces**",
		"- [ ] **Step 1: Write failing pure refresh-schedule traces**", 1)
	if mutated == string(raw) {
		t.Fatal("failed to construct delivered-as-unchecked mutation")
	}
	if err := validateIssue151M3Checklist(mutated); err == nil {
		t.Fatal("delivered M3 step presented as unchecked passed the boundary contract")
	}
}

func TestIssue151M3SourceConceptsAppearAtExactPlanPaths(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", ".."))
	planPath := findPlanArtifact(t, root, "000151-hierarchical-thread-menu-plan.md")
	raw, err := os.ReadFile(planPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateIssue151M3SourceConcepts(root, string(raw)); err != nil {
		t.Fatal(err)
	}

	mutated := strings.Replace(string(raw), "| `ParkedResumeObservation` | `cmd/internal/couchcore/actionableinventory.go` | new | M3 | present |\n", "", 1)
	if mutated == string(raw) {
		t.Fatal("failed to construct omitted-source-concept mutation")
	}
	if err := validateIssue151M3SourceConcepts(root, mutated); err == nil {
		t.Fatal("omitted M3 source concept passed the architectural inventory contract")
	}
}

func TestIssue151M3DeclarationDispositionSourceSetMatchesMilestoneDiff(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", ".."))
	if _, err := os.Stat(filepath.Join(root, ".git")); err != nil {
		t.Skip("source archive has no Git metadata; the checked-in disposition set remains the oracle")
	}
	want := append(append([]string(nil), issue151M3GoSources...), issue151M3DeletedGoSources...)
	sort.Strings(want)
	command := exec.Command("git", "-C", root, "diff", "--name-only", issue151M3Base, issue151M3Head, "--", "*.go")
	raw, err := command.Output()
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Fields(string(raw))
	sort.Strings(got)
	if !equalStrings(got, want) {
		t.Fatalf("M3 declaration disposition sources = %v, want exact milestone diff %v", got, want)
	}
}

func TestIssue151M3DeclarationDispositionSetIsClosed(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", ".."))
	got, err := issue151M3SourceDeclarationDigest(root, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != issue151M3DeclarationDigest {
		t.Fatalf("M3 declaration set changed without an explicit architectural/detail/retired disposition: got %s, want %s", got, issue151M3DeclarationDigest)
	}
}

func TestIssue151M3ArchitecturalLedgerIsClosed(t *testing.T) {
	if got := issue151M3ArchitecturalLedgerFingerprint("", ""); got != issue151M3ArchitecturalLedgerDigest {
		t.Fatalf("M3 architectural ledger changed without an explicit entity/dependency disposition: got %s, want %s", got, issue151M3ArchitecturalLedgerDigest)
	}
	if got := issue151M3ArchitecturalLedgerFingerprint("Couch.ActionableThreadInventory", "cmd/internal/couchcore/threadstore.go"); got == issue151M3ArchitecturalLedgerDigest {
		t.Fatal("removing an enumerated dependency left the architectural ledger unchanged")
	}
}

func TestIssue151M3IntegrationDependenciesMatchPinnedSource(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", ".."))
	index, err := buildIssue151DeclarationIndex(root)
	if err != nil {
		t.Fatal(err)
	}
	for name, declaration := range issue151M3ArchitecturalDeclarations {
		if declaration.kind != "integration" {
			continue
		}
		got, err := index.dependencies(root, name, declaration.source, nil)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		want := make([]string, 0, len(declaration.present))
		for _, path := range declaration.present {
			if path != declaration.source {
				want = append(want, path)
			}
		}
		sort.Strings(want)
		if !equalStrings(got, want) {
			t.Errorf("%s pinned source dependencies = %v, ledger = %v", name, got, want)
		}
	}
}

func TestIssue151M3ImplementationDependencyRemovalFails(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", ".."))
	index, err := buildIssue151DeclarationIndex(root)
	if err != nil {
		t.Fatal(err)
	}
	path := "cmd/internal/couchcore/actionableinventory.go"
	raw, err := issue151M3SourceAtHead(root, path)
	if err != nil {
		t.Fatal(err)
	}
	removed := "\t\t\tif _, pathErr := c.Path.Physical(record.WorkingPath); pathErr != nil {\n\t\t\t\tcontinue\n\t\t\t}\n"
	mutated := strings.Replace(string(raw), removed, "", 1)
	mutated = strings.Replace(mutated, " || c.Path == nil", "", 1)
	if mutated == string(raw) {
		t.Fatal("failed to remove pinned PathOps dependency")
	}
	got, err := index.dependencies(root, "Couch.ActionableThreadInventory", path, map[string][]byte{path: []byte(mutated)})
	if err != nil {
		t.Fatal(err)
	}
	if containsString(got, "cmd/internal/couchcore/pathops.go") {
		t.Fatal("removed PathOps reference remained in mechanically derived dependencies")
	}
	declaration := issue151M3ArchitecturalDeclarations["Couch.ActionableThreadInventory"]
	var want []string
	for _, dependency := range declaration.present {
		if dependency != declaration.source {
			want = append(want, dependency)
		}
	}
	sort.Strings(want)
	if equalStrings(got, want) {
		t.Fatal("implementation dependency removal still matched the closed ledger")
	}
}

type issue151DeclarationIndex struct {
	files        map[string]*ast.File
	imports      map[string]map[string]string
	declarations map[string]map[string]string
	methods      map[string]map[string]string
	fields       map[string]map[string]map[string]string
}

func buildIssue151DeclarationIndex(root string) (*issue151DeclarationIndex, error) {
	index := &issue151DeclarationIndex{
		files: make(map[string]*ast.File), imports: make(map[string]map[string]string),
		declarations: make(map[string]map[string]string), methods: make(map[string]map[string]string),
		fields: make(map[string]map[string]map[string]string),
	}
	err := filepath.WalkDir(filepath.Join(root, "cmd", "internal"), func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return walkErr
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		dir := filepath.ToSlash(filepath.Dir(rel))
		index.files[rel] = file
		index.imports[rel] = make(map[string]string)
		for _, spec := range file.Imports {
			importPath := strings.Trim(spec.Path.Value, "\"")
			if !strings.HasPrefix(importPath, "github.com/xianxu/pair/") {
				continue
			}
			alias := filepath.Base(importPath)
			if spec.Name != nil {
				alias = spec.Name.Name
			}
			index.imports[rel][alias] = strings.TrimPrefix(importPath, "github.com/xianxu/pair/")
		}
		if index.declarations[dir] == nil {
			index.declarations[dir] = make(map[string]string)
			index.methods[dir] = make(map[string]string)
			index.fields[dir] = make(map[string]map[string]string)
		}
		for _, decl := range file.Decls {
			switch typed := decl.(type) {
			case *ast.FuncDecl:
				if typed.Recv == nil {
					if ast.IsExported(typed.Name.Name) || issue151M3ArchitecturalDeclarations[typed.Name.Name].source != "" {
						index.declarations[dir][typed.Name.Name] = rel
					}
					continue
				}
				receiver := strings.TrimPrefix(issue149ReceiverName(typed.Recv.List[0].Type), "*")
				if ast.IsExported(typed.Name.Name) || issue151M3ArchitecturalDeclarations[receiver+"."+typed.Name.Name].source != "" {
					index.methods[dir][receiver+"."+typed.Name.Name] = rel
				}
			case *ast.GenDecl:
				for _, spec := range typed.Specs {
					typeSpec, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}
					if ast.IsExported(typeSpec.Name.Name) || issue151M3ArchitecturalDeclarations[typeSpec.Name.Name].source != "" || issue151M3ArchitecturalDeclarations[file.Name.Name+"."+typeSpec.Name.Name].source != "" {
						index.declarations[dir][typeSpec.Name.Name] = rel
					}
					structType, ok := typeSpec.Type.(*ast.StructType)
					if !ok {
						continue
					}
					index.fields[dir][typeSpec.Name.Name] = make(map[string]string)
					for _, field := range structType.Fields.List {
						ref := issue151TypeReference(field.Type)
						for _, name := range field.Names {
							index.fields[dir][typeSpec.Name.Name][name.Name] = ref
						}
					}
				}
			}
		}
		return nil
	})
	return index, err
}

func issue151TypeReference(expr ast.Expr) string {
	switch typed := expr.(type) {
	case *ast.Ident:
		return typed.Name
	case *ast.StarExpr:
		return issue151TypeReference(typed.X)
	case *ast.IndexExpr:
		return issue151TypeReference(typed.X)
	case *ast.IndexListExpr:
		return issue151TypeReference(typed.X)
	case *ast.SelectorExpr:
		if pkg, ok := typed.X.(*ast.Ident); ok {
			return pkg.Name + "." + typed.Sel.Name
		}
	}
	return ""
}

func (index *issue151DeclarationIndex) dependencies(root, entity, source string, override map[string][]byte) ([]string, error) {
	raw, ok := override[source]
	if !ok {
		var err error
		raw, err = issue151M3SourceAtHead(root, source)
		if err != nil {
			return nil, err
		}
	}
	file, err := parser.ParseFile(token.NewFileSet(), source, raw, 0)
	if err != nil {
		return nil, err
	}
	dir := filepath.ToSlash(filepath.Dir(source))
	short := strings.TrimPrefix(entity, file.Name.Name+".")
	parts := strings.Split(short, ".")
	var roots []ast.Node
	for _, decl := range file.Decls {
		switch typed := decl.(type) {
		case *ast.FuncDecl:
			if len(parts) == 2 && typed.Recv != nil && strings.TrimPrefix(issue149ReceiverName(typed.Recv.List[0].Type), "*") == parts[0] && strings.HasPrefix(typed.Name.Name, parts[1]) {
				roots = append(roots, typed)
			} else if len(parts) == 1 && typed.Recv == nil && typed.Name.Name == parts[0] {
				roots = append(roots, typed)
			} else if len(parts) == 1 && typed.Recv != nil && strings.TrimPrefix(issue149ReceiverName(typed.Recv.List[0].Type), "*") == parts[0] {
				roots = append(roots, typed)
			}
		case *ast.GenDecl:
			for _, spec := range typed.Specs {
				if typeSpec, ok := spec.(*ast.TypeSpec); ok && len(parts) == 1 && typeSpec.Name.Name == parts[0] {
					roots = append(roots, typeSpec)
				}
			}
		}
	}
	if len(roots) == 0 {
		return nil, fmt.Errorf("cannot resolve pinned declaration")
	}
	dependencies := make(map[string]bool)
	addRef := func(ref string) {
		refParts := strings.Split(ref, ".")
		refDir := dir
		name := refParts[len(refParts)-1]
		if len(refParts) == 2 {
			if imported := index.imports[source][refParts[0]]; imported != "" {
				refDir = imported
			}
		}
		if path := index.declarations[refDir][name]; path != "" && path != source {
			dependencies[path] = true
		}
	}
	for _, root := range roots {
		variables := make(map[string]string)
		if function, ok := root.(*ast.FuncDecl); ok {
			if function.Recv != nil {
				for _, field := range function.Recv.List {
					for _, name := range field.Names {
						variables[name.Name] = issue151TypeReference(field.Type)
					}
				}
			}
			if function.Type.Params != nil {
				for _, field := range function.Type.Params.List {
					for _, name := range field.Names {
						variables[name.Name] = issue151TypeReference(field.Type)
					}
				}
			}
		}
		ast.Inspect(root, func(node ast.Node) bool {
			switch typed := node.(type) {
			case *ast.Ident:
				addRef(typed.Name)
			case *ast.SelectorExpr:
				if base, ok := typed.X.(*ast.Ident); ok {
					addRef(base.Name + "." + typed.Sel.Name)
					if variableType := variables[base.Name]; variableType != "" {
						typeParts := strings.Split(variableType, ".")
						typeDir := dir
						typeName := typeParts[len(typeParts)-1]
						if len(typeParts) == 2 && index.imports[source][typeParts[0]] != "" {
							typeDir = index.imports[source][typeParts[0]]
						}
						if ref := index.fields[typeDir][typeName][typed.Sel.Name]; ref != "" {
							if typeDir != dir && !strings.Contains(ref, ".") {
								ref = filepath.Base(typeDir) + "." + ref
							}
							addRef(ref)
						}
					}
				}
			}
			return true
		})
	}
	out := make([]string, 0, len(dependencies))
	for path := range dependencies {
		out = append(out, path)
	}
	sort.Strings(out)
	return out, nil
}

func issue151M3ArchitecturalLedgerFingerprint(removeEntity, removePath string) string {
	keys := make([]string, 0, len(issue151M3ArchitecturalDeclarations))
	for name, declaration := range issue151M3ArchitecturalDeclarations {
		paths := append([]string(nil), declaration.present...)
		if name == removeEntity {
			filtered := paths[:0]
			for _, path := range paths {
				if path != removePath {
					filtered = append(filtered, path)
				}
			}
			paths = filtered
		}
		sort.Strings(paths)
		absent := append([]string(nil), declaration.absent...)
		sort.Strings(absent)
		keys = append(keys, strings.Join([]string{name, declaration.kind, declaration.delivery, declaration.current, declaration.source, strings.Join(paths, ","), strings.Join(absent, ",")}, "|"))
	}
	sort.Strings(keys)
	digest := sha256.Sum256([]byte(strings.Join(keys, "\n")))
	return fmt.Sprintf("%x", digest)
}

func TestIssue151M3HistoricalOracleRejectsMovingHeadAndWorktreeBytes(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", ".."))
	path := "cmd/internal/couchcore/plan_contract_test.go"
	if _, err := issue151M3SourceAtRef(root, "HEAD", path); err == nil {
		t.Fatal("moving HEAD was accepted as the immutable M3 source ref")
	}
	worktree, err := os.ReadFile(filepath.Join(root, path))
	if err != nil {
		t.Fatal(err)
	}
	got, digestErr := issue151M3SourceDeclarationDigest(root, map[string][]byte{path: worktree}, nil)
	if digestErr == nil && got == issue151M3DeclarationDigest {
		t.Fatal("mutable worktree bytes reproduced the immutable M3 declaration snapshot")
	}
}

func TestIssue151M3DeclarationDispositionMutationsFailClosed(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", ".."))
	path := "cmd/internal/couchcore/actionableinventory.go"
	raw, err := issue151M3SourceAtHead(root, path)
	if err != nil {
		t.Fatal(err)
	}
	mutated := append(append([]byte(nil), raw...), []byte("\ntype ReviewAddedM3Authority struct{}\n")...)
	if got, digestErr := issue151M3SourceDeclarationDigest(root, map[string][]byte{path: mutated}, nil); digestErr == nil && got == issue151M3DeclarationDigest {
		t.Fatal("unlisted declaration left the closed disposition snapshot unchanged")
	}
	for name, disposition := range map[string]map[string]string{
		"architectural to detail": {"ParkedResumeObservation": "detail"},
		"detail to architectural": {"ActionableThreadState": "architectural"},
	} {
		t.Run(name, func(t *testing.T) {
			got, digestErr := issue151M3SourceDeclarationDigest(root, nil, disposition)
			if digestErr == nil && got == issue151M3DeclarationDigest {
				t.Fatal("classification mutation left the closed declaration disposition unchanged")
			}
		})
	}
}

func issue151M3SourceDeclarationDigest(root string, sourceOverride map[string][]byte, dispositionOverride map[string]string) (string, error) {
	keys := make([]string, 0, len(issue151M3GoSources)+len(issue151M3DeletedGoSources))
	foundArchitectural := make(map[string]string, len(issue151M3ArchitecturalDeclarations))
	for _, rel := range issue151M3GoSources {
		raw, ok := sourceOverride[rel]
		if !ok {
			var err error
			raw, err = issue151M3SourceAtHead(root, rel)
			if err != nil {
				return "", fmt.Errorf("read M3 disposition source %s: %w", rel, err)
			}
		}
		file, err := parser.ParseFile(token.NewFileSet(), rel, raw, 0)
		if err != nil {
			return "", fmt.Errorf("parse M3 disposition source %s: %w", rel, err)
		}
		for _, decl := range file.Decls {
			switch typed := decl.(type) {
			case *ast.FuncDecl:
				receiver := ""
				if typed.Recv != nil && len(typed.Recv.List) == 1 {
					receiver = issue149ReceiverName(typed.Recv.List[0].Type)
				}
				name := typed.Name.Name
				if receiver != "" {
					name = strings.TrimPrefix(receiver, "*") + "." + name
				} else if _, ok := issue151M3ArchitecturalDeclarations[file.Name.Name+"."+name]; ok {
					name = file.Name.Name + "." + name
				}
				disposition := issue151M3Disposition(name, rel, dispositionOverride)
				if disposition == "architectural" {
					foundArchitectural[name] = rel
				}
				keys = append(keys, rel+"|func|"+receiver+"|"+typed.Name.Name+"|"+disposition)
			case *ast.GenDecl:
				for _, spec := range typed.Specs {
					switch item := spec.(type) {
					case *ast.TypeSpec:
						name := item.Name.Name
						if _, ok := issue151M3ArchitecturalDeclarations[file.Name.Name+"."+name]; ok {
							name = file.Name.Name + "." + name
						}
						disposition := issue151M3Disposition(name, rel, dispositionOverride)
						if disposition == "architectural" {
							foundArchitectural[name] = rel
						}
						keys = append(keys, rel+"|"+typed.Tok.String()+"|"+item.Name.Name+"|"+disposition)
					case *ast.ValueSpec:
						for _, name := range item.Names {
							qualified := name.Name
							if _, ok := issue151M3ArchitecturalDeclarations[file.Name.Name+"."+qualified]; ok {
								qualified = file.Name.Name + "." + qualified
							}
							disposition := issue151M3Disposition(qualified, rel, dispositionOverride)
							if disposition == "architectural" {
								foundArchitectural[qualified] = rel
							}
							keys = append(keys, rel+"|"+typed.Tok.String()+"|"+name.Name+"|"+disposition)
						}
					}
				}
			default:
				return "", fmt.Errorf("unclassified declaration kind %T in %s", decl, rel)
			}
		}
	}
	for _, rel := range issue151M3DeletedGoSources {
		if _, err := issue151M3SourceAtHead(root, rel); err == nil {
			return "", fmt.Errorf("retired M3 source %s exists at pinned head", rel)
		} else if !isGitObjectMissing(err) {
			return "", fmt.Errorf("inspect retired M3 source %s: %w", rel, err)
		}
		keys = append(keys, rel+"|retired")
	}
	for name, declaration := range issue151M3ArchitecturalDeclarations {
		if !containsString(issue151M3GoSources, declaration.source) {
			continue
		}
		if foundArchitectural[name] != declaration.source {
			return "", fmt.Errorf("M3 architectural declaration %s classified at %q, want %q", name, foundArchitectural[name], declaration.source)
		}
	}
	for name, rel := range foundArchitectural {
		declaration, ok := issue151M3ArchitecturalDeclarations[name]
		if !ok || declaration.source != rel {
			return "", fmt.Errorf("unexpected M3 architectural declaration %s at %s", name, rel)
		}
	}
	sort.Strings(keys)
	digest := sha256.Sum256([]byte(strings.Join(keys, "\n")))
	return fmt.Sprintf("%x", digest), nil
}

func issue151M3Disposition(name, rel string, override map[string]string) string {
	if disposition := override[name]; disposition != "" {
		return disposition
	}
	if declaration, ok := issue151M3ArchitecturalDeclarations[name]; ok && declaration.source == rel {
		return "architectural"
	}
	return "detail"
}

func issue151M3SourceAtHead(root, rel string) ([]byte, error) {
	return issue151M3SourceAtRef(root, issue151M3Head, rel)
}

func issue151M3SourceAtRef(root, ref, rel string) ([]byte, error) {
	resolved, err := exec.Command("git", "-C", root, "rev-parse", ref+"^{commit}").Output()
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(string(resolved)) != issue151M3Head {
		return nil, fmt.Errorf("M3 source ref %q resolves to %s, want immutable %s", ref, strings.TrimSpace(string(resolved)), issue151M3Head)
	}
	command := exec.Command("git", "-C", root, "show", issue151M3Head+":"+rel)
	raw, err := command.Output()
	if err != nil {
		return nil, err
	}
	return raw, nil
}

func isGitObjectMissing(err error) bool {
	var exitErr *exec.ExitError
	return errors.As(err, &exitErr) && exitErr.ExitCode() != 0
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func validateIssue151M3SourceConcepts(root, plan string) error {
	coreParts := strings.SplitN(plan, "## Core concepts", 2)
	if len(coreParts) != 2 {
		return errors.New("plan has no Core concepts section")
	}
	coreParts = strings.SplitN(coreParts[1], "## Function-level test strategies", 2)
	if len(coreParts) != 2 {
		return errors.New("Core concepts section has no boundary")
	}
	core := coreParts[0]
	if strings.Contains(core, "existing test seams") {
		return errors.New("Core concepts contains a non-resolvable source location")
	}
	if _, err := issue151M3SourceDeclarationDigest(root, nil, nil); err != nil {
		return err
	}
	for name, declaration := range issue151M3ArchitecturalDeclarations {
		if len(declaration.absent) == 0 {
			found, err := issue151M3PinnedDeclarationExists(root, name, declaration.source)
			if err != nil {
				return err
			}
			if !found {
				return fmt.Errorf("M3 architectural declaration %s is absent from pinned source %s", name, declaration.source)
			}
		}
		rowFound := false
		for _, planLine := range strings.Split(core, "\n") {
			if strings.HasPrefix(planLine, "| `"+name+"` |") && strings.Contains(planLine, "`"+declaration.source+"`") {
				rowFound = true
				break
			}
		}
		if !rowFound {
			return fmt.Errorf("M3 architectural declaration %s at %s is absent from an exact Core concepts row", name, declaration.source)
		}
	}
	return nil
}

func issue151M3PinnedDeclarationExists(root, want, rel string) (bool, error) {
	raw, err := issue151M3SourceAtHead(root, rel)
	if err != nil {
		return false, fmt.Errorf("read pinned architectural source %s: %w", rel, err)
	}
	file, err := parser.ParseFile(token.NewFileSet(), rel, raw, 0)
	if err != nil {
		return false, fmt.Errorf("parse pinned architectural source %s: %w", rel, err)
	}
	for _, decl := range file.Decls {
		switch typed := decl.(type) {
		case *ast.FuncDecl:
			name := typed.Name.Name
			if typed.Recv != nil && len(typed.Recv.List) == 1 {
				name = strings.TrimPrefix(issue149ReceiverName(typed.Recv.List[0].Type), "*") + "." + name
			} else if file.Name.Name+"."+name == want {
				name = want
			}
			if name == want {
				return true, nil
			}
		case *ast.GenDecl:
			for _, spec := range typed.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				name := typeSpec.Name.Name
				if file.Name.Name+"."+name == want {
					name = want
				}
				if name == want {
					return true, nil
				}
			}
		}
	}
	return false, nil
}

func validateIssue151M3Checklist(plan string) error {
	wantSteps := map[int]int{10: 9, 11: 33, 12: 6, 13: 9}
	for task, count := range wantSteps {
		startMarker := fmt.Sprintf("### Task %d:", task)
		start := strings.Index(plan, startMarker)
		if start < 0 {
			return fmt.Errorf("missing M3 Task %d", task)
		}
		section := plan[start+len(startMarker):]
		if next := strings.Index(section, "\n### Task "); next >= 0 {
			section = section[:next]
		} else if next := strings.Index(section, "\n## Revisions"); next >= 0 {
			section = section[:next]
		}
		steps := make([]bool, 0, count)
		for _, line := range strings.Split(section, "\n") {
			if strings.HasPrefix(line, "- [x] **Step ") {
				steps = append(steps, true)
			} else if strings.HasPrefix(line, "- [ ] **Step ") {
				steps = append(steps, false)
			}
		}
		if len(steps) != count {
			return fmt.Errorf("M3 Task %d has %d checklist steps, want %d", task, len(steps), count)
		}
		for index, checked := range steps {
			wantChecked := task != 13 || index < 7
			if checked != wantChecked {
				return fmt.Errorf("M3 Task %d Step %d checked=%t, want %t", task, index+1, checked, wantChecked)
			}
		}
	}
	return nil
}

func validateIssue151CurrentConcepts(root, plan string) error {
	core := strings.SplitN(plan, "## Core concepts", 2)
	if len(core) != 2 {
		return errors.New("plan has no Core concepts section")
	}
	core = strings.SplitN(core[1], "## Function-level test strategies", 2)
	if len(core) != 2 {
		return errors.New("Core concepts section has no boundary")
	}
	if strings.Count(core[0], "Current at M3 boundary") != 2 || strings.Contains(core[0], "Current after M3 Task") {
		return errors.New("Core concepts current-state columns do not name the M3 boundary")
	}
	seen := make(map[string]bool, len(issue151M3ArchitecturalDeclarations))
	kind := "pure"
	for _, line := range strings.Split(core[0], "\n") {
		if line == "### Integration points" {
			kind = "integration"
			continue
		}
		if !strings.HasPrefix(line, "| ") {
			continue
		}
		cells := strings.Split(line, "|")
		if len(cells) < 7 {
			continue
		}
		nameCell := strings.TrimSpace(cells[1])
		if nameCell == "Name" || strings.HasPrefix(nameCell, "---") {
			continue
		}
		names := backtickedPlanValues(nameCell)
		if len(names) != 1 || nameCell != "`"+names[0]+"`" {
			return fmt.Errorf("Core concepts row must name exactly one architectural entity: %q", nameCell)
		}
		name := names[0]
		contract, ok := issue151M3ArchitecturalDeclarations[name]
		if !ok {
			return fmt.Errorf("unclassified Core concepts row %q", name)
		}
		if seen[name] {
			return fmt.Errorf("duplicate Core concepts row %q", name)
		}
		seen[name] = true
		if kind != contract.kind {
			return fmt.Errorf("%s kind = %q, want %q", name, kind, contract.kind)
		}
		delivery := strings.TrimSpace(cells[4])
		current := strings.TrimSpace(cells[5])
		if delivery != contract.delivery || current != contract.current {
			return fmt.Errorf("%s boundary = delivery %q current %q, want %q / %q", name, delivery, current, contract.delivery, contract.current)
		}
		wantPaths := append(append([]string(nil), contract.present...), contract.absent...)
		gotPaths := backtickedPlanValues(strings.TrimSpace(cells[2]))
		sort.Strings(wantPaths)
		sort.Strings(gotPaths)
		if !equalStrings(gotPaths, wantPaths) {
			return fmt.Errorf("%s paths = %v, want exact repo-relative paths %v", name, gotPaths, wantPaths)
		}
		for _, path := range contract.present {
			if _, err := os.Stat(filepath.Join(root, path)); err != nil {
				return fmt.Errorf("%s says present but %s is unavailable: %w", name, path, err)
			}
		}
		for _, path := range contract.absent {
			if _, err := os.Stat(filepath.Join(root, path)); !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("%s says absent but %s exists or cannot be classified: %v", name, path, err)
			}
		}
	}
	if len(seen) != len(issue151M3ArchitecturalDeclarations) {
		return fmt.Errorf("classified %d Core concepts rows, want %d", len(seen), len(issue151M3ArchitecturalDeclarations))
	}
	return nil
}

func backtickedPlanValues(cell string) []string {
	var values []string
	for {
		start := strings.IndexByte(cell, '`')
		if start < 0 {
			return values
		}
		cell = cell[start+1:]
		end := strings.IndexByte(cell, '`')
		if end < 0 {
			return values
		}
		values = append(values, cell[:end])
		cell = cell[end+1:]
	}
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func requireGoDeclaration(path, qualifiedName string) error {
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		return err
	}
	receiver, name, isMethod := strings.Cut(qualifiedName, ".")
	if !isMethod {
		name = receiver
	}
	for _, declaration := range file.Decls {
		switch value := declaration.(type) {
		case *ast.GenDecl:
			if isMethod {
				continue
			}
			for _, spec := range value.Specs {
				if typeSpec, ok := spec.(*ast.TypeSpec); ok && typeSpec.Name.Name == name {
					return nil
				}
				if valueSpec, ok := spec.(*ast.ValueSpec); ok {
					for _, declared := range valueSpec.Names {
						if declared.Name == name {
							return nil
						}
					}
				}
			}
		case *ast.FuncDecl:
			if value.Name.Name != name {
				continue
			}
			if !isMethod && value.Recv == nil {
				return nil
			}
			if isMethod && receiverName(value.Recv) == receiver {
				return nil
			}
		}
	}
	return fmt.Errorf("Go declaration not found")
}

func receiverName(fields *ast.FieldList) string {
	if fields == nil || len(fields.List) != 1 {
		return ""
	}
	expression := fields.List[0].Type
	if pointer, ok := expression.(*ast.StarExpr); ok {
		expression = pointer.X
	}
	if identifier, ok := expression.(*ast.Ident); ok {
		return identifier.Name
	}
	return ""
}
