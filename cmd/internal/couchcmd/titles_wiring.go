package couchcmd

import (
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchtty"
)

// wireTitles installs the sole supervisor title owner using the same explicit
// environment and data root as the rest of this Couch instance.
func wireTitles(console *couchtty.Console, c *couchcore.Couch, getenv func(string) string) {
	source, ok := c.Artifacts.(interface{ PairLifecycleDataDir() string })
	if !ok {
		return
	}
	updater := newCouchTitleUpdater(source.PairLifecycleDataDir(), getenv("HOME"), getenv("CMUX_WORKSPACE_ID"), c.TitleTargetCurrent)
	console.SetActivityPass(updater.Update)
}
