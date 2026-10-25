package zellijquery

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/testpair"
)

func TestNamespaceCanonicalAliasesBeforeSocketDirectoryExists(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "missing", "native")
	alias := filepath.Join(root, "alias")
	if err := os.Symlink(target, alias); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{alias, filepath.Join(alias, "nested")} {
		got, err := canonicalNamespace(path)
		want := target
		if path != alias {
			want = filepath.Join(target, "nested")
		}
		if err != nil || got != want {
			t.Errorf("canonicalNamespace(%q) = %q, %v; want %q", path, got, err, want)
		}
	}
	if _, err := os.Lstat(target); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("resolution created target: %v", err)
	}
}

func TestReadyMarkerRoundTripRetainsNestedUnknownNumbers(t *testing.T) {
	_, identity := privateReadinessEnv(t)
	rawIdentity, err := json.Marshal(identity)
	if err != nil {
		t.Fatal(err)
	}
	raw := []byte(`{"future":{"n":1e1000},"identity":` + strings.TrimSuffix(string(rawIdentity), "}") + `,"future":{"nested":[1e1000,true]}}}`)
	marker, err := DecodeReadyMarker(raw)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := EncodeReadyMarker(marker)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeReadyMarker(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if !decoded.Identity.Equal(identity) || string(decoded.Unknown["future"]) != `{"n":1e1000}` || string(decoded.IdentityUnknown["future"]) != `{"nested":[1e1000,true]}` {
		t.Fatalf("lost additive metadata: %s", encoded)
	}
}

func TestReadyPublicationKeepsSameGenerationBytes(t *testing.T) {
	env, identity := privateReadinessEnv(t)
	creator, err := AcquireCreator(t.Context(), env, identity)
	if err != nil {
		t.Fatal(err)
	}
	defer creator.Close()
	if err := creator.Prepare(t.Context()); err != nil {
		t.Fatal(err)
	}
	path, err := creator.store.paths.ZellijReadyMarker(identity.Namespace, identity.Session)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	identityRaw, _ := json.Marshal(identity)
	raw := []byte("{\n \"future\": {\"n\": 1e1000}, \"identity\": " + strings.TrimSuffix(string(identityRaw), "}") + `,"future":{"v":true}}}` + "\n")
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	if err := creator.PublishReady(t.Context(), identity); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != string(raw) {
		t.Fatalf("publication rewrote existing generation: %q %v", got, err)
	}
}

func TestCreatorCapacityRefusesBeforePreparationAndRetainsMarkers(t *testing.T) {
	env, identity := privateReadinessEnv(t)
	store, err := newReadinessStore(env)
	if err != nil {
		t.Fatal(err)
	}
	dir, err := store.paths.ZellijReadyDirectory(identity.Namespace)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < MaxReadySessions; i++ {
		id := identity
		id.Session = fmt.Sprintf("existing-%03d", i)
		path, err := store.paths.ZellijReadyMarker(identity.Namespace, id.Session)
		if err != nil {
			t.Fatal(err)
		}
		raw, err := EncodeReadyMarker(ReadyMarker{Identity: id})
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	creator, err := AcquireCreator(t.Context(), env, identity)
	if err != nil {
		t.Fatal(err)
	}
	defer creator.Close()
	if err := creator.Prepare(t.Context()); err == nil {
		t.Fatal("129th marker admitted")
	}
	if creator.prepared {
		t.Fatal("capacity refusal marked creator prepared")
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != MaxReadySessions {
		t.Fatalf("existing markers changed: %d %v", len(entries), err)
	}
}

func TestLeaseOwnerDeathReleasesStableInodeWithoutReadiness(t *testing.T) {
	if os.Getenv("PAIR_TEST_LEASE_OWNER") != "" {
		lease, err := AcquireFileLease(context.Background(), os.Getenv("PAIR_TEST_LEASE_OWNER"))
		if err != nil {
			panic(err)
		}
		defer lease.Close()
		fmt.Println("held")
		select {}
	}
	env, identity := privateReadinessEnv(t)
	root := testpair.ShortRoot(t)
	env = append(env, "ZELLIJ_SOCKET_DIR="+root)
	if err := os.MkdirAll(filepath.Join(root, "contract_version_1"), 0700); err != nil {
		t.Fatal(err)
	}
	endpoint, err := net.Listen("unix", filepath.Join(root, "contract_version_1", identity.Session))
	if err != nil {
		t.Fatal(err)
	}
	defer endpoint.Close()
	store, err := newReadinessStore(env)
	if err != nil {
		t.Fatal(err)
	}
	path, err := store.paths.ZellijNamespaceLease(store.namespace)
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestLeaseOwnerDeathReleasesStableInodeWithoutReadiness$")
	cmd.Env = append(os.Environ(), "PAIR_TEST_LEASE_OWNER="+path)
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cmd.Process.Kill(); _ = cmd.Wait() }()
	line := make([]byte, 5)
	if _, err := io.ReadFull(out, line); err != nil || string(line) != "held\n" {
		t.Fatalf("child did not acquire: %q %v", line, err)
	}
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Millisecond)
	defer cancel()
	if lease, err := AcquireFileLease(ctx, path); !errors.Is(err, context.DeadlineExceeded) {
		if lease != nil {
			lease.Close()
		}
		t.Fatalf("live owner did not exclude: %v", err)
	}
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = cmd.Wait()
	ctx2, cancel2 := context.WithTimeout(t.Context(), time.Second)
	defer cancel2()
	lease, err := AcquireFileLease(ctx2, path)
	if err != nil {
		t.Fatal(err)
	}
	if err := lease.Close(); err != nil {
		t.Fatal(err)
	}
	after, err := os.Stat(path)
	if err != nil || !os.SameFile(before, after) {
		t.Fatalf("lease inode replaced: %v", err)
	}
	calls := 0
	if _, err := (Source{Env: env, Run: func(*exec.Cmd) error { calls++; return nil }}).ReadContext(ctx2, "list-sessions"); err != nil || calls != 1 {
		t.Fatalf("dead creator's unmarked survivor was denied: calls=%d err=%v", calls, err)
	}
}

func TestFileLeaseRefusesSymlinkAndNonregularInodes(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target")
	if err := os.WriteFile(target, nil, 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{link, root} {
		lease, err := AcquireFileLease(t.Context(), path)
		if lease != nil {
			lease.Close()
		}
		if err == nil {
			t.Errorf("admitted unsafe inode %s", path)
		}
	}
}

func TestNamespaceCanonicalRelativeDanglingAliasAndLoop(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "missing", "native")
	alias := filepath.Join(root, "alias")
	if err := os.Symlink(filepath.Join("missing", "native"), alias); err != nil {
		t.Fatal(err)
	}
	got, err := canonicalNamespace(alias)
	if err != nil || got != target {
		t.Fatalf("relative alias = %q %v", got, err)
	}
	loop := filepath.Join(root, "loop")
	if err := os.Symlink("loop", loop); err != nil {
		t.Fatal(err)
	}
	if _, err := canonicalNamespace(loop); err == nil {
		t.Fatal("cyclic namespace accepted")
	}
}

func TestReadyMarkerRejectsAmbiguousAndOversizedMetadata(t *testing.T) {
	_, identity := privateReadinessEnv(t)
	raw, err := EncodeReadyMarker(ReadyMarker{Identity: identity})
	if err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{
		`null`, `{}`, `{"identity":null}`, `{"identity":3}`,
		`{"identity":` + string(mustIdentityJSON(t, identity)) + `,"identity":{}}`,
		strings.Replace(string(raw), `"nonce":"start-private"`, `"nonce":false`, 1),
		strings.Replace(string(raw), `"nonce":"start-private"`, `"nonce":"a","nonce":"b"`, 1),
		`{"future":{"x":1,"x":2},` + strings.TrimPrefix(string(raw), "{"),
		string(raw) + strings.Repeat(" ", 8192),
	} {
		if _, err := DecodeReadyMarker([]byte(bad)); err == nil {
			t.Errorf("accepted %s", bad)
		}
	}
}

func mustIdentityJSON(t *testing.T, identity ReadyIdentity) []byte {
	t.Helper()
	raw, err := json.Marshal(identity)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestLegacyEndpointCountDoesNotDenyNativeObservation(t *testing.T) {
	env, identity := privateReadinessEnv(t)
	root := filepath.Join(identity.Namespace, "contract_version_1")
	for i := 0; i <= MaxReadySessions; i++ {
		if err := os.WriteFile(filepath.Join(root, fmt.Sprintf("candidate-%03d", i)), nil, 0600); err != nil {
			t.Fatal(err)
		}
	}
	calls := 0
	result, err := (Source{Env: env, Run: func(cmd *exec.Cmd) error {
		calls++
		_, err := cmd.Stdout.Write([]byte("existing-session\n"))
		return err
	}}).ReadContext(t.Context(), "list-sessions", "--no-formatting")
	if err != nil || calls != 1 || string(result.Stdout) != "existing-session\n" {
		t.Fatalf("legacy endpoint count denied observation: result=%+v calls=%d err=%v", result, calls, err)
	}
}

func TestNamespaceCanonicalDanglingAliasKeepsSymlinkParentTraversal(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "physical", "nested")
	if err := os.MkdirAll(nested, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(nested, filepath.Join(root, "jump")); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(root, "alias")
	if err := os.Symlink("jump/../missing", alias); err != nil {
		t.Fatal(err)
	}
	got, err := canonicalNamespace(alias)
	want := filepath.Join(root, "physical", "missing")
	if err != nil || got != want {
		t.Fatalf("symlink parent traversal = %q %v; want %q", got, err, want)
	}
}

func TestLegacyUnmarkedEndpointPreservesAutomaticAndExplicitAccess(t *testing.T) {
	env, identity := privateReadinessEnv(t)
	shortRoot := testpair.ShortRoot(t)
	identity.Namespace = shortRoot
	env = append(env, "ZELLIJ_SOCKET_DIR="+shortRoot)
	if err := os.MkdirAll(filepath.Join(shortRoot, "contract_version_1"), 0700); err != nil {
		t.Fatal(err)
	}
	// This endpoint predates the readiness contract. Its absent receipt cannot
	// deny ordinary listing or explicit recovery after exclusion is acquired.
	path := filepath.Join(identity.Namespace, "contract_version_1", "unmarked")
	listener, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	calls := 0
	run := func(cmd *exec.Cmd) error {
		calls++
		_, err := cmd.Stdout.Write([]byte("existing-session\n"))
		return err
	}
	if _, err := (Source{Env: env, Run: run}).ReadContext(t.Context(), "list-sessions"); err != nil || calls != 1 {
		t.Fatalf("automatic probe: calls=%d err=%v", calls, err)
	}
	if _, err := (Source{Env: env, Run: run, ExplicitRecovery: true}).ReadContext(t.Context(), "list-sessions"); err != nil || calls != 2 {
		t.Fatalf("explicit recovery: calls=%d err=%v", calls, err)
	}
}

func TestExplicitRecoveryWaitsForCreatorAndHonorsCancellation(t *testing.T) {
	env, identity := privateReadinessEnv(t)
	creator, err := AcquireCreator(t.Context(), env, identity)
	if err != nil {
		t.Fatal(err)
	}
	defer creator.Close()
	for _, explicit := range []bool{false, true} {
		ctx, cancel := context.WithTimeout(t.Context(), 25*time.Millisecond)
		calls := 0
		_, err := (Source{Env: env, ExplicitRecovery: explicit, Run: func(*exec.Cmd) error { calls++; return nil }}).ReadContext(ctx, "list-sessions")
		cancel()
		if !errors.Is(err, context.DeadlineExceeded) || calls != 0 {
			t.Fatalf("explicit=%v calls=%d err=%v", explicit, calls, err)
		}
	}
}

func TestLegacyProbeAdmissionDoesNotScanNamespaceEntries(t *testing.T) {
	for _, count := range []int{8, 16} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			env, identity := privateReadinessEnv(t)
			for i := 0; i < count; i++ {
				path := filepath.Join(identity.Namespace, "contract_version_1", fmt.Sprintf("untrusted-%d", i))
				// Passive namespace contents cannot be an admission prerequisite.
				if err := os.WriteFile(path, []byte("not a socket or marker"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			calls := 0
			_, err := (Source{Env: env, Run: func(*exec.Cmd) error { calls++; return nil }}).ReadContext(t.Context(), "list-sessions")
			if err != nil || calls != 1 {
				t.Fatalf("count=%d native invocations=%d err=%v", count, calls, err)
			}
		})
	}
}

func TestExplicitCommandCreatorPermitCannotEscape(t *testing.T) {
	env, identity := privateReadinessEnv(t)
	creator, err := AcquireCreator(t.Context(), env, identity)
	if err != nil {
		t.Fatal(err)
	}
	defer creator.Close()
	var escaped context.Context
	if err := creator.Preflight(t.Context(), func(ctx context.Context) error {
		escaped = ctx
		lease, err := AcquireExplicitCommand(ctx, env)
		if err != nil {
			return err
		}
		if err := lease.Close(); err != nil {
			return err
		}
		foreign, _ := privateReadinessEnv(t)
		if lease, err := AcquireExplicitCommand(ctx, foreign); !errors.Is(err, ErrReadinessUnknown) {
			if lease != nil {
				lease.Close()
			}
			return fmt.Errorf("foreign permit accepted: %v", err)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if lease, err := AcquireExplicitCommand(escaped, env); !errors.Is(err, ErrReadinessUnknown) {
		if lease != nil {
			lease.Close()
		}
		t.Fatalf("expired permit accepted: %v", err)
	}
}

func TestBlockedAdmissionsAtEightAndSixteenNeverRunNative(t *testing.T) {
	for _, count := range []int{8, 16} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			env, identity := privateReadinessEnv(t)
			creator, err := AcquireCreator(t.Context(), env, identity)
			if err != nil {
				t.Fatal(err)
			}
			defer creator.Close()
			calls := 0
			for i := 0; i < count; i++ {
				ctx, cancel := context.WithTimeout(t.Context(), time.Millisecond)
				_, err := (Source{Env: env, Run: func(*exec.Cmd) error { calls++; return nil }}).ReadContext(ctx, "list-sessions")
				cancel()
				if !errors.Is(err, context.DeadlineExceeded) {
					t.Fatalf("admission %d err=%v", i, err)
				}
			}
			if calls != 0 {
				t.Fatalf("blocked native scans=%d", calls)
			}
		})
	}
}

func TestLegacyUnmarkedEndpointPreservesAccessWithLongTMPDIR(t *testing.T) {
	longRoot := filepath.Join(t.TempDir(), strings.Repeat("long", 40))
	if err := os.MkdirAll(longRoot, 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TMPDIR", longRoot)
	t.Run("socket-safe-regression", TestLegacyUnmarkedEndpointPreservesAutomaticAndExplicitAccess)
	if os.Getenv("TMPDIR") != longRoot {
		t.Fatal("socket fixture changed inherited temporary directory")
	}
}

func TestLegacyMarkerContentsCannotDenyProbe(t *testing.T) {
	for _, kind := range []string{"absent", "malformed", "foreign", "stale"} {
		t.Run(kind, func(t *testing.T) {
			env, identity := privateReadinessEnv(t)
			root := testpair.ShortRoot(t)
			env = append(env, "ZELLIJ_SOCKET_DIR="+root)
			if err := os.MkdirAll(filepath.Join(root, "contract_version_1"), 0700); err != nil {
				t.Fatal(err)
			}
			listener, err := net.Listen("unix", filepath.Join(root, "contract_version_1", identity.Session))
			if err != nil {
				t.Fatal(err)
			}
			defer listener.Close()
			store, err := newReadinessStore(env)
			if err != nil {
				t.Fatal(err)
			}
			if kind != "absent" {
				path, err := store.paths.ZellijReadyMarker(store.namespace, identity.Session)
				if err != nil {
					t.Fatal(err)
				}
				raw := []byte("malformed receipt")
				if kind != "malformed" {
					identity.Namespace = store.namespace
					if kind == "foreign" {
						identity.Session = "another-session"
					} else {
						identity.Nonce = "start-old-generation"
					}
					raw, err = EncodeReadyMarker(ReadyMarker{Identity: identity})
					if err != nil {
						t.Fatal(err)
					}
				}
				if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, raw, 0600); err != nil {
					t.Fatal(err)
				}
			}
			calls := 0
			for _, explicit := range []bool{false, true} {
				_, err := (Source{Env: env, ExplicitRecovery: explicit, Run: func(*exec.Cmd) error { calls++; return nil }}).ReadContext(t.Context(), "list-sessions")
				if err != nil {
					t.Fatalf("explicit=%v: %v", explicit, err)
				}
			}
			if calls != 2 {
				t.Fatalf("native invocations=%d", calls)
			}
		})
	}
}
