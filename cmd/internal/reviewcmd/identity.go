package reviewcmd

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const identityLimit = 8 << 20
const identityCommitLimit = 10000

type ReviewIdentity struct {
	Status            string `json:"status"`
	Repo              string `json:"repo"`
	Branch            string `json:"branch"`
	Head              string `json:"head"`
	File              string `json:"file"`
	Diagnostic        string `json:"diagnostic,omitempty"`
	LatestAgentBody   string `json:"latest_agent_body"`
	LatestAgentCommit string `json:"latest_agent_commit,omitempty"`
	HumanRound        int    `json:"human_round"`
	AgentRound        int    `json:"agent_round"`
}

type identityRound struct {
	commit, subject, body string
	paths                 []string
}

func classifyIdentity(branch string, rounds []identityRound) ReviewIdentity {
	out := ReviewIdentity{Branch: branch, Status: "missing"}
	if !strings.HasPrefix(branch, "review/") || len(branch) == len("review/") {
		out.Status = "non_review"
		return out
	}
	pattern := regexp.MustCompile(`^review\(` + regexp.QuoteMeta(strings.TrimPrefix(branch, "review/")) + `\): (human|agent) r([0-9]+)( — .*)?$`)
	paths := map[string]bool{}
	for _, r := range rounds {
		match := pattern.FindStringSubmatch(r.subject)
		if match == nil {
			continue
		}
		n, err := strconv.Atoi(match[2])
		if err != nil {
			out.Status = "invalid"
			out.Diagnostic = "invalid round number"
			return out
		}
		if match[1] == "agent" {
			if n > out.AgentRound {
				out.AgentRound = n
			}
			if out.LatestAgentCommit == "" {
				out.LatestAgentCommit = r.commit
				out.LatestAgentBody = r.body
			}
		} else if n > out.HumanRound {
			out.HumanRound = n
		}
		for _, p := range r.paths {
			if !safeIdentityPath(p) {
				out.Status = "invalid"
				out.Diagnostic = "unsafe review path"
				return out
			}
			paths[p] = true
		}
	}
	if len(paths) > 1 {
		out.Status = "ambiguous"
		out.Diagnostic = "matching review rounds name multiple documents"
		return out
	}
	for p := range paths {
		out.File = p
		out.Status = "resolved"
	}
	return out
}
func safeIdentityPath(p string) bool {
	return p != "" && !strings.ContainsRune(p, 0) && !filepath.IsAbs(p) && filepath.Clean(p) == p && p != "." && p != ".." && !strings.HasPrefix(p, "../")
}

// Each record begins with NUL, then hash, subject and body as NUL fields.
// Git inserts one newline before the NUL-delimited changed paths. Empty paths
// cannot occur, so the next empty token unambiguously begins another record.
func parseIdentityHistory(raw string) ([]identityRound, error) {
	if raw == "" {
		return nil, nil
	}
	parts := strings.Split(raw, "\x00")
	var out []identityRound
	for i := 0; i < len(parts)-1; {
		if parts[i] != "" || i+3 >= len(parts) {
			return nil, fmt.Errorf("malformed Git history framing")
		}
		r := identityRound{commit: parts[i+1], subject: parts[i+2], body: parts[i+3]}
		i += 4
		if r.commit == "" {
			return nil, fmt.Errorf("missing commit identity")
		}
		first := true
		for i < len(parts)-1 && parts[i] != "" {
			p := parts[i]
			if first {
				if !strings.HasPrefix(p, "\n") {
					return nil, fmt.Errorf("malformed Git path framing")
				}
				p = p[1:]
				first = false
			}
			r.paths = append(r.paths, p)
			i++
		}
		out = append(out, r)
		if len(out) > identityCommitLimit {
			return nil, fmt.Errorf("review history exceeds %d matching commits", identityCommitLimit)
		}
	}
	if parts[len(parts)-1] != "" {
		return nil, fmt.Errorf("truncated Git history")
	}
	return out, nil
}

func resolveIdentity(rt Runtime, dir, selected, expectedHead string) ReviewIdentity {
	out := ReviewIdentity{Status: "invalid"}
	fail := func(err error) ReviewIdentity { out.Status = "invalid"; out.Diagnostic = err.Error(); return out }
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	remaining := identityLimit
	git := func(args ...string) (string, error) {
		s, err := rt.GitContext(ctx, remaining, dir, args...)
		remaining -= len(s)
		if remaining < 0 {
			return "", fmt.Errorf("review history exceeds 8 MiB")
		}
		return s, err
	}
	root, err := git("rev-parse", "--show-toplevel")
	if err != nil {
		return fail(fmt.Errorf("cannot read repository: %w", err))
	}
	out.Repo, err = rt.CanonicalDir(strings.TrimSpace(root))
	if err != nil {
		return fail(err)
	}
	dir = out.Repo
	branch, err := git("symbolic-ref", "--quiet", "--short", "HEAD")
	if err != nil {
		return fail(fmt.Errorf("cannot resolve branch (detached HEAD or Git failure): %w", err))
	}
	out.Branch = strings.TrimSpace(branch)
	head, err := git("rev-parse", "--verify", "HEAD")
	if err != nil {
		return fail(err)
	}
	out.Head = strings.TrimSpace(head)
	if !strings.HasPrefix(out.Branch, "review/") {
		out.Status = "non_review"
		return out
	}
	// The broad fixed-string grep bounds collection to the current slug; the
	// pure classifier enforces the exact subject grammar, never the body grep.
	raw, err := git("log", out.Head, "--fixed-strings", "--grep=review("+strings.TrimPrefix(out.Branch, "review/")+"): ", "--max-count=10001", "--format=%x00%H%x00%s%x00%b", "--name-only", "-z", "--no-renames", "--root")
	if err != nil {
		return fail(fmt.Errorf("cannot read review history: %w", err))
	}
	rounds, err := parseIdentityHistory(raw)
	if err != nil {
		return fail(err)
	}
	result := classifyIdentity(out.Branch, rounds)
	result.Repo = out.Repo
	result.Head = out.Head
	out = result
	if out.Status == "missing" && selected != "" {
		if expectedHead != out.Head {
			return fail(fmt.Errorf("selection receipt HEAD changed"))
		}
		out.File = selected
		out.Status = "resolved"
	}
	if out.Status == "resolved" {
		if !safeIdentityPath(out.File) {
			return fail(fmt.Errorf("unsafe selected path"))
		}
		tracked, e := git("ls-files", "-z", "--error-unmatch", "--", out.File)
		if e != nil || tracked != out.File+"\x00" {
			return fail(fmt.Errorf("review document is not tracked"))
		}
		if err = rt.RegularFileWithin(out.Repo, out.File); err != nil {
			return fail(err)
		}
	}
	after, err := git("rev-parse", "--verify", "HEAD")
	if err != nil || strings.TrimSpace(after) != out.Head {
		return fail(fmt.Errorf("checkout HEAD changed during resolution"))
	}
	after, err = git("symbolic-ref", "--quiet", "--short", "HEAD")
	if err != nil || strings.TrimSpace(after) != out.Branch {
		return fail(fmt.Errorf("checkout branch changed during resolution"))
	}
	return out
}
