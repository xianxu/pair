package couchcore

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

func TestAllocateThreadTagAtomicallyClaimsAgainstArtifactProducers(t *testing.T) {
	dataDir := t.TempDir()
	scope := launcher.RepoScope{Key: "0123456789abcdef"}
	producer, err := launcher.ClaimNewThreadAddress(dataDir, scope, "couch-0000000000000000")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = producer.Release() })
	store, ns := newTestThreadStore(t)
	entropy := append(make([]byte, 8), []byte{1, 2, 3, 4, 5, 6, 7, 8}...)
	got, err := store.AllocateThreadTag(scope.Key, ns.Dir(), time.Now(), bytes.NewReader(entropy), NewScopedThreadArtifactCollisionChecker(dataDir))
	if err != nil {
		t.Fatal(err)
	}
	if got.Address.Tag != "couch-0102030405060708" {
		t.Fatalf("allocation reused producer-owned address: %+v", got.Address)
	}
}

func TestScopedArtifactCheckerReportsPairRegistrationEvidence(t *testing.T) {
	dataDir := t.TempDir()
	address := ThreadAddress{RepoScope: "0123456789abcdef", Tag: "couch-0001020304050607"}
	checker := NewScopedThreadArtifactCollisionChecker(dataDir)
	claim, err := checker.Claim(address)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = claim.Release() })
	if got, err := checker.Registration(address); err != nil || got != RegistrationAbsent {
		t.Fatalf("reserved registration = %q, %v", got, err)
	}
	if err := launcher.EnsureThreadAddressForPair(dataDir, launcher.RepoScope{Key: address.RepoScope}, string(address.Tag), true); err != nil {
		t.Fatal(err)
	}
	if got, err := checker.Registration(address); err != nil || got != RegistrationEstablished {
		t.Fatalf("established registration = %q, %v", got, err)
	}
}

func TestScopedArtifactCheckerRegistrationRequiresExactEstablishedMarker(t *testing.T) {
	address := ThreadAddress{RepoScope: "0123456789abcdef", Tag: "couch-0001020304050607"}
	tests := []struct {
		name    string
		marker  *string
		dir     bool
		want    RegistrationEvidence
		wantErr bool
	}{
		{name: "missing", want: RegistrationAbsent},
		{name: "reserved", marker: stringPointer(`{"schema":1,"scope":"0123456789abcdef","tag":"couch-0001020304050607","state":"reserved"}`), want: RegistrationAbsent},
		{name: "established", marker: stringPointer(`{"schema":1,"scope":"0123456789abcdef","tag":"couch-0001020304050607","state":"established"}`), want: RegistrationEstablished},
		{name: "malformed", marker: stringPointer(`not-json`), want: RegistrationUnknown, wantErr: true},
		{name: "wrong schema", marker: stringPointer(`{"schema":2,"scope":"0123456789abcdef","tag":"couch-0001020304050607","state":"established"}`), want: RegistrationUnknown, wantErr: true},
		{name: "wrong scope", marker: stringPointer(`{"schema":1,"scope":"fedcba9876543210","tag":"couch-0001020304050607","state":"established"}`), want: RegistrationUnknown, wantErr: true},
		{name: "wrong tag", marker: stringPointer(`{"schema":1,"scope":"0123456789abcdef","tag":"couch-fedcba9876543210","state":"established"}`), want: RegistrationUnknown, wantErr: true},
		{name: "invalid state", marker: stringPointer(`{"schema":1,"scope":"0123456789abcdef","tag":"couch-0001020304050607","state":"live"}`), want: RegistrationUnknown, wantErr: true},
		{name: "unknown field", marker: stringPointer(`{"schema":1,"scope":"0123456789abcdef","tag":"couch-0001020304050607","state":"established","future":true}`), want: RegistrationUnknown, wantErr: true},
		{name: "duplicate state", marker: stringPointer(`{"schema":1,"scope":"0123456789abcdef","tag":"couch-0001020304050607","state":"reserved","state":"established"}`), want: RegistrationUnknown, wantErr: true},
		{name: "filesystem read failure", dir: true, want: RegistrationUnknown, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dataDir := t.TempDir()
			paths := launcher.NewScopedPaths(dataDir, launcher.RepoScope{Key: address.RepoScope}, string(address.Tag))
			if tt.marker != nil || tt.dir {
				if err := os.MkdirAll(paths.ScopeDir(), 0o700); err != nil {
					t.Fatal(err)
				}
				if tt.dir {
					if err := os.Mkdir(paths.ThreadClaim(), 0o700); err != nil {
						t.Fatal(err)
					}
				} else if err := os.WriteFile(paths.ThreadClaim(), []byte(*tt.marker), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			got, err := NewScopedThreadArtifactCollisionChecker(dataDir).Registration(address)
			if got != tt.want || (err != nil) != tt.wantErr {
				t.Fatalf("Registration = %q, %v; want %q, error=%v", got, err, tt.want, tt.wantErr)
			}
		})
	}
}

func FuzzScopedArtifactCheckerRegistrationRejectsArbitraryMarkerBytes(f *testing.F) {
	f.Add("")
	f.Add("garbage")
	f.Add(strings.Repeat("x", 4096))
	f.Fuzz(func(t *testing.T, arbitrary string) {
		dataDir := t.TempDir()
		address := ThreadAddress{RepoScope: "0123456789abcdef", Tag: "couch-0001020304050607"}
		paths := launcher.NewScopedPaths(dataDir, launcher.RepoScope{Key: address.RepoScope}, string(address.Tag))
		if err := os.MkdirAll(paths.ScopeDir(), 0o700); err != nil {
			t.Fatal(err)
		}
		// The payload is always structurally foreign, however adversarial its
		// string content. Registration must fail closed rather than authorize it.
		raw := []byte(`{"arbitrary":` + strconvQuote(arbitrary) + `}`)
		if err := os.WriteFile(paths.ThreadClaim(), raw, 0o600); err != nil {
			t.Fatal(err)
		}
		got, err := NewScopedThreadArtifactCollisionChecker(dataDir).Registration(address)
		if got != RegistrationUnknown || err == nil {
			t.Fatalf("arbitrary marker authorized: %q, %v", got, err)
		}
	})
}

func stringPointer(value string) *string { return &value }

func strconvQuote(value string) string {
	raw, _ := json.Marshal(value)
	return string(raw)
}

type fakeSessionDeleter struct{ deleted []string }

func (d *fakeSessionDeleter) DeleteSession(name string) error {
	d.deleted = append(d.deleted, name)
	return nil
}

func TestScopedArtifactCheckerQuiescesExactIndexedSession(t *testing.T) {
	global := t.TempDir()
	address := ThreadAddress{RepoScope: "0123456789abcdef", Tag: "couch-0001020304050607"}
	entry := launcher.SessionNameEntry{SessionName: "📁pair-couch", ScopeKey: address.RepoScope, RepoRoot: "/repo", RepoName: "repo", Tag: string(address.Tag)}
	paths := launcher.NewScopedPaths(global, launcher.RepoScope{Key: address.RepoScope}, string(address.Tag))
	if err := os.MkdirAll(paths.ScopeDir(), 0o700); err != nil {
		t.Fatal(err)
	}
	line, err := launcher.BuildSessionNameIndexLine(entry)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.SessionBindings(), []byte(line+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	deleter := &fakeSessionDeleter{}
	checker := NewScopedThreadArtifactCollisionChecker(global)
	checker.Sessions = deleter
	if err := checker.Quiesce(address); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(deleter.deleted, []string{entry.SessionName}) {
		t.Fatalf("deleted = %v", deleter.deleted)
	}
}

func TestScopedArtifactCollisionCheckerFindsEveryTagNameShape(t *testing.T) {
	dataDir := t.TempDir()
	address := ThreadAddress{RepoScope: "0123456789abcdef", Tag: "couch-0001020304050607"}
	paths := launcher.NewScopedPaths(dataDir, launcher.RepoScope{Key: address.RepoScope}, string(address.Tag))
	checker := NewScopedThreadArtifactCollisionChecker(dataDir)
	scopeDir := paths.ScopeDir()
	if err := os.MkdirAll(scopeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{
		paths.Ledger(), paths.Draft(), paths.Log(), paths.QueueDir(), paths.Agent(),
		paths.AgentPID(), paths.AgentOutput(), paths.AgentPicks(), paths.AdaptLog(),
		paths.OuterTTY(), paths.NvimDraftPID(), paths.NvimScrollbackPID(),
		paths.Config("codex"), paths.LegacyCodexConfig(), paths.AgentReady("claude"),
		paths.Pane("codex"), paths.ScrollbackRaw("codex"), paths.ScrollbackANSI("codex"),
		paths.ScrollbackEvents("codex"), paths.ScrollbackViewport("codex"),
		paths.Changelog("codex"), paths.AgentDraft("codex"),
	} {
		name := filepath.Base(path)
		t.Run(name, func(t *testing.T) {
			if err := os.WriteFile(path, []byte("owned"), 0o600); err != nil {
				t.Fatal(err)
			}
			claim, err := checker.Claim(address)
			if !errors.Is(err, launcher.ErrThreadAddressClaimed) || claim != nil {
				t.Fatalf("Claim = %T, %v", claim, err)
			}
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
		})
	}
	if err := os.WriteFile(filepath.Join(scopeDir, "draft-couch-00010203040506070.md"), []byte("neighbor"), 0o600); err != nil {
		t.Fatal(err)
	}
	claim, err := checker.Claim(address)
	if err != nil {
		t.Fatalf("neighbor tag claim: %v", err)
	}
	_ = claim.Release()
}

func TestScopedArtifactCollisionCheckerFindsDetachedSessionBinding(t *testing.T) {
	dataDir := t.TempDir()
	address := ThreadAddress{RepoScope: "0123456789abcdef", Tag: "couch-0001020304050607"}
	paths := launcher.NewScopedPaths(dataDir, launcher.RepoScope{Key: address.RepoScope}, string(address.Tag))
	if err := os.MkdirAll(paths.ScopeDir(), 0o700); err != nil {
		t.Fatal(err)
	}
	line, err := launcher.BuildSessionNameIndexLine(launcher.SessionNameEntry{
		SessionName: "📁repo-work", ScopeKey: address.RepoScope, RepoRoot: "/repo", RepoName: "repo", Tag: string(address.Tag),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.SessionBindings(), []byte(line+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	claim, err := NewScopedThreadArtifactCollisionChecker(dataDir).Claim(address)
	if !errors.Is(err, launcher.ErrThreadAddressClaimed) || claim != nil {
		t.Fatalf("session binding claim = %T, %v", claim, err)
	}
}

func TestScopedArtifactClaimerRejectsNonScopedPathsAndFutureFamilies(t *testing.T) {
	dataDir := t.TempDir()
	address := ThreadAddress{RepoScope: "0123456789abcdef", Tag: "couch-0001020304050607"}
	paths := launcher.NewScopedPaths(dataDir, launcher.RepoScope{Key: address.RepoScope}, string(address.Tag))
	if err := os.MkdirAll(paths.ScopeDir(), 0o700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{
		"draft-pane-couch-0001020304050607.json",
		"image-capture-couch-0001020304050607.done",
		"parked-scrollback-couch-0001020304050607-20260826.raw",
		"last-terminal-pane-couch-0001020304050607",
		"review-definition-request-couch-0001020304050607.json",
		"future-family-couch-0001020304050607-variant.bin",
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(paths.ScopeDir(), name)
			if err := os.WriteFile(path, []byte("owned"), 0o600); err != nil {
				t.Fatal(err)
			}
			claim, err := NewScopedThreadArtifactCollisionChecker(dataDir).Claim(address)
			if !errors.Is(err, launcher.ErrThreadAddressClaimed) || claim != nil {
				t.Fatalf("Claim = %T, %v", claim, err)
			}
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
		})
	}
}
