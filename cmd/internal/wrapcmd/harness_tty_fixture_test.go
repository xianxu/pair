package wrapcmd

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"
)

type ttyFixtureMetadata struct {
	Agent      string            `json:"agent"`
	Version    string            `json:"version"`
	CapturedAt string            `json:"captured_at"`
	Command    []string          `json:"command"`
	Files      map[string]string `json:"files"`
}

var ttyFixtureVersionToken = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

func ttyFixtureVersionDir(version string) string {
	fields := strings.Fields(strings.TrimSpace(version))
	for i := len(fields) - 1; i >= 0; i-- {
		candidate := strings.Trim(fields[i], "()[]{}.,;")
		if strings.IndexFunc(candidate, func(r rune) bool { return r >= '0' && r <= '9' }) < 0 {
			continue
		}
		if ttyFixtureVersionToken.MatchString(candidate) {
			return candidate
		}
	}
	return ""
}

func TestHarnessTTYFixtureVersionDir(t *testing.T) {
	tests := map[string]string{
		"Muse Code 0.1.0 (0.1.0-R708.1)": "0.1.0-R708.1",
		"codex-cli 0.42.0":               "0.42.0",
		"agy version 1.2.3-beta+unsafe":  "",
		"no-version":                     "",
	}
	for version, want := range tests {
		if got := ttyFixtureVersionDir(version); got != want {
			t.Errorf("ttyFixtureVersionDir(%q) = %q, want %q", version, got, want)
		}
	}
}

func TestHarnessTTYFixtureConformance(t *testing.T) {
	root := filepath.Join("testdata", "tty")
	required := map[string]bool{}
	for harness, profile := range harnessTTYProfiles {
		if profile.composerGate == composerGatePositive {
			required[harness] = false
		}
	}
	if len(required) == 0 {
		t.Fatal("positive-gated harness fixture inventory is empty")
	}

	metadataPaths, err := filepath.Glob(filepath.Join(root, "*", "*", "metadata.json"))
	if err != nil {
		t.Fatalf("glob fixture metadata: %v", err)
	}
	if len(metadataPaths) == 0 {
		t.Fatal("fixture metadata inventory is empty")
	}
	for _, metadataPath := range metadataPaths {
		metadata, rawFiles := readHarnessTTYFixture(t, metadataPath)
		if _, ok := required[metadata.Agent]; !ok {
			t.Errorf("%s: agent %q has no positive-gated profile", metadataPath, metadata.Agent)
			continue
		}
		required[metadata.Agent] = true
		if filepath.Base(filepath.Dir(metadataPath)) != ttyFixtureVersionDir(metadata.Version) {
			t.Errorf("%s: version directory %q does not match normalized version %q", metadataPath, filepath.Base(filepath.Dir(metadataPath)), ttyFixtureVersionDir(metadata.Version))
		}
		if _, ok := rawFiles["composer.raw"]; !ok {
			t.Errorf("%s: missing composer.raw", metadataPath)
		}
		for _, name := range sortedKeys(rawFiles) {
			wantComposer, ok := ttyFixtureExpectation[name]
			if !ok {
				t.Errorf("%s: raw file %q has no recorded Return expectation", metadataPath, name)
				continue
			}
			t.Run(fixtureLabel(metadata)+"/"+name, func(t *testing.T) {
				replayHarnessTTYFixture(t, metadata.Agent, rawFiles[name], wantComposer)
			})
		}
	}

	var missing []string
	for harness, found := range required {
		if !found {
			missing = append(missing, harness)
		}
	}
	sort.Strings(missing)
	if len(missing) != 0 {
		t.Fatalf("required positive-gated fixtures missing: %s", strings.Join(missing, ", "))
	}
}

func readHarnessTTYFixture(t *testing.T, metadataPath string) (ttyFixtureMetadata, map[string][]byte) {
	t.Helper()
	encoded, err := os.ReadFile(metadataPath)
	if err != nil {
		t.Fatalf("read %s: %v", metadataPath, err)
	}
	var metadata ttyFixtureMetadata
	if err := json.Unmarshal(encoded, &metadata); err != nil {
		t.Fatalf("decode %s: %v", metadataPath, err)
	}
	if metadata.Agent == "" || strings.TrimSpace(metadata.Version) != metadata.Version || metadata.Version == "" {
		t.Errorf("%s: agent/version must be non-empty and version exactly trimmed", metadataPath)
	}
	if _, err := time.Parse(time.RFC3339, metadata.CapturedAt); err != nil {
		t.Errorf("%s: captured_at %q is not RFC3339: %v", metadataPath, metadata.CapturedAt, err)
	}
	if len(metadata.Command) == 0 {
		t.Errorf("%s: command argv is empty", metadataPath)
	}
	for i, arg := range metadata.Command {
		if arg == "" {
			t.Errorf("%s: command argv[%d] is empty", metadataPath, i)
		}
	}

	dir := filepath.Dir(metadataPath)
	rawFiles := map[string][]byte{}
	err = filepath.WalkDir(dir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".raw" {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		rawFiles[entry.Name()] = data
		return nil
	})
	if err != nil {
		t.Fatalf("inventory raw files beside %s: %v", metadataPath, err)
	}
	if len(metadata.Files) != len(rawFiles) {
		t.Errorf("%s: metadata has %d digests for %d raw files", metadataPath, len(metadata.Files), len(rawFiles))
	}
	for name, data := range rawFiles {
		digest := sha256.Sum256(data)
		want := hex.EncodeToString(digest[:])
		got, ok := metadata.Files[name]
		if !ok {
			t.Errorf("%s: raw file %q has no digest", metadataPath, name)
		} else if got != want {
			t.Errorf("%s: digest for %q = %q, want %q", metadataPath, name, got, want)
		}
	}
	for name, digest := range metadata.Files {
		if _, ok := rawFiles[name]; !ok {
			t.Errorf("%s: dangling digest for %q", metadataPath, name)
		}
		if len(digest) != sha256.Size*2 || strings.ToLower(digest) != digest {
			t.Errorf("%s: digest for %q is not lowercase SHA-256: %q", metadataPath, name, digest)
		}
		if _, err := hex.DecodeString(digest); err != nil {
			t.Errorf("%s: digest for %q is not hex: %v", metadataPath, name, err)
		}
	}
	if len(rawFiles) == 0 {
		t.Errorf("%s: fixture contains no raw files", metadataPath)
	}
	return metadata, rawFiles
}

func fixtureLabel(metadata ttyFixtureMetadata) string {
	return fmt.Sprintf("%s/%s", metadata.Agent, ttyFixtureVersionDir(metadata.Version))
}

// ttyFixtureExpectation records what each captured state must decide for a
// plain Return. A composer capture must remap to a newline; an overlay capture
// must pass a bare CR through so the overlay confirms.
var ttyFixtureExpectation = map[string]bool{
	"composer.raw": true,
	"overlay.raw":  false,
}

// harnessTTYReplayResult is the observable result of replaying a fixture
// through the production seam: what the recognizer saw and what a plain Return
// would actually emit.
type harnessTTYReplayResult struct {
	recognized   bool
	overlayArmed bool
	returnBytes  string
	reason       string
}

// replayHarnessTTYFixture feeds raw through the production proxy unsplit to
// establish a baseline, then repeats the feed at every possible split point.
// Chunk boundaries are an artifact of PTY scheduling, so no split may change
// what the user's next Return does.
func replayHarnessTTYFixture(t *testing.T, harness string, raw []byte, wantComposer bool) {
	t.Helper()
	baseline := replayHarnessTTYSplit(t, harness, raw, len(raw))
	if baseline.recognized != wantComposer {
		t.Fatalf("unsplit recognizer = %t, want %t (%s)", baseline.recognized, wantComposer, baseline.reason)
	}
	wantReturn := "\r"
	if wantComposer {
		wantReturn = "\n"
	}
	if baseline.returnBytes != wantReturn {
		t.Fatalf("unsplit plain Return = %q, want %q (%s)", baseline.returnBytes, wantReturn, baseline.reason)
	}
	for split := 0; split <= len(raw); split++ {
		if got := replayHarnessTTYSplit(t, harness, raw, split); got != baseline {
			t.Fatalf("split at %d/%d changed the Return decision: %+v, want %+v", split, len(raw), got, baseline)
		}
	}
}

// replayHarnessTTYSplit feeds raw as two chunks divided at split and reports
// what a plain Return would emit afterwards.
func replayHarnessTTYSplit(t *testing.T, harness string, raw []byte, split int) harnessTTYReplayResult {
	t.Helper()
	p := &proxy{agentBasename: harness}
	if err := p.configureHarnessTTY(true, 120, 38); err != nil {
		t.Fatalf("configure %s replay proxy: %v", harness, err)
	}
	defer func() {
		if err := p.closeTerminal(); err != nil {
			t.Errorf("close %s replay proxy: %v", harness, err)
		}
	}()
	var rolling []byte
	p.handleChunk(raw[:split], &rolling)
	p.handleChunk(raw[split:], &rolling)

	snapshot := p.terminal.Snapshot()
	result := harnessTTYReplayResult{
		recognized:   p.ttyProfile.recognize(snapshot),
		overlayArmed: p.pickerActive.Load(),
	}
	// emitPlainCR is the production Return path: it consumes overlay state
	// under the same lock the detector arms it with, then decides.
	result.returnBytes = string(p.emitPlainCR(nil))
	result.reason = decidePlainReturn(*p.ttyProfile, result.overlayArmed, &snapshot).reason
	return result
}

func sortedKeys(files map[string][]byte) []string {
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
