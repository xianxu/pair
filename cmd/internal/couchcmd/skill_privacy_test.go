package couchcmd

import (
	"strings"
	"testing"
)

// This guards the shipped guidance, not an agent's interpretation of it. Exercise
// private and shareable submissions separately in fresh-context skill tests.
func TestSkillBroadcastPrivacyContract(t *testing.T) {
	skill := strings.Join(strings.Fields(couchSkill), " ")
	for _, tc := range []struct {
		name string
		want string
	}{
		{"per-item scope", "Read each version-3 note or sketch's `privacy` and `privacy_rule` before selecting its evidence"},
		{"sketch image privacy", "and attach no crops or sketch images"},
		{"historical sketch provenance", "Its image has `replay-rendered` provenance"},
		{"literal IDs", "literal `id`/`data-*` attributes"},
		{"sketch export", "sketch-image --sketch ID --out DIR [--at T]"},
		{"legacy fallback", "(the submission-level fields on older versions)"},
		{"explicit grant", "Only an explicit `privacy: shareable` decision makes that item's content eligible"},
		{"aggregate is not a grant", "The version-3 top-level privacy is a conservative summary, never a grant for a private item"},
		{"private default", "Absent, private, or unrecognized privacy values and invalid policy files mean private"},
		{"private evidence", "summarize without quoting `box_text` and attach no crops"},
		{"invalid policy result", "`privacy_rule: 'invalid policy file: <reason>'`"},
		{"invalid policy notice", "a Couch notice; it does not fail the submission"},
		{"text first", "Write the issue text first"},
		{"safe URL", "page URL without secrets"},
		{"element", "element/classes"},
		{"request", "operator's request"},
		{"box text", "appropriate shareable box text or a private summary"},
		{"crop explanation", "one line explaining what a crop would demonstrate"},
		{"asset permission", "explicitly shareable and the repository's asset policy allows them"},
		{"asset placement", "ariadne#337 issue asset folder convention"},
		{"unknown policy", "If that policy is unknown, keep the issue text-only"},
		{"operator authority", "Only the operator edits the origin sharing list"},
		{"no self grant", "never edit it to obtain a grant"},
		{"local retention", "Recordings stay local and may expire"},
		{"no recording upload", "Never upload a recording or make the issue depend on a recording or replay link"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if !strings.Contains(skill, tc.want) {
				t.Errorf("shipped skill missing privacy contract: %s", tc.want)
			}
		})
	}
}
