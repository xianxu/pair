package couchcore

import (
	"context"
	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/sessionledger"
	"os"
	"path/filepath"
	"testing"
)

func TestContinuationSourceLatestAcrossAgents(t *testing.T) {
	data := t.TempDir()
	address := validThreadRecord(t).Address
	paths, err := artifactpath.Resolve(artifactpath.Address{DataDir: data, RepoScope: address.RepoScope, Tag: string(address.Tag)})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(paths.ScopeDir(), 0700); err != nil {
		t.Fatal(err)
	}
	var raw []byte
	for _, agent := range []string{"claude", "codex"} {
		row, err := sessionledger.EncodeRecord(sessionledger.Record{Version: 1, Kind: sessionledger.RecordLaunch, ScopeKey: address.RepoScope, Tag: string(address.Tag), Agent: agent})
		if err != nil {
			t.Fatal(err)
		}
		raw = append(raw, append(row, '\n')...)
	}
	if err := os.WriteFile(paths.Ledger(), raw, 0600); err != nil {
		t.Fatal(err)
	}
	scope, err := artifactpath.ResolveSelectedScope(paths.ScopeDir())
	if err != nil {
		t.Fatal(err)
	}
	line, err := launcher.BuildSessionNameIndexLine(launcher.SessionNameEntry{ScopeKey: address.RepoScope, Tag: string(address.Tag), SessionName: "📁repo-test", RepoRoot: "/repo", RepoName: "repo"})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(scope.SessionBindings(), []byte(line+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	proof, err := (OSContinuationSourceReader{DataDir: data}).Read(context.Background(), address)
	if err != nil || proof.Agent != "codex" || proof.LaunchOrdinal != 2 || proof.Session != "📁repo-test" {
		t.Fatalf("proof %+v %v", proof, err)
	}
	if err := os.WriteFile(filepath.Clean(paths.Ledger()), append(raw, []byte("broken\n")...), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := (OSContinuationSourceReader{DataDir: data}).Read(context.Background(), address); err == nil {
		t.Fatal("malformed ledger granted source authority")
	}
}
