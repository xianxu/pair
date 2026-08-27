package couchcore

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

const testPolicyDigest = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func TestPolicyResolverDecodesStrictSuccessVariants(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want PolicyResult
	}{
		{
			name: "bounded reject",
			raw:  `{"ok":true,"value":{"policy_version":1,"policy_digest":"` + testPolicyDigest + `","repo_identity":"/repo/.git","admission_key":"/repo","capacity":{"kind":"bounded","limit":2},"on_capacity":"reject"}}`,
			want: PolicyResult{PolicyVersion: 1, PolicyDigest: testPolicyDigest, RepoIdentity: "/repo/.git", AdmissionKey: "/repo", Capacity: PolicyCapacity{Kind: CapacityBounded, Limit: 2}, OnCapacity: CapacityReject},
		},
		{
			name: "bounded provision worktree",
			raw:  `{"ok":true,"value":{"policy_version":1,"policy_digest":"` + testPolicyDigest + `","repo_identity":"/repo/.git","admission_key":"/repo/wt","capacity":{"kind":"bounded","limit":1},"on_capacity":"provision-worktree"}}`,
			want: PolicyResult{PolicyVersion: 1, PolicyDigest: testPolicyDigest, RepoIdentity: "/repo/.git", AdmissionKey: "/repo/wt", Capacity: PolicyCapacity{Kind: CapacityBounded, Limit: 1}, OnCapacity: CapacityProvisionWorktree},
		},
		{
			name: "unbounded",
			raw:  `{"ok":true,"value":{"policy_version":1,"policy_digest":"` + testPolicyDigest + `","repo_identity":"/brain/.git","admission_key":"/brain","capacity":{"kind":"unbounded"}}}`,
			want: PolicyResult{PolicyVersion: 1, PolicyDigest: testPolicyDigest, RepoIdentity: "/brain/.git", AdmissionKey: "/brain", Capacity: PolicyCapacity{Kind: CapacityUnbounded}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DecodePolicyResponse([]byte(tt.raw), nil, 0)
			if err != nil {
				t.Fatalf("DecodePolicyResponse: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("result = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestPolicyResolverRejectsInvalidEnvelopeValueAndProcessCombinations(t *testing.T) {
	valid := `{"ok":true,"value":{"policy_version":1,"policy_digest":"` + testPolicyDigest + `","repo_identity":"/repo/.git","admission_key":"/repo","capacity":{"kind":"unbounded"}}}`
	tests := []struct {
		name     string
		stdout   string
		stderr   string
		exitCode int
	}{
		{name: "missing discriminator", stdout: `{"value":{}}`},
		{name: "success with diagnostic", stdout: `{"ok":true,"value":{"policy_version":1,"policy_digest":"` + testPolicyDigest + `","repo_identity":"/r","admission_key":"/r","capacity":{"kind":"unbounded"}},"diagnostic":{"code":"bad","message":"bad"}}`},
		{name: "unknown top field", stdout: strings.TrimSuffix(valid, "}") + `,"extra":true}`},
		{name: "unknown nested field", stdout: strings.Replace(valid, `"kind":"unbounded"`, `"kind":"unbounded","extra":true`, 1)},
		{name: "duplicate key", stdout: strings.Replace(valid, `"ok":true`, `"ok":true,"ok":true`, 1)},
		{name: "trailing value", stdout: valid + `{}`},
		{name: "unsupported version", stdout: strings.Replace(valid, `"policy_version":1`, `"policy_version":2`, 1)},
		{name: "invalid digest", stdout: strings.Replace(valid, testPolicyDigest, "not-a-digest", 1)},
		{name: "empty repo identity", stdout: strings.Replace(valid, `"repo_identity":"/repo/.git"`, `"repo_identity":""`, 1)},
		{name: "bounded without limit", stdout: strings.Replace(valid, `"kind":"unbounded"`, `"kind":"bounded"`, 1)},
		{name: "unbounded with limit", stdout: strings.Replace(valid, `"kind":"unbounded"`, `"kind":"unbounded","limit":1`, 1)},
		{name: "success nonzero", stdout: valid, exitCode: 1},
		{name: "success stderr", stdout: valid, stderr: "warning\n"},
		{name: "crash exit", stdout: valid, exitCode: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, err := DecodePolicyResponse([]byte(tt.stdout), []byte(tt.stderr), tt.exitCode); err == nil {
				t.Fatalf("accepted invalid response as %#v", got)
			}
		})
	}
}

func TestPolicyResolverReturnsTypedDiagnosticOnlyForExactRefusal(t *testing.T) {
	raw := `{"ok":false,"diagnostic":{"code":"missing-policy","message":"fleet policy declaration is missing","path":"/repo/.sdlc/fleet.json"}}`
	stderr := "Error: fleet policy refused: missing-policy\n"
	_, err := DecodePolicyResponse([]byte(raw), []byte(stderr), 1)
	var refusal *PolicyRefusal
	if !errors.As(err, &refusal) {
		t.Fatalf("err = %T %v, want *PolicyRefusal", err, err)
	}
	if refusal.Diagnostic.Code != "missing-policy" || refusal.Diagnostic.Path != "/repo/.sdlc/fleet.json" {
		t.Fatalf("diagnostic = %#v", refusal.Diagnostic)
	}

	for _, mismatch := range []struct {
		stderr string
		exit   int
	}{
		{stderr: stderr, exit: 0},
		{stderr: "Error: fleet policy refused: other\n", exit: 1},
		{stderr: stderr + "Usage: sdlc fleet policy\n", exit: 1},
	} {
		_, err := DecodePolicyResponse([]byte(raw), []byte(mismatch.stderr), mismatch.exit)
		if errors.As(err, &refusal) {
			t.Fatalf("mismatched process response became typed refusal: stderr=%q exit=%d", mismatch.stderr, mismatch.exit)
		}
		if err == nil {
			t.Fatalf("accepted mismatched refusal: stderr=%q exit=%d", mismatch.stderr, mismatch.exit)
		}
	}
}

func TestPolicyResolverFakeIsStateful(t *testing.T) {
	fake := NewFakePolicyResolver()
	first := PolicyResult{PolicyVersion: 1, PolicyDigest: testPolicyDigest, RepoIdentity: "/repo/.git", AdmissionKey: "/repo", Capacity: PolicyCapacity{Kind: CapacityBounded, Limit: 1}, OnCapacity: CapacityReject}
	second := first
	second.Capacity.Limit = 2
	fake.Queue("/repo", first, nil)
	fake.Queue("/repo", second, nil)

	gotFirst, err := fake.ResolvePolicy(context.Background(), "/repo")
	if err != nil || gotFirst.Capacity.Limit != 1 {
		t.Fatalf("first = %#v, %v", gotFirst, err)
	}
	gotSecond, err := fake.ResolvePolicy(context.Background(), "/repo")
	if err != nil || gotSecond.Capacity.Limit != 2 {
		t.Fatalf("second = %#v, %v", gotSecond, err)
	}
	if !reflect.DeepEqual(fake.Calls(), []string{"/repo", "/repo"}) {
		t.Fatalf("calls = %v", fake.Calls())
	}
}

func TestPolicyResolverExecUsesDeadlineAndExactCommand(t *testing.T) {
	var gotName string
	var gotArgs []string
	runner := PolicyCommandFunc(func(ctx context.Context, name string, args ...string) (PolicyCommandOutput, error) {
		gotName, gotArgs = name, append([]string{}, args...)
		if _, ok := ctx.Deadline(); !ok {
			t.Fatal("provider command has no deadline")
		}
		return PolicyCommandOutput{Stdout: []byte(`{"ok":true,"value":{"policy_version":1,"policy_digest":"` + testPolicyDigest + `","repo_identity":"/repo/.git","admission_key":"/repo","capacity":{"kind":"unbounded"}}}`)}, nil
	})
	resolver := ExecPolicyResolver{Binary: "/ariadne/bin/sdlc", Timeout: time.Second, Command: runner}
	if _, err := resolver.ResolvePolicy(context.Background(), "/repo/task"); err != nil {
		t.Fatalf("ResolvePolicy: %v", err)
	}
	if gotName != "/ariadne/bin/sdlc" || !reflect.DeepEqual(gotArgs, []string{"fleet", "policy", "--path", "/repo/task", "--json"}) {
		t.Fatalf("command = %q %v", gotName, gotArgs)
	}
}

func TestPolicyResolverExecCancelsHungProvider(t *testing.T) {
	runner := PolicyCommandFunc(func(ctx context.Context, _ string, _ ...string) (PolicyCommandOutput, error) {
		<-ctx.Done()
		return PolicyCommandOutput{}, ctx.Err()
	})
	resolver := ExecPolicyResolver{Binary: "sdlc", Timeout: 10 * time.Millisecond, Command: runner}
	_, err := resolver.ResolvePolicy(context.Background(), "/repo")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("hung provider err = %v, want deadline exceeded", err)
	}
}

func FuzzPolicyResolverDecodeNeverPanics(f *testing.F) {
	f.Add([]byte(`{"ok":true}`), []byte{}, 0)
	f.Add([]byte(`{"ok":false,"diagnostic":{"code":"x","message":"y"}}`), []byte("Error: fleet policy refused: x\n"), 1)
	f.Fuzz(func(t *testing.T, stdout, stderr []byte, exitCode int) {
		_, _ = DecodePolicyResponse(stdout, stderr, exitCode)
	})
}
