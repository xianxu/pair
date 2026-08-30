// Package pairlifecycle owns the versioned, process-independent protocol used
// to request and prove Pair's full-quit lifecycle. It is intentionally a leaf
// package so launcher and Couch can share the vocabulary without importing one
// another.
package pairlifecycle

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const SchemaVersion = 1

type CleanupMode string

const CleanupPreserveScrollback CleanupMode = "preserve-scrollback"

type CompletionOutcome string

const (
	CompletionSuccess CompletionOutcome = "success"
	CompletionFailure CompletionOutcome = "failure"
)

type FailureCode string

const (
	FailureRequestPublishFailed   FailureCode = "request_publish_failed"
	FailureCleanupFailed          FailureCode = "cleanup_failed"
	FailureTimeout                FailureCode = "timeout"
	FailureCompletionMissing      FailureCode = "completion_missing"
	FailureStaleCompletion        FailureCode = "stale_completion"
	FailureRevisionConflict       FailureCode = "revision_conflict"
	FailureReplacementIncarnation FailureCode = "replacement_incarnation"
)

// Identity is stable for every attempt in one park transaction.
type Identity struct {
	Nonce           string `json:"nonce"`
	RepoScope       string `json:"repo_scope"`
	Tag             string `json:"tag"`
	PID             int    `json:"pid"`
	ProcessIdentity string `json:"process_identity"`
}

type QuitRequest struct {
	SchemaVersion int         `json:"schema_version"`
	Identity      Identity    `json:"identity"`
	Attempt       uint64      `json:"attempt"`
	Session       string      `json:"session"`
	Mode          CleanupMode `json:"mode"`
	CompletionKey string      `json:"completion_key"`
}

type QuitCompletion struct {
	SchemaVersion int               `json:"schema_version"`
	Identity      Identity          `json:"identity"`
	Attempt       uint64            `json:"attempt"`
	Session       string            `json:"session"`
	Mode          CleanupMode       `json:"mode"`
	CompletionKey string            `json:"completion_key"`
	Outcome       CompletionOutcome `json:"outcome"`
	FailureCode   FailureCode       `json:"failure_code,omitempty"`
	CompletedAt   time.Time         `json:"completed_at"`
}

var safeComponent = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

func ValidateQuitRequest(request QuitRequest) error {
	if request.SchemaVersion != SchemaVersion {
		return fmt.Errorf("quit request schema_version %d is unsupported", request.SchemaVersion)
	}
	if err := validateIdentity(request.Identity); err != nil {
		return err
	}
	if request.Attempt == 0 {
		return fmt.Errorf("quit request attempt must be positive")
	}
	if !validSession(request.Session) {
		return fmt.Errorf("quit request session %q is invalid", request.Session)
	}
	if request.Mode != CleanupPreserveScrollback {
		return fmt.Errorf("quit request mode %q is unsupported", request.Mode)
	}
	if !safeComponent.MatchString(request.CompletionKey) {
		return fmt.Errorf("quit request completion_key %q is invalid", request.CompletionKey)
	}
	return nil
}

func ValidateQuitCompletion(completion QuitCompletion) error {
	requestFields := QuitRequest{
		SchemaVersion: completion.SchemaVersion,
		Identity:      completion.Identity,
		Attempt:       completion.Attempt,
		Session:       completion.Session,
		Mode:          completion.Mode,
		CompletionKey: completion.CompletionKey,
	}
	if err := ValidateQuitRequest(requestFields); err != nil {
		return fmt.Errorf("quit completion: %w", err)
	}
	if completion.CompletedAt.IsZero() {
		return fmt.Errorf("quit completion completed_at is required")
	}
	switch completion.Outcome {
	case CompletionSuccess:
		if completion.FailureCode != "" {
			return fmt.Errorf("quit completion failure_code must be empty for success")
		}
	case CompletionFailure:
		if !validFailureCode(completion.FailureCode) {
			return fmt.Errorf("quit completion failure_code %q is invalid", completion.FailureCode)
		}
	default:
		return fmt.Errorf("quit completion outcome %q is invalid", completion.Outcome)
	}
	return nil
}

// MatchQuitCompletion accepts only a valid completion repeating every request
// binding. Both arguments are values so validation cannot rewrite caller data.
func MatchQuitCompletion(request QuitRequest, completion QuitCompletion) error {
	if err := ValidateQuitRequest(request); err != nil {
		return err
	}
	if err := ValidateQuitCompletion(completion); err != nil {
		return err
	}
	checks := []struct {
		name string
		ok   bool
	}{
		{"schema_version", request.SchemaVersion == completion.SchemaVersion},
		{"nonce", request.Identity.Nonce == completion.Identity.Nonce},
		{"repo_scope", request.Identity.RepoScope == completion.Identity.RepoScope},
		{"tag", request.Identity.Tag == completion.Identity.Tag},
		{"pid", request.Identity.PID == completion.Identity.PID},
		{"process_identity", request.Identity.ProcessIdentity == completion.Identity.ProcessIdentity},
		{"attempt", request.Attempt == completion.Attempt},
		{"session", request.Session == completion.Session},
		{"mode", request.Mode == completion.Mode},
		{"completion_key", request.CompletionKey == completion.CompletionKey},
	}
	for _, check := range checks {
		if !check.ok {
			return fmt.Errorf("quit completion %s does not match request", check.name)
		}
	}
	return nil
}

func validateIdentity(identity Identity) error {
	for _, field := range []struct {
		name, value string
	}{
		{"nonce", identity.Nonce},
		{"repo_scope", identity.RepoScope},
		{"tag", identity.Tag},
	} {
		if !safeComponent.MatchString(field.value) {
			return fmt.Errorf("quit identity %s %q is invalid", field.name, field.value)
		}
	}
	if identity.PID <= 0 {
		return fmt.Errorf("quit identity pid must be positive")
	}
	if strings.TrimSpace(identity.ProcessIdentity) == "" {
		return fmt.Errorf("quit identity process_identity is required")
	}
	return nil
}

func validSession(session string) bool {
	return session != "" && session != "." && session != ".." && filepath.Base(session) == session && !strings.ContainsRune(session, '\x00')
}

func validFailureCode(code FailureCode) bool {
	switch code {
	case FailureRequestPublishFailed, FailureCleanupFailed, FailureTimeout,
		FailureCompletionMissing, FailureStaleCompletion, FailureRevisionConflict,
		FailureReplacementIncarnation:
		return true
	default:
		return false
	}
}
