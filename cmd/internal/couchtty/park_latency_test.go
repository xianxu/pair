package couchtty

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

func TestParkLatencySmoke(t *testing.T) {
	if os.Getenv("PAIR_PARK_LATENCY_SMOKE") != "1" {
		t.Skip("set PAIR_PARK_LATENCY_SMOKE=1 on the target macOS Apple M2 Max")
	}
	ns, err := couchcore.ResolveCouchNamespace(t.TempDir(), "/repo")
	if err != nil {
		t.Fatal(err)
	}
	store := couchcore.NewThreadStore(ns)
	policy := couchcore.PolicyResult{
		PolicyVersion: 1, PolicyDigest: strings.Repeat("a", 64), RepoIdentity: "repo", AdmissionKey: "repo",
		Capacity: couchcore.PolicyCapacity{Kind: couchcore.CapacityUnbounded}, OnCapacity: couchcore.CapacityActionUnknown,
	}
	profile := couchcore.LaunchProfile{Agent: "claude", Argv: []string{}}
	record, err := store.CreateThread(couchcore.ThreadRecord{
		SchemaVersion: couchcore.ThreadSchemaVersion,
		Address:       couchcore.ThreadAddress{RepoScope: "816fc349d3faebf8", Tag: "couch-0102030405060708"},
		StartingPath:  "/repo", WorkingPath: "/repo", CreatedAt: time.Unix(1, 0).UTC(), LastActiveAt: time.Unix(1, 0).UTC(),
		Incarnations:        []couchcore.ThreadIncarnation{{PID: 42, Identity: "helper", State: couchcore.IncarnationLive, Policy: &policy, LaunchProfile: &profile}},
		LatestLaunchProfile: &profile, Revision: 1,
	})
	if err != nil {
		t.Fatal(err)
	}

	queue := newOperationQueue(1)
	stop := make(chan struct{})
	defer close(stop)
	go queue.Run(stop)
	feedback := make([]time.Duration, 0, 100)
	commits := make([]time.Duration, 0, 100)
	for sample := 0; sample < 100; sample++ {
		started := time.Now()
		accepted, err := queue.Enqueue(operationRequest{key: fmt.Sprintf("park-%d", sample), name: "park", run: func() error {
			current, err := store.GetThread(record.Address)
			if err != nil {
				return err
			}
			_, err = store.UpdateExistingThread(record.Address, current.Revision, func(next *couchcore.ThreadRecord) error {
				next.Description = fmt.Sprintf("sample-%d", sample)
				return nil
			})
			return err
		}})
		feedback = append(feedback, time.Since(started))
		if !accepted || err != nil {
			t.Fatalf("sample %d enqueue = %v, %v", sample, accepted, err)
		}
		result := <-queue.results
		commits = append(commits, time.Since(started))
		if result.err != nil {
			t.Fatalf("sample %d: %v", sample, result.err)
		}
	}

	overload := newOperationQueue(1)
	_, _ = overload.Enqueue(operationRequest{key: "held", name: "park", run: func() error { return nil }})
	accepted, overloadErr := overload.Enqueue(operationRequest{key: "overflow", name: "park", run: func() error {
		t.Fatal("overloaded request executed")
		return nil
	}})
	if accepted || !errors.Is(overloadErr, errOperationQueueOverloaded) {
		t.Fatalf("overload = accepted %v, err %v", accepted, overloadErr)
	}
	feedbackP95, commitP95, commitMax := nearestRankP95(feedback), nearestRankP95(commits), maxDuration(commits)
	t.Logf("target=macOS Apple M2 Max feedback_p95=%s commit_p95=%s commit_max=%s overload=refused", feedbackP95, commitP95, commitMax)
	if feedbackP95 >= 100*time.Millisecond || commitP95 >= 100*time.Millisecond || commitMax >= time.Second {
		t.Fatalf("park operating envelope exceeded")
	}
}

func nearestRankP95(values []time.Duration) time.Duration {
	copy := append([]time.Duration(nil), values...)
	sort.Slice(copy, func(i, j int) bool { return copy[i] < copy[j] })
	return copy[(95*len(copy)+99)/100-1]
}

func maxDuration(values []time.Duration) time.Duration {
	var max time.Duration
	for _, value := range values {
		if value > max {
			max = value
		}
	}
	return max
}
