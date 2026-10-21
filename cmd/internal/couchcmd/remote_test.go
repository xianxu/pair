package couchcmd

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

func TestRemoteCLIClosedActionsAndNoLink(t *testing.T) {
	for _, action := range []string{"on", "off", "status"} {
		inv, err := ParseCLI([]string{"--remote", action}, nil)
		if err != nil {
			t.Fatal(err)
		}
		var out, stderr bytes.Buffer
		calls := 0
		code := runMessageCLIWithCall(inv, messageRuntime(), &out, &stderr, func(_ context.Context, _ string, request, response any) error {
			calls++
			r := request.(couchmessage.Request)
			if r != (couchmessage.Request{Op: "remote-" + action}) {
				t.Fatalf("unexpected fields: %+v", r)
			}
			*response.(*couchmessage.Response) = couchmessage.Response{Code: "ok", Remote: &couchmessage.RemoteStatus{Status: "on", Detail: "waiting for authenticated phone"}}
			return nil
		})
		if code != 0 || calls != 1 {
			t.Fatalf("%d %d %s", code, calls, &stderr)
		}
		if strings.Contains(out.String(), "https:") || !strings.Contains(out.String(), "Remote "+action+": on") {
			t.Fatal("not action/status only", out.String())
		}
	}
	for _, args := range [][]string{{"--remote"}, {"--remote", "reset"}, {"--remote", "kill"}, {"--remote", "on", "extra"}, {"--remote", "on", "--json"}} {
		if _, err := ParseCLI(args, nil); err == nil {
			t.Fatal("open remote action", args)
		}
	}
}
func TestRemoteCLINeverRetriesLostReply(t *testing.T) {
	inv, err := ParseCLI([]string{"--remote", "on"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	var out, stderr bytes.Buffer
	calls := 0
	code := runMessageCLIWithCall(inv, messageRuntime(), &out, &stderr, func(context.Context, string, any, any) error { calls++; return errors.New("lost reply") })
	if code != 1 || calls != 1 || !strings.Contains(stderr.String(), "unconfirmed") {
		t.Fatalf("%d %d %s", code, calls, &stderr)
	}
}
func TestRemoteSettingsRequireExplicitProtection(t *testing.T) {
	valid := map[string]string{"COUCH_REMOTE_ACCESS_TEAM": "operator", "COUCH_REMOTE_ACCESS_AUDIENCE": strings.Repeat("a", 64), "COUCH_REMOTE_ALLOWED_EMAILS": "human@example.com"}
	named := broadcast.Cloudflared{Named: &broadcast.NamedTunnel{Name: "pair", Hostname: "remote.example.com"}, Guard: []string{"pair", "guard"}, Records: filepath.Join(t.TempDir(), "records")}
	for _, fault := range []string{"none", "anonymous", "local", "missing-team", "missing-audience", "missing-identity"} {
		t.Run(fault, func(t *testing.T) {
			env := map[string]string{}
			for k, v := range valid {
				env[k] = v
			}
			var tunnel broadcast.Tunnel = named
			switch fault {
			case "anonymous":
				tunnel = broadcast.Cloudflared{}
			case "local":
				tunnel = broadcast.LocalOnly{}
			case "missing-team":
				delete(env, "COUCH_REMOTE_ACCESS_TEAM")
			case "missing-audience":
				delete(env, "COUCH_REMOTE_ACCESS_AUDIENCE")
			case "missing-identity":
				delete(env, "COUCH_REMOTE_ALLOWED_EMAILS")
			}
			cfg, err := remoteSettings(func(k string) string { return env[k] }, t.TempDir(), tunnel)
			if (err == nil) != (fault == "none") {
				t.Fatalf("config=%+v err=%v", cfg, err)
			}
		})
	}
}
