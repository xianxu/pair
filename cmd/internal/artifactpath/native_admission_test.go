package artifactpath

import "testing"

func TestNativeAdmissionCachePaths(t *testing.T) {
	p, e := ResolvePairCache("/home/user")
	if e != nil {
		t.Fatal(e)
	}
	store, e := p.NativeAdmissions()
	if e != nil || store != "/home/user/.cache/pair/native-admission-v1.json" {
		t.Fatalf("store=%s err=%v", store, e)
	}
	lock, e := p.NativeAdmissionLease()
	if e != nil || lock != "/home/user/.cache/pair/native-admission-v1.lock" {
		t.Fatalf("lock=%s err=%v", lock, e)
	}
	if _, e = (PairCachePaths{}).NativeAdmissions(); e == nil {
		t.Fatal("zero store path")
	}
	if _, e = (PairCachePaths{}).NativeAdmissionLease(); e == nil {
		t.Fatal("zero lock path")
	}
}
