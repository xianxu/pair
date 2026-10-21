package launcher

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"unicode/utf8"

	"github.com/xianxu/pair/cmd/internal/zellijexec"
)

// NativeQueryEvidence is diagnostic data only, never authority for native effects.
// A nil exit code means the subprocess exit was not observed.
type NativeQueryEvidence struct {
	Args                             []string
	Executable                       string
	Stdout, Stderr                   []byte
	StdoutBytes, StderrBytes         int
	StdoutTruncated, StderrTruncated bool
	ExitCode                         *int
	Version                          string
}

// Bounded returns an independently owned, bounded evidence snapshot.
func (e NativeQueryEvidence) Bounded() NativeQueryEvidence { return boundNativeQueryEvidence(e) }

func boundNativeQueryEvidence(e NativeQueryEvidence) NativeQueryEvidence {
	e.StdoutBytes = max(e.StdoutBytes, len(e.Stdout))
	e.StderrBytes = max(e.StderrBytes, len(e.Stderr))
	e.StdoutTruncated = e.StdoutTruncated || e.StdoutBytes > 4096
	e.StderrTruncated = e.StderrTruncated || e.StderrBytes > 4096
	e.Stdout = append([]byte(nil), e.Stdout[:min(len(e.Stdout), 4096)]...)
	e.Stderr = append([]byte(nil), e.Stderr[:min(len(e.Stderr), 4096)]...)
	e.Executable = queryMetadataPrefix(e.Executable, 1024)
	var args []string
	remaining := 1024 - len(e.Executable)
	for _, arg := range e.Args[:min(len(e.Args), 32)] {
		if remaining == 0 {
			break
		}
		arg = queryMetadataPrefix(arg, remaining)
		args = append(args, arg)
		remaining -= len(arg)
	}
	e.Args = args
	e.Version = queryMetadataPrefix(e.Version, 256)
	if e.ExitCode != nil {
		v := *e.ExitCode
		e.ExitCode = &v
	}
	return e
}

func queryMetadataPrefix(s string, limit int) string {
	s = strings.ToValidUTF8(s, "�")
	s = s[:min(len(s), limit)]
	for !utf8.ValidString(s) {
		s = s[:len(s)-1]
	}
	return s
}

type nativeQueryFailure struct {
	diagnostic string
	evidence   NativeQueryEvidence
}

func (e *nativeQueryFailure) Error() string                            { return e.diagnostic }
func (e *nativeQueryFailure) NativeQueryEvidence() NativeQueryEvidence { return e.evidence.Bounded() }

type diagnosticCauseError struct {
	error
	cause error
}

func (e *diagnosticCauseError) Unwrap() error { return errors.Join(e.error, e.cause) }

// WithDiagnosticCause keeps the concise caller message while retaining evidence.
func (o SessionOwnerObservation) WithDiagnosticCause(err error) error {
	if o.DiagnosticCause == nil {
		return err
	}
	return &diagnosticCauseError{error: err, cause: o.DiagnosticCause}
}

type queryPrefixWriter struct{ data []byte }

func (w *queryPrefixWriter) Write(p []byte) (int, error) {
	n := len(p)
	w.data = append(w.data, p[:min(len(p), 256-len(w.data))]...)
	return n, nil
}

// Failure-only version lookup uses the exact observed executable and remaining
// proof budget. Its absence is explicit and never changes ownership authority.
func nativeQueryFailureEvidence(ctx context.Context, event ZellijQueryObservation, diagnostic string) error {
	e := NativeQueryEvidence{Args: event.Args, Executable: event.Executable, Stdout: event.Stdout, Stderr: event.Stderr, StdoutBytes: event.StdoutBytes, StderrBytes: event.StderrBytes, StdoutTruncated: event.StdoutTruncated, StderrTruncated: event.StderrTruncated, Version: "unavailable"}
	if event.ExitCode >= 0 {
		code := event.ExitCode
		e.ExitCode = &code
	}
	e.Version = queryZellijVersion(ctx, event.Executable)
	return &nativeQueryFailure{diagnostic: diagnostic, evidence: e.Bounded()}
}

// Snapshot before chained observers receive their mutable copies. Keep the real
// executable intact until the failure-only version read has used that same path.
func snapshotNativeQueryObservation(event ZellijQueryObservation) ZellijQueryObservation {
	e := NativeQueryEvidence{Args: event.Args, Stdout: event.Stdout, Stderr: event.Stderr, StdoutBytes: event.StdoutBytes, StderrBytes: event.StderrBytes, StdoutTruncated: event.StdoutTruncated, StderrTruncated: event.StderrTruncated}.Bounded()
	event.Args, event.Stdout, event.Stderr = e.Args, e.Stdout, e.Stderr
	event.StdoutBytes, event.StderrBytes = e.StdoutBytes, e.StderrBytes
	event.StdoutTruncated, event.StderrTruncated = e.StdoutTruncated, e.StderrTruncated
	return event
}

func queryZellijVersion(ctx context.Context, executable string) string {
	if err := ctx.Err(); err != nil {
		return "unavailable: " + err.Error()
	}
	if executable == "" {
		return "unavailable"
	}
	cmd := exec.CommandContext(ctx, executable, "--version")
	var output, stderr queryPrefixWriter
	cmd.Stdout = &output
	cmd.Stderr = &stderr
	if err := zellijexec.Run(cmd); err != nil {
		return "unavailable: " + err.Error()
	}
	if v := strings.TrimSpace(string(output.data)); v != "" {
		return v
	}
	return "unavailable"
}
