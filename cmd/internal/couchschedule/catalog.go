package couchschedule

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/osfs"
	"golang.org/x/sys/unix"
)

type CatalogGit interface {
	RunContext(context.Context, string, ...string) (string, error)
}

// Catalog only visits configured roots. ReadFile receives an already admitted
// regular descriptor; the default reader checks stable bounded metadata.
type Catalog struct {
	Git      CatalogGit
	ReadFile func(context.Context, *os.File, int64) ([]byte, error)
}

func (c Catalog) read(ctx context.Context, dir *os.File, name string, max int64) ([]byte, error) {
	f, err := openScheduleChild(dir, name, unix.O_RDONLY)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	reader := c.ReadFile
	if reader == nil {
		reader = osfs.ReadOpenedRegularMetadata
	}
	return reader(ctx, f, max)
}

func catalogRevision(raw, team []byte, operator, reason string) string {
	bytes, _ := json.Marshal([]string{string(raw), string(team), operator, reason})
	hash := sha256.Sum256(bytes)
	return hex.EncodeToString(hash[:])
}

func (c Catalog) Discover(ctx context.Context, configuredRoots []string) ([]Declaration, error) {
	if ctx == nil {
		return nil, fmt.Errorf("catalog requires context")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var result []Declaration
	seenRoots := map[string]bool{}
	for _, root := range configuredRoots {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if seenRoots[root] {
			continue
		}
		seenRoots[root] = true
		hub, err := openScheduleDir(ctx, root, false)
		if err != nil {
			return nil, err
		}
		scheduled, err := openScheduleDir(ctx, filepath.Join(root, "scripts", "scheduled"), false)
		if errors.Is(err, os.ErrNotExist) {
			hub.Close()
			continue
		}
		if err != nil {
			hub.Close()
			return nil, err
		}
		entries, err := scheduled.ReadDir(257)
		if err != nil && !errors.Is(err, io.EOF) {
			scheduled.Close()
			hub.Close()
			return nil, err
		}
		if len(result)+len(entries) > 256 {
			scheduled.Close()
			hub.Close()
			return nil, fmt.Errorf("configured scheduler exceeds 256 tasks")
		}
		// File.ReadDir preserves directory order; deterministic ordering is supplied
		// below by filename before ownership and revision observations.
		sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
		folded := map[string]int{}
		for _, entry := range entries {
			folded[strings.ToLower(entry.Name())]++
		}
		start := len(result)
		raws := map[string][]byte{}
		needsTeam := false
		for _, entry := range entries {
			if err := ctx.Err(); err != nil {
				scheduled.Close()
				hub.Close()
				return nil, err
			}
			name := entry.Name()
			d := Declaration{Key: TaskKey{Hub: root, Task: name}, Allowed: true}
			if !ValidTaskName(name) {
				d.Key.Task = refusedTaskIdentity(name)
			}
			refuse := func(err error) { d.Allowed = false; d.Reason = err.Error() }
			if !ValidTaskName(name) || folded[strings.ToLower(name)] > 1 {
				refuse(fmt.Errorf("invalid or case-colliding task name %q", name))
			} else {
				dir, openErr := openScheduleChild(scheduled, name, unix.O_RDONLY|unix.O_DIRECTORY)
				if openErr != nil {
					refuse(fmt.Errorf("task directory %s: %w", name, openErr))
				} else {
					raw, readErr := c.read(ctx, dir, "schedule", maxTaskSpecBytes)
					raws[name] = raw
					if readErr != nil {
						refuse(fmt.Errorf("schedule %s: %w", name, readErr))
					} else {
						d.Spec, err = ParseTaskSpec(raw)
						if err != nil {
							refuse(err)
						}
					}
					if d.Allowed {
						main, mainErr := openScheduleChild(dir, "main.go", unix.O_RDONLY)
						if mainErr != nil {
							refuse(fmt.Errorf("main.go: %w", mainErr))
						} else {
							st, statErr := main.Stat()
							main.Close()
							if statErr != nil || !st.Mode().IsRegular() {
								refuse(fmt.Errorf("main.go must be regular"))
							}
						}
					}
					dir.Close()
				}
			}
			if d.Allowed && d.Spec.Scope == "team" {
				needsTeam = true
			}
			result = append(result, d)
		}
		var team []byte
		var ops []string
		var owner string
		var teamErr error
		if needsTeam {
			team, teamErr = c.read(ctx, hub, "team.md", 1024*1024)
			if teamErr == nil {
				ops, teamErr = ParseOperators(team)
			}
			if teamErr == nil {
				git := c.Git
				if git == nil {
					git = couchcore.ExecGit{}
				}
				name, gitErr := git.RunContext(ctx, root, "config", "--get", "user.name")
				if gitErr != nil {
					teamErr = fmt.Errorf("effective git user.name: %w", gitErr)
				} else {
					owner = OperatorSlug(name)
				}
			}
		}
		for i := start; i < len(result); i++ {
			d := &result[i]
			if d.Allowed && d.Spec.Scope == "team" {
				if teamErr != nil {
					d.Allowed = false
					d.Reason = teamErr.Error()
				} else {
					d.Allowed, d.Reason = OwnerAdmission(d.Spec, ops, owner)
				}
			}
			d.Revision = catalogRevision(raws[d.Key.Task], team, owner, d.Reason)
		}
		scheduled.Close()
		hub.Close()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return result, nil
}
