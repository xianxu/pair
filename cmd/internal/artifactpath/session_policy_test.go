package artifactpath

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func validSessionPolicy() SessionLaunchPolicy {
	return SessionLaunchPolicy{Version: 1, Socket: "/tmp/sockets/session", SocketDevice: 1, SocketInode: 2, Name: "session", ServerPID: 123, ServerIdentity: "1234.5678", Scope: "0123456789abcdef", Tag: "work", Agent: "codex", Nonce: "launch-1", ConfigDigest: strings.Repeat("a", 64), PolicyVersion: 1}
}
func TestSessionLaunchPolicyStrictDecode(t *testing.T) {
	p := validSessionPolicy()
	raw, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	invalidUTF8 := []byte(strings.Replace(string(raw), "launch-1", "launch-\xff", 1))
	if _, err := DecodeSessionLaunchPolicy(invalidUTF8); err == nil {
		t.Fatal("accepted invalid UTF-8")
	}
	got, err := DecodeSessionLaunchPolicy(raw)
	if err != nil || got != p {
		t.Fatalf("roundtrip: %+v %v", got, err)
	}
	for name, input := range map[string][]byte{
		"unknown":   append(append([]byte(nil), raw[:len(raw)-1]...), []byte(`,"unknown":1}`)...),
		"duplicate": append(append([]byte(nil), raw[:len(raw)-1]...), []byte(`,"version":1}`)...),
		"trailing":  append(append([]byte(nil), raw...), []byte(` {}`)...),
		"null":      []byte(`null`), "oversize": []byte(strings.Repeat(" ", MaxSessionLaunchPolicyBytes+1)),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := DecodeSessionLaunchPolicy(input); err == nil {
				t.Fatal("accepted invalid policy")
			}
		})
	}
}
func TestSessionLaunchPolicyRejectsInvalidIdentity(t *testing.T) {
	for name, change := range map[string]func(*SessionLaunchPolicy){
		"version": func(p *SessionLaunchPolicy) { p.Version = 2 }, "policy": func(p *SessionLaunchPolicy) { p.PolicyVersion = 2 },
		"socket": func(p *SessionLaunchPolicy) { p.Socket = "relative" }, "unclean": func(p *SessionLaunchPolicy) { p.Socket = "/tmp/../session" },
		"name": func(p *SessionLaunchPolicy) { p.Name = "other" }, "pid": func(p *SessionLaunchPolicy) { p.ServerPID = 0 },
		"start": func(p *SessionLaunchPolicy) { p.ServerIdentity = "" }, "inode": func(p *SessionLaunchPolicy) { p.SocketInode = 0 },
		"tag": func(p *SessionLaunchPolicy) { p.Tag = "../work" }, "agent": func(p *SessionLaunchPolicy) { p.Agent = "" },
		"nonce":      func(p *SessionLaunchPolicy) { p.Nonce = "" },
		"agent path": func(p *SessionLaunchPolicy) { p.Agent = "../codex" }, "digest": func(p *SessionLaunchPolicy) { p.ConfigDigest = "wrong" },
		"scope": func(p *SessionLaunchPolicy) { p.Scope = "../scope" }, "control": func(p *SessionLaunchPolicy) { p.ServerIdentity = "x\ny" },
	} {
		t.Run(name, func(t *testing.T) {
			p := validSessionPolicy()
			change(&p)
			if ValidateSessionLaunchPolicy(p) == nil {
				t.Fatal("accepted invalid identity")
			}
		})
	}
}
func TestTerminalPolicyPathsUseCanonicalSocket(t *testing.T) {
	cache, _ := ResolvePairCache("/home/test")
	socket := "/tmp/sockets/session"
	lock, err := cache.TerminalLease(socket)
	if err != nil {
		t.Fatal(err)
	}
	policy, err := cache.SessionPolicy(socket)
	if err != nil {
		t.Fatal(err)
	}
	key := sha256.Sum256([]byte(socket))
	if want := fmt.Sprintf("/home/test/.cache/pair/session-locks-v2/%02x.lock", key[0]); lock != want {
		t.Fatalf("lock = %s, want %s", lock, want)
	}
	if want := fmt.Sprintf("/home/test/.cache/pair/session-policy/%x.json", key); policy != want {
		t.Fatalf("policy = %s, want %s", policy, want)
	}
	for _, bad := range []string{"relative", "/tmp/../session", "/", "/tmp/s\x00"} {
		if _, err := cache.TerminalLease(bad); err == nil {
			t.Fatalf("accepted %q", bad)
		}
	}
	if _, err := (PairCachePaths{}).TerminalLease(socket); err == nil {
		t.Fatal("accepted unresolved cache")
	}
}

func TestTerminalLeasePathsHaveFixedBound(t *testing.T) {
	cache, _ := ResolvePairCache("/home/test")
	locks := map[string]bool{}
	policies := map[string]bool{}
	for i := 0; i < 10000; i++ {
		socket := fmt.Sprintf("/tmp/sockets/session-%d", i)
		lock, err := cache.TerminalLease(socket)
		if err != nil {
			t.Fatal(err)
		}
		again, err := cache.TerminalLease(socket)
		if err != nil || lock != again {
			t.Fatalf("unstable key: %s %s %v", lock, again, err)
		}
		locks[lock] = true
		policy, err := cache.SessionPolicy(socket)
		if err != nil {
			t.Fatal(err)
		}
		policies[policy] = true
	}
	if len(locks) > 256 {
		t.Fatalf("%d unique lock paths exceeds fixed bound 256", len(locks))
	}
	if len(policies) != 10000 {
		t.Fatalf("policy identities merged: %d", len(policies))
	}
}
func FuzzDecodeSessionLaunchPolicy(f *testing.F) {
	raw, _ := json.Marshal(validSessionPolicy())
	f.Add(raw)
	f.Add([]byte(`null`))
	f.Fuzz(func(t *testing.T, raw []byte) {
		p, err := DecodeSessionLaunchPolicy(raw)
		if err == nil {
			if err := ValidateSessionLaunchPolicy(p); err != nil {
				t.Fatalf("decoded invalid policy: %v", err)
			}
		}
	})
}
