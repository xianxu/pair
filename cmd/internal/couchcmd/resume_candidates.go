package couchcmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/rowtext"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

const recoveryPreviewTimeout = couchmessage.RecoveryPreviewTimeout

func runResumeCandidatesCLI(inv cliInvocation, rt Runtime, out, diagnostic io.Writer, call messageCall) int {
	socket, err := couchmessage.SocketPath(rt.Getenv("COUCH_STORE_DIR"), "broker")
	if rt.Getenv("COUCH_STORE_DIR") == "" {
		err = fmt.Errorf("--resume-candidates requires a live Couch slot")
	}
	if err != nil {
		fmt.Fprintln(diagnostic, "couch:", err)
		return 1
	}
	request := callerIdentity(rt, "resume-candidates")
	request.Target = inv.ref
	if err = couchmessage.ValidateRequest(request); err != nil {
		fmt.Fprintln(diagnostic, "couch:", err)
		return 1
	}
	ctx, cancel := context.WithTimeout(context.Background(), recoveryPreviewTimeout)
	defer cancel()
	var response couchmessage.Response
	if err = call(ctx, socket, request, &response); err != nil {
		fmt.Fprintln(diagnostic, "couch:", err)
		return 1
	}
	if response.Code != "ok" || response.RecoveryPreview == nil {
		detail := response.Error
		if detail == "" {
			detail = "candidate preview unavailable (" + response.Code + ")"
		}
		fmt.Fprintln(diagnostic, "couch:", detail)
		return 1
	}
	preview := response.RecoveryPreview
	if inv.jsonOutput {
		if err = json.NewEncoder(out).Encode(preview); err != nil {
			fmt.Fprintln(diagnostic, err)
			return 1
		}
		return 0
	}
	fmt.Fprintln(out, "Recovery choices for", inv.ref, "(explicit selection required)")
	choices := []couchcore.RecoveryChoice{}
	if preview.Proven != nil {
		choices = append(choices, *preview.Proven)
	}
	candidateStart := len(choices)
	limit := 5
	if preview.Proven != nil {
		limit = 4
	}
	candidates := preview.Candidates[:min(len(preview.Candidates), limit)]
	choices = append(choices, candidates...)
	if preview.Fresh != nil {
		choices = append(choices, *preview.Fresh)
	}
	for i, c := range choices {
		label := c.Label
		nativeOnly := c.SourceKind == sessioninventory.RecoverySourceNative
		if nativeOnly && !strings.Contains(strings.ToLower(label), "unconfirmed") {
			label += " (unconfirmed)"
		}
		if !nativeOnly && i == candidateStart && len(preview.Candidates) > 0 {
			label = "best · " + label
		}
		activity := "unknown"
		if !c.LastActivity.IsZero() {
			activity = c.LastActivity.Local().Format("2006-01-02 15:04 MST")
		}
		evidence := "original saved screen"
		if c.NativeID == "" && c.Key == "" {
			evidence = "fresh conversation"
		}
		if c.NativeID != "" && c.Key == "" {
			evidence = "established native binding"
		}
		if c.NativeID != "" && c.Key != "" {
			evidence = fmt.Sprintf("matched %d/%d · tail %.0f%%", c.Matched, c.Total, c.TailMatch)
		}
		activityLabel := "last activity"
		if nativeOnly {
			activityLabel = "native activity"
			evidence = "native transcript · physical cwd " + c.CWD + " · launch association unknown"
		}
		command := "couch --resume " + inv.ref + " --candidate " + c.Key
		if c.Key == "" {
			if c.NativeID != "" {
				command = "couch --resume " + inv.ref
			} else {
				command = couchcore.SlotOperationCommand("reboot", inv.ref)
			}
		}
		if nativeOnly && c.Key == "" {
			command = "Refresh candidates before selecting this transcript."
		}
		fmt.Fprintf(out, "%d. %s [%s] %s\n   %s %s · %s\n   %s\n   %s\n", i+1, rowtext.Sanitize(label), rowtext.Sanitize(c.Agent), rowtext.Sanitize(c.NativeID), activityLabel, activity, rowtext.Sanitize(evidence), rowtext.Sanitize(c.Excerpt), command)
	}
	if !preview.Screen.LastActivity.IsZero() {
		source := preview.Screen.ActivitySource
		if source == "" {
			source = "capture-file activity"
		}
		fmt.Fprintln(out, rowtext.Sanitize(source), preview.Screen.LastActivity.Local().Format("2006-01-02 15:04 MST"))
	}
	for _, reason := range preview.Reasons {
		fmt.Fprintln(out, rowtext.Sanitize(reason))
	}
	if preview.Incomplete {
		fmt.Fprintln(out, "Candidate scan incomplete.")
	}
	if preview.Truncated {
		fmt.Fprintln(out, "Additional candidates omitted.")
	}
	return 0
}

// Resolve the durable slot row without filtering it through today's action list.
func recoveryTargetForReference(ctx context.Context, c *couchcore.Couch, reference string) (couchcore.ResumeTarget, error) {
	ref, ok, err := couchcore.ParseWorkspaceReference(reference)
	if err != nil || !ok || ref.Repo == "" {
		return couchcore.ResumeTarget{}, fmt.Errorf("recovery requires an exact repo:N slot")
	}
	path, ok, err := c.WorkspaceReferencePath(ctx, reference)
	if err != nil {
		return couchcore.ResumeTarget{}, err
	}
	if !ok || path == "" {
		return couchcore.ResumeTarget{}, fmt.Errorf("unknown slot %s", reference)
	}
	if ref.Number > 0 {
		return couchcore.ResumeTarget{Path: path}, nil
	}
	rows, err := c.ActionableThreadInventoryContext(ctx, nil)
	if err != nil {
		return couchcore.ResumeTarget{}, err
	}
	row, err := couchcore.SelectSlotRow(rows, ref.Number, path)
	if err != nil {
		return couchcore.ResumeTarget{}, err
	}
	return couchcore.ResumeTarget{Address: row.Address}, nil
}

func (s *messageService) handleResumeCandidates(ctx context.Context, r couchmessage.Request) couchmessage.Response {
	if err := couchmessage.ValidateRequest(r); err != nil {
		return couchmessage.Response{Code: "invalid-request", Error: err.Error()}
	}
	if _, err := s.liveCaller(ctx, r); err != nil {
		return couchmessage.Response{Code: "unavailable", Error: err.Error()}
	}
	if s.slotOps == nil || s.slotOps.preview == nil {
		return couchmessage.Response{Code: "unsupported", Error: "resume candidates unavailable"}
	}
	ctx, cancel := context.WithTimeout(ctx, recoveryPreviewTimeout)
	defer cancel()
	preview, err := s.slotOps.preview(ctx, r.Target)
	if err != nil {
		return couchmessage.Response{Code: "unavailable", Error: err.Error()}
	}
	return couchmessage.Response{Code: "ok", RecoveryPreview: &preview}
}
