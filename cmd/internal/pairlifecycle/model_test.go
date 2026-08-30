package pairlifecycle

import (
	"strings"
	"testing"
	"time"
)

func validQuitRequest() QuitRequest {
	return QuitRequest{
		SchemaVersion: SchemaVersion,
		Identity: Identity{
			Nonce:           "park-nonce_1",
			RepoScope:       "github_com-xianxu-pair",
			Tag:             "work_1",
			PID:             42,
			ProcessIdentity: "pid-start:123456",
		},
		Attempt:       1,
		Session:       "pair-work_1",
		Mode:          CleanupPreserveScrollback,
		CompletionKey: "quit-completion-1",
	}
}

func validQuitCompletion() QuitCompletion {
	r := validQuitRequest()
	return QuitCompletion{
		SchemaVersion: r.SchemaVersion,
		Identity:      r.Identity,
		Attempt:       r.Attempt,
		Session:       r.Session,
		Mode:          r.Mode,
		CompletionKey: r.CompletionKey,
		Outcome:       CompletionSuccess,
		CompletedAt:   time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC),
	}
}

func TestValidateQuitRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*QuitRequest)
		want   string
	}{
		{name: "schema", mutate: func(r *QuitRequest) { r.SchemaVersion++ }, want: "schema_version"},
		{name: "nonce empty", mutate: func(r *QuitRequest) { r.Identity.Nonce = "" }, want: "nonce"},
		{name: "nonce traversal", mutate: func(r *QuitRequest) { r.Identity.Nonce = "../park" }, want: "nonce"},
		{name: "repo scope", mutate: func(r *QuitRequest) { r.Identity.RepoScope = "bad/scope" }, want: "repo_scope"},
		{name: "tag", mutate: func(r *QuitRequest) { r.Identity.Tag = "bad/tag" }, want: "tag"},
		{name: "pid", mutate: func(r *QuitRequest) { r.Identity.PID = 0 }, want: "pid"},
		{name: "process identity", mutate: func(r *QuitRequest) { r.Identity.ProcessIdentity = "" }, want: "process_identity"},
		{name: "attempt", mutate: func(r *QuitRequest) { r.Attempt = 0 }, want: "attempt"},
		{name: "session empty", mutate: func(r *QuitRequest) { r.Session = "" }, want: "session"},
		{name: "session traversal", mutate: func(r *QuitRequest) { r.Session = "../pair" }, want: "session"},
		{name: "mode", mutate: func(r *QuitRequest) { r.Mode = "detach" }, want: "mode"},
		{name: "completion key empty", mutate: func(r *QuitRequest) { r.CompletionKey = "" }, want: "completion_key"},
		{name: "completion key traversal", mutate: func(r *QuitRequest) { r.CompletionKey = "../completion" }, want: "completion_key"},
	}

	if err := ValidateQuitRequest(validQuitRequest()); err != nil {
		t.Fatalf("valid request: %v", err)
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := validQuitRequest()
			test.mutate(&r)
			if err := ValidateQuitRequest(r); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("ValidateQuitRequest() error = %v, want containing %q", err, test.want)
			}
		})
	}
}

func TestValidateQuitCompletion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*QuitCompletion)
		want   string
	}{
		{name: "request fields", mutate: func(c *QuitCompletion) { c.Identity.PID = -1 }, want: "pid"},
		{name: "completed at", mutate: func(c *QuitCompletion) { c.CompletedAt = time.Time{} }, want: "completed_at"},
		{name: "outcome", mutate: func(c *QuitCompletion) { c.Outcome = "unknown" }, want: "outcome"},
		{name: "success with failure", mutate: func(c *QuitCompletion) { c.FailureCode = FailureTimeout }, want: "failure_code"},
		{name: "failure without code", mutate: func(c *QuitCompletion) { c.Outcome = CompletionFailure }, want: "failure_code"},
		{name: "unknown failure code", mutate: func(c *QuitCompletion) { c.Outcome = CompletionFailure; c.FailureCode = "other" }, want: "failure_code"},
	}

	if err := ValidateQuitCompletion(validQuitCompletion()); err != nil {
		t.Fatalf("valid completion: %v", err)
	}
	failure := validQuitCompletion()
	failure.Outcome = CompletionFailure
	failure.FailureCode = FailureCleanupFailed
	if err := ValidateQuitCompletion(failure); err != nil {
		t.Fatalf("valid failure completion: %v", err)
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := validQuitCompletion()
			test.mutate(&c)
			if err := ValidateQuitCompletion(c); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("ValidateQuitCompletion() error = %v, want containing %q", err, test.want)
			}
		})
	}
}

func TestMatchQuitCompletion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*QuitCompletion)
		want   string
	}{
		{name: "schema", mutate: func(c *QuitCompletion) { c.SchemaVersion++ }, want: "schema_version"},
		{name: "nonce", mutate: func(c *QuitCompletion) { c.Identity.Nonce = "other" }, want: "nonce"},
		{name: "repo scope", mutate: func(c *QuitCompletion) { c.Identity.RepoScope = "other" }, want: "repo_scope"},
		{name: "tag", mutate: func(c *QuitCompletion) { c.Identity.Tag = "other" }, want: "tag"},
		{name: "pid", mutate: func(c *QuitCompletion) { c.Identity.PID++ }, want: "pid"},
		{name: "process identity", mutate: func(c *QuitCompletion) { c.Identity.ProcessIdentity = "other" }, want: "process_identity"},
		{name: "attempt", mutate: func(c *QuitCompletion) { c.Attempt++ }, want: "attempt"},
		{name: "session", mutate: func(c *QuitCompletion) { c.Session = "other" }, want: "session"},
		{name: "mode", mutate: func(c *QuitCompletion) { c.Mode = "other" }, want: "mode"},
		{name: "completion key", mutate: func(c *QuitCompletion) { c.CompletionKey = "other" }, want: "completion_key"},
	}

	r := validQuitRequest()
	c := validQuitCompletion()
	if err := MatchQuitCompletion(r, c); err != nil {
		t.Fatalf("matching completion: %v", err)
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := c
			test.mutate(&got)
			if err := MatchQuitCompletion(r, got); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("MatchQuitCompletion() error = %v, want containing %q", err, test.want)
			}
		})
	}

	beforeRequest, beforeCompletion := r, c
	_ = MatchQuitCompletion(r, c)
	if r != beforeRequest || c != beforeCompletion {
		t.Fatal("matching mutated caller-owned records")
	}
}
