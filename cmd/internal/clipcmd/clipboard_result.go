package clipcmd

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"strings"

	"github.com/xianxu/pair/cmd/internal/rowtext"
)

const clipboardStderrLimit = 4096

// ClipboardCopyMetadata can cross diagnostic boundaries: it has no input or
// free-form tool output. ExitCode is absent when the tool did not exit normally.
type ClipboardCopyMetadata struct {
	Tool            string `json:"tool,omitempty"`
	Executable      string `json:"executable,omitempty"`
	Status          string `json:"status"`
	Started         bool   `json:"started"`
	ExitCode        *int   `json:"exit_code,omitempty"`
	TimedOut        bool   `json:"timeout"`
	Canceled        bool   `json:"canceled"`
	StderrBytes     int64  `json:"stderr_bytes"`
	StderrHash      string `json:"stderr_hash,omitempty"`
	StderrTruncated bool   `json:"stderr_truncated"`
}

// ClipboardCopyResult keeps local display evidence apart from log-safe metadata.
// Stderr is never sent to an observer or serialized as a supervision event.
type ClipboardCopyResult struct {
	Metadata ClipboardCopyMetadata
	Stderr   string
}

type ClipboardCopyError struct {
	Result ClipboardCopyResult
	Cause  error
}

func (e *ClipboardCopyError) Unwrap() error { return e.Cause }
func (e *ClipboardCopyError) Error() string {
	tool := e.Result.Metadata.Tool
	if tool == "" {
		tool = "local clipboard"
	}
	message := fmt.Sprintf("%s: %s", tool, e.Cause)
	if e.Result.Stderr != "" {
		message += ": " + e.Result.Stderr
	}
	if e.Result.Metadata.StderrTruncated {
		message += " [stderr truncated]"
	}
	return message
}

// clipboardStderrSummary drains all output without retaining more than one
// bounded prefix. Hashing the full stream distinguishes equal-prefix failures.
type clipboardStderrSummary struct {
	prefix []byte
	bytes  int64
	digest hash.Hash
}

func (s *clipboardStderrSummary) Write(p []byte) (int, error) {
	if s.digest == nil {
		s.digest = sha256.New()
	}
	_, _ = s.digest.Write(p)
	s.bytes += int64(len(p))
	remaining := clipboardStderrLimit - len(s.prefix)
	s.prefix = append(s.prefix, p[:min(remaining, len(p))]...)
	return len(p), nil
}
func (s *clipboardStderrSummary) apply(result *ClipboardCopyResult) {
	result.Metadata.StderrBytes = s.bytes
	result.Metadata.StderrTruncated = s.bytes > clipboardStderrLimit
	if s.digest != nil {
		result.Metadata.StderrHash = hex.EncodeToString(s.digest.Sum(nil)[:8])
	}
	result.Stderr = strings.TrimSpace(rowtext.Sanitize(string(s.prefix)))
}
