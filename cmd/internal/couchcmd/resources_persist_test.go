package couchcmd

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

func resourcePersistenceDir(t *testing.T) string {
	t.Helper()
	d, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return d
}
func TestResourcePersistenceCorruptHistoryPreservesCheckpointAndAlerts(t *testing.T) {
	d, _, endpoint, cp, _ := resourceBrokerRig(t)
	dir := resourcePersistenceDir(t)
	cp.State.Episodes[0].Effect.Workspace = resourcePersistenceWorkspace(dir)
	cp.State.Provenance = []couchcore.ResourceProvenance{{ResourceRoot: couchcore.ResourceRoot{Key: couchcore.ResourceProcessKey{PID: 200, Identity: "old"}, Slot: "pair:9", Workspace: resourcePersistenceWorkspace(dir)}, Root: couchcore.ResourceProcessKey{PID: 200, Identity: "old"}}}
	p := newResourcePersistence(dir, d)
	now := time.Now()
	state, _, err := p.persist(context.Background(), cp.State, couchcore.ResourceView{SampledAt: now})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-endpoint.deliveries:
	case <-time.After(time.Second):
		t.Fatal("initial alert missing")
	}
	if err := os.WriteFile(filepath.Join(dir, "resources", "history.json"), []byte("broken history"), 0600); err != nil {
		t.Fatal(err)
	}
	recovered := newResourcePersistence(dir, d)
	if len(recovered.checkpoint.State.Episodes) != 1 || recovered.checkpoint.State.Episodes[0].Phase != "notified" {
		t.Fatal("lost valid suppression checkpoint")
	}
	if len(recovered.checkpoint.State.Provenance) != 1 || recovered.checkpoint.State.Provenance[0].Key != cp.State.Provenance[0].Key || recovered.checkpoint.State.Previous != nil {
		t.Fatal("lost valid provenance or restored rate baseline")
	}
	next := state.Episodes[0]
	next.Key = "rss/201/new"
	next.Phase = "pending"
	next.Delivery = ""
	effect := *next.Effect
	effect.ID = "resource-new-after-history-corruption"
	next.Effect = &effect
	state.Episodes = append(state.Episodes, next)
	state, cost, err := recovered.persist(context.Background(), state, couchcore.ResourceView{SampledAt: now.Add(time.Minute)})
	if err == nil || !strings.Contains(err.Error(), "continuity") {
		t.Fatal("missing explicit history continuity loss", err)
	}
	if cost.StorageWrites < 2 || len(state.Episodes) != 2 || state.Episodes[1].Phase != "notified" {
		t.Fatalf("recovery disabled persistence/alerts: %+v %+v %v", cost, state, err)
	}
	select {
	case m := <-endpoint.deliveries:
		if m.ID != effect.ID {
			t.Fatal("duplicated old admission", m.ID)
		}
	case <-time.After(time.Second):
		t.Fatal("new eligible alert suppressed")
	}
	select {
	case m := <-endpoint.deliveries:
		t.Fatal("duplicate admission", m.ID)
	default:
	}
	raw, err := os.ReadFile(filepath.Join(dir, "resources", "history.json"))
	if err != nil || couchcore.ValidateResourceHistory(raw) != nil {
		t.Fatal("history not rebuilt", string(raw), err)
	}
	if !strings.Contains(string(raw), "continuity") {
		t.Fatal("history lost recovery diagnostic")
	}
	again := newResourcePersistence(dir, d)
	if len(again.checkpoint.State.Episodes) != 2 {
		t.Fatal("repaired checkpoint not restartable")
	}
}
func TestResourcePersistenceCorruptCheckpointStartsFreshAndVisible(t *testing.T) {
	d, _, endpoint, cp, _ := resourceBrokerRig(t)
	dir := resourcePersistenceDir(t)
	cp.State.Episodes[0].Effect.Workspace = resourcePersistenceWorkspace(dir)
	if err := os.Mkdir(filepath.Join(dir, "resources"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "resources", "checkpoint.json"), []byte("broken checkpoint"), 0600); err != nil {
		t.Fatal(err)
	}
	p := newResourcePersistence(dir, d)
	if len(p.checkpoint.State.Provenance) != 0 || len(p.checkpoint.State.Episodes) != 0 || p.checkpoint.State.Previous != nil {
		t.Fatal("fabricated recovered state")
	}
	_, cost, err := p.persist(context.Background(), cp.State, couchcore.ResourceView{SampledAt: time.Now()})
	if err == nil || !strings.Contains(err.Error(), "continuity") || !strings.Contains(err.Error(), "deduplication") {
		t.Fatal("missing continuity policy", err)
	}
	if cost.StorageWrites < 2 {
		t.Fatal("corruption permanently disabled fresh monitoring", cost)
	}
	select {
	case <-endpoint.deliveries:
	case <-time.After(time.Second):
		t.Fatal("fresh episode never admitted")
	}
}
func TestResourcePersistenceUnsafeCheckpointWaitsThenRecovers(t *testing.T) {
	d, _, endpoint, cp, _ := resourceBrokerRig(t)
	dir := resourcePersistenceDir(t)
	cp.State.Episodes[0].Effect.Workspace = resourcePersistenceWorkspace(dir)
	resources := filepath.Join(dir, "resources")
	if err := os.Mkdir(resources, 0700); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(dir, "outside")
	raw, err := json.Marshal(cp)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(outside, raw, 0600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(resources, "checkpoint.json")
	if err = os.Symlink(outside, path); err != nil {
		t.Fatal(err)
	}
	p := newResourcePersistence(dir, d)
	_, cost, err := p.persist(context.Background(), couchcore.ResourceState{}, couchcore.ResourceView{SampledAt: time.Now()})
	if err == nil || cost.StorageWrites != 0 {
		t.Fatal("unsafe checkpoint overwritten", cost, err)
	}
	got, err := os.ReadFile(outside)
	if err != nil || string(got) != string(raw) {
		t.Fatal("symlink target mutated")
	}
	select {
	case <-endpoint.deliveries:
		t.Fatal("alert without checkpoint authority")
	default:
	}
	if err = os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	state, _, _ := p.persist(context.Background(), couchcore.ResourceState{}, couchcore.ResourceView{SampledAt: time.Now()})
	if len(state.Episodes) != 1 {
		t.Fatal("recovered checkpoint was not restored")
	}
	_, _, err = p.persist(context.Background(), state, couchcore.ResourceView{SampledAt: time.Now().Add(time.Minute)})
	if err != nil {
		t.Fatal("read failure latched permanently", err)
	}
	select {
	case <-endpoint.deliveries:
	case <-time.After(time.Second):
		t.Fatal("recovered eligible alert missing")
	}
}

func resourcePersistenceWorkspace(dir string) couchcore.WorkspaceIdentity {
	primary := filepath.Join(dir, "pair")
	slot := 0
	address := "pair:0"
	rest := "main"
	return couchcore.WorkspaceIdentity{SchemaVersion: 2, Repo: "pair", RepoIdentity: filepath.Join(primary, ".git"), PrimaryRoot: primary, FleetRoot: dir, EnvironmentRoot: dir, WorktreeRoot: primary, Kind: "primary", Slot: &slot, Address: &address, RestingBranch: &rest}
}

func TestResourcePersistenceUnsafeHistoryDoesNotBlockSafeCheckpoint(t *testing.T) {
	d, _, endpoint, cp, _ := resourceBrokerRig(t)
	dir := resourcePersistenceDir(t)
	cp.State.Episodes[0].Effect.Workspace = resourcePersistenceWorkspace(dir)
	p := newResourcePersistence(dir, d)
	raw, err := encodeResourceCheckpoint(*cp)
	if err != nil {
		t.Fatal(err)
	}
	if err = p.store.SaveCheckpoint(raw); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(dir, "outside")
	original := []byte(`{"unrelated":true}`)
	if err = os.WriteFile(outside, original, 0600); err != nil {
		t.Fatal(err)
	}
	historyPath := filepath.Join(dir, "resources", "history.json")
	if err = os.Symlink(outside, historyPath); err != nil {
		t.Fatal(err)
	}
	p = newResourcePersistence(dir, d)
	now := time.Now()
	state, cost, err := p.persist(context.Background(), p.checkpoint.State, couchcore.ResourceView{SampledAt: now})
	if err == nil || cost.StorageWrites < 1 || state.Episodes[0].Phase != "notified" {
		t.Fatal("unsafe history disabled safe checkpoint/alert", cost, err)
	}
	select {
	case <-endpoint.deliveries:
	case <-time.After(time.Second):
		t.Fatal("eligible alert missing")
	}
	got, err := os.ReadFile(outside)
	if err != nil || string(got) != string(original) {
		t.Fatal("unsafe history target overwritten")
	}
	if err = os.Remove(historyPath); err != nil {
		t.Fatal(err)
	}
	_, cost, err = p.persist(context.Background(), state, couchcore.ResourceView{SampledAt: now.Add(time.Minute)})
	if err != nil || cost.StorageWrites != 2 {
		t.Fatal("safe history did not resume", cost, err)
	}
	select {
	case <-endpoint.deliveries:
		t.Fatal("recovery duplicated admission")
	default:
	}
}
