package operationalrun

import (
	"context"
	"errors"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const RetentionPeriod = 30 * 24 * time.Hour
const MaxRunsPerTask = 300

type Cursor struct {
	Hub       string `json:"hub,omitempty"`
	Task      string `json:"task,omitempty"`
	Run       string `json:"run,omitempty"`
	AfterTask bool   `json:"after_task,omitempty"`
	AfterHub  bool   `json:"after_hub,omitempty"`
}
type Row struct {
	Key      RunKey `json:"key"`
	Path     string `json:"path"`
	Bytes    int64  `json:"bytes"`
	Eligible bool   `json:"eligible"`
	Reason   string `json:"reason,omitempty"`
}

func (c Cursor) token() string { return c.Hub + "/" + c.Task + "/" + c.Run }
func (c Cursor) valid() bool {
	if c.Hub == "" {
		return c.Task == "" && c.Run == "" && !c.AfterHub && !c.AfterTask
	}
	if !component.MatchString(c.Hub) {
		return false
	}
	if c.AfterHub {
		return c.Task == "" && c.Run == ""
	}
	return component.MatchString(c.Task) && len(c.Run) <= 255 && !strings.ContainsAny(c.Run, "/\\\x00")
}

// SweepPage visits bounded run rows in lexical order. Each task inventory is
// bounded separately; damaged tasks produce a blocked row and do not hide peers.
func (s Store) SweepPage(ctx context.Context, now time.Time, after Cursor, apply bool, limit int) (rows []Row, next Cursor, complete bool, err error) {
	next = after
	if !after.valid() {
		return nil, next, false, errors.New("invalid operational cursor")
	}
	if limit < 1 || limit > 100 {
		limit = 100
	}
	if e := ctx.Err(); e != nil {
		return nil, after, false, e
	}
	work := 0
	if !filepath.IsAbs(s.Namespace) || filepath.Clean(s.Namespace) != s.Namespace {
		return nil, after, false, errors.New("invalid operational namespace")
	}
	root := filepath.Join(s.Namespace, "schedule")
	hubs, e := names(root)
	if e != nil {
		return nil, after, false, e
	}
	for _, hub := range hubs {
		if e = ctx.Err(); e != nil {
			return rows, after, false, e
		}
		if !component.MatchString(hub) {
			continue
		}
		if after.Hub != "" && (hub < after.Hub || hub == after.Hub && after.AfterHub) {
			continue
		}
		if len(rows) >= limit || work >= limit {
			return rows, next, false, nil
		}
		tasks, e := names(filepath.Join(root, hub))
		if e != nil {
			rows = append(rows, Row{Path: filepath.Join(root, hub), Reason: e.Error()})
			next = Cursor{Hub: hub, AfterHub: true}
			work++
			continue
		}
		if len(tasks) == 0 {
			next = Cursor{Hub: hub, AfterHub: true}
			work++
			continue
		}
		for _, task := range tasks {
			if !component.MatchString(task) {
				continue
			}
			base := Cursor{Hub: hub, Task: task}
			if after.Hub == hub && (task < after.Task || task == after.Task && after.AfterTask) {
				continue
			}
			if e = ctx.Err(); e != nil {
				return rows, after, false, e
			}
			if len(rows) >= limit || work >= limit {
				return rows, next, false, nil
			}
			dir := filepath.Join(root, hub, task)
			unlock, e := taskLock(ctx, dir, false)
			if errors.Is(e, ErrBusy) || ctx.Err() != nil {
				return rows, after, false, e
			}
			if e != nil {
				rows = append(rows, Row{Path: dir, Reason: e.Error()})
				next = Cursor{Hub: hub, Task: task, AfterTask: true}
				continue
			}
			if apply {
				e = cleanupStages(ctx, dir)
				if e == nil {
					e = s.recoverCollection(ctx, dir)
				}
			}
			if e != nil {
				unlock()
				if ctx.Err() != nil {
					return rows, after, false, ctx.Err()
				}
				rows = append(rows, Row{Path: dir, Reason: e.Error()})
				next = Cursor{Hub: hub, Task: task, AfterTask: true}
				continue
			}
			ns, e := names(filepath.Join(dir, "runs"))
			if e != nil {
				unlock()
				rows = append(rows, Row{Path: dir, Reason: e.Error()})
				next = Cursor{Hub: hub, Task: task, AfterTask: true}
				continue
			}
			records := map[string]Record{}
			failures := map[string]error{}
			var ordered []Record
			for _, id := range ns {
				if e = ctx.Err(); e != nil {
					unlock()
					return rows, after, false, e
				}
				key := RunKey{Hub: hub, Task: task, ID: id}
				r, e := s.readMetadata(key)
				if e != nil {
					failures[id] = e
					continue
				}
				records[id] = r
				if terminal(r) && !r.EndedAt.After(now) {
					ordered = append(ordered, r)
				}
			}
			sort.Slice(ordered, func(i, j int) bool {
				if ordered[i].EndedAt.Equal(ordered[j].EndedAt) {
					return ordered[i].Key.ID > ordered[j].Key.ID
				}
				return ordered[i].EndedAt.After(ordered[j].EndedAt)
			})
			excess := map[string]bool{}
			for i := MaxRunsPerTask; i < len(ordered); i++ {
				excess[ordered[i].Key.ID] = true
			}
			for _, id := range ns {
				cur := base
				cur.Run = id
				if cur.token() <= after.token() {
					continue
				}
				if len(rows) >= limit {
					unlock()
					return rows, next, false, nil
				}
				if e = ctx.Err(); e != nil {
					unlock()
					return rows, after, false, e
				}
				key := RunKey{Hub: hub, Task: task, ID: id}
				row := Row{Key: key, Path: filepath.Join(dir, "runs", id)}
				if e = failures[id]; e != nil {
					row.Reason = e.Error()
				} else {
					r := records[id]
					row.Bytes = int64(0)
					if b, e := bundleIdentity(row.Path); e == nil {
						row.Bytes = b.Record.Size + b.Output.Size
					}
					switch {
					case !terminal(r):
						row.Reason = "run is active, incomplete or uncertain"
					case r.EndedAt.After(now):
						row.Reason = "run completion is in the future"
					case !now.Before(r.EndedAt.Add(RetentionPeriod)):
						row.Eligible = true
						row.Reason = "operational run expired after 30 days"
					case excess[id]:
						row.Eligible = true
						row.Reason = "outside newest 300 completed runs"
					default:
						row.Reason = "within operational retention"
					}
					if row.Eligible && apply {
						if e = s.collect(ctx, key, dir); e != nil {
							if ctx.Err() != nil {
								unlock()
								return rows, after, false, ctx.Err()
							}
							row.Eligible = false
							row.Reason = e.Error()
						}
					}
				}
				rows = append(rows, row)
				work++
				next = cur
			}
			unlock()
			next = Cursor{Hub: hub, Task: task, AfterTask: true}
			work++
		}
	}
	return rows, next, true, nil
}
