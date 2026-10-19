package sessioninventory

// RootTranscriptID reads the native conversation ID from a root transcript's
// path under its native root, using the same parser the scanners use. Only a
// root transcript counts: a subagent file or an unknown agent is not one.
func RootTranscriptID(agent Agent, relativePath string) (string, bool) {
	switch agent {
	case AgentClaude, AgentQoder:
		id, _, role, ok := claudePathFact(relativePath)
		return id, ok && role == RoleRoot
	case AgentCodex:
		return codexPathID(relativePath)
	case AgentGrok:
		return grokPathFact(relativePath)
	default:
		return "", false
	}
}
