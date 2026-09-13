package launcher

import (
	"encoding/json"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/strictjson"
	"strings"
)

// AgentCommandEnv carries the final invocation across the layout shell without
// shell splitting. Only explicit pair wrap --from-launch-env consumes it.
const AgentCommandEnv = "PAIR_AGENT_COMMAND"

type AgentCommand struct {
	Executable string   `json:"executable"`
	Argv       []string `json:"argv"`
}

func (c AgentCommand) validate() error {
	if c.Executable == "" || strings.ContainsRune(c.Executable, 0) || c.Argv == nil {
		return fmt.Errorf("invalid agent command")
	}
	for _, arg := range c.Argv {
		if strings.ContainsRune(arg, 0) {
			return fmt.Errorf("agent command argument contains NUL")
		}
	}
	return nil
}
func EncodeAgentCommand(command AgentCommand) (string, error) {
	if err := command.validate(); err != nil {
		return "", err
	}
	raw, err := json.Marshal(command)
	return string(raw), err
}
func DecodeAgentCommand(raw string) (AgentCommand, error) {
	var command AgentCommand
	if err := strictjson.Decode([]byte(raw), &command); err != nil {
		return AgentCommand{}, fmt.Errorf("decode agent command: %w", err)
	}
	if err := command.validate(); err != nil {
		return AgentCommand{}, err
	}
	return command, nil
}
