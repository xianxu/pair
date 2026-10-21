package couchmessage

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestSocketPathUnsetPreservesExactAddresses(t *testing.T) {
	t.Setenv("COUCH_ISOLATED_ROOT", "")
	namespace := "/repo/namespace"
	digest := sha256.Sum256([]byte(namespace + "\x00broker"))
	//lint:allow tmp-literal — pins the historical production address
	want := filepath.Join("/tmp", fmt.Sprintf("pair-message-%d", os.Getuid()), fmt.Sprintf("%x.sock", digest[:20]))
	got, err := SocketPath(namespace, "broker")
	if err != nil || got != want {
		t.Fatalf("legacy broker = %q, %v; want %q", got, err, want)
	}
	binding := protocolBinding("pair:0")
	raw, _ := json.Marshal(binding)
	digest = sha256.Sum256([]byte(namespace + "\x00wrapper:" + string(raw)))
	want = filepath.Join(filepath.Dir(want), fmt.Sprintf("wrapper-%d-%x.sock", binding.PID, digest[:20]))
	got, err = EndpointSocket(namespace, binding)
	if err != nil || got != want {
		t.Fatalf("legacy endpoint = %q, %v; want %q", got, err, want)
	}
}

func TestSocketPathIsolationConfinesBrokerAndWrapperWithoutEffects(t *testing.T) {
	root := filepath.Dir(transportSocket(t))
	t.Setenv("COUCH_ISOLATED_ROOT", root)
	namespace := filepath.Join(root, "store")
	broker, err := SocketPath(namespace, "broker")
	if err != nil {
		t.Fatal(err)
	}
	wrapper, err := EndpointSocket(namespace, protocolBinding("pair:0"))
	if err != nil {
		t.Fatal(err)
	}
	physical, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	wantParent := filepath.Join(physical, "m")
	if filepath.Dir(broker) != wantParent || filepath.Dir(wrapper) != wantParent || broker == wrapper {
		t.Fatalf("unconfined paths: broker=%q wrapper=%q want parent=%q", broker, wrapper, wantParent)
	}
	if entries, err := os.ReadDir(root); err != nil || len(entries) != 0 {
		t.Fatalf("derivation created artifacts: %v %v", entries, err)
	}
}

func TestSocketPathIsolationRejectsInvalidPhysicalScopeBeforeEffects(t *testing.T) {
	for _, scenario := range []string{"relative-root", "file-root", "namespace-escape", "namespace-symlink", "parent-symlink", "dangling-root", "overlong-root", "overlong-physical-root"} {
		t.Run(scenario, func(t *testing.T) {
			root := filepath.Dir(transportSocket(t))
			outside := filepath.Dir(transportSocket(t))
			namespace := filepath.Join(root, "store")
			selected := root
			switch scenario {
			case "relative-root":
				selected = "relative"
			case "file-root":
				selected = filepath.Join(root, "file")
				if err := os.WriteFile(selected, nil, 0600); err != nil {
					t.Fatal(err)
				}
				namespace = filepath.Join(selected, "store")
			case "namespace-escape":
				namespace = filepath.Join(outside, "store")
			case "namespace-symlink":
				if err := os.Symlink(outside, namespace); err != nil {
					t.Fatal(err)
				}
			case "parent-symlink":
				if err := os.Symlink(outside, filepath.Join(root, "m")); err != nil {
					t.Fatal(err)
				}
			case "dangling-root":
				selected = filepath.Join(root, "dangling")
				if err := os.Symlink(filepath.Join(outside, "missing"), selected); err != nil {
					t.Fatal(err)
				}
				namespace = filepath.Join(selected, "store")
			case "overlong-root":
				selected = filepath.Join(root, strings.Repeat("x", 100))
				namespace = filepath.Join(selected, "store")
			case "overlong-physical-root":
				target := filepath.Join(outside, strings.Repeat("x", 100))
				if err := os.Mkdir(target, 0700); err != nil {
					t.Fatal(err)
				}
				selected = filepath.Join(root, "alias")
				if err := os.Symlink(target, selected); err != nil {
					t.Fatal(err)
				}
				namespace = filepath.Join(selected, "store")
			}
			before, err := os.ReadDir(outside)
			if err != nil {
				t.Fatal(err)
			}
			t.Setenv("COUCH_ISOLATED_ROOT", selected)
			if socket, err := SocketPath(namespace, "broker"); err == nil {
				t.Fatalf("accepted %s: %s", scenario, socket)
			}
			if entries, err := os.ReadDir(outside); err != nil || len(entries) != len(before) {
				t.Fatalf("outside effects: %v %v", entries, err)
			}
		})
	}
}

func deadWrapperResidue(t *testing.T, dir string) (string, os.FileInfo) {
	t.Helper()
	const absentPID = 2147483647
	if err := syscall.Kill(absentPID, 0); err != syscall.ESRCH {
		t.Fatalf("fixture PID not proven dead: %v", err)
	}
	path := filepath.Join(dir, fmt.Sprintf("wrapper-%d-%s.sock", absentPID, strings.Repeat("a", 40)))
	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: path, Net: "unix"})
	if err != nil {
		t.Fatal(err)
	}
	listener.SetUnlinkOnClose(false)
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	return path, info
}

func TestIsolatedStartServerSweepsOnlyOwnedParent(t *testing.T) {
	shared := filepath.Dir(transportSocket(t))
	foreign, identity := deadWrapperResidue(t, shared)
	root := filepath.Join(shared, "i")
	if err := os.Mkdir(root, 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("COUCH_ISOLATED_ROOT", root)
	socket, err := SocketPath(filepath.Join(root, "store"), "broker")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Dir(socket), 0700); err != nil {
		t.Fatal(err)
	}
	owned, _ := deadWrapperResidue(t, filepath.Dir(socket))
	server, err := StartServer(context.Background(), socket, socketCleanupEcho)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	if _, err := os.Lstat(owned); !os.IsNotExist(err) {
		t.Fatalf("owned dead socket not swept: %v", err)
	}
	if info, err := os.Lstat(foreign); err != nil || !os.SameFile(identity, info) {
		t.Fatalf("foreign inode changed: %v", err)
	}
	if err := server.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(socket); !os.IsNotExist(err) {
		t.Fatalf("listener socket retained: %v", err)
	}
	if info, err := os.Lstat(foreign); err != nil || !os.SameFile(identity, info) {
		t.Fatalf("foreign inode changed on close: %v", err)
	}
}

func TestUnsetStartServerStillSweepsDisposableSharedParent(t *testing.T) {
	t.Setenv("COUCH_ISOLATED_ROOT", "")
	socket := transportSocket(t)
	dead, _ := deadWrapperResidue(t, filepath.Dir(socket))
	server, err := StartServer(context.Background(), socket, socketCleanupEcho)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	if _, err := os.Lstat(dead); !os.IsNotExist(err) {
		t.Fatalf("legacy sweep changed: %v", err)
	}
}

func TestIsolatedListenerRefusesForeignParentBeforeSweep(t *testing.T) {
	root := filepath.Dir(transportSocket(t))
	foreignSocket := transportSocket(t)
	foreign, identity := deadWrapperResidue(t, filepath.Dir(foreignSocket))
	t.Setenv("COUCH_ISOLATED_ROOT", root)
	if server, err := StartServer(context.Background(), foreignSocket, socketCleanupEcho); err == nil {
		server.Close()
		t.Fatal("isolated listener admitted a foreign parent")
	}
	if info, err := os.Lstat(foreign); err != nil || !os.SameFile(identity, info) {
		t.Fatalf("refused startup touched foreign inode: %v", err)
	}
	if entries, err := os.ReadDir(root); err != nil || len(entries) != 0 {
		t.Fatalf("refused startup created paths: %v %v", entries, err)
	}
}

func TestIsolatedEndpointRejectsFinalAddressOverflow(t *testing.T) {
	root := filepath.Dir(transportSocket(t))
	selected := filepath.Join(root, strings.Repeat("x", 25))
	t.Setenv("COUCH_ISOLATED_ROOT", selected)
	namespace := filepath.Join(selected, "store")
	if _, err := SocketPath(namespace, "broker"); err != nil {
		t.Fatalf("fixture broker should fit: %v", err)
	}
	binding := protocolBinding("pair:0")
	binding.PID = 2147483647
	if socket, err := EndpointSocket(namespace, binding); err == nil {
		t.Fatalf("overlong wrapper admitted: %q (%d bytes)", socket, len(socket))
	}
	if entries, err := os.ReadDir(root); err != nil || len(entries) != 0 {
		t.Fatalf("overflow created paths: %v %v", entries, err)
	}
}
