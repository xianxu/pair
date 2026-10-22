package couchcmd

import (
	"errors"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/couchtty"
	"github.com/xianxu/pair/cmd/internal/remotesession"
)

type remoteControl struct {
	supervisor *remotesession.Supervisor
	reason     string
}

// initializeOriginsLocked runs before socket publication, or under
// originAdmission. An opened or failed owner is retained for this service's
// lifetime; exposure configuration never opens another writer.
func (s *messageService) initializeOriginsLocked(namespace string) error {
	if s.originNamespace != "" {
		if s.originNamespace != namespace {
			return errors.New("origin owner namespace changed; restart Couch")
		}
		return s.originErr
	}
	s.originNamespace = namespace
	paths, err := artifactpath.ResolveCouchRemote(namespace)
	if err == nil {
		s.origins, err = remotesession.OpenOriginStore(paths.Origins())
	}
	s.originErr = err
	return err
}

func (s *messageService) configureRemoteSession(console *couchtty.Console, namespace string, getenv func(string) string) {
	s.originAdmission.Lock()
	defer s.originAdmission.Unlock()
	control := &remoteControl{}
	// The leased service owns provenance even when remote exposure is absent
	// or misconfigured. A supervisor always borrows this same owner.
	err := s.initializeOriginsLocked(namespace)
	cast, settingsErr := broadcastSettings(getenv, namespace, guardArgv())
	if err == nil {
		err = settingsErr
	} else {
		err = errors.New("remote origin evidence unavailable; inspect local state before enabling remote")
	}
	var reset func() error
	if err == nil {
		var cfg remotesession.SupervisorConfig
		cfg, err = remoteSettings(getenv, namespace, cast.Tunnel)
		if err == nil {
			cfg.Origins = s.origins
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
