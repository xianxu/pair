package artifactpath

import (
	"encoding/json"
	"path/filepath"
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
func TestTerminalPolicyPathsUseSameSocketKey(t *testing.T) {
	cache, err := ResolvePairCache("/home/test")
	if err != nil {
		t.Fatal(err)
	}
	lock, err := cache.TerminalLease("/tmp/sockets/session")
	if err != nil {
		t.Fatal(err)
	}
	policy, err := cache.SessionPolicy("/tmp/sockets/session")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(lock) != "/home/test/.cache/pair/session-locks" || filepath.Dir(policy) != "/home/test/.cache/pair/session-policy" {
		t.Fatalf("wrong namespaces: %s %s", lock, policy)
	}
	key := strings.TrimSuffix(filepath.Base(lock), ".lock")
	if len(key) != 64 || filepath.Base(policy) != key+".json" {
		t.Fatalf("keys disagree: %s %s", lock, policy)
	}
	other, _ := cache.TerminalLease("/tmp/other/session")
	if lock == other {
		t.Fatal("same name in different socket directories collided")
	}
	for _, bad := range []string{"relative", "/tmp/../session", "/", "/tmp/s\x00"} {
		if _, err := cache.TerminalLease(bad); err == nil {
			t.Fatalf("accepted %q", bad)
		}
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
