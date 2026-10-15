package couchcmd

import (
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/wrapcmd"
)

// receiverDisplayNames maps inventory agent ids to the names the shipped skill
// uses. A new inventory agent fails the test until it is named here.
var receiverDisplayNames = map[string]string{
	"claude": "Claude Code",
	"codex":  "Codex CLI",
	"qoder":  "Qoder CLI",
	"agy":    "Agy",
	"muse":   "Muse",
	"grok":   "Grok",
}

// TestSkillNamesReceiverProfiles pins the skill's receiver paragraph to the
// AgentInventory receiver policies (#558), so adding, removing or flipping an
// agent's policy fails here instead of leaving the shipped skill stale.
func TestSkillNamesReceiverProfiles(t *testing.T) {
	policy := wrapcmd.PeerReceiverAgents()
	var enabled, disabled []string
	for _, agent := range launcher.AgentInventory() {
		p, ok := policy[agent]
		if !ok {
			t.Fatalf("inventory agent %q has no receiver policy", agent)
		}
		name, ok := receiverDisplayNames[agent]
		if !ok {
			t.Fatalf("inventory agent %q has no display name; name it here and in the skill", agent)
		}
		if p {
			enabled = append(enabled, name)
		} else {
			disabled = append(disabled, name)
		}
	}
	skill := strings.Join(strings.Fields(couchSkill), " ")
	for _, want := range []string{
		"receiver profile (" + humanJoin(enabled) + ")",
		humanJoin(disabled) + " have no receiver profile and are explicitly disabled",
	} {
		if !strings.Contains(skill, want) {
			t.Errorf("shipped skill omits receiver list %q", want)
		}
	}
}

func humanJoin(names []string) string {
	switch len(names) {
	case 0:
		return ""
	case 1:
		return names[0]
	case 2:
		return names[0] + " and " + names[1]
	default:
		return strings.Join(names[:len(names)-1], ", ") + " and " + names[len(names)-1]
	}
}
