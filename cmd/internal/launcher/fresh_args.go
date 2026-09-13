package launcher

import (
	"fmt"
	"strings"
)

// ValidateFreshAgentArgs rejects native context selectors rather than changing
// accepted argv. Option values and text after -- remain literal data.
func ValidateFreshAgentArgs(agent string, argv []string) error {
	if !IsSupportedAgent(agent) {
		return fmt.Errorf("unsupported fresh agent %q", agent)
	}
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
		forbidden := false
		switch agent {
		case "claude":
			switch flag {
			case "--resume", "--continue", "--session-id", "--fork-session", "--from-pr", "--teleport", "--cloud":
				forbidden = true
			}
			if strings.HasPrefix(arg, "-") && !strings.HasPrefix(arg, "--") {
				for _, r := range arg[1:] {
					if r == 'c' || r == 'r' {
						forbidden = true
						break
					}
					if r == 'n' || r == 'w' || r == 'd' {
						break
					}
				}
			}
		case "agy":
			switch flag {
			case "-c", "--continue", "-continue", "--conversation", "-conversation":
				forbidden = true
			}
		}
		if agent == "muse" && flag == "--session-id" {
			forbidden = true
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
				i++
				if agent == "claude" && freshVariadicOption(flag) {
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
	}
	for _, value := range strings.Fields(flags) {
		if value == flag {
			return true
		}
	}
	return false
}

func freshVariadicOption(flag string) bool {
	switch flag {
	case "--add-dir", "--allowedTools", "--allowed-tools", "--disallowedTools", "--disallowed-tools", "--betas", "--file", "--mcp-config", "--tools":
		return true
	}
	return false
}
