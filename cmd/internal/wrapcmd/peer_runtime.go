package wrapcmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/strictjson"
)

// peerReceiverAgents are the agents with a peer-delivery receiver profile
// (composer recognition and submit). Any installed version may receive: an
// exact-version allowlist silently dropped every slot after each agent
// auto-update. Upgrade safety is to come from evidence gathered in daily use
// (#368); until then the composer checks still refuse unknown input states.
var peerReceiverAgents = map[string]bool{"codex": true, "claude": true}

type peerRuntime struct {
	cancel   context.CancelFunc
	server   *couchmessage.Server
	workers  sync.WaitGroup
	delivery *peerDelivery
}

func (r *peerRuntime) Close() {
	r.cancel()
	r.delivery.mu.Lock()
	r.delivery.exited = true
	r.delivery.mu.Unlock()
	r.delivery.signal()
	_ = r.server.Close()
	r.workers.Wait()
}

func (d *peerDelivery) handleEndpoint(_ context.Context, raw []byte) ([]byte, error) {
	var request couchmessage.EndpointRequest
	if err := strictjson.Decode(raw, &request); err != nil {
		return nil, err
	}
	response := couchmessage.EndpointResponse{}
	if err := couchmessage.ValidateEndpointRequest(request); err != nil {
		return json.Marshal(couchmessage.EndpointResponse{Error: err.Error()})
	}
	if request.Binding != d.binding {
		return json.Marshal(couchmessage.EndpointResponse{Error: "wrapper identity mismatch"})
	}
	var err error
	switch request.Op {
	case "observe":
		d.mu.Lock()
		response.Observation = couchmessage.Observation{LastActivity: d.lastActivity, Sequence: d.sequence, Submission: d.submissions}
		d.mu.Unlock()
	case "reserve":
		err = d.reserve(request.ID, request.Sequence)
	case "commit":
		err = d.enqueue(*request.Message)
		if err == nil {
			r := d.receipt()
			response.Receipt = &r
		}
	case "status":
		r := d.receipt()
		if r.Message.ID != request.ID {
			err = errors.New("unknown delivery")
		} else {
			response.Receipt = &r
		}
	case "release":
		d.mu.Lock()
		if d.reservation == request.ID {
			d.reservation = ""
			if d.current.Message.ID == request.ID && !d.current.Status.Terminal() {
				d.interrupted = true
			}
		}
		d.mu.Unlock()
		d.signal()
	}
	if err != nil {
		response.Error = err.Error()
	}
	return json.Marshal(response)
}

func (p *proxy) startPeerRuntime(executable string) (*peerRuntime, error) {
	namespace := os.Getenv("COUCH_STORE_DIR")
	if namespace == "" || os.Getenv("COUCH_THREAD_SCOPE") == "" || p.ttyProfile == nil {
		return nil, nil
	}
	if !peerReceiverAgents[p.agentBasename] {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), couchmessage.AdmissionTimeout)
	defer cancel()
	raw, err := exec.CommandContext(ctx, executable, "--version").Output()
	if err != nil {
		return nil, err
	}
	// Recorded in the binding so receipts and listings name the version that
	// received; it no longer gates delivery.
	version := strings.TrimSpace(string(raw))
	if version == "" {
		return nil, fmt.Errorf("%s --version printed nothing", p.agentBasename)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	identity, err := couchcore.NewOSSlotCatalog(couchcore.OSProvisionIO{}).ResolveWorkspace(ctx, cwd)
	if err != nil {
		return nil, err
	}
	if identity.Address == nil {
		return nil, errors.New("peer delivery needs an addressable Couch slot")
	}
	process, err := (couchcore.OSProcOps{}).Current()
	if err != nil {
		return nil, err
	}
	binding := couchmessage.Binding{Slot: *identity.Address, Repository: identity.RepoIdentity, Scope: os.Getenv("COUCH_THREAD_SCOPE"), Tag: os.Getenv("COUCH_THREAD_TAG"), Session: os.Getenv("PAIR_SESSION_NAME"), Nonce: os.Getenv("PAIR_LAUNCH_NONCE"), Agent: p.agentBasename, Version: version, PID: process.PID, Start: process.Identity}
	socket, err := couchmessage.EndpointSocket(namespace, binding)
	if err != nil {
		return nil, err
	}
	brokerSocket, err := couchmessage.SocketPath(namespace, "broker")
	if err != nil {
		return nil, err
	}
	lifetime, stop := context.WithCancel(context.Background())
	d := newPeerDelivery(binding, time.Now)
	server, err := couchmessage.StartServer(lifetime, socket, d.handleEndpoint)
	if err != nil {
		stop()
		return nil, err
	}
	r := &peerRuntime{cancel: stop, server: server, delivery: d}
	p.peer = d
	r.workers.Add(1)
	go func() {
		defer r.workers.Done()
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			request := couchmessage.Request{Op: "register", Binding: &binding}
			callCtx, c := context.WithTimeout(lifetime, couchmessage.AdmissionTimeout)
			var response couchmessage.Response
			_ = couchmessage.Call(callCtx, brokerSocket, request, &response)
			c()
			select {
			case <-lifetime.Done():
				return
			case <-ticker.C:
			case <-d.submit:
				// A wakeup only: the broker observes the wrapper's monotonic
				// submission count and never replenishes the same generation twice.
				submitCtx, c := context.WithTimeout(lifetime, couchmessage.AdmissionTimeout)
				_ = couchmessage.Call(submitCtx, brokerSocket, couchmessage.Request{Op: "operator-submit", Binding: &binding}, &response)
				c()
			}
		}
	}()
	return r, nil
}
