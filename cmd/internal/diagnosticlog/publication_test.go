package diagnosticlog

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestRegistryPagesDoNotConfuseResiduesWithCompletion(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for n := 0; n < 101; n++ {
		path := filepath.Join(root, fmt.Sprintf("trace%d.log", n))
		if err := register(context.Background(), root, RegistryEntry{Version: 1, Path: path, Directory: directory(path), Lock: lockPath(path)}); err != nil {
			t.Fatal(err)
		}
	}
	for n := 0; n < 101; n++ {
		if err := os.WriteFile(filepath.Join(RegistryDirectory(root), fmt.Sprintf(".pending-%d", n)), nil, 0600); err != nil {
			t.Fatal(err)
		}
	}
	count := 0
	offset := 0
	for pages := 0; pages < 10; pages++ {
		page, err := EnumerateRoot(context.Background(), root, offset, 100)
		if err != nil {
			t.Fatal(err)
		}
		count += len(page.Entries)
		if page.Complete {
			if count != 101 {
				t.Fatalf("premature completion %d/101", count)
			}
			return
		}
		if page.NextOffset <= offset {
			t.Fatal("cursor stalled")
		}
		offset = page.NextOffset
	}
	t.Fatal("registry never completed")
}

func TestRegistryKilledPublisherAndBusyRecovery(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	log := filepath.Join(root, "trace.log")
	cmd := exec.Command(os.Args[0], "-test.run=^TestRegistryPublicationChild$")
	cmd.Env = append(os.Environ(), "PAIR_REGISTRY_CHILD="+root)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	done := false
	defer func() {
		if !done {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	}()
	ready := make(chan string, 1)
	go func() { line, _ := bufio.NewReader(stdout).ReadString('\n'); ready <- line }()
	select {
	case line := <-ready:
		if line != "staged\n" {
			t.Fatalf("child stage %q", line)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("publisher did not stage")
	}
	if err := RecoverRegistry(context.Background(), root); err != ErrBusy {
		t.Fatalf("active publisher cleanup %v", err)
	}
	other := RegistryEntry{Version: 1, Path: log, Directory: directory(log), Lock: lockPath(log)}
	if err := register(context.Background(), root, other); err != ErrBusy {
		t.Fatalf("concurrent registry writer ignored lock %v", err)
	}
	page, err := EnumerateRoot(context.Background(), root, 0, 100)
	if err != nil || len(page.Entries) != 0 {
		t.Fatal("unpublished registry acquired authority", page, err)
	}
	if _, err := os.Stat(publicationPath(RegistryDirectory(root))); err != nil {
		t.Fatal("live stage removed", err)
	}
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = cmd.Wait()
	done = true
	if err := RecoverRegistry(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(publicationPath(RegistryDirectory(root))); !os.IsNotExist(err) {
		t.Fatal("dead publication retained", err)
	}
	if err := register(context.Background(), root, other); err != nil {
		t.Fatal(err)
	}
	page, err = EnumerateRoot(context.Background(), root, 0, 100)
	if err != nil || len(page.Entries) != 1 {
		t.Fatal("registry retry failed", page, err)
	}
}

func TestRegistryPublicationChild(t *testing.T) {
	root := os.Getenv("PAIR_REGISTRY_CHILD")
	if root == "" {
		return
	}
	path := filepath.Join(root, "trace.log")
	err := registerWithHook(context.Background(), root, RegistryEntry{Version: 1, Path: path, Directory: directory(path), Lock: lockPath(path)}, func() error {
		fmt.Println("staged")
		for {
			time.Sleep(time.Hour)
		}
	})
	t.Fatal("stage hook returned", err)
}

func TestDiagnosticPublicationRecoveryIsExactAndReadOnlyPreview(t *testing.T) {
	path, _, opts := fixture(t)
	w, err := Open(path, opts)
	if err != nil {
		t.Fatal(err)
	}
	w.Close()
	stage := publicationPath(directory(path))
	if err := os.WriteFile(stage, []byte("interrupted"), 0600); err != nil {
		t.Fatal(err)
	}
	old := filepath.Join(directory(path), ".pending-not-owned")
	if err := os.WriteFile(old, []byte("retain"), 0600); err != nil {
		t.Fatal(err)
	}
	_, _ = Preview(path, opts, 100)
	if _, err := os.Stat(stage); err != nil {
		t.Fatal("preview removed stage", err)
	}
	if _, err := Collect(path, opts, 100); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(stage); !os.IsNotExist(err) {
		t.Fatal("apply retained stage", err)
	}
	if raw, err := os.ReadFile(old); err != nil || string(raw) != "retain" {
		t.Fatal("cleanup swept generic residue", err)
	}
}

func TestDiagnosticMetadataKilledPublisherUsesCentralStage(t *testing.T) {
	for _, kind := range []string{"state", "segment"} {
		t.Run(kind, func(t *testing.T) {
			path, _, opts := fixture(t)
			w, err := Open(path, opts)
			if err != nil {
				t.Fatal(err)
			}
			w.Close()
			target := filepath.Join(directory(path), "state.json")
			if kind == "segment" {
				name := "segment-0123456789abcdef0123456789abcdef.log"
				if err := ensureSegmentDirectory(path, name, opts); err != nil {
					t.Fatal(err)
				}
				target = segmentPath(path, name) + ".json"
			}
			before, beforeErr := os.ReadFile(target)
			cmd := exec.Command(os.Args[0], "-test.run=^TestDiagnosticMetadataPublisherChild$")
			cmd.Env = append(os.Environ(), "PAIR_METADATA_LOG="+path, "PAIR_METADATA_TARGET="+target)
			stdout, err := cmd.StdoutPipe()
			if err != nil {
				t.Fatal(err)
			}
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			done := false
			defer func() {
				if !done {
					_ = cmd.Process.Kill()
					_ = cmd.Wait()
				}
			}()
			ready := make(chan string, 1)
			go func() { line, _ := bufio.NewReader(stdout).ReadString('\n'); ready <- line }()
			select {
			case line := <-ready:
				if line != "staged\n" {
					t.Fatalf("no staged publication %q", line)
				}
			case <-time.After(10 * time.Second):
				t.Fatal("metadata child timeout")
			}
			if _, err := Collect(path, opts, 100); err != ErrBusy {
				t.Fatalf("collector ignored live publisher lock %v", err)
			}
			after, afterErr := os.ReadFile(target)
			if !bytes.Equal(before, after) || os.IsNotExist(beforeErr) != os.IsNotExist(afterErr) {
				t.Fatal("target acquired authority before rename")
			}
			if err := cmd.Process.Kill(); err != nil {
				t.Fatal(err)
			}
			_ = cmd.Wait()
			done = true
			if _, err := Collect(path, opts, 100); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(publicationPath(directory(path))); !os.IsNotExist(err) {
				t.Fatal("central stage retained", err)
			}
		})
	}
}

func TestDiagnosticMetadataPublisherChild(t *testing.T) {
	path := os.Getenv("PAIR_METADATA_LOG")
	if path == "" {
		return
	}
	err := locked(path, false, func() error {
		return publishJSON(directory(path), os.Getenv("PAIR_METADATA_TARGET"), map[string]string{"test": "unpublished"}, true, Options{}, func() error {
			fmt.Println("staged")
			for {
				time.Sleep(time.Hour)
			}
		})
	})
	t.Fatal("metadata barrier returned", err)
}

func TestDiagnosticPublicationCancellationBeforeRename(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target.json")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err := publishJSON(dir, target, map[string]string{"unpublished": "value"}, true, Options{Context: ctx}, func() error { cancel(); return nil })
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("context ignored %v", err)
	}
	for _, path := range []string{target, publicationPath(dir)} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatal("canceled write retained authority or stage", path, err)
		}
	}
}

func TestCanceledRegistryEnumerationAndRecoveryMakeNoProgress(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "trace.log")
	if err := register(context.Background(), root, RegistryEntry{Version: 1, Path: path, Directory: directory(path), Lock: lockPath(path)}); err != nil {
		t.Fatal(err)
	}
	stage := publicationPath(RegistryDirectory(root))
	if err := os.WriteFile(stage, []byte("retain until writable recovery"), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	page, err := EnumerateRoot(ctx, root, 100000, 100)
	if !errors.Is(err, context.Canceled) || page.NextOffset != 0 || page.Complete || len(page.Entries) != 0 {
		t.Fatalf("canceled enumeration advanced %+v %v", page, err)
	}
	if err := RecoverRegistry(ctx, root); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled registry recovery %v", err)
	}
	raw, err := os.ReadFile(stage)
	if err != nil || string(raw) != "retain until writable recovery" {
		t.Fatal("canceled recovery changed stage", err)
	}
}
