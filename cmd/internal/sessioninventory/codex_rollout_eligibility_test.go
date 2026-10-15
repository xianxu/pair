package sessioninventory

import (
	"encoding/json"
	"testing"
)

func codexEligibilityLine(t *testing.T, kind string, payload map[string]any) []byte {
	t.Helper()
	raw, err := json.Marshal(map[string]any{"timestamp": "2026-08-28T10:00:00Z", "type": kind, "payload": payload})
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
func codexEligibilityMeta() map[string]any {
	return map[string]any{"id": "019d1111-1111-7111-8111-000000000001", "timestamp": "", "cwd": "/work", "originator": "", "cli_version": ""}
}
func codexEligibilityTurn() map[string]any {
	return map[string]any{"cwd": "/work", "approval_policy": "never", "sandbox_policy": map[string]any{"type": "read-only"}, "model": ""}
}
func TestCodexLatestNativeOptionalDefaultsAndPresentShapes(t *testing.T) {
	// Native SessionMetaLine backfills session_id; source and history_mode have
	// defaults. Omitted optional fields must not become invented requirements.
	if !codexNativeSessionMetaEligible(codexEligibilityLine(t, "session_meta", codexEligibilityMeta())) {
		t.Fatal("native-default metadata rejected")
	}
	for _, tc := range []struct {
		key   string
		value any
		want  bool
	}{
		{"session_id", nil, false}, {"source", nil, false}, {"source", "future-label", true}, {"source", map[string]any{"custom": "fixture"}, true}, {"source", map[string]any{"custom": 1}, false}, {"history_mode", "legacy", true}, {"history_mode", "paginated", true}, {"history_mode", nil, false}, {"runtime_workspace_roots", []string{"/work"}, true}, {"runtime_workspace_roots", nil, true}, {"selected_capability_roots", []any{}, true}, {"selected_capability_roots", nil, false}, {"base_instructions", map[string]any{"text": ""}, true}, {"base_instructions", map[string]any{"text": 1}, false}, {"base_instructions", map[string]any{"text": "", "provenance": map[string]any{}}, false}, {"git", map[string]any{}, false}, {"dynamic_tools", []any{}, true}, {"dynamic_tools", []any{map[string]any{}}, false},
	} {
		t.Run("meta-"+tc.key, func(t *testing.T) {
			payload := codexEligibilityMeta()
			payload[tc.key] = tc.value
			if got := codexNativeSessionMetaEligible(codexEligibilityLine(t, "session_meta", payload)); got != tc.want {
				t.Fatalf("supported eligibility=%v want %v", got, tc.want)
			}
		})
	}
	for _, tc := range []struct {
		key   string
		value any
		want  bool
	}{
		{"approval_policy", "on-failure", true}, {"approval_policy", map[string]any{"granular": map[string]any{"sandbox_approval": true, "rules": false, "mcp_elicitations": true}}, true}, {"approval_policy", map[string]any{"granular": map[string]any{"sandbox_approval": true, "rules": false}}, false},
		{"sandbox_policy", map[string]any{"type": "workspace-write"}, true}, {"sandbox_policy", map[string]any{"type": "read-only"}, true}, {"sandbox_policy", map[string]any{"type": "external-sandbox"}, true}, {"sandbox_policy", map[string]any{"type": "danger-full-access"}, true}, {"sandbox_policy", map[string]any{"type": "workspace-write", "writable_roots": []string{"/work"}}, true}, {"sandbox_policy", map[string]any{"type": "workspace-write", "writable_roots": []string{"relative"}}, false}, {"sandbox_policy", map[string]any{"type": "read-only", "network_access": nil}, false},
		{"disabled_plugin_ids", []string{}, true}, {"disabled_plugin_ids", nil, true}, {"realtime_active", false, true}, {"realtime_active", "false", false}, {"summary", nil, true}, {"permission_profile", map[string]any{}, false},
	} {
		t.Run("turn-"+tc.key, func(t *testing.T) {
			payload := codexEligibilityTurn()
			payload[tc.key] = tc.value
			_, got := codexNativeTurnContextCWD(codexEligibilityLine(t, "turn_context", payload))
			if got != tc.want {
				t.Fatalf("supported eligibility=%v want %v", got, tc.want)
			}
		})
	}
}
func TestCodexLatestNativeJSONUnicodeEligibility(t *testing.T) {
	for _, tc := range []struct {
		name, value string
		want        bool
	}{
		{"utf8", "a\xffb", false},
		{"lone-high", `\uD800`, false},
		{"lone-low", `\udfff`, false},
		{"paired-unmodeled", `\uD83D\uDE00`, false},
		{"unicode", "工作🙂", true},
		{"escaped-ascii", `\u0061`, true},
		{"literal-surrogate-text", `\\uD800`, true},
		{"odd-backslash-surrogate", `\\\uD800`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := []byte(`{"timestamp":"2026-08-28T10:00:00Z","type":"turn_context","payload":{"cwd":"/work","model":"` + tc.value + `","approval_policy":"never","sandbox_policy":{"type":"read-only"}}}`)
			_, got := codexNativeTurnContextCWD(raw)
			if got != tc.want {
				t.Fatalf("native Unicode eligibility=%v want %v", got, tc.want)
			}
			meta := []byte(`{"timestamp":"2026-08-28T10:00:00Z","type":"session_meta","payload":{"id":"019d1111-1111-7111-8111-000000000001","timestamp":"","cwd":"/work","originator":"` + tc.value + `","cli_version":""}}`)
			if got := codexNativeSessionMetaEligible(meta); got != tc.want {
				t.Fatalf("native metadata Unicode eligibility=%v want %v", got, tc.want)
			}
		})
	}
}
