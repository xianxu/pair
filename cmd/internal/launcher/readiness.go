package launcher

import (
	"fmt"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/readiness"
)

type ReadyExpectation struct {
	Tag     string
	Agent   string
	Session string
	Nonce   string
}

func AgentReadyPath(dataDir, tag, agent string) string {
	paths, err := artifactpath.ResolveScoped(dataDir, tag)
	if err != nil {
		return ""
	}
	return paths.AgentReady(agentDefaultPathComponent(agent))
}

func MatchReadyRecord(expect ReadyExpectation, record readiness.ReadyRecord, pidAlive func(int) bool) error {
	if record.Tag != expect.Tag {
		return fmt.Errorf("ready record tag %q does not match %q", record.Tag, expect.Tag)
	}
	if record.Agent != expect.Agent {
		return fmt.Errorf("ready record agent %q does not match %q", record.Agent, expect.Agent)
	}
	if record.Session != expect.Session {
		return fmt.Errorf("ready record session %q does not match %q", record.Session, expect.Session)
	}
	if record.Nonce != expect.Nonce {
		return fmt.Errorf("ready record nonce %q does not match %q", record.Nonce, expect.Nonce)
	}
	if pidAlive == nil || !pidAlive(record.PID) {
		return fmt.Errorf("ready record pid %d is not live", record.PID)
	}
	return nil
}
