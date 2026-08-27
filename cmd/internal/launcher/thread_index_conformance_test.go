package launcher_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/launcher"
)

func TestStandalonePairReadsCouchThreadStoreAndPreservesScopedArtifacts(t *testing.T) {
	root := t.TempDir()
	repo := filepath.Join(root, "repo")
	if err := os.MkdirAll(repo, 0o700); err != nil {
		t.Fatal(err)
	}
	scope, err := launcher.ResolveRepoScope(repo)
	if err != nil {
		t.Fatal(err)
	}
	storeDir := filepath.Join(root, "couch")
	namespace, err := couchcore.ResolveCouchNamespace(storeDir, root)
	if err != nil {
		t.Fatal(err)
	}
	store := couchcore.NewThreadStore(namespace)
	record, err := store.CreateThread(couchcore.ThreadRecord{
		SchemaVersion: couchcore.ThreadSchemaVersion,
		Address: couchcore.ThreadAddress{
			RepoScope: scope.Key,
			Tag:       "couch-0102030405060708",
		},
		StartingPath: repo,
		WorkingPath:  repo,
		CreatedAt:    time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC),
		Revision:     1,
	})
	if err != nil {
		t.Fatal(err)
	}
	name := "compiler"
	if _, err := store.ApplyThreadMetadata(record.Address, record.Revision, couchcore.ThreadMetadataPatch{Name: &name}); err != nil {
		t.Fatal(err)
	}

	globalData := filepath.Join(root, "pair-data")
	paths := launcher.NewScopedPaths(globalData, scope, string(record.Address.Tag))
	if err := os.MkdirAll(paths.ScopeDir(), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.Draft(), []byte("durable draft"), 0o600); err != nil {
		t.Fatal(err)
	}

	index, err := launcher.LoadThreadIndex(storeDir, func(path string) (string, error) {
		raw, err := os.ReadFile(path)
		return string(raw), err
	})
	if err != nil {
		t.Fatal(err)
	}
	matches, err := launcher.ResolveThreadIndexReference(index.Entries, scope.Key, "compiler")
	if err != nil || len(matches) != 1 || matches[0].Address.Tag != string(record.Address.Tag) {
		t.Fatalf("standalone resolution = %+v, %v", matches, err)
	}
	if raw, err := os.ReadFile(paths.Draft()); err != nil || string(raw) != "durable draft" {
		t.Fatalf("scoped artifact changed during lookup: %q, %v", raw, err)
	}
}

func TestCouchAndStandalonePairRejectTheSameInvalidPersistedRecords(t *testing.T) {
	type mutation func(map[string]any)
	incarnation := func(record map[string]any) map[string]any {
		return record["incarnations"].([]any)[0].(map[string]any)
	}
	start := func(record map[string]any) map[string]any {
		return incarnation(record)["start"].(map[string]any)
	}
	policy := func(record map[string]any) map[string]any {
		return incarnation(record)["policy"].(map[string]any)
	}
	mutations := map[string]mutation{
		"missing starting path":        func(r map[string]any) { delete(r, "starting_path") },
		"missing working path":         func(r map[string]any) { delete(r, "working_path") },
		"missing creation time":        func(r map[string]any) { delete(r, "created_at") },
		"missing revision":             func(r map[string]any) { delete(r, "revision") },
		"missing claim generation":     func(r map[string]any) { delete(r, "claim_generation") },
		"schema version":               func(r map[string]any) { r["schema_version"] = 2 },
		"invalid repository scope":     func(r map[string]any) { r["address"].(map[string]any)["repo_scope"] = "/" },
		"invalid tag":                  func(r map[string]any) { r["address"].(map[string]any)["tag"] = "/" },
		"relative starting path":       func(r map[string]any) { r["starting_path"] = "repo" },
		"relative working path":        func(r map[string]any) { r["working_path"] = "repo" },
		"zero creation time":           func(r map[string]any) { r["created_at"] = "0001-01-01T00:00:00Z" },
		"zero revision":                func(r map[string]any) { r["revision"] = 0 },
		"zero claim generation":        func(r map[string]any) { r["claim_generation"] = 0 },
		"path address mismatch":        func(r map[string]any) { r["address"].(map[string]any)["tag"] = "couch-1112131415161718" },
		"invalid incarnation state":    func(r map[string]any) { incarnation(r)["state"] = "done" },
		"negative incarnation pid":     func(r map[string]any) { incarnation(r)["pid"] = -1 },
		"start outside creating":       func(r map[string]any) { incarnation(r)["state"] = "live" },
		"invalid start nonce":          func(r map[string]any) { start(r)["nonce"] = "/" },
		"missing start owner pid":      func(r map[string]any) { delete(start(r), "owner_pid") },
		"missing start owner identity": func(r map[string]any) { delete(start(r), "owner_identity") },
		"helper pid without identity":  func(r map[string]any) { delete(incarnation(r), "identity") },
		"helper identity without pid":  func(r map[string]any) { delete(incarnation(r), "pid") },
		"two tracked starts": func(r map[string]any) {
			copy := map[string]any{}
			for key, value := range incarnation(r) {
				copy[key] = value
			}
			claim := map[string]any{}
			for key, value := range start(r) {
				claim[key] = value
			}
			claim["nonce"] = "second"
			copy["start"] = claim
			r["incarnations"] = append(r["incarnations"].([]any), copy)
		},
		"unknown record field":      func(r map[string]any) { r["unknown"] = true },
		"unknown address field":     func(r map[string]any) { r["address"].(map[string]any)["unknown"] = true },
		"unknown incarnation field": func(r map[string]any) { incarnation(r)["unknown"] = true },
		"unknown start field":       func(r map[string]any) { start(r)["unknown"] = true },
		"unknown policy field":      func(r map[string]any) { policy(r)["unknown"] = true },
		"unknown capacity field":    func(r map[string]any) { policy(r)["capacity"].(map[string]any)["unknown"] = true },
	}

	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			repo := filepath.Join(root, "repo")
			if err := os.MkdirAll(repo, 0o700); err != nil {
				t.Fatal(err)
			}
			scope, err := launcher.ResolveRepoScope(repo)
			if err != nil {
				t.Fatal(err)
			}
			storeDir := filepath.Join(root, "couch")
			namespace, err := couchcore.ResolveCouchNamespace(storeDir, root)
			if err != nil {
				t.Fatal(err)
			}
			store := couchcore.NewThreadStore(namespace)
			created, err := store.CreateThread(couchcore.ThreadRecord{
				SchemaVersion: couchcore.ThreadSchemaVersion,
				Address:       couchcore.ThreadAddress{RepoScope: scope.Key, Tag: "couch-0102030405060708"},
				StartingPath:  repo,
				WorkingPath:   repo,
				CreatedAt:     time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC),
				Revision:      1,
				Incarnations: []couchcore.ThreadIncarnation{{
					PID: 42, Identity: "helper", State: couchcore.IncarnationCreating,
					Start: &couchcore.ThreadStartClaim{Nonce: "nonce", OwnerPID: 7, OwnerIdentity: "owner"},
					Policy: &couchcore.PolicyResult{
						PolicyVersion: 1, PolicyDigest: "digest", RepoIdentity: "repo", AdmissionKey: repo,
						Capacity: couchcore.PolicyCapacity{Kind: couchcore.CapacityUnbounded},
					},
				}},
			})
			if err != nil {
				t.Fatal(err)
			}
			recordPath := filepath.Join(namespace.Dir(), "threadstore", "records", scope.Key, string(created.Address.Tag)+".json")
			raw, err := os.ReadFile(recordPath)
			if err != nil {
				t.Fatal(err)
			}
			var document map[string]any
			if err := json.Unmarshal(raw, &document); err != nil {
				t.Fatal(err)
			}
			mutate(document)
			mutated, err := json.Marshal(document)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(recordPath, mutated, 0o600); err != nil {
				t.Fatal(err)
			}

			if _, err := store.Snapshot(); err == nil {
				t.Fatal("Couch accepted invalid persisted record")
			}
			if _, err := launcher.LoadThreadIndex(namespace.Dir(), func(path string) (string, error) {
				raw, err := os.ReadFile(path)
				return string(raw), err
			}); err == nil {
				t.Fatal("standalone Pair accepted invalid persisted record")
			}
		})
	}
}
