package launcher

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

// AgentDefault is the repo-scoped launch default for one agent. It deliberately
// carries no tag and no native session id; those remain tag-specific config.
type AgentDefault struct {
	Agent string   `json:"agent"`
	Args  []string `json:"args"`
}

func AgentDefaultPath(dataDir, agent string) string {
	return filepath.Join(dataDir, "agent-default-"+agentDefaultPathComponent(agent)+".json")
}

func ParseAgentDefault(expectedAgent, raw string) (AgentDefault, error) {
	var d AgentDefault
	if err := json.Unmarshal([]byte(raw), &d); err != nil {
		return AgentDefault{}, err
	}
	if expectedAgent == "" {
		return AgentDefault{}, fmt.Errorf("agent default: expected agent is empty")
	}
	if d.Agent == "" {
		return AgentDefault{}, fmt.Errorf("agent default: agent is empty")
	}
	if d.Agent != expectedAgent {
		return AgentDefault{}, fmt.Errorf("agent default: agent %q does not match %q", d.Agent, expectedAgent)
	}
	d.Args = append([]string(nil), d.Args...)
	if d.Args == nil {
		d.Args = []string{}
	}
	return d, nil
}

func BuildAgentDefault(agent string, args []string) (string, error) {
	if agent == "" {
		return "", fmt.Errorf("agent default: agent is empty")
	}
	copied := append([]string(nil), args...)
	if copied == nil {
		copied = []string{}
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(AgentDefault{Agent: agent, Args: copied}); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func agentDefaultPathComponent(agent string) string {
	var b strings.Builder
	for _, r := range agent {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	if b.Len() == 0 {
		return "agent"
	}
	return b.String()
}
