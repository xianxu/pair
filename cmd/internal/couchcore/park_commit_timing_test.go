package couchcore

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// This opt-in measurement deliberately uses the user's cache volume, not the
// optional test RAM disk: RAM-backed fsync cannot measure the incident's SSD
// constraint. Only a newly allocated private directory is read or written.
func TestParkCommitTimingRealStore(t *testing.T) {
	if os.Getenv("PAIR_PARK_COMMIT_TIMING") != "1" {
		t.Skip("opt-in SSD durability measurement for #827")
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		t.Fatal(err)
	}
	root, err := os.MkdirTemp(cache, "pair-827-commit-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(root); err != nil {
			t.Error(err)
		}
	})
	for _, n := range []int{15, 30} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			namespace, err := ResolveCouchNamespace(filepath.Join(root, fmt.Sprint(n)), root)
			if err != nil {
				t.Fatal(err)
			}
			store := NewThreadStore(namespace)
			var stagedBytes atomic.Int64
			store.hooks.AfterPublicationWrite = func(string) error {
				info, err := os.Stat(store.publicationPath())
				if err != nil {
					return err
				}
				if stagedBytes.Add(info.Size()) > 64<<20 {
					return fmt.Errorf("measurement staging exceeded 64 MiB")
				}
				return nil
			}
			records := make([]ThreadRecord, n)
			for i := range records {
				record := validThreadRecord(t)
				record.Address.Tag = ThreadTag(fmt.Sprintf("couch-%016x", i+1))
				record.StartingPath, record.WorkingPath = namespace.Dir(), namespace.Dir()
				profile := LaunchProfile{Agent: []string{"claude", "codex"}[i%2]}
				record.LatestLaunchProfile = &profile
				record.Incarnations = []ThreadIncarnation{{
					PID: 1000000 + i, Identity: fmt.Sprintf("fixture-pair-%d", i),
					State: IncarnationLive, LaunchProfile: &profile,
				}}
				records[i], err = store.CreateThread(record)
				if err != nil {
					t.Fatal(err)
				}
			}
			type sample struct {
				Tag       ThreadTag `json:"tag"`
				ElapsedNS int64     `json:"elapsed_ns"`
				Phase     ParkPhase `json:"phase"`
				Error     string    `json:"error,omitempty"`
			}
			samples := make([]sample, n)
			units := make(chan struct{}, LifecycleParallelism)
			var active, peak atomic.Int32
			var wg sync.WaitGroup
			beforeBytes := stagedBytes.Load()
			started := time.Now()
			for i, record := range records {
				units <- struct{}{}
				wg.Add(1)
				go func() {
					defer wg.Done()
					defer func() { <-units }()
					inFlight := active.Add(1)
					defer active.Add(-1)
					for old := peak.Load(); inFlight > old; old = peak.Load() {
						if peak.CompareAndSwap(old, inFlight) {
							break
						}
					}
					identity := ParkIdentity{
						Nonce: fmt.Sprintf("park-%032x", i+1), Address: record.Address,
						PID: record.Incarnations[0].PID, ProcessIdentity: record.Incarnations[0].Identity,
					}
					begin := time.Now()
					committed, err := store.BeginPark(record.Address, record.Revision, identity)
					samples[i] = sample{Tag: record.Address.Tag, ElapsedNS: time.Since(begin).Nanoseconds()}
					if err != nil {
						samples[i].Error = err.Error()
					} else if committed.Park != nil {
						samples[i].Phase = committed.Park.Phase
					}
				}()
			}
			wg.Wait()
			wall := time.Since(started)
			elapsed := make([]int64, len(samples))
			for i, sample := range samples {
				elapsed[i] = sample.ElapsedNS
				if sample.Error != "" || sample.Phase != ParkRequested {
					t.Errorf("commit %s phase=%s error=%s", sample.Tag, sample.Phase, sample.Error)
				}
			}
			sort.Slice(elapsed, func(i, j int) bool { return elapsed[i] < elapsed[j] })
			body, err := json.Marshal(map[string]any{
				"slots": n, "compiler": runtime.Version(), "root": namespace.Dir(),
				"parallelism": LifecycleParallelism, "peak": peak.Load(), "wall_ns": wall.Nanoseconds(),
				"p50_ns": timingNearestRank(elapsed, .50), "p95_ns": timingNearestRank(elapsed, .95),
				"max_ns": elapsed[len(elapsed)-1], "staged_payload_bytes": stagedBytes.Load() - beforeBytes,
				"current_threshold_ns": parkCommitDeadline.Nanoseconds(), "samples": samples,
			})
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("PARK_COMMIT_TIMING %s", body)
		})
	}
}

func timingNearestRank(sorted []int64, quantile float64) int64 {
	return sorted[int(math.Ceil(float64(len(sorted))*quantile))-1]
}

func TestTimingNearestRank(t *testing.T) {
	sorted := []int64{1, 2, 3, 4, 5}
	if got := timingNearestRank(sorted, .50); got != 3 {
		t.Fatalf("median=%d", got)
	}
	if got := timingNearestRank(sorted, .95); got != 5 {
		t.Fatalf("p95=%d", got)
	}
}
