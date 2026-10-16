package sessioninventory

import (
	"context"
	"encoding/json"
)

func collectNativeSelector(ctx context.Context, rt Runtime, q NativeSelectorRequest) NativeSelectorSnapshot {
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
