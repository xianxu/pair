package sessioninventory

import (
	"bytes"
	"errors"
	"testing"
)

func TestRunCLIUnreachableRenderAndPrivacyFailures(t *testing.T) {
	t.Parallel()
	runtime := emptyCLIRuntime{}
	t.Run("inventory serialization", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		renderers := defaultCLIRenderers()
		renderers.inventory = func(Inventory, RenderFormat) ([]byte, error) { return nil, errors.New("encode") }
		code := runCLIOptionsWithRenderers(cliOptions{agents: []Agent{AgentCodex}, json: true}, runtime, &stdout, &stderr, renderers)
		if code != 2 || stdout.Len() != 0 || stderr.String() != "pair session-inventory: render failed\n" {
			t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
		}
	})
	t.Run("conformance serialization", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		renderers := defaultCLIRenderers()
		renderers.conformance = func(ConformanceReport) ([]byte, error) { return nil, errors.New("encode") }
		code := runCLIOptionsWithRenderers(cliOptions{agents: []Agent{AgentCodex}, conformance: true}, runtime, &stdout, &stderr, renderers)
		if code != 2 || stdout.Len() != 0 || stderr.String() != "pair session-inventory: render failed\n" {
			t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
		}
	})
	t.Run("conformance privacy", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		renderers := defaultCLIRenderers()
		renderers.conformance = func(ConformanceReport) ([]byte, error) { return []byte(`{"home":"/home/private"}` + "\n"), nil }
		code := runCLIOptionsWithRenderers(cliOptions{agents: []Agent{AgentCodex}, conformance: true}, runtime, &stdout, &stderr, renderers)
		if code != 2 || stdout.Len() != 0 || stderr.String() != "pair session-inventory: conformance privacy check failed\n" {
			t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
		}
	})
}

type emptyCLIRuntime struct{}

func (emptyCLIRuntime) NativeRoots(Agent) []StorageRoot            { return nil }
func (emptyCLIRuntime) PairDataRoot() StorageRoot                  { return StorageRoot{} }
func (emptyCLIRuntime) ListFiles(StorageRoot) ([]FileEntry, error) { return nil, nil }
func (emptyCLIRuntime) ReadFile(Artifact, int64) ([]byte, error)   { return nil, errors.New("unused") }
func (emptyCLIRuntime) ReadAt(Artifact, int64, int64) ([]byte, bool, error) {
	return nil, true, errors.New("unused")
}
func (emptyCLIRuntime) QuerySQLite(Artifact, string, int64) (SQLiteResult, error) {
	return SQLiteResult{}, errors.New("unused")
}
func (emptyCLIRuntime) ProcessChildren() map[string][]string { return nil }
func (emptyCLIRuntime) ProcessIdentity(string) string        { return "" }
func (emptyCLIRuntime) OpenFiles(string) []string            { return nil }
