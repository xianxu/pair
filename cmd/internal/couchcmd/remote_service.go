package couchcmd

import (
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/couchtty"
	"github.com/xianxu/pair/cmd/internal/remotesession"
)

type remoteControl struct {
	supervisor *remotesession.Supervisor
	reason     string
}

func (s *messageService) configureRemoteSession(console *couchtty.Console, namespace string, getenv func(string) string) {
	control := &remoteControl{}
	cast, err := broadcastSettings(getenv, namespace, guardArgv())
	var reset func() error
	if err == nil {
		var cfg remotesession.SupervisorConfig
		cfg, err = remoteSettings(getenv, namespace, cast.Tunnel)
		if err == nil {
			control.supervisor, reset, err = remotesession.NewSupervisor(cfg)
		}
	}
	if err != nil {
		control.reason = err.Error()
	}
	// The only production capture of reset authority. No service/CLI request
	// retains that closure; the console's owned host-key branch alone invokes it.
	console.SetRemoteSession(control.supervisor, reset, control.reason)
	s.remote.Store(control)
}

func (s *messageService) handleRemote(r couchmessage.Request) couchmessage.Response {
	if err := couchmessage.ValidateRequest(r); err != nil {
		return couchmessage.Response{Code: "invalid-request", Error: err.Error()}
	}
	control := s.remote.Load()
	if control == nil {
		return couchmessage.Response{Code: "unavailable", Error: "remote console unavailable"}
	}
	if control.supervisor == nil {
		if r.Op == "remote-status" {
			return couchmessage.Response{Code: "ok", Remote: &couchmessage.RemoteStatus{Status: "blocked", Detail: control.reason}}
		}
		return couchmessage.Response{Code: "refused", Error: control.reason}
	}
	remote := control.supervisor
	switch r.Op {
	case "remote-on":
		if err := remote.On(); err != nil {
			return couchmessage.Response{Code: "refused", Error: err.Error()}
		}
	case "remote-off":
		remote.Off()
	case "remote-status":
	default:
		return couchmessage.Response{Code: "invalid-request", Error: "unsupported remote action"}
	}
	status := remote.Snapshot()
	return couchmessage.Response{Code: "ok", Remote: &couchmessage.RemoteStatus{Status: status.Status, Detail: status.Detail, CanReset: status.CanReset}}
}
