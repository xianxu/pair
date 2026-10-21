package couchcmd

import (
	"errors"
	"strings"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/remotesession"
)

// remoteSettings reuses the ordinary cast's named connector/lease namespace.
// Missing protection refuses remote mode; it never changes ordinary casts.
func remoteSettings(getenv func(string) string, namespace string, tunnel broadcast.Tunnel) (remotesession.SupervisorConfig, error) {
	cloud, ok := tunnel.(broadcast.Cloudflared)
	if !ok || cloud.Named == nil {
		return remotesession.SupervisorConfig{}, errors.New("remote requires COUCH_BROADCAST_TUNNEL and COUCH_BROADCAST_HOSTNAME for a protected named tunnel")
	}
	team := getenv("COUCH_REMOTE_ACCESS_TEAM")
	audience := getenv("COUCH_REMOTE_ACCESS_AUDIENCE")
	protected, err := broadcast.NewProtectedTunnel(cloud, broadcast.AccessEnforcement{TeamName: team, Audience: audience})
	if err != nil {
		return remotesession.SupervisorConfig{}, errors.New("remote requires a named tunnel and explicit COUCH_REMOTE_ACCESS_TEAM and COUCH_REMOTE_ACCESS_AUDIENCE")
	}
	emails := strings.Split(getenv("COUCH_REMOTE_ALLOWED_EMAILS"), ",")
	for i := range emails {
		emails[i] = strings.TrimSpace(emails[i])
	}
	verifier, err := remotesession.NewAccessVerifier(remotesession.AccessConfig{Issuer: "https://" + team + ".cloudflareaccess.com", Audience: audience, AllowedEmails: emails})
	if err != nil {
		return remotesession.SupervisorConfig{}, errors.New("remote requires valid COUCH_REMOTE_ACCESS_TEAM, COUCH_REMOTE_ACCESS_AUDIENCE and exact COUCH_REMOTE_ALLOWED_EMAILS")
	}
	host, err := broadcast.NewProtectedHost(protected)
	if err != nil {
		return remotesession.SupervisorConfig{}, err
	}
	return remotesession.SupervisorConfig{Namespace: namespace, Origin: "https://" + cloud.Named.Hostname, Host: host, Verifier: verifier, CaptureConfigured: getenv("COUCH_CAPTURE_DIR") != ""}, nil
}
