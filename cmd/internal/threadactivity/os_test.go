package threadactivity

import "testing"

// Every thread's probe shares the runtime's one ledger cache (#507), so a
// ledger read for one pass is reused by the next.
func TestOSRuntimeSharesOneLedgerCacheAcrossThreads(t *testing.T) {
	r := NewOSRuntime("/home")
	a, b := r.inventory("/data/repos/a").OwnerLaunchCache(), r.inventory("/data/repos/b").OwnerLaunchCache()
	if a == nil || a != b {
		t.Fatalf("probes do not share one cache: %p %p", a, b)
	}
	if NewOSRuntime("/home").inventory("/data/repos/a").OwnerLaunchCache() == a {
		t.Fatal("two runtimes share a cache")
	}
}

func TestOSRuntimeOwnsActivityCache(t *testing.T) {
	r := NewOSRuntime("/home")
	if r.activities == nil {
		t.Fatal("long-lived activity runtime has no cache")
	}
	if NewOSRuntime("/home").activities == r.activities {
		t.Fatal("activity evidence shared across caller lifetimes")
	}
}
