package couchcmd

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/rowtext"
)

// Each untrusted line is sanitized independently; only this renderer supplies
// terminal line breaks and indentation, never the captured process controls.
func renderLaunchFailureText(w io.Writer, label, text string) {
	if text == "" {
		return
	}
	fmt.Fprintf(w, "  %s:\n", label)
	for _, line := range strings.Split(text, "\n") {
		fmt.Fprintf(w, "    %s\n", rowtext.Sanitize(line))
	}
}
func launchFailureExit(code *int) string {
	if code == nil {
		return "unobserved"
	}
	return strconv.Itoa(*code)
}
func renderLaunchFailures(w io.Writer, failures []couchcore.LaunchFailure, readError string) {
	if readError != "" {
		renderLaunchFailureText(w, "lifecycle diagnostics unavailable", readError)
	}
	for _, f := range failures {
		if f.SupersededAt != nil {
			continue
		}
		if f.Operation != "" || f.Phase != "" || f.Slot != "" {
			phase := f.Phase
			if phase == "" {
				phase = f.Operation
			}
			outcome := f.Outcome
			if outcome == "" {
				outcome = "failed"
			}
			fmt.Fprintf(w, "lifecycle failure %s (%s, %s) at %s; attempt %s\n", rowtext.Sanitize(f.Slot), rowtext.Sanitize(phase), rowtext.Sanitize(outcome), f.FailedAt.Format(time.RFC3339Nano), rowtext.Sanitize(f.Nonce))
		} else {
			fmt.Fprintf(w, "launch failure %s/%s (%s, %s) at %s; attempt %s\n", rowtext.Sanitize(f.Address.RepoScope), rowtext.Sanitize(string(f.Address.Tag)), rowtext.Sanitize(f.Agent), rowtext.Sanitize(f.Stage), f.FailedAt.Format(time.RFC3339Nano), rowtext.Sanitize(f.Nonce))
		}
		for _, holder := range f.Holders {
			fmt.Fprintf(w, "  holder: %s pid %d identity %s\n", rowtext.Sanitize(holder.Kind), holder.Process.PID, rowtext.Sanitize(holder.Process.Identity))
		}
		fmt.Fprintf(w, "  helper started: %t\n", f.HelperStarted)
		fmt.Fprintf(w, "  exit before cleanup: %s; exit after cleanup: %s\n", launchFailureExit(f.BeforeExit), launchFailureExit(f.AfterExit))
		renderLaunchFailureText(w, "operation", f.Operation)
		renderLaunchFailureText(w, "agent", f.Agent)
		renderLaunchFailureText(w, "launch stage", f.Stage)
		renderLaunchFailureText(w, "thread", f.Address.RepoScope+"/"+string(f.Address.Tag))
		renderLaunchFailureText(w, "starting path", f.StartingPath)
		renderLaunchFailureText(w, "launch error", f.LaunchError)
		renderLaunchFailureText(w, "final operation error", f.FinalError)
		renderLaunchFailureText(w, "cleanup error", f.CleanupError)
		output := f.Output
		if len(output) > couchcore.LaunchFailureOutputLimit {
			output = output[len(output)-couchcore.LaunchFailureOutputLimit:]
		}
		label := rowtext.Sanitize(f.Stream)
		if label == "" {
			label = "output"
		}
		renderLaunchFailureText(w, label+" tail", string(output))
	}
}
