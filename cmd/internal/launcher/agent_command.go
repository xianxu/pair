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

// AgentVersionEnv carries the probed agent CLI version from the launcher to the
// wrapper's ready record (pair#891).
const AgentVersionEnv = "PAIR_AGENT_VERSION"

type AgentCommand struct {
	Executable string   `json:"executable"`
	Argv       []string `json:"argv"`
	// Prompt is a fresh conversation's one-time first message (#522). Only
	// the wrapper's exec adds it (PromptArgs); Argv stays what a later fresh
	// start re-derives from.
	Prompt string `json:"prompt,omitempty"`
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
	if c.Prompt != "" {
		return ValidatePrompt(c.Prompt)
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
