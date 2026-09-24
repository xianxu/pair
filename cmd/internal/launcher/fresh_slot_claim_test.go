package launcher

import (
	"bytes"
	"os"
	"testing"
)

func TestFreshSlotLaunchUsesRealClaim(t *testing.T) {
	for _, tc := range []struct {
		name, state         string
		owned, resume, want bool
	}{
		{"new", "reserved", true, false, true},
		{"existing", "established", true, false, true},
		{"resume cannot establish", "reserved", true, true, false},
		{"unowned cannot establish", "reserved", false, false, false},
		{"missing", "", true, false, false},
		{"malformed", "malformed", true, false, false},
		{"wrong identity", "wrong", true, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			global := t.TempDir()
			scope, err := ResolveRepoScope("/home/u/work")
			if err != nil {
				t.Fatal(err)
			}
			paths := NewScopedPaths(global, scope, "work")
			if tc.state != "" {
				claim, err := ClaimNewThreadAddress(global, scope, "work")
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = claim.Release() })
				if tc.state == "established" {
					if err := EnsureThreadAddressForPair(global, scope, "work", true); err != nil {
						t.Fatal(err)
					}
				} else if tc.state == "malformed" || tc.state == "wrong" {
					raw := []byte("not-json")
					if tc.state == "wrong" {
						raw = []byte(`{"schema":1,"scope":"wrong","tag":"work","state":"reserved"}`)
					}
					if err := os.WriteFile(paths.ThreadClaim(), raw, 0600); err != nil {
						t.Fatal(err)
					}
				}
			}
			before, _ := os.ReadFile(paths.ThreadClaim())
			fake := newFakeRuntime()
			rt := checkpointClaimRuntime{fake, global}
			opts := baseOpts(LaunchArgs{Agent: "claude", ForcedTag: "work", FreshRequired: !tc.resume, ResumeRequired: tc.resume, RequiredSessionID: ""})
			if tc.resume {
				opts.Args.RequiredSessionID = "native"
			}
			if tc.owned {
				opts.Env.CouchThreadScope = scope.Key
				opts.Env.CouchThreadTag = "work"
			}
			var diagnostic bytes.Buffer
			code, err := RunLaunch(opts, rt, &diagnostic)
			if err != nil || (code == 0) != tc.want || (fake.launchCount == 1) != tc.want {
				t.Fatalf("code=%d err=%v launches=%d: %s", code, err, fake.launchCount, diagnostic.String())
			}
			after, _ := os.ReadFile(paths.ThreadClaim())
			if !tc.want || tc.state == "established" {
				if !bytes.Equal(before, after) {
					t.Fatal("claim mutated")
				}
			} else if established, err := ThreadAddressEstablished(global, scope, "work"); err != nil || !established {
				t.Fatalf("fresh claim not established: %v %v", established, err)
			}
		})
	}
}
