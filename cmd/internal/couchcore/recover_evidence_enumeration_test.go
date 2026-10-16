package couchcore

import (
	"crypto/sha256"
	"fmt"
	"hash"
	"testing"
)

// The fingerprint pins the ordered pre-#611 domain. Pruning may avoid impossible
// prefixes, but must neither drop an admitted point nor change the stride's
// sample. Delimiters keep adjacent string-valued dimensions unambiguous.
type evidenceFingerprint struct {
	hash  hash.Hash
	count int
	buf   []byte
}

func (f *evidenceFingerprint) add(e SlotEvidence) {
	if f.hash == nil {
		f.hash = sha256.New()
	}
	f.buf = f.buf[:0]
	for _, value := range []string{
		string(e.Dir), string(e.Couch), string(e.Agent), string(e.Threads), string(e.Offer),
		string(e.Branch), string(e.Claims), string(e.Quality), string(e.GitSource),
		string(e.Dirty), string(e.Unlanded), string(e.Operation), string(e.DepClaims), string(e.DepTree),
	} {
		f.buf = append(f.buf, value...)
		f.buf = append(f.buf, 0)
	}
	for _, value := range []bool{e.Workspace, e.Elsewhere} {
		if value {
			f.buf = append(f.buf, 1)
		} else {
			f.buf = append(f.buf, 0)
		}
	}
	_, _ = f.hash.Write(f.buf)
	f.count++
}

func (f *evidenceFingerprint) String() string {
	return fmt.Sprintf("%d:%x", f.count, f.hash.Sum(nil))
}

func TestEvidenceEnumerationPreservesDomain(t *testing.T) {
	t.Parallel()
	var all, stride, bases, projectedBases, variants evidenceFingerprint
	index := 0
	forEachEvidence(func(e SlotEvidence) {
		all.add(e)
		index++
		if index%23 == 0 {
			stride.add(e)
		}
		if e.DepClaims != DepNone || e.DepTree != DepTreeClean {
			return
		}
		bases.add(e)
	})
	forEachEvidenceWithDeps([]EvidenceDepClaims{DepNone}, []EvidenceDepTree{DepTreeClean}, func(e SlotEvidence) {
		projectedBases.add(e)
		for _, dep := range AllEvidenceDepClaims() {
			for _, tree := range AllEvidenceDepTrees() {
				v := withDependencyFacts(e, dep, tree)
				if consistent(v) {
					variants.add(v)
				}
			}
		}
	})
	for name, got := range map[string]*evidenceFingerprint{"all": &all, "stride23": &stride, "host-bases": &bases, "projected-bases": &projectedBases, "host-variants": &variants} {
		want := map[string]string{
			"all":             "2406528:80cc493d212304a2e81fb801b99d8d23f20cfb167ce2386a25317b6a328626ea",
			"stride23":        "104631:dd099f5e277a63ff512dfb24cfdb98506940a0ff19257684406aa8a813960f92",
			"projected-bases": "102204:1fe1b6257091f69cd111b740f8aef12825a491b6069551ff22c2140f8d17b9ab",
			"host-bases":      "102204:1fe1b6257091f69cd111b740f8aef12825a491b6069551ff22c2140f8d17b9ab",
			"host-variants":   "2406528:80cc493d212304a2e81fb801b99d8d23f20cfb167ce2386a25317b6a328626ea",
		}[name]
		if got.String() != want {
			t.Errorf("%s = %s; want %s", name, got, want)
		}
	}
}
