package couchcore

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func metadataThread(scope string, tag ThreadTag, path, name string) ThreadRecord {
	return ThreadRecord{
		SchemaVersion:    ThreadSchemaVersion,
		Address:          ThreadAddress{RepoScope: scope, Tag: tag},
		StartingPath:     path,
		WorkingPath:      path,
		CreatedAt:        time.Unix(1, 0).UTC(),
		Revision:         1,
		ClaimGeneration:  1,
		Name:             name,
		Description:      "operator description",
		PublishedSummary: "agent summary",
	}
}

func metadataValue(value string) *string { return &value }

func TestThreadStoreMetadataUpdateUsesRevisionCAS(t *testing.T) {
	store, _ := newTestThreadStore(t)
	record := metadataThread("816fc349d3faebf8", "couch-0102030405060708", "/repo/task", "old name")
	created, err := store.CreateThread(record)
	if err != nil {
		t.Fatal(err)
	}

	updated, err := store.ApplyThreadMetadata(created.Address, created.Revision, ThreadMetadataPatch{
		Description:      metadataValue("new description"),
		PublishedSummary: metadataValue("new agent summary"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Revision != created.Revision+1 || updated.Name != "old name" || updated.Description != "new description" || updated.PublishedSummary != "new agent summary" {
		t.Fatalf("updated metadata = %+v", updated)
	}

	_, err = store.ApplyThreadMetadata(created.Address, created.Revision, ThreadMetadataPatch{Name: metadataValue("stale overwrite")})
	var stale *ThreadRevisionError
	if !errors.As(err, &stale) {
		t.Fatalf("stale update error = %v, want ThreadRevisionError", err)
	}
	got, err := store.GetThread(created.Address)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "old name" || got.Description != "new description" || got.PublishedSummary != "new agent summary" {
		t.Fatalf("stale update changed record: %+v", got)
	}
}

func TestResolveThreadReferenceExactTagPrecedesFuzzyMatches(t *testing.T) {
	records := []ThreadRecord{
		metadataThread("scope-b", "work", "/repo/one", "other"),
		metadataThread("scope-a", "work", "/other/one", "other"),
		metadataThread("scope-a", "other", "/repo/work", "work"),
	}

	for _, test := range []struct {
		name      string
		repoScope string
		want      []ThreadAddress
		ambiguous bool
	}{
		{name: "scoped exact", repoScope: "scope-a", want: []ThreadAddress{records[1].Address}},
		{name: "global repeated exact", want: []ThreadAddress{records[1].Address, records[0].Address}, ambiguous: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := ResolveThreadReference(records, test.repoScope, "  work  ")
			var ambiguous *AmbiguousThreadReferenceError
			if test.ambiguous != errors.As(err, &ambiguous) {
				t.Fatalf("error = %v, want ambiguous %v", err, test.ambiguous)
			}
			if !test.ambiguous && err != nil {
				t.Fatal(err)
			}
			if addresses := threadAddresses(got); !reflect.DeepEqual(addresses, test.want) {
				t.Fatalf("addresses = %+v, want %+v", addresses, test.want)
			}
			if ambiguous != nil && !reflect.DeepEqual(ambiguous.Candidates, test.want) {
				t.Fatalf("ambiguous candidates = %+v, want %+v", ambiguous.Candidates, test.want)
			}
		})
	}
}

func TestResolveThreadReferenceScopedFuzzyMatchesAreDeterministicallyOrdered(t *testing.T) {
	records := []ThreadRecord{
		metadataThread("scope-b", "tag-z", "/repo/compiler-z", "unrelated"),
		metadataThread("scope-a", "tag-b", "/repo/b", "COMPILER"),
		metadataThread("scope-a", "tag-a", "/repo/compiler-a", "unrelated"),
		metadataThread("scope-c", "tag-a", "/repo/compiler-c", "unrelated"),
	}
	want := []ThreadAddress{records[2].Address, records[1].Address}

	for shift := range records {
		permutation := append(append([]ThreadRecord{}, records[shift:]...), records[:shift]...)
		got, err := ResolveThreadReference(permutation, "scope-a", "  CoMpIlEr  ")
		var ambiguous *AmbiguousThreadReferenceError
		if !errors.As(err, &ambiguous) {
			t.Fatalf("shift %d: error = %v, want ambiguity", shift, err)
		}
		if addresses := threadAddresses(got); !reflect.DeepEqual(addresses, want) {
			t.Fatalf("shift %d: addresses = %+v, want %+v", shift, addresses, want)
		}
		if !reflect.DeepEqual(ambiguous.Candidates, want) || ambiguous.Reference != "CoMpIlEr" {
			t.Fatalf("shift %d: ambiguity = %+v, want trimmed reference and candidates %+v", shift, ambiguous, want)
		}
	}
}

func TestResolveThreadReferenceReturnsDeepClones(t *testing.T) {
	record := metadataThread("scope-a", "work", "/repo/work", "compiler")
	record.Incarnations = []ThreadIncarnation{{
		Identity:      "original",
		Start:         &ThreadStartClaim{LaunchProfile: &LaunchProfile{Argv: []string{"original-start"}}},
		Policy:        &PolicyResult{PolicyDigest: "original-policy"},
		LaunchProfile: &LaunchProfile{Argv: []string{"original-launch"}},
	}}
	records := []ThreadRecord{cloneThreadRecord(record)}

	got, err := ResolveThreadReference(records, "scope-a", "compiler")
	if err != nil {
		t.Fatal(err)
	}
	got[0].Incarnations[0].Identity = "changed"
	got[0].Incarnations[0].Start.LaunchProfile.Argv[0] = "changed"
	got[0].Incarnations[0].Policy.PolicyDigest = "changed"
	got[0].Incarnations[0].LaunchProfile.Argv[0] = "changed"
	if !reflect.DeepEqual(records[0], record) {
		t.Fatalf("mutating result changed input:\n got  %+v\n want %+v", records[0], record)
	}
}

func TestResolveThreadReferenceUsesCouchOwnedNotFoundErrors(t *testing.T) {
	records := []ThreadRecord{metadataThread("scope-a", "work", "/repo/work", "compiler")}
	for _, test := range []struct {
		name string
		ref  string
		want string
	}{
		{name: "empty", ref: " \t\n ", want: "thread reference not found: empty reference"},
		{name: "malformed NUL", ref: "bad\x00reference", want: "thread reference not found: malformed reference"},
		{name: "missing", ref: "  absent  ", want: `thread reference not found: "absent"`},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := ResolveThreadReference(records, "scope-a", test.ref)
			if got != nil || !errors.Is(err, ErrThreadReferenceNotFound) || err.Error() != test.want {
				t.Fatalf("resolution = %+v, %v; want nil, %q", got, err, test.want)
			}
			if strings.Contains(err.Error(), "launcher") {
				t.Fatalf("error leaked launcher ownership: %v", err)
			}
		})
	}
}

func threadAddresses(records []ThreadRecord) []ThreadAddress {
	addresses := make([]ThreadAddress, len(records))
	for i := range records {
		addresses[i] = records[i].Address
	}
	return addresses
}
