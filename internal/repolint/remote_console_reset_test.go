package repolint

import (
	"go/parser"
	"go/token"
	"testing"
)

func TestRemoteConsoleResetGuard(t *testing.T) {
	for _, tc := range []struct {
		name, source string
		want         int
	}{
		{"foreign call", `package couchtty;func (c *Console) unsafe(){c.remoteConsoleReset()}`, 1},
		{"foreign alias", `package couchtty;func (c *Console) unsafe(){reset:=c.remoteConsoleReset;reset()}`, 1},
		{"test name bypass", `package couchtty;func TestReset(){c.remoteConsoleReset()}`, 1},
		{"real test", `package couchtty;import "testing";func TestReset(t *testing.T){c.remoteConsoleReset()}`, 0},
		{"owned key", `package couchtty;func (c *Console) routeRemoteSessionKey(key uv.KeyPressEvent){reset:=c.remoteConsoleReset;reset()}`, 0},
		{"owned key closure escape", `package couchtty;func (c *Console) routeRemoteSessionKey(key uv.KeyPressEvent)func()error{reset:=c.remoteConsoleReset;return reset}`, 1},
		{"owned key async escape", `package couchtty;func (c *Console) routeRemoteSessionKey(key uv.KeyPressEvent){reset:=c.remoteConsoleReset;go reset()}`, 1},
		{"foreign method named owner", `package couchtty;func (c *Other) routeRemoteSessionKey(key uv.KeyPressEvent){c.remoteConsoleReset()}`, 1},
		{"foreign key route", `package couchtty;func (c *Console) brokerKeys(){c.routeRemoteSessionKey(key)}`, 1},
		{"paste branch", `package couchtty;func(c *Console)routeInputEvent(event terminal.InputEvent){switch event.Event.(type){case uv.PasteEvent:c.routeRemoteSessionKey(key)}}`, 1},
		{"key branch", `package couchtty;func(c *Console)routeInputEvent(event terminal.InputEvent){switch event.Event.(type){case uv.KeyPressEvent:c.routeRemoteSessionKey(key)}}`, 0},
		{"foreign constructor", `package couchcmd;import remote "github.com/xianxu/pair/cmd/internal/remotesession";func unsafe(){_,reset,_:=remote.NewSupervisor(cfg);reset()}`, 1},
		{"composition constructor", `package couchcmd;import "github.com/xianxu/pair/cmd/internal/remotesession";func(s *messageService)configureRemoteSession(){_,reset,_:=remotesession.NewSupervisor(cfg);c.SetRemoteSession(s,reset,"")}`, 0},
		{"foreign setter", `package elsewhere;func wire(){c.SetRemoteSession(s,reset,"")}`, 1},
		{"composition capture", `package couchcmd;func(s *messageService)configureRemoteSession(){c.SetRemoteSession(s,reset,"")}`, 0},
		{"store escape", `package remotesession;func expose(s *Store)func()error{return s.consoleReset}`, 1},
		{"internal store reset", `package remotesession;func(s *Supervisor)consoleReset(){s.store.consoleReset()}`, 0},
		{"internal constructor", `package remotesession;func NewSupervisor(){return s.consoleReset}`, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			file, err := parser.ParseFile(token.NewFileSet(), "fixture.go", tc.source, 0)
			if err != nil {
				t.Fatal(err)
			}
			matches := matchRemoteConsoleReset(file)
			if len(matches) != tc.want {
				t.Fatalf("matches=%d want=%d", len(matches), tc.want)
			}
		})
	}
}

func TestRemoteConsoleResetCatalogRejectsSeededForeignCall(t *testing.T) {
	source := []byte("package couchtty\nfunc(c *Console)brokerKeys(){reset:=c.remoteConsoleReset;reset()}\n")
	repo := newFakeRepository()
	repo.index["cmd/internal/couchtty/seed.go"] = source
	findings, _, count, err := scanTrackedWithRepository(repo, "repo", func(string) ([]byte, error) { return source, nil })
	if err != nil || count != 1 || len(findings) != 1 || findings[0].Finding.Rule != "remote-console-reset" {
		t.Fatalf("findings=%v count=%d err=%v", findings, count, err)
	}
}

func TestAdditivePTYEnvironmentGuard(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		want       int
	}{
		{"inherits live identity", `env:=testpair.Environment(t,root);ptychild.Start(ptychild.Options{Env:env})`, 1},
		{"clears identity", `env:=testpair.Environment(t,root);testpair.ApplyEnvironment(t,root,env...);ptychild.Start(ptychild.Options{Env:env})`, 0},
		{"clears too late", `env:=testpair.Environment(t,root);ptychild.Start(ptychild.Options{Env:env});testpair.ApplyEnvironment(t,root,env...)`, 1},
		{"conditional clearing", `env:=testpair.Environment(t,root);if condition {testpair.ApplyEnvironment(t,root,env...)};ptychild.Start(ptychild.Options{Env:env})`, 1},
		{"unrelated launch", `exec.Command("fixture")`, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := `package fixture;import("github.com/xianxu/pair/cmd/internal/testpair";"github.com/xianxu/pair/cmd/internal/ptychild");func TestFixture(){` + tc.body + `}`
			file, err := parser.ParseFile(token.NewFileSet(), "fixture_test.go", source, 0)
			if err != nil {
				t.Fatal(err)
			}
			if got := len(matchAdditivePTYEnvironment(file)); got != tc.want {
				t.Fatalf("got=%d want=%d", got, tc.want)
			}
		})
	}
}
