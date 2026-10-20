package couchcore

import (
	"context"
	"errors"
	"io"
	"os"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/readiness"
)

// AgentTerminal reads the agent the thread's wrapper last published as ready:
// its exact process and controlling terminal (pair#695 P1). The record is the
// wrapper's alone; ok is false when none is published.
func (c ScopedThreadArtifactCollisionChecker) AgentTerminal(ctx context.Context, address ThreadAddress, agent string) (AgentTerminal, bool, error) {
	if err := ctx.Err(); err != nil {
		return AgentTerminal{}, false, err
	}
	paths, err := artifactpath.Resolve(artifactpath.Address{DataDir: c.GlobalDataDir, RepoScope: address.RepoScope, Tag: string(address.Tag)})
	if err != nil {
		return AgentTerminal{}, false, err
	}
	path, err := paths.AgentReadyChecked(agent)
	if err != nil {
		return AgentTerminal{}, false, err
	}
	file, err := openSwitchRegularFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return AgentTerminal{}, false, nil
	}
	if err != nil {
		return AgentTerminal{}, false, err
	}
	defer file.Close()
	raw, err := io.ReadAll(io.LimitReader(file, 16*1024+1))
	if err != nil {
		return AgentTerminal{}, false, err
	}
	if len(raw) > 16*1024 {
		return AgentTerminal{}, false, errors.New("ready record exceeds size limit")
	}
	ready, err := readiness.Decode(string(raw))
	if err != nil {
		return AgentTerminal{}, false, err
	}
	if ready.Tag != string(address.Tag) || ready.Agent != agent || ready.AgentIdentity == "" {
		return AgentTerminal{}, false, nil
	}
	return AgentTerminal{Process: ProcessIdentity{PID: ready.PID, Identity: ready.AgentIdentity}, TTY: ready.TTY}, true, nil
}

var _ AgentTerminalReader = ScopedThreadArtifactCollisionChecker{}
