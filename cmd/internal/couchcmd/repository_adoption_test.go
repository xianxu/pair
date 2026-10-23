package couchcmd

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchtty"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
)

func TestRepositoryConfigurationPublicationFollowsActualConsoleAttach(t *testing.T) {
	for _, scenario := range []string{"attached", "attach-refused", "publication-refused"} {
		t.Run(scenario, func(t *testing.T) {
			rt := newRT(t, "/repo")
			rt.runner = couchcore.NewFakeRunner()
			c, err := rt.NewCouch()
			if err != nil {
				t.Fatal(err)
			}
			actor, handle, err := c.Spawn(couchcore.StartArgs{Worktree: "/repo", Stack: "codex", ExtraArgs: []string{"--model", "accepted"}})
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { rt.runner.SetExited(handle.ID(), 0) })
			start, _ := (couchcore.StartResult{Record: actor, Handle: handle}).Started()
			if start.Configuration == nil {
				t.Fatal("explicit start lost learning intent")
			}
			path := filepath.Join(c.Namespace.Dir(), "threadstore", "repo-preferences.json")
			if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("registration published settings: %v", err)
			}
			if scenario == "publication-refused" {
				if err := os.Mkdir(path, 0700); err != nil {
					t.Fatal(err)
				}
			}
			console := couchtty.New(hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 100}), strings.NewReader(""))
			t.Cleanup(console.Stop)
			wireResolver(console, c)
			if scenario == "attach-refused" {
				start.Record.PID++
			}
			err = dispatchInitialAttach(console, start)
			if scenario == "attach-refused" {
				if err == nil {
					t.Fatal("mismatched attachment accepted")
				}
				if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("failed adoption published: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if scenario == "publication-refused" {
				if !handle.Alive() || !strings.Contains(start.Warning(), "startup parameters were not remembered") {
					t.Fatalf("live=%v warning=%q", handle.Alive(), start.Warning())
				}
				wrapped := notedResult{value: start, note: "existing warning"}
				if !strings.Contains(wrapped.Warning(), "existing warning") || !strings.Contains(wrapped.Warning(), "startup parameters were not remembered") {
					t.Fatal(wrapped.Warning())
				}
				return
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			doc, err := couchcore.DecodeRepoPreferences(raw)
			if err != nil {
				t.Fatal(err)
			}
			got := doc.Repositories["/repo/.git"].Agents["codex"]
			if !got.Confirmed || !reflect.DeepEqual(got.Argv, []string{"--model", "accepted"}) {
				t.Fatal(string(raw))
			}
			// The opaque publication capability is never durable registry state.
			var record map[string]json.RawMessage
			encoded, err := json.Marshal(actor)
			if err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(encoded, &record); err != nil {
				t.Fatal(err)
			}
			if _, ok := record["configuration"]; ok {
				t.Fatal("publication capability serialized")
			}
		})
	}
}
