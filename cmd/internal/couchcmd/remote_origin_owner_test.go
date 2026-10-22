package couchcmd

import (
	"bytes"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/couchtty"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
)

func TestRemoteOriginOwnerExistsWithoutProtectedConfiguration(t *testing.T) {
	namespace := t.TempDir()
	console := couchtty.New(hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 100}), bytes.NewReader(nil))
	t.Cleanup(func() { console.Stop(); console.Run() })
	service := &messageService{}
	service.configureRemoteSession(console, namespace, func(string) string { return "" })
	if service.origins == nil || service.originErr != nil || service.originNamespace != namespace {
		t.Fatalf("local origin owner missing: %+v", service.originErr)
	}
	const id = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	if err := service.origins.MarkRemote(id); err != nil {
		t.Fatal(err)
	}
	if service.remote.Load().supervisor != nil {
		t.Fatal("unprotected configuration enabled remote")
	}
}

func TestRemoteOriginOwnerFailurePreventsSupervisorFallback(t *testing.T) {
	namespace := t.TempDir()
	paths, err := artifactpath.ResolveCouchRemote(namespace)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(paths.Origins()), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.Origins(), []byte("corrupt"), 0600); err != nil {
		t.Fatal(err)
	}
	console := couchtty.New(hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 100}), bytes.NewReader(nil))
	t.Cleanup(func() { console.Stop(); console.Run() })
	service := &messageService{}
	env := map[string]string{"COUCH_BROADCAST_TUNNEL": "remote-test", "COUCH_BROADCAST_HOSTNAME": "remote.example.com", "COUCH_REMOTE_ACCESS_TEAM": "operator", "COUCH_REMOTE_ACCESS_AUDIENCE": strings.Repeat("a", 64), "COUCH_REMOTE_ALLOWED_EMAILS": "human@example.com"}
	service.configureRemoteSession(console, namespace, func(key string) string { return env[key] })
	if service.origins != nil || service.originErr == nil {
		t.Fatal("corrupt origin owner admitted")
	}
	if service.remote.Load().supervisor != nil {
		t.Fatal("fallback supervisor created a second owner")
	}
}

func TestRemoteOriginOwnerPublicationOrdersLocalForget(t *testing.T) {
	namespace := t.TempDir()
	console := couchtty.New(hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 100}), bytes.NewReader(nil))
	t.Cleanup(func() { console.Stop(); console.Run() })
	service := &messageService{}
	var workers sync.WaitGroup
	workers.Add(2)
	start := make(chan struct{})
	go func() {
		defer workers.Done()
		<-start
		service.configureRemoteSession(console, namespace, func(string) string { return "" })
	}()
	go func() {
		defer workers.Done()
		<-start
		for range 100 {
			service.handleSessionForget(couchmessage.Request{Op: "session-forget", Target: filepath.Join(namespace, "absent.md")})
		}
	}()
	close(start)
	workers.Wait()
}

func TestRemoteOriginOwnerReusedForConfiguredNamespace(t *testing.T) {
	namespace := t.TempDir()
	console := couchtty.New(hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 100}), bytes.NewReader(nil))
	t.Cleanup(func() { console.Stop(); console.Run() })
	service := &messageService{}
	absent := func(string) string { return "" }
	service.configureRemoteSession(console, namespace, absent)
	owner := service.origins
	if owner == nil {
		t.Fatal(service.originErr)
	}
	service.configureRemoteSession(console, namespace, absent)
	if service.origins != owner {
		t.Fatal("configuration replaced sole origin writer")
	}
}
