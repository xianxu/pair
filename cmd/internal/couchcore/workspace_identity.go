package couchcore

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/xianxu/pair/cmd/internal/strictjson"
)

type WorkspaceIdentity struct {
	SchemaVersion   int              `json:"schema_version"`
	Repo            string           `json:"repo"`
	RepoIdentity    string           `json:"repo_identity"`
	PrimaryRoot     string           `json:"primary_root"`
	FleetRoot       string           `json:"fleet_root"`
	EnvironmentRoot string           `json:"environment_root"`
	EnvironmentHost *EnvironmentHost `json:"environment_host,omitempty"`
	WorktreeRoot    string           `json:"worktree_root"`
	Kind            string           `json:"kind"`
	Address         *string          `json:"address"`
	Slot            *int             `json:"slot"`
	Branch          *string          `json:"branch"`
	Head            *string          `json:"head"`
	RestingBranch   *string          `json:"resting_branch"`
}
type EnvironmentHost struct {
	Repo         string `json:"repo"`
	Slot         int    `json:"slot"`
	RepoIdentity string `json:"repo_identity"`
	PrimaryRoot  string `json:"primary_root"`
	WorktreeRoot string `json:"worktree_root"`
}

// ParseWorkspaceIdentity validates SDLC's observation transport; it does not
// establish filesystem membership or grant permission for subsequent mutation.
func ParseWorkspaceIdentity(raw []byte) (WorkspaceIdentity, error) {
	var id WorkspaceIdentity
	if len(raw) > 1<<20 {
		return id, fmt.Errorf("workspace identity exceeds 1 MiB")
	}
	if err := strictjson.Decode(raw, &id); err != nil {
		return WorkspaceIdentity{}, fmt.Errorf("workspace identity: %w", err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return WorkspaceIdentity{}, err
	}
	for _, key := range []string{"schema_version", "repo", "repo_identity", "primary_root", "fleet_root", "environment_root", "worktree_root", "kind", "address", "slot", "branch", "head", "resting_branch"} {
		if _, ok := fields[key]; !ok {
			return WorkspaceIdentity{}, fmt.Errorf("workspace identity: missing %s", key)
		}
	}
	if err := id.validate(); err != nil {
		return WorkspaceIdentity{}, fmt.Errorf("workspace identity: %w", err)
	}
	return id, nil
}
func workspaceAbsolute(p string) bool {
	return filepath.IsAbs(p) && filepath.Clean(p) == p && utf8.ValidString(p) && !strings.ContainsRune(p, 0)
}
func workspaceRepoName(s string) bool {
	return s != "" && s != "." && s != ".." && utf8.ValidString(s) && !strings.ContainsAny(s, "/\\:") && strings.IndexFunc(s, unicode.IsControl) < 0
}
func (id WorkspaceIdentity) validate() error {
	if id.SchemaVersion != 2 {
		return fmt.Errorf("unsupported schema version %d", id.SchemaVersion)
	}
	if !workspaceRepoName(id.Repo) || filepath.Base(id.PrimaryRoot) != id.Repo {
		return fmt.Errorf("inconsistent repository name")
	}
	for _, p := range []string{id.RepoIdentity, id.PrimaryRoot, id.FleetRoot, id.EnvironmentRoot, id.WorktreeRoot} {
		if !workspaceAbsolute(p) {
			return fmt.Errorf("invalid absolute path %q", p)
		}
	}
	if id.Head != nil && !validWorkspaceOID(*id.Head) {
		return fmt.Errorf("invalid HEAD OID")
	}
	if id.Branch != nil && (*id.Branch == "" || strings.IndexFunc(*id.Branch, unicode.IsControl) >= 0) {
		return fmt.Errorf("invalid branch")
	}
	host := id.EnvironmentHost
	if host == nil {
		if id.EnvironmentRoot != id.FleetRoot || filepath.Dir(id.PrimaryRoot) != id.FleetRoot {
			return fmt.Errorf("inconsistent ordinary environment")
		}
	} else {
		if !workspaceRepoName(host.Repo) || host.Slot <= 0 || !workspaceAbsolute(host.RepoIdentity) || host.PrimaryRoot != filepath.Join(id.FleetRoot, host.Repo) {
			return fmt.Errorf("invalid environment host")
		}
		env := filepath.Join(id.FleetRoot, "worktree", host.Repo+"-slot"+strconv.Itoa(host.Slot))
		if id.EnvironmentRoot != env || host.WorktreeRoot != filepath.Join(env, host.Repo) {
			return fmt.Errorf("inconsistent numbered environment")
		}
		if id.RepoIdentity == host.RepoIdentity {
			if id.PrimaryRoot != host.PrimaryRoot || id.Repo != host.Repo {
				return fmt.Errorf("inconsistent host repository")
			}
		} else if id.PrimaryRoot != filepath.Join(env, id.Repo) {
			return fmt.Errorf("inconsistent dependency primary")
		}
	}
	switch id.Kind {
	case "primary", "slot":
		if id.Slot == nil || id.Address == nil || id.RestingBranch == nil {
			return fmt.Errorf("missing workspace address")
		}
		n := *id.Slot
		if n < 0 || *id.Address != id.Repo+":"+strconv.Itoa(n) {
			return fmt.Errorf("inconsistent workspace address")
		}
		if id.Kind == "primary" {
			if n != 0 || host != nil || id.WorktreeRoot != id.PrimaryRoot || *id.RestingBranch != "main" {
				return fmt.Errorf("inconsistent primary")
			}
		} else {
			rest := "main-slot" + strconv.Itoa(n)
			if n <= 0 || host == nil || host.Slot != n || host.Repo != id.Repo || host.RepoIdentity != id.RepoIdentity || host.PrimaryRoot != id.PrimaryRoot || host.WorktreeRoot != id.WorktreeRoot || *id.RestingBranch != rest || id.Head == nil {
				return fmt.Errorf("inconsistent slot")
			}
			if id.Branch != nil {
				b := *id.Branch
				if b == "main" {
					return fmt.Errorf("slot on primary resting branch")
				}
				if strings.HasPrefix(b, "main-slot") {
					suffix := strings.TrimPrefix(b, "main-slot")
					m, e := strconv.Atoi(suffix)
					if e == nil && m > 0 && strconv.Itoa(m) == suffix && b != rest {
						return fmt.Errorf("slot on another resting branch")
					}
				}
			}
		}
	case "dependency", "worktree":
		if id.Address != nil || id.Slot != nil || id.RestingBranch != nil {
			return fmt.Errorf("ordinary workspace has numbered address")
		}
		if id.Kind == "dependency" {
			if host == nil || id.RepoIdentity == host.RepoIdentity || id.WorktreeRoot != id.PrimaryRoot || filepath.Dir(id.PrimaryRoot) != id.EnvironmentRoot {
				return fmt.Errorf("inconsistent dependency")
			}
		} else if id.WorktreeRoot == id.PrimaryRoot {
			return fmt.Errorf("ordinary worktree equals primary")
		}
	default:
		return fmt.Errorf("unknown kind %q", id.Kind)
	}
	return nil
}
func validWorkspaceOID(s string) bool {
	if len(s) != 40 && len(s) != 64 {
		return false
	}
	nonzero := false
	for _, r := range s {
		if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f') {
			return false
		}
		nonzero = nonzero || r != '0'
	}
	return nonzero
}
