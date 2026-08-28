package sessioninventory

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestRunCLIUnreachableRenderAndPrivacyFailures(t *testing.T) {
	t.Parallel()
	runtime := emptyCLIRuntime{}
	var matrix []failureGoldenResult
	t.Run("inventory serialization", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		renderers := defaultCLIRenderers()
		renderers.inventory = func(Inventory, RenderFormat) ([]byte, error) { return nil, errors.New("encode") }
		code := runCLIOptionsWithRenderers(cliOptions{agents: []Agent{AgentCodex}, json: true}, runtime, &stdout, &stderr, renderers)
		matrix = append(matrix, failureGoldenResult{Name: "inventory serialization", Exit: code, Stdout: stdout.String(), Stderr: stderr.String()})
		if code != 2 || stdout.Len() != 0 || stderr.String() != "pair session-inventory: render failed\n" {
			t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
		}
	})
	t.Run("human serialization", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		renderers := defaultCLIRenderers()
		renderers.inventory = func(Inventory, RenderFormat) ([]byte, error) { return nil, errors.New("encode") }
		code := runCLIOptionsWithRenderers(cliOptions{agents: []Agent{AgentCodex}}, runtime, &stdout, &stderr, renderers)
		matrix = append(matrix, failureGoldenResult{Name: "human serialization", Exit: code, Stdout: stdout.String(), Stderr: stderr.String()})
		if code != 2 || stdout.Len() != 0 || stderr.String() != "pair session-inventory: render failed\n" {
			t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
		}
	})
	t.Run("conformance serialization", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		renderers := defaultCLIRenderers()
		renderers.conformance = func(ConformanceReport) ([]byte, error) { return nil, errors.New("encode") }
		code := runCLIOptionsWithRenderers(cliOptions{agents: []Agent{AgentCodex}, conformance: true}, runtime, &stdout, &stderr, renderers)
		matrix = append(matrix, failureGoldenResult{Name: "conformance serialization", Exit: code, Stdout: stdout.String(), Stderr: stderr.String()})
		if code != 2 || stdout.Len() != 0 || stderr.String() != "pair session-inventory: render failed\n" {
			t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
		}
	})
	t.Run("conformance privacy", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		renderers := defaultCLIRenderers()
		renderers.conformance = func(ConformanceReport) ([]byte, error) { return []byte(`{"home":"/home/private"}` + "\n"), nil }
		code := runCLIOptionsWithRenderers(cliOptions{agents: []Agent{AgentCodex}, conformance: true}, runtime, &stdout, &stderr, renderers)
		matrix = append(matrix, failureGoldenResult{Name: "conformance privacy", Exit: code, Stdout: stdout.String(), Stderr: stderr.String()})
		if code != 2 || stdout.Len() != 0 || stderr.String() != "pair session-inventory: conformance privacy check failed\n" {
			t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
		}
	})
	t.Run("conformance writer", func(t *testing.T) {
		var stderr bytes.Buffer
		code := runCLIOptionsWithRenderers(cliOptions{agents: []Agent{AgentCodex}, conformance: true}, runtime, failureWriter{}, &stderr, defaultCLIRenderers())
		matrix = append(matrix, failureGoldenResult{Name: "conformance writer", Exit: code, Stderr: stderr.String()})
		if code != 2 || stderr.String() != "pair session-inventory: render write failed\n" {
			t.Fatalf("code=%d stderr=%q", code, stderr.String())
		}
	})
	assertFailureGolden(t, "cli-failure-matrix.json", matrix)
}

type failureGoldenResult struct {
	Name   string `json:"name"`
	Exit   int    `json:"exit"`
	Stdout string `json:"stdout"`
	Stderr string `json:"stderr"`
}

func assertFailureGolden(t *testing.T, name string, results []failureGoldenResult) {
	t.Helper()
	got, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	got = append(got, '\n')
	want, err := os.ReadFile(filepath.Join("testdata", "golden", name))
	if err != nil {
		t.Fatalf("read golden: %v\nwant:\n%s", err, got)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("CLI failure matrix differs from %s\nwant:\n%s\ngot:\n%s", name, want, got)
	}
}

type emptyCLIRuntime struct{}

type failureWriter struct{}

func (failureWriter) Write([]byte) (int, error) { return 0, errors.New("write failed") }

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
