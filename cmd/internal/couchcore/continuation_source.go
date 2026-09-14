package couchcore

import (
	"context"
	"errors"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/sessionledger"
	"os"
)

// Source authority is the newest launch at the address, across every agent.
// Filtering first by the submitting agent would authorize an obsolete pane.
type OSContinuationSourceReader struct{ DataDir string }

func (r OSContinuationSourceReader) Read(ctx context.Context, address ThreadAddress) (ContinuationSource, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return ContinuationSource{}, err
	}
	paths, err := artifactpath.Resolve(artifactpath.Address{DataDir: r.DataDir, RepoScope: address.RepoScope, Tag: string(address.Tag)})
	if err != nil {
		return ContinuationSource{}, err
	}
	raw, err := os.ReadFile(paths.Ledger())
	if err != nil {
		return ContinuationSource{}, err
	}
	ledger := sessionledger.ParseLedger(raw)
	if len(ledger.MalformedOrdinals) > 0 {
		return ContinuationSource{}, errors.New("continuation source ledger contains malformed rows")
	}
	var result ContinuationSource
	for _, row := range ledger.Records {
		if row.Kind != sessionledger.RecordLaunch {
			continue
		}
		if row.ScopeKey != address.RepoScope || row.Tag != string(address.Tag) {
			return result, errors.New("continuation ledger contains a foreign launch")
		}
		if row.Ordinal > result.LaunchOrdinal {
			result.Agent = row.Agent
			result.LaunchOrdinal = row.Ordinal
		}
	}
	for _, ordinal := range ledger.CompatibilityOrdinals {
		if ordinal > result.LaunchOrdinal {
			return result, errors.New("continuation latest ledger generation is not typed")
		}
	}
	if !launcher.IsSupportedAgent(result.Agent) || result.LaunchOrdinal == 0 {
		return result, errors.New("continuation source has no exact current launch")
	}
	global, err := artifactpath.ResolveLegacyRoot(r.DataDir)
	if err != nil {
		return result, err
	}
	scoped, err := artifactpath.ResolveSelectedScope(paths.ScopeDir())
	if err != nil {
		return result, err
	}
	for _, path := range []string{global.SessionBindings(), scoped.SessionBindings()} {
		raw, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return result, err
		}
		index, err := launcher.DecodeSessionNameIndex(string(raw))
		if err != nil {
			return result, err
		}
		for _, entry := range index.Entries {
			if entry.ScopeKey == address.RepoScope && entry.Tag == string(address.Tag) {
				result.Session = entry.SessionName
			}
		}
	}
	if result.Session == "" {
		return result, fmt.Errorf("continuation source %s has no exact Pair session binding", address.Tag)
	}
	return result, nil
}
