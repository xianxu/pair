package launcher

import (
	"fmt"
	"strings"
)

// freshAgentSpec is the per-agent context-selector surface
// ValidateFreshAgentArgs consults: forbidden standalone flags plus the
// short-flag cluster letters, split into selectors (rejected anywhere in the
// cluster) and value-taking letters (a letter that takes a value ends the
// cluster scan — the rest of the cluster is that option's glued value).
type freshAgentSpec struct {
	forbiddenFlags string
	forbiddenShort string
	valueShort     string
}

var freshAgentSpecs = map[string]freshAgentSpec{
	"claude": {
		forbiddenFlags: "--resume --continue --session-id --fork-session --from-pr --teleport --cloud",
		forbiddenShort: "cr",
		valueShort:     "nwd",
	},
	"agy": {
		forbiddenFlags: "-c --continue -continue --conversation -conversation",
	},
	"qoder": {
		forbiddenFlags: "--resume --continue --session-id --fork-session --remote --remote-session --teleport --remote-control --list-sessions --delete-session",
		forbiddenShort: "cr",
		valueShort:     "mniwo",
	},
}

func (spec freshAgentSpec) forbidsFlag(flag string) bool {
	for _, value := range strings.Fields(spec.forbiddenFlags) {
		if value == flag {
			return true
		}
	}
	return false
}

func (spec freshAgentSpec) forbidsCluster(cluster string) bool {
	for _, r := range cluster[1:] {
		if strings.ContainsRune(spec.valueShort, r) {
			return false
		}
		if strings.ContainsRune(spec.forbiddenShort, r) {
			return true
		}
	}
	return false
}

// ValidateFreshAgentArgs rejects native context selectors rather than changing
// accepted argv. Option values and text after -- remain literal data.
func ValidateFreshAgentArgs(agent string, argv []string) error {
	if !IsSupportedAgent(agent) {
		return fmt.Errorf("unsupported fresh agent %q", agent)
	}
	spec := freshAgentSpecs[agent]
	commandSeen := false
	execSeen := false
	for i := 0; i < len(argv); i++ {
		arg := argv[i]
		if strings.ContainsRune(arg, 0) {
			return fmt.Errorf("fresh arguments contain NUL")
		}
		if arg == "--" {
			for _, tail := range argv[i+1:] {
				if strings.ContainsRune(tail, 0) {
					return fmt.Errorf("fresh arguments contain NUL")
				}
			}
			return nil
		}
		flag, _, inline := strings.Cut(arg, "=")
		forbidden := spec.forbidsFlag(flag)
		if agent == "muse" && flag == "--session-id" {
			forbidden = true
		}
		if !forbidden && spec.forbiddenShort != "" && strings.HasPrefix(arg, "-") && !strings.HasPrefix(arg, "--") {
			forbidden = spec.forbidsCluster(arg)
		}
		if forbidden {
			return fmt.Errorf("%s argument %q selects an existing conversation", agent, arg)
		}
		if strings.HasPrefix(arg, "-") {
			if agent == "muse" && !inline && (flag == "-w" || flag == "--worktree") && i+1 < len(argv) {
				switch argv[i+1] {
				case "off", "create", "existing":
					i++
				}
				continue
			}
			if !inline && freshValueOption(agent, flag) && i+1 < len(argv) {
				if agent == "claude" && (flag == "-d" || flag == "--debug" || flag == "-w" || flag == "--worktree" || flag == "--prompt-suggestions" || flag == "--remote-control") && strings.HasPrefix(argv[i+1], "-") {
					continue
				}
				if agent == "qoder" && flag == "--worktree" && strings.HasPrefix(argv[i+1], "-") {
					continue
				}
				i++
				if freshVariadicOption(agent, flag) {
					for i+1 < len(argv) && !strings.HasPrefix(argv[i+1], "-") {
						if strings.ContainsRune(argv[i], 0) {
							return fmt.Errorf("fresh arguments contain NUL")
						}
						i++
					}
				}
				if strings.ContainsRune(argv[i], 0) {
					return fmt.Errorf("fresh arguments contain NUL")
				}
			}
			continue
		}
		if !commandSeen && agent == "codex" && !execSeen && (arg == "exec" || arg == "e") {
			execSeen = true
			continue
		}
		if !commandSeen {
			if (agent == "codex" && (arg == "resume" || arg == "fork")) || (agent == "muse" && arg == "resume") || (agent == "claude" && (arg == "attach" || arg == "respawn")) {
				return fmt.Errorf("%s command %q selects an existing conversation", agent, arg)
			}
			commandSeen = true
		}
	}
	return nil
}

func freshValueOption(agent, flag string) bool {
	if agent == "codex" {
		return codexValueGlobalOption(flag)
	}
	var flags string
	switch agent {
	case "claude":
		flags = "--add-dir --agent --agents --allowedTools --allowed-tools --append-system-prompt --autocompact --betas -d --debug --debug-file --disallowedTools --disallowed-tools --effort --environment --fallback-model --file --input-format --json-schema --max-budget-usd --mcp-config --model -n --name --output-format --permission-mode --permission-prompts --plugin-dir --plugin-url --prompt-suggestions --remote-control --remote-control-session-name-prefix --setting-sources --settings --system-prompt --system-prompt-snapshot --tools -w --worktree"
	case "agy":
		flags = "--add-dir --agent --effort -i --prompt-interactive --input-format --json-schema --log-file --mode --model --new-project --output-format -p --print --prompt --print-timeout --project"
	case "muse":
		flags = "--agents --provider --preset --model --reasoning-effort --base-url --image --workspace --worktree-base --worktree-existing --approval-mode --approval-judge --echo-delay-ms --sandbox-network"
	case "qoder":
		flags = "--model --reasoning-effort --thinking --thinking-budget --context-window --prompt-interactive --cwd --config-dir --permission-mode --allowed-mcp-server-names --allowed-tools --disallowed-tools --attachment --plugin-dir --name --add-dir --output-format --input-format --max-output-tokens --agent --agents --append-system-prompt --system-prompt --output-style --max-model-request-retries --mcp-config --setting-sources --settings --worktree --tools -m -i -w -n -o"
	}
	for _, value := range strings.Fields(flags) {
		if value == flag {
			return true
		}
	}
	return false
}

// freshVariadicOption reports flags that take multiple values (consumed until
// the next flag token). Strictly per-agent: a flag from another agent's table
// must not swallow positionals a later command check would reject.
func freshVariadicOption(agent, flag string) bool {
	var flags string
	switch agent {
	case "claude":
		flags = "--add-dir --allowedTools --allowed-tools --disallowedTools --disallowed-tools --betas --file --mcp-config --tools"
	case "qoder":
		flags = "--tools"
	}
	for _, value := range strings.Fields(flags) {
		if value == flag {
			return true
		}
	}
	return false
}
