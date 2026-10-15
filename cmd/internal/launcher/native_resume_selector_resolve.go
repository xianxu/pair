package launcher

import (
	"context"
	"errors"
	"path/filepath"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

type NativeResumeRequest struct {
	Agent    string
	CWD      string
	Selector NativeResumeSelector
	Config   sessioninventory.NativeSelectorConfig
}
type NativeResumeResolution struct {
	CodexScope *sessioninventory.CodexSelectionScope
	Status     sessioninventory.NativeSelectorStatus
	NativeID   string
	Args       []string
	Reason     string
}

func ResolveNativeResume(ctx context.Context, rt sessioninventory.Runtime, q NativeResumeRequest) (NativeResumeResolution, error) {
	s, err := ParseNativeResumeSelector(q.Agent, q.Selector.Args)
	if err != nil {
		return NativeResumeResolution{}, err
	}
	if s.Kind != q.Selector.Kind || s.NativeID != q.Selector.NativeID {
		return NativeResumeResolution{}, errors.New("native selector changed before resolution")
	}
	if s.Kind != NativeResumeExplicit && s.Kind != NativeResumeLatest {
		return NativeResumeResolution{Status: sessioninventory.NativeSelectorUnsupported, Reason: "native selector does not identify an exact or latest session"}, nil
	}
	cwd := q.CWD
	if s.CWD != "" {
		cwd = s.CWD
		if !filepath.IsAbs(cwd) {
			cwd = filepath.Join(q.CWD, cwd)
		}
	}
	q.Config.Remote = q.Config.Remote || s.Remote
	q.Config.ScopeExpanded = q.Config.ScopeExpanded || s.ScopeExpanded
	result, err := sessioninventory.ResolveNativeSelector(ctx, rt, sessioninventory.NativeSelectorRequest{Agent: sessioninventory.Agent(q.Agent), NativeID: s.NativeID, CWD: cwd, Latest: s.Kind == NativeResumeLatest, IncludeNonInteractive: s.IncludeNonInteractive, All: s.All, Config: q.Config})
	out := NativeResumeResolution{Status: result.Status, NativeID: result.NativeID, Reason: result.Reason, CodexScope: q.Config.CodexScope}
	if err != nil || result.Status != sessioninventory.NativeSelectorResolved {
		return out, err
	}
	out.Args, err = PinNativeResumeSelector(s, result.NativeID)
	return out, err
}
