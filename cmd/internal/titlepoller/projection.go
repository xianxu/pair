package titlepoller

import (
	"errors"
	"github.com/xianxu/pair/cmd/internal/contextcmd"
	"github.com/xianxu/pair/cmd/internal/ctxmeter"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"path/filepath"
	"time"
)

// titleProjection owns one exact invocation's compact display facts. Activity
// begins one poll; occupancy reads that same result without a second scan.
type titleProjection struct {
	env        contextcmd.Env
	runtime    sessioninventory.OSRuntime
	ordinal    uint64
	core       *sessioninventory.TitleProjection
	tag, agent string
	result     sessioninventory.TitleProjectionResult
	err        error
}

func newTitleProjection(env contextcmd.Env, ordinal uint64) *titleProjection {
	data := env.PairDataDir
	if data == "" {
		base := env.XDGDataHome
		if base == "" {
			base = filepath.Join(env.Home, ".local", "share")
		}
		data = filepath.Join(base, "pair")
	}
	return &titleProjection{env: env, runtime: sessioninventory.NewOSRuntime(env.Home, data), ordinal: ordinal}
}
func (p *titleProjection) activity(tag, agent string) (time.Time, bool) {
	if p.err != nil {
		return time.Time{}, false
	}
	if p.core == nil {
		p.tag, p.agent = tag, agent
		p.core = sessioninventory.NewTitleProjection(p.runtime, p.env.PairScopeKey, tag, sessioninventory.Agent(agent), p.ordinal)
	}
	if tag != p.tag || agent != p.agent {
		p.err = errors.New("title projection owner changed")
		return time.Time{}, false
	}
	p.result, p.err = p.core.Poll()
	return p.result.Activity, p.err == nil && !p.result.Awaiting && !p.result.Activity.IsZero()
}
func (p *titleProjection) count(tag, agent string) string {
	if p.err != nil || p.core == nil || p.result.Awaiting || !p.result.HasUsage || tag != p.tag || agent != p.agent {
		return ""
	}
	return ctxmeter.Humanize(p.result.Usage.InputTokens)
}
func (p *titleProjection) Err() error { return p.err }
