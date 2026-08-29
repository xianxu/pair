package sessioninventory

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestLiveProviderContractConformance(t *testing.T) {
	if os.Getenv("PAIR_LIVE_NATIVE_SESSIONS") != "1" {
		t.Skip("set PAIR_LIVE_NATIVE_SESSIONS=1 for installed-provider comparison")
	}
	runtime, err := DefaultOSRuntime(t.TempDir())
	if err != nil {
		t.Fatal("live provider runtime unavailable")
	}
	for _, agent := range []Agent{AgentClaude, AgentCodex, AgentMuse} {
		agent := agent
		t.Run(string(agent), func(t *testing.T) {
			if !findValidLiveJSONLSample(runtime, agent) {
				t.Fatalf("%s provider contract drift or no bounded recognized sample", agent)
			}
		})
	}
	t.Run("agy", func(t *testing.T) {
		if ok, summary := validateCopiedLiveAgySample(t, runtime); !ok {
			t.Fatalf("agy provider contract drift: %s", summary)
		}
	})
}

func findValidLiveJSONLSample(runtime OSRuntime, agent Agent) bool {
	for _, root := range runtime.NativeRoots(agent) {
		files, err := runtime.ListFiles(root)
		if err != nil && len(files) == 0 {
			continue
		}
		sort.Slice(files, func(i, j int) bool { return files[i].Size < files[j].Size })
		attempts := 0
		for _, entry := range files {
			if !recognizedLiveJSONL(agent, entry.Artifact.RelativePath) || entry.Size > 8<<20 {
				continue
			}
			attempts++
			raw, readErr := runtime.ReadFile(entry.Artifact, 8<<20)
			if readErr != nil {
				continue
			}
			records, state, frameErr := FrameJSONLSuffix(JSONLFrameState{}, raw, jsonRecordLimit)
			if frameErr != nil || len(state.IncompleteTail) != 0 {
				continue
			}
			entry.Artifact.Kind = ArtifactTranscript
			var scanner ScannerState
			var validateErr error
			switch agent {
			case AgentClaude:
				scanner, _, validateErr = ValidateClaudeDelta(entry, nil, records)
			case AgentCodex:
				scanner, _, validateErr = ValidateCodexDelta(entry, nil, records)
			case AgentMuse:
				scanner, _, validateErr = ValidateMuseDelta(entry, nil, records)
			}
			if validateErr == nil && !scanner.Disputed && (agent != AgentCodex || scanner.FirstRecordValidated) {
				return true
			}
			if attempts >= 8 {
				break
			}
		}
	}
	return false
}

func recognizedLiveJSONL(agent Agent, relativePath string) bool {
	switch agent {
	case AgentClaude:
		_, _, _, ok := claudePathFact(relativePath)
		return ok
	case AgentCodex:
		_, ok := codexPathID(relativePath)
		return ok
	case AgentMuse:
		_, _, _, ok := musePathFact(relativePath)
		return ok
	default:
		return false
	}
}

func validateCopiedLiveAgySample(t *testing.T, installed OSRuntime) (bool, string) {
	t.Helper()
	databases := map[string]FileEntry{}
	transcripts := map[string]FileEntry{}
	for _, root := range installed.NativeRoots(AgentAgy) {
		files, err := installed.ListFiles(root)
		if err != nil && len(files) == 0 {
			continue
		}
		for _, entry := range files {
			if root.Name == "agy-conversations" {
				if id, ok := agyDatabasePathID(entry.Artifact.RelativePath); ok && entry.Size <= 64<<20 {
					databases[id] = entry
				}
			} else if root.Name == "agy-brain" {
				if id, ok := agyTranscriptPathID(entry.Artifact.RelativePath); ok && entry.Size <= 8<<20 {
					transcripts[id] = entry
				}
			}
		}
	}
	pairs, readable, framed, validated := 0, 0, 0, 0
	rejections := map[string]int{}
	for id, database := range databases {
		transcript, ok := transcripts[id]
		if !ok {
			continue
		}
		pairs++
		transcriptRaw, transcriptErr := installed.ReadFile(transcript.Artifact, 8<<20)
		sourceDatabase, dbErr := installed.resolveArtifact(database.Artifact)
		if dbErr != nil || transcriptErr != nil {
			continue
		}
		home := t.TempDir()
		dbPath := filepath.Join(home, ".gemini", "antigravity-cli", "conversations", id+".db")
		transcriptPath := filepath.Join(home, ".gemini", "antigravity-cli", "brain", filepath.FromSlash(transcript.Artifact.RelativePath))
		if os.MkdirAll(filepath.Dir(dbPath), 0o700) != nil || os.MkdirAll(filepath.Dir(transcriptPath), 0o700) != nil || copySQLiteSnapshot(sourceDatabase, dbPath) != nil || os.WriteFile(transcriptPath, transcriptRaw, 0o600) != nil {
			return false, "temporary-copy setup failed"
		}
		readable++
		copied := NewOSRuntime(home, t.TempDir())
		database.Artifact = Artifact{StorageRoot: "agy-conversations", RelativePath: id + ".db", Kind: ArtifactDatabase}
		transcript.Artifact = Artifact{StorageRoot: "agy-brain", RelativePath: transcript.Artifact.RelativePath, Kind: ArtifactTranscript}
		records, frame, frameErr := FrameJSONLSuffix(JSONLFrameState{}, transcriptRaw, jsonRecordLimit)
		if frameErr != nil || len(frame.IncompleteTail) != 0 {
			continue
		}
		framed++
		state, diagnostics, validateErr := ValidateAgyDelta(copied, database, transcript, nil, records)
		if validateErr == nil && !state.Disputed && state.FirstRecordValidated {
			return true, "ok"
		}
		validated++
		if validateErr != nil {
			rejections["validator input"]++
		} else if len(diagnostics) != 0 {
			rejections[diagnostics[0].Detail]++
		} else {
			rejections["disputed without diagnostic"]++
		}
	}
	return false, fmt.Sprintf("databases=%d transcripts=%d joined=%d readable=%d framed=%d rejected=%d reasons=%v", len(databases), len(transcripts), pairs, readable, framed, validated, rejections)
}

func copySQLiteSnapshot(source, destination string) error {
	quotedDestination := "'" + strings.ReplaceAll(destination, "'", "''") + "'"
	command := exec.Command("sqlite3", "-readonly", source, "VACUUM INTO "+quotedDestination)
	return command.Run()
}
