package couchcore

import (
	"fmt"
	"strconv"
	"strings"
)

// WorkspaceReference addresses a workspace number, including primary (:0).
// An empty Repo requests the caller's verified repository context.
type WorkspaceReference struct {
	Repo   string
	Number int
}

func (r WorkspaceReference) String() string { return r.Repo + ":" + strconv.Itoa(r.Number) }

// ParseWorkspaceReference recognizes explicit :N and repo:N references. Ordinary
// paths and opaque tags return matched=false and remain the caller's concern.
// A recognized but malformed reference returns matched=true with an error.
func ParseWorkspaceReference(raw string) (ref WorkspaceReference, matched bool, err error) {
	if !strings.Contains(raw, ":") {
		return ref, false, nil
	}
	if !strings.HasPrefix(raw, ":") && strings.ContainsAny(raw, `/\\`) {
		return ref, false, nil
	}
	repo, number, _ := strings.Cut(raw, ":")
	invalid := func() (WorkspaceReference, bool, error) {
		return WorkspaceReference{}, true, fmt.Errorf("invalid workspace reference %q: use :0, :N or repo:N with a canonical nonnegative number", raw)
	}
	if repo != "" && (!workspaceRepoName(repo) || strings.TrimSpace(repo) != repo) {
		return invalid()
	}
	if number == "" || (len(number) > 1 && number[0] == '0') {
		return invalid()
	}
	for _, c := range number {
		if c < '0' || c > '9' {
			return invalid()
		}
	}
	n, e := strconv.Atoi(number)
	if e != nil || n < 0 {
		return invalid()
	}
	return WorkspaceReference{Repo: repo, Number: n}, true, nil
}
