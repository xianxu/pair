package couchcore

import (
	"bytes"
	"context"
	"os"
	"reflect"
	"testing"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

func TestRepoConfigurationSessionRegistrationPreservesLaterRepositoryEdit(t *testing.T) {
	for _, tc := range []struct {
		agent string
		argv  []string
	}{
		{"codex", []string{"--sandbox", "danger-full-access"}},
		{"claude", []string{"--dangerously-skip-permissions"}},
	} {
		t.Run(tc.agent, func(t *testing.T) {
			env, in := acceptedTrackedFixture(t, false)
			stale := LaunchProfile{Agent: tc.agent, Argv: []string{"--model", "stale-session"}}
			var err error
			in.Thread, err = env.Couch.Threads.updateExistingThread(in.Thread.Address, in.Thread.Revision, func(next *ThreadRecord) error {
				next.Incarnations[0].Start.LaunchProfile = &stale
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			in.Args.Stack = tc.agent
			in.ProfileRaw, err = launcher.BuildCouchLaunchProfile(string(in.Thread.Address.Tag), tc.agent, stale.Argv, "explicit", "explicit")
			if err != nil {
				t.Fatal(err)
			}
			wanted := LaunchProfile{Agent: tc.agent, Argv: tc.argv}
			if err := env.Couch.Threads.rememberRepoConfiguration("/repo/.git", wanted, nil); err != nil {
				t.Fatal(err)
			}
			before, err := os.ReadFile(env.Couch.Threads.repoPreferencesPath())
			if err != nil {
				t.Fatal(err)
			}

			// The old session may persist its exact invocation again, but that is
			// historical evidence, not another acceptance of repository settings.
			record, handle, err := env.Couch.launchTrackedThread(in)
			if err != nil {
				t.Fatal(err)
			}
			env.Couch.PublishAdoptedConfiguration(StartResult{Record: record, Handle: handle})
			after, err := os.ReadFile(env.Couch.Threads.repoPreferencesPath())
			if err != nil || !bytes.Equal(before, after) {
				t.Fatalf("saved session replaced the repository edit: %s, %v", after, err)
			}
			history, err := env.Couch.Threads.GetThread(in.Thread.Address)
			if err != nil || history.LatestLaunchProfile == nil || !reflect.DeepEqual(*history.LatestLaunchProfile, stale) {
				t.Fatalf("historical invocation was not preserved: %+v, %v", history.LatestLaunchProfile, err)
			}
			resolved, err := env.Couch.resolveOrdinaryStartResolution(context.Background(), StartArgs{Cwd: "/repo", Stack: tc.agent})
			if err != nil || !reflect.DeepEqual(resolved.Profile, wanted) || resolved.ArgvSource != ArgvSourceRepository {
				t.Fatalf("next launch ignored the Couch repository edit: %+v, %v", resolved, err)
			}
		})
	}
}
