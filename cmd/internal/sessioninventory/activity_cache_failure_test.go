package sessioninventory_test

import (
	"context"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessioninventorytest"
)

type failedCatalogReadRuntime struct {
	*sessioninventorytest.FakeRuntime
	fail    bool
	failure error
	loads   int
}

func (r *failedCatalogReadRuntime) LoadSessionInventoryCatalog() (sessioninventory.Catalog, error) {
	r.loads++
	if r.fail {
		if r.failure != nil {
			return sessioninventory.Catalog{}, r.failure
		}
		return sessioninventory.Catalog{}, os.ErrPermission
	}
	return r.FakeRuntime.LoadSessionInventoryCatalog()
}
func (r *failedCatalogReadRuntime) PublishSessionInventoryValidations(v []sessioninventory.TargetValidation) error {
	// Successful publication of already persisted validations need not change
	// metadata. Preserve that complete stateful catalog while reads fail.
	before, err := r.FakeRuntime.LoadSessionInventoryCatalog()
	if err != nil {
		return err
	}
	next, err := sessioninventory.CatalogWithTargetValidations(before, v)
	if err != nil {
		return err
	}
	if reflect.DeepEqual(before, next) {
		return nil
	}
	return r.FakeRuntime.PublishSessionInventoryValidations(v)
}
func TestActivityCacheFailedCatalogObservationNeverPublishesFallback(t *testing.T) {
	for _, failure := range []error{os.ErrPermission, os.ErrNotExist} {
		t.Run(failure.Error(), func(t *testing.T) {
			at := time.Now()
			fake, _, _ := proofBackedCodexFixture(t, "019d1111-1111-7111-8111-111111111111", &at)
			// Seed the catalog through the real query before failing only reads.
			if _, err := sessioninventory.QuerySession(fake, "scope", "work", sessioninventory.AgentCodex); err != nil {
				t.Fatal(err)
			}
			rt := &failedCatalogReadRuntime{FakeRuntime: fake, fail: true, failure: failure}
			cache := sessioninventory.NewActivityCache()
			query := func() {
				t.Helper()
				if _, ok, err := cache.Query(context.Background(), rt, "scope", "work", sessioninventory.AgentCodex); err != nil || !ok {
					t.Fatalf("proof fallback=%v,%v", ok, err)
				}
			}
			query()
			before := rt.loads
			query()
			if rt.loads == before {
				t.Fatal("incomplete catalog observation published reusable fallback")
			}
			rt.fail = false
			before = rt.loads
			query()
			if rt.loads == before {
				t.Fatal("unchanged recovery reused incomplete observation")
			}
			before = rt.loads
			query()
			if rt.loads != before {
				t.Fatal("complete recovered observation did not become warm")
			}
		})
	}
}

// A failed optional write is also incomplete evidence, even when the loaded
// catalog and all native fingerprints remain unchanged.
type failedCatalogPublishRuntime struct{ *failedCatalogReadRuntime }

func (r failedCatalogPublishRuntime) PublishSessionInventoryValidations(v []sessioninventory.TargetValidation) error {
	if r.fail {
		return os.ErrPermission
	}
	return r.failedCatalogReadRuntime.PublishSessionInventoryValidations(v)
}
func (r failedCatalogPublishRuntime) LoadSessionInventoryCatalog() (sessioninventory.Catalog, error) {
	r.loads++
	return r.FakeRuntime.LoadSessionInventoryCatalog()
}
func TestActivityCacheFailedCatalogPublishNeverPublishesFallback(t *testing.T) {
	at := time.Now()
	fake, _, _ := proofBackedCodexFixture(t, "019d1111-1111-7111-8111-111111111111", &at)
	rt := failedCatalogPublishRuntime{&failedCatalogReadRuntime{FakeRuntime: fake, fail: true}}
	cache := sessioninventory.NewActivityCache()
	for i := 0; i < 2; i++ {
		before := rt.loads
		if _, ok, err := cache.Query(context.Background(), rt, "scope", "work", sessioninventory.AgentCodex); err != nil || !ok {
			t.Fatalf("fallback=%v,%v", ok, err)
		}
		if rt.loads == before {
			t.Fatal("failed catalog publication became warm")
		}
	}
}
