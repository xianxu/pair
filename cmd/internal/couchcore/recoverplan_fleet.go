package couchcore

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/xianxu/pair/cmd/internal/strictjson"
)

// FleetSchemaVersion is the only `sdlc fleet inventory --json` schema this
// decoder reads (pair#367). A bump is an explicit decoder change.
const FleetSchemaVersion = 1

// ErrFleetSchemaUnsupported is a document this build cannot read as a v1
// inventory with slots and claims: another schema_version, or a v1 build from
// before ariadne#288/#289 added `slots`, `machine`, `dangling_claims` and
// `rows[].claims_state` without a bump. Callers degrade the fleet to
// `unsupported`; it never reads as zero slots.
var ErrFleetSchemaUnsupported = errors.New("unsupported sdlc fleet inventory")

// maxFleetInventoryBytes matches ProvisionIO's stdout cap.
const maxFleetInventoryBytes = 1 << 20

// FleetInventory is sdlc's machine-local fleet observation, decoded once at
// this boundary (ARCH-SECURE). Fields Couch does not read are not modelled;
// additive v1 fields are accepted and ignored.
type FleetInventory struct {
	SchemaVersion  int                  `json:"schema_version"`
	Machine        FleetMachine         `json:"machine"`
	Rows           []FleetRow           `json:"rows"`
	Slots          []FleetSlot          `json:"slots"`
	DanglingClaims []FleetDanglingClaim `json:"dangling_claims"`
	Diagnostics    []FleetDiagnostic    `json:"diagnostics"`
}

// FleetMachine is this machine's identity as sdlc read it.
type FleetMachine struct {
	State       string `json:"state"`
	Fingerprint string `json:"fingerprint,omitempty"`
	Name        string `json:"name,omitempty"`
	Error       string `json:"error,omitempty"`
}

// FleetRow is one worktree's measured git facts, branch-named issues and this
// machine's claims on it.
type FleetRow struct {
	RepoIdentity string       `json:"repo_identity"`
	RepoRoot     string       `json:"repo_root"`
	TreePath     string       `json:"tree_path"`
	Branch       string       `json:"branch,omitempty"`
	Detached     bool         `json:"detached"`
	Bare         bool         `json:"bare"`
	Facts        FleetFacts   `json:"facts"`
	Issues       []FleetIssue `json:"issues"`
	Claims       []FleetClaim `json:"claims"`
	ClaimsState  string       `json:"claims_state"`
	ClaimsError  string       `json:"claims_error,omitempty"`
}

// FleetFacts are a row's measured counts; a nil count was not measured.
type FleetFacts struct {
	Available     bool   `json:"available"`
	Error         string `json:"error,omitempty"`
	BaseAvailable bool   `json:"base_available"`
	Ahead         *int   `json:"ahead,omitempty"`
	Behind        *int   `json:"behind,omitempty"`
	DirtyCount    *int   `json:"dirty_count,omitempty"`
}

// FleetIssue is the issue a row's branch prefix names.
type FleetIssue struct {
	Ref            string `json:"ref"`
	DeclaredStatus string `json:"declared_status"`
	Provenance     string `json:"provenance"`
	StaleStatus    bool   `json:"stale_status,omitempty"`
}

// FleetClaim is one tracker claim this machine holds on a row.
type FleetClaim struct {
	Ref      string        `json:"ref"`
	Status   string        `json:"status"`
	Revision string        `json:"revision"`
	Claimant FleetClaimant `json:"claimant"`
}

// FleetClaimant is the card's responsibility record. Workspace is empty for a
// claim made from a dependency clone or a plain clone (ariadne claimant.go).
type FleetClaimant struct {
	Operator    string `json:"operator"`
	Machine     string `json:"machine"`
	MachineName string `json:"machine_name"`
	Workspace   string `json:"workspace,omitempty"`
	Worktree    string `json:"worktree"`
	Repository  string `json:"repository"`
}

// FleetDanglingClaim is this machine's claim whose worktree is no inventory row.
type FleetDanglingClaim struct {
	RepoIdentity string `json:"repo_identity"`
	RepoRoot     string `json:"repo_root"`
	FleetClaim
}

// FleetSlot is one workspace's readiness verdict and members.
type FleetSlot struct {
	Address         string        `json:"address"`
	Repo            string        `json:"repo"`
	Slot            int           `json:"slot"`
	EnvironmentRoot string        `json:"environment_root,omitempty"`
	RestingBranch   string        `json:"resting_branch"`
	Verdict         string        `json:"verdict"`
	Members         []FleetMember `json:"members"`
}

// FleetMember is one checkout of a slot and its own verdict and reasons.
type FleetMember struct {
	Role          string   `json:"role"`
	Path          string   `json:"path"`
	RestingBranch string   `json:"resting_branch"`
	Branch        string   `json:"branch,omitempty"`
	Verdict       string   `json:"verdict"`
	Reasons       []string `json:"reasons"`
	Errors        []string `json:"errors,omitempty"`
}

// FleetDiagnostic is a worktree sdlc could not read.
type FleetDiagnostic struct {
	RepoIdentity string `json:"repo_identity,omitempty"`
	RepoPath     string `json:"repo_path"`
	TreePath     string `json:"tree_path,omitempty"`
	Stage        string `json:"stage"`
	Message      string `json:"message"`
}

// Fleet verdicts (ariadne fleet/slots.go). An unknown string decodes and is
// read as unknown downstream.
const (
	FleetVerdictReady         = "ready"
	FleetVerdictHoldsWork     = "holds-work"
	FleetVerdictUnknown       = "unknown"
	FleetVerdictMissing       = "missing"
	FleetVerdictNeedsRecovery = "needs-recovery"
)

var knownFleetVerdicts = map[string]bool{
	FleetVerdictReady: true, FleetVerdictHoldsWork: true, FleetVerdictUnknown: true,
	FleetVerdictMissing: true, FleetVerdictNeedsRecovery: true,
}

// Claim read qualities (ariadne fleet/claims.go).
const (
	FleetClaimsPresent = "present"
	FleetClaimsStale   = "stale"
	FleetClaimsPartial = "partial"
	FleetClaimsUnknown = "unknown"
	FleetClaimsAbsent  = "absent"
)

var knownClaimsStates = map[string]bool{
	FleetClaimsPresent: true, FleetClaimsStale: true, FleetClaimsPartial: true,
	FleetClaimsUnknown: true, FleetClaimsAbsent: true,
}

// knownFleetReason is the member reason grammar: dirty | detached | missing |
// unlanded-commits | operation:<x> | open-issue:<ref> | claimed:<ref> |
// probe:<x>.
func knownFleetReason(reason string) bool {
	switch reason {
	case "dirty", "detached", "missing", "unlanded-commits":
		return true
	}
	kind, value, ok := strings.Cut(reason, ":")
	if !ok || value == "" {
		return false
	}
	switch kind {
	case "operation", "open-issue", "claimed", "probe":
		return true
	}
	return false
}

// DecodeFleetInventory reads `sdlc fleet inventory --json` v1. The version is
// checked before anything else, then the presence of every section #288/#289
// added, so an older v1 build is ErrFleetSchemaUnsupported rather than an
// inventory with no slots. Malformed input is an ordinary error.
func DecodeFleetInventory(raw []byte) (FleetInventory, error) {
	if len(raw) > maxFleetInventoryBytes {
		return FleetInventory{}, fmt.Errorf("sdlc fleet inventory exceeds 1 MiB")
	}
	if err := strictjson.RejectDuplicateKeys(raw); err != nil {
		return FleetInventory{}, fmt.Errorf("sdlc fleet inventory: %w", err)
	}
	var top map[string]json.RawMessage
	if err := json.Unmarshal(raw, &top); err != nil || top == nil {
		return FleetInventory{}, fmt.Errorf("sdlc fleet inventory: not a JSON object: %v", err)
	}
	var version int
	if v, ok := top["schema_version"]; !ok || json.Unmarshal(v, &version) != nil || version != FleetSchemaVersion {
		return FleetInventory{}, fmt.Errorf("%w: schema_version %s, want %d", ErrFleetSchemaUnsupported, top["schema_version"], FleetSchemaVersion)
	}
	for _, key := range []string{"slots", "machine", "dangling_claims", "rows"} {
		if !fleetFieldPresent(top, key) {
			return FleetInventory{}, fmt.Errorf("%w: no %s (sdlc predates #288/#289; upgrade the slot's sdlc)", ErrFleetSchemaUnsupported, key)
		}
	}
	var rows []map[string]json.RawMessage
	if err := json.Unmarshal(top["rows"], &rows); err != nil {
		return FleetInventory{}, fmt.Errorf("sdlc fleet inventory rows: %w", err)
	}
	for i, row := range rows {
		if !fleetFieldPresent(row, "claims_state") {
			return FleetInventory{}, fmt.Errorf("%w: row %d has no claims_state (sdlc predates #288/#289; upgrade the slot's sdlc)", ErrFleetSchemaUnsupported, i)
		}
	}
	var inv FleetInventory
	if err := json.Unmarshal(raw, &inv); err != nil {
		return FleetInventory{}, fmt.Errorf("sdlc fleet inventory: %w", err)
	}
	return inv, nil
}

func fleetFieldPresent(object map[string]json.RawMessage, key string) bool {
	v, ok := object[key]
	return ok && string(v) != "null"
}
