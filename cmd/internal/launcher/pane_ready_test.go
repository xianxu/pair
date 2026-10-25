package launcher

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/panebirth"
)

func panePublisherEnv(t *testing.T, agent string) ([]string, string) {
	t.Helper()
	root := t.TempDir()
	data := filepath.Join(root, "data", "repos", "0123456789abcdef")
	paths, err := artifactpath.ResolveScoped(data, "1-test-1")
	if err != nil {
		t.Fatal(err)
	}
	path, err := paths.PaneChecked(agent)
	if err != nil {
		t.Fatal(err)
	}
	return []string{
		"HOME=" + filepath.Join(root, "home"), "ZELLIJ_SOCKET_DIR=" + filepath.Join(root, "native"),
		"PAIR_DATA_DIR=" + data, "PAIR_SCOPE_KEY=0123456789abcdef", "PAIR_TAG=1-test-1",
		"PAIR_AGENT=" + agent, "PAIR_LAUNCH_NONCE=start-publisher", "ZELLIJ_SESSION_NAME=private-session",
		"ZELLIJ_PANE_ID=7", "PAIR_AGENT_PANE_PATH=" + path,
	}, path
}

func TestPanePublisherWritesExactLaunchWithoutNativeCommand(t *testing.T) {
	for _, agent := range []string{"claude", "codex"} {
		t.Run(agent, func(t *testing.T) {
			env, path := panePublisherEnv(t, agent)
			if err := PublishPaneReady(t.Context(), env); err != nil {
				t.Fatal(err)
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			record, err := panebirth.DecodeRecord(raw)
			if err != nil {
				t.Fatal(err)
			}
			if record.PaneID != "7" || record.Identity.Agent != agent || record.Identity.Session != "private-session" || record.Identity.Nonce != "start-publisher" || record.Identity.Validate() != nil {
				t.Fatalf("published wrong launch: %+v", record)
			}
		})
	}
}

func TestPanePublisherRefusesForeignPathAndCancelledWrite(t *testing.T) {
	env, path := panePublisherEnv(t, "claude")
	foreign := filepath.Join(t.TempDir(), "foreign")
	if err := PublishPaneReady(t.Context(), append(env, "PAIR_AGENT_PANE_PATH="+foreign)); err == nil {
		t.Fatal("foreign sidecar binding accepted")
	}
	if _, err := os.Lstat(foreign); !os.IsNotExist(err) {
		t.Fatalf("foreign path touched: %v", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := PublishPaneReady(ctx, env); err == nil {
		t.Fatal("cancelled publication succeeded")
	}
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		t.Fatalf("cancelled write touched path: %v", err)
	}
}
