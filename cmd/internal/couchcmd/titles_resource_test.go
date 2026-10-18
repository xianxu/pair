package couchcmd

import "github.com/xianxu/pair/cmd/internal/couchcore"

// Only source-root selection differs between the real fixtures. Commands use
// their shared disposable shim and the production scoped command implementation.
type fleetOSTitleSources struct {
	*observedOSTitles
	neighbor *observedOSTitles
}

func (f fleetOSTitleSources) Open(target couchcore.TitleTarget) (couchTitleSource, error) {
	if target.Address.RepoScope == "neighbor" {
		return f.neighbor.Open(target)
	}
	return f.observedOSTitles.Open(target)
}
