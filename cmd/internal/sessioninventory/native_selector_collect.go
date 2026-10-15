package sessioninventory

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

func collectNativeSelector(ctx context.Context, rt Runtime, q NativeSelectorRequest) NativeSelectorSnapshot {
	if q.Agent == AgentCodex {
		return collectCodexNativeSelector(ctx, rt, q)
	}
	if q.Agent != AgentClaude && q.Agent != AgentQoder {
		return NativeSelectorSnapshot{Unsupported: true, Reason: "native latest ordering has not been mapped for this provider"}
	}
	io, ok := rt.(RecoveryRuntime)
	if !ok {
		return NativeSelectorSnapshot{Unsupported: true, Reason: "bounded native selector IO unavailable"}
	}
	out := NativeSelectorSnapshot{Complete: true}
	metadata, err := newNativeSelectorRuntime(ctx, rt)
	if err != nil {
		return NativeSelectorSnapshot{Unsupported: true, Reason: err.Error()}
	}
	budget := RecoveryEvidence{}
	seen := map[string]bool{}
	for _, root := range rt.NativeRoots(q.Agent) {
		files, err := metadata.ListFiles(root)
		if err != nil {
			out.Complete = false
			out.Reason = "native metadata listing incomplete"
		}
		for _, e := range files {
			_, recognized := observeEntry(q.Agent, e)
			if !recognized {
				continue
			}
			// Child filenames have no independent native latest root eligibility.
			id := observationNativeID(q.Agent, e.Artifact)
			if id == "" {
				continue
			}
			var state *ScannerState
			cwd := ""
			excluded := false
			unmapped := false
			hasMessage := false
			err := readRecoverySnapshot(ctx, io, e, &budget, func(records []FramedJSONLRecord) error {
				for _, r := range records {
					var next ScannerState
					var err error
					if q.Agent == AgentClaude {
						next, _, err = ValidateClaudeDelta(e, state, []FramedJSONLRecord{r})
					} else {
						next, _, err = ValidateQoderDelta(e, state, []FramedJSONLRecord{r})
					}
					if err != nil {
						return err
					}
					state = &next
					var row struct {
						Type         string          `json:"type"`
						CWD          string          `json:"cwd"`
						RelocatedCWD string          `json:"relocatedCwd"`
						Sidechain    bool            `json:"isSidechain"`
						Team         string          `json:"teamName"`
						Kind         string          `json:"sessionKind"`
						Entrypoint   string          `json:"entrypoint"`
						Continued    string          `json:"continuedInSessionId"`
						Loop         bool            `json:"isLoopSession"`
						Bookkeeping  bool            `json:"bookkeepingOnly"`
						Message      json.RawMessage `json:"message"`
						Worktree     json.RawMessage `json:"worktreeSession"`
					}
					if err := json.Unmarshal(r.Bytes, &row); err != nil {
						return err
					}
					if row.CWD != "" {
						cwd = row.CWD
					}
					if row.RelocatedCWD != "" {
						cwd = row.RelocatedCWD
					}
					if row.Type == "user" || row.Type == "assistant" {
						hasMessage = true
					}
					if q.Agent == AgentClaude {
						if row.Sidechain || row.Team != "" || row.Kind == "daemon" || row.Kind == "daemon-worker" || row.Entrypoint == "sdk-cli" || row.Entrypoint == "sdk-ts" || row.Entrypoint == "sdk-py" || row.Loop || row.Bookkeeping {
							excluded = true
						}
						if row.Continued != "" || row.Kind != "" {
							unmapped = true
						}
					} else if len(row.Worktree) > 0 && string(row.Worktree) != "null" {
						unmapped = true
					}
				}
				return nil
			})
			if err != nil {
				out.Complete = false
				out.Reason = "native transcript metadata incomplete"
				continue
			}
			if state == nil || state.Disputed || !state.FirstRecordValidated || state.Role != RoleRoot {
				continue
			}
			physical, err := io.ResolveRecoveryCWD(ctx, cwd)
			if err != nil {
				out.Complete = false
				out.Reason = "native directory metadata unavailable"
				continue
			}
			if physical != q.CWD {
				continue
			}
			if excluded {
				continue
			}
			if unmapped {
				out.Unsupported = true
				out.Reason = "native latest requires provider lifecycle or worktree metadata not mapped by this adapter"
				continue
			}
			if !hasMessage {
				continue
			}
			if e.ModTime == nil || e.ModTime.IsZero() {
				out.Complete = false
				out.Reason = "native latest activity unavailable"
				continue
			}
			if seen[state.NativeID] {
				out.Complete = false
				out.Reason = "duplicate native root artifacts"
				continue
			}
			seen[state.NativeID] = true
			out.Candidates = append(out.Candidates, NativeSelectorCandidate{Agent: q.Agent, NativeID: state.NativeID, CWD: physical, UpdatedAt: *e.ModTime, Eligible: true})
		}
	}
	if ctx.Err() != nil {
		out.Complete = false
		out.Reason = ctx.Err().Error()
	}
	return out
}

func collectCodexNativeSelector(ctx context.Context, rt Runtime, q NativeSelectorRequest) NativeSelectorSnapshot {
	if !q.Config.ProviderResolved || !q.Config.SQLiteHomeResolved {
		return NativeSelectorSnapshot{Unsupported: true, Reason: "effective Codex provider and SQLite home are not resolved"}
	}
	queryRT, err := nativeSelectorCodexRuntime(rt, q.Config.SQLiteHome)
	if err != nil {
		return NativeSelectorSnapshot{Unsupported: true, Reason: err.Error()}
	}
	bounded, err := newNativeSelectorRuntime(ctx, queryRT)
	if err != nil {
		return NativeSelectorSnapshot{Unsupported: true, Reason: err.Error()}
	}
	source := "'cli','vscode'"
	if q.IncludeNonInteractive {
		source += ",'exec','mcp'"
	}
	quote := func(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }
	provider := ""
	if q.Config.Provider != "" {
		provider = " AND model_provider=" + quote(q.Config.Provider)
	}
	query := "SELECT id, rollout_path, updated_at_ms FROM threads WHERE archived=0 AND preview<>'' AND cwd=" + quote(q.CWD) + " AND source IN (" + source + ")" + provider + " ORDER BY updated_at_ms DESC LIMIT 2"
	result, err := bounded.QuerySQLite(Artifact{StorageRoot: "codex-state", RelativePath: "state_5.sqlite", Kind: ArtifactMetadata}, query, 1<<20)
	if err != nil {
		return NativeSelectorSnapshot{Reason: "Codex state metadata unavailable or unsupported"}
	}
	if len(result.Columns) != 3 || result.Columns[0] != "id" || result.Columns[1] != "rollout_path" || result.Columns[2] != "updated_at_ms" || len(result.Rows) > 2 {
		return NativeSelectorSnapshot{Reason: "unexpected Codex selector metadata shape"}
	}
	out := NativeSelectorSnapshot{Complete: true}
	for _, row := range result.Rows {
		if len(row) != 3 {
			return NativeSelectorSnapshot{Reason: "malformed Codex selector row"}
		}
		ms, err := strconv.ParseInt(row[2], 10, 64)
		if err != nil || ms <= 0 {
			return NativeSelectorSnapshot{Reason: "Codex native activity unavailable"}
		}
		if err := validateCodexSelectorPath(rt, row[0], row[1]); err != nil {
			return NativeSelectorSnapshot{Reason: "Codex selected rollout path is not an authorized canonical root"}
		}
		out.Candidates = append(out.Candidates, NativeSelectorCandidate{Agent: AgentCodex, NativeID: row[0], CWD: q.CWD, UpdatedAt: time.UnixMilli(ms), Eligible: true})
	}
	return out
}
