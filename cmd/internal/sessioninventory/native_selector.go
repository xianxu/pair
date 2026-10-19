package sessioninventory

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"time"
)

type NativeSelectorStatus string

const (
	NativeSelectorResolved    NativeSelectorStatus = "resolved"
	NativeSelectorNone        NativeSelectorStatus = "none"
	NativeSelectorAmbiguous   NativeSelectorStatus = "ambiguous"
	NativeSelectorIncomplete  NativeSelectorStatus = "incomplete"
	NativeSelectorUnsupported NativeSelectorStatus = "unsupported"
)

// NativeSelectorConfig contains already resolved native configuration, not
// Pair defaults. A positively resolved empty Provider means the native query
// intentionally has no provider filter. False resolution flags never imply
// openai or the default Codex home.
type NativeSelectorConfig struct {
	Provider           string
	ProviderResolved   bool
	SQLiteHome         string
	SQLiteHomeResolved bool
	Remote             bool
	ScopeExpanded      bool
}

type NativeSelectorRequest struct {
	Agent                 Agent
	NativeID              string
	CWD                   string
	Latest                bool
	IncludeNonInteractive bool
	All                   bool
	Config                NativeSelectorConfig
}

type NativeSelectorResolution struct {
	Status   NativeSelectorStatus
	NativeID string
	Reason   string
}

type NativeSelectorCandidate struct {
	Agent     Agent
	NativeID  string
	CWD       string
	UpdatedAt time.Time
	Eligible  bool
}

type NativeSelectorSnapshot struct {
	Complete    bool
	Unsupported bool
	Reason      string
	Candidates  []NativeSelectorCandidate
}

// SelectNativeLatest resolves only complete provider-native metadata. Recovery
// scores, capture times and filesystem fallback activity are not selector facts.
func SelectNativeLatest(q NativeSelectorRequest, s NativeSelectorSnapshot) NativeSelectorResolution {
	if q.Agent == AgentCodex {
		return NativeSelectorResolution{Status: NativeSelectorUnsupported, Reason: "Codex indirect selectors require native selection"}
	}
	if s.Unsupported {
		return NativeSelectorResolution{Status: NativeSelectorUnsupported, Reason: s.Reason}
	}
	if !s.Complete {
		return NativeSelectorResolution{Status: NativeSelectorIncomplete, Reason: s.Reason}
	}
	var best NativeSelectorCandidate
	tied := false
	for _, c := range s.Candidates {
		if c.Agent != q.Agent || c.CWD != q.CWD || !c.Eligible {
			continue
		}
		if c.NativeID == "" || c.UpdatedAt.IsZero() {
			return NativeSelectorResolution{Status: NativeSelectorIncomplete, Reason: "eligible native session has no canonical identity or activity"}
		}
		if best.NativeID == "" || c.UpdatedAt.After(best.UpdatedAt) {
			best = c
			tied = false
		} else if c.UpdatedAt.Equal(best.UpdatedAt) && c.NativeID != best.NativeID {
			tied = true
		}
	}
	if best.NativeID == "" {
		return NativeSelectorResolution{Status: NativeSelectorNone, Reason: "no eligible native session for this directory"}
	}
	if tied {
		return NativeSelectorResolution{Status: NativeSelectorAmbiguous, Reason: "latest native activity is tied"}
	}
	return NativeSelectorResolution{Status: NativeSelectorResolved, NativeID: best.NativeID}
}

// ResolveNativeSelector validates explicit canonical roots for all supported
// scanners. Latest selection additionally requires a complete provider-native
// metadata snapshot; unresolved internal states never authorize execution.
func ResolveNativeSelector(ctx context.Context, rt Runtime, q NativeSelectorRequest) (NativeSelectorResolution, error) {
	if ctx == nil || rt == nil {
		return NativeSelectorResolution{}, errors.New("native selector requires context and inventory")
	}
	if err := ctx.Err(); err != nil {
		return NativeSelectorResolution{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, RecoveryScanTimeout)
	defer cancel()
	if !validAgent(q.Agent) {
		return NativeSelectorResolution{Status: NativeSelectorUnsupported, Reason: "unsupported native agent"}, nil
	}
	if q.Agent == AgentCodex && (q.Latest || !IsCanonicalCodexSessionID(q.NativeID)) {
		return NativeSelectorResolution{Status: NativeSelectorUnsupported, Reason: "Codex indirect selectors require native selection"}, nil
	}
	if q.Config.Remote || (q.Latest && (q.Config.ScopeExpanded || q.All)) {
		return NativeSelectorResolution{Status: NativeSelectorUnsupported, Reason: "native selector scope cannot be mapped to one local directory"}, nil
	}
	if q.Latest {
		if !filepath.IsAbs(q.CWD) {
			return NativeSelectorResolution{Status: NativeSelectorUnsupported, Reason: "native selector requires an absolute directory"}, nil
		}
		physical, ok := rt.(RecoveryRuntime)
		if !ok {
			return NativeSelectorResolution{Status: NativeSelectorUnsupported, Reason: "bounded native selector IO unavailable"}, nil
		}
		cwd, err := physical.ResolveRecoveryCWD(ctx, q.CWD)
		if err != nil {
			return NativeSelectorResolution{Status: NativeSelectorIncomplete, Reason: "physical native selector directory unavailable"}, nil
		}
		q.CWD = cwd
		snapshot := collectNativeSelector(ctx, rt, q)
		result := SelectNativeLatest(q, snapshot)
		if result.Status != NativeSelectorResolved {
			return result, ctx.Err()
		}
		q.NativeID = result.NativeID
	}
	if q.NativeID == "" || strings.HasPrefix(q.NativeID, "-") || strings.ContainsAny(q.NativeID, "\x00\r\n/") {
		return NativeSelectorResolution{Status: NativeSelectorNone, Reason: "native selector is not a canonical ID"}, nil
	}
	bounded, err := newNativeSelectorRuntime(ctx, rt)
	if err != nil {
		return NativeSelectorResolution{Status: NativeSelectorUnsupported, Reason: err.Error()}, nil
	}
	if q.Agent == AgentClaude && uuidPattern.MatchString(q.NativeID) {
		bounded.exactClaudeID = q.NativeID
	}
	validation, diagnostics, err := NewIncrementalInventory(bounded, Catalog{}).ValidateNamedContent(q.Agent, q.NativeID)
	if contextErr := ctx.Err(); contextErr != nil {
		return NativeSelectorResolution{}, contextErr
	}
	if bounded.failure != nil {
		return NativeSelectorResolution{Status: NativeSelectorIncomplete, Reason: "native root inspection incomplete"}, nil
	}
	for _, d := range diagnostics {
		if d.Code == DiagnosticParentConflict && d.NativeID != nil && *d.NativeID == q.NativeID {
			return NativeSelectorResolution{Status: NativeSelectorNone, Reason: "native root identity evidence is contradictory"}, nil
		}
	}
	if err != nil || validation.State.Disputed || validation.State.Role != RoleRoot || !validation.Fact.Resumable || validation.State.NativeID != q.NativeID {
		return NativeSelectorResolution{Status: NativeSelectorNone, Reason: "native ID has no unique validated resumable root"}, nil
	}
	return NativeSelectorResolution{Status: NativeSelectorResolved, NativeID: validation.State.NativeID}, nil
}

// IsCanonicalCodexSessionID recognizes the same complete UUID shape as native
// Codex scanner authority. Existence and root eligibility remain separate checks.
func IsCanonicalCodexSessionID(id string) bool { return uuidPattern.MatchString(id) }
