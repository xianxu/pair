package sessioninventory

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strconv"
	"unicode/utf8"
)

// These predicates recognize a conservative, source-verified subset of native
// rollout decoding, not the general inventory scanner's weaker identity shape.
// Optional/default fields may be omitted. Unsupported non-null nested metadata
// and unknown fields keep native compatibility rather than implying validity.
// Source rust-v0.162.1: rollout/src/lib.rs decode_rollout_line;
// protocol/src/protocol.rs SessionMetaLine, SessionMeta, TurnContextItem,
// AskForApproval and SandboxPolicy. No policy or model is executed here.
type codexNativeObject map[string]json.RawMessage
type codexNativeField func(json.RawMessage) bool

func codexNativeMap(raw json.RawMessage) (codexNativeObject, bool) {
	var out codexNativeObject
	err := json.Unmarshal(raw, &out)
	return out, err == nil && out != nil
}
func codexNativeString(raw json.RawMessage) bool {
	var s string
	return len(raw) > 0 && raw[0] == '"' && json.Unmarshal(raw, &s) == nil
}
func codexNativeBool(raw json.RawMessage) bool {
	return bytes.Equal(raw, []byte("true")) || bytes.Equal(raw, []byte("false"))
}
func codexNativeUint(raw json.RawMessage) bool {
	_, err := strconv.ParseUint(string(raw), 10, 64)
	return err == nil
}
func codexNativeNull(raw json.RawMessage) bool { return bytes.Equal(raw, []byte("null")) }
func codexNativeOptional(f codexNativeField) codexNativeField {
	return func(raw json.RawMessage) bool { return codexNativeNull(raw) || f(raw) }
}
func codexNativeUUID(raw json.RawMessage) bool {
	var s string
	return codexNativeString(raw) && json.Unmarshal(raw, &s) == nil && IsCanonicalCodexSessionID(s)
}
func codexNativeAbsolute(raw json.RawMessage) bool {
	var s string
	return codexNativeString(raw) && json.Unmarshal(raw, &s) == nil && filepath.IsAbs(s)
}
func codexNativeStrings(raw json.RawMessage) bool {
	var values []json.RawMessage
	if len(raw) == 0 || raw[0] != '[' || json.Unmarshal(raw, &values) != nil {
		return false
	}
	for _, v := range values {
		if !codexNativeString(v) {
			return false
		}
	}
	return true
}
func codexNativeEmptyArray(raw json.RawMessage) bool {
	var values []json.RawMessage
	return len(raw) > 0 && raw[0] == '[' && json.Unmarshal(raw, &values) == nil && len(values) == 0
}
func codexNativeEnum(values ...string) codexNativeField {
	return func(raw json.RawMessage) bool {
		var s string
		if !codexNativeString(raw) || json.Unmarshal(raw, &s) != nil {
			return false
		}
		for _, v := range values {
			if s == v {
				return true
			}
		}
		return false
	}
}
func codexNativeFields(o codexNativeObject, required, optional map[string]codexNativeField) bool {
	for key, check := range required {
		raw, ok := o[key]
		if !ok || !check(raw) {
			return false
		}
	}
	for key, raw := range o {
		if _, ok := required[key]; ok {
			continue
		}
		check, ok := optional[key]
		if !ok || !check(raw) {
			return false
		}
	}
	return true
}
func codexNativeEnvelope(raw []byte, kind string) (codexNativeObject, bool) {
	if !codexNativeJSONUnicode(raw) {
		return nil, false
	}
	o, ok := codexNativeMap(raw)
	if !ok {
		return nil, false
	}
	if !codexNativeFields(o, map[string]codexNativeField{"timestamp": codexNativeString, "type": codexNativeEnum(kind), "payload": func(v json.RawMessage) bool { _, ok := codexNativeMap(v); return ok }}, map[string]codexNativeField{"ordinal": codexNativeOptional(codexNativeUint)}) {
		return nil, false
	}
	return codexNativeMap(o["payload"])
}
func codexNativeSessionSource(raw json.RawMessage) bool {
	// Serde's other unit variant accepts unfamiliar string labels, but the three
	// data-carrying variants cannot be represented as bare strings.
	var label string
	if codexNativeString(raw) && json.Unmarshal(raw, &label) == nil {
		return label != "custom" && label != "internal" && label != "subagent"
	}
	o, ok := codexNativeMap(raw)
	if !ok || len(o) != 1 {
		return false
	}
	if v, ok := o["custom"]; ok {
		return codexNativeString(v)
	}
	if v, ok := o["internal"]; ok {
		return codexNativeEnum("memory_consolidation", "guardian")(v)
	}
	// Subagent payloads do not supply root eligibility to this adapter.
	return false
}
func codexNativeBaseInstructions(raw json.RawMessage) bool {
	o, ok := codexNativeMap(raw)
	return ok && codexNativeFields(o, map[string]codexNativeField{"text": codexNativeString}, map[string]codexNativeField{"provenance": codexNativeNull})
}
func codexNativeSessionMetaEligible(raw []byte) bool {
	o, ok := codexNativeEnvelope(raw, "session_meta")
	if !ok {
		return false
	}
	// SessionMetaLine fills session_id from id if omitted. Neither source nor
	// history_mode is required: their native defaults are vscode and legacy.
	optional := map[string]codexNativeField{
		"session_id": codexNativeUUID, "source": codexNativeSessionSource,
		"history_mode":              codexNativeEnum("legacy", "paginated"),
		"runtime_workspace_roots":   codexNativeOptional(codexNativeStrings),
		"base_instructions":         codexNativeOptional(codexNativeBaseInstructions),
		"dynamic_tools":             codexNativeOptional(codexNativeEmptyArray),
		"selected_capability_roots": codexNativeEmptyArray,
	}
	for _, key := range []string{"creator_user_id", "creator_account_id", "agent_nickname", "agent_role", "agent_type", "agent_path", "model_provider", "memory_mode", "thread_source"} {
		optional[key] = codexNativeOptional(codexNativeString)
	}
	for _, key := range []string{"forked_from_id", "parent_thread_id"} {
		optional[key] = codexNativeOptional(codexNativeUUID)
	}
	for _, key := range []string{"forked_from_ordinal_exclusive", "subagent_history_start_ordinal"} {
		optional[key] = codexNativeOptional(codexNativeUint)
	}
	for _, key := range []string{"history_base", "multi_agent_version", "context_window", "git"} {
		optional[key] = codexNativeNull
	}
	if _, role := o["agent_role"]; role {
		if _, alias := o["agent_type"]; alias {
			return false
		}
	}
	return codexNativeFields(o, map[string]codexNativeField{"id": codexNativeUUID, "timestamp": codexNativeString, "cwd": codexNativeString, "originator": codexNativeString, "cli_version": codexNativeString}, optional)
}
func codexNativeApproval(raw json.RawMessage) bool {
	if codexNativeEnum("untrusted", "on-request", "on-failure", "never")(raw) {
		return true
	}
	o, ok := codexNativeMap(raw)
	if !ok || len(o) != 1 {
		return false
	}
	g, ok := codexNativeMap(o["granular"])
	if !ok {
		return false
	}
	return codexNativeFields(g, map[string]codexNativeField{"sandbox_approval": codexNativeBool, "rules": codexNativeBool, "mcp_elicitations": codexNativeBool}, map[string]codexNativeField{"skill_approval": codexNativeBool, "request_permissions": codexNativeBool})
}
func codexNativeSandbox(raw json.RawMessage) bool {
	o, ok := codexNativeMap(raw)
	if !ok {
		return false
	}
	var kind string
	if json.Unmarshal(o["type"], &kind) != nil {
		return false
	}
	optional := map[string]codexNativeField{}
	switch kind {
	case "danger-full-access":
	case "read-only":
		optional["network_access"] = codexNativeBool
	case "external-sandbox":
		optional["network_access"] = codexNativeEnum("restricted", "enabled")
	case "workspace-write":
		for _, k := range []string{"network_access", "exclude_tmpdir_env_var", "exclude_slash_tmp"} {
			optional[k] = codexNativeBool
		}
		optional["writable_roots"] = func(v json.RawMessage) bool {
			var roots []json.RawMessage
			if len(v) == 0 || v[0] != '[' || json.Unmarshal(v, &roots) != nil {
				return false
			}
			for _, root := range roots {
				if !codexNativeAbsolute(root) {
					return false
				}
			}
			return true
		}
	default:
		return false
	}
	return codexNativeFields(o, map[string]codexNativeField{"type": codexNativeEnum(kind)}, optional)
}
func codexNativeTurnContextCWD(raw []byte) (string, bool) {
	o, ok := codexNativeEnvelope(raw, "turn_context")
	if !ok {
		return "", false
	}
	optional := map[string]codexNativeField{"disabled_plugin_ids": codexNativeOptional(codexNativeStrings), "realtime_active": codexNativeOptional(codexNativeBool)}
	for _, key := range []string{"turn_id", "root_turn_id", "comp_hash"} {
		optional[key] = codexNativeOptional(codexNativeString)
	}
	for _, key := range []string{"approvals_reviewer", "permission_profile", "active_permission_profile", "file_system_sandbox_policy", "collaboration_mode", "multi_agent_version", "cyber_access_program", "effort", "summary"} {
		optional[key] = codexNativeNull
	}
	if !codexNativeFields(o, map[string]codexNativeField{"cwd": codexNativeAbsolute, "approval_policy": codexNativeApproval, "sandbox_policy": codexNativeSandbox, "model": codexNativeString}, optional) {
		return "", false
	}
	var cwd string
	_ = json.Unmarshal(o["cwd"], &cwd)
	return cwd, true
}

// Go's JSON decoder replaces invalid UTF-8 and unpaired UTF-16 surrogates,
// while native serde rejects them. Conservatively leave all surrogate escapes
// to native, including valid pairs; literal escaped-backslash text is safe.
func codexNativeJSONUnicode(raw []byte) bool {
	if !utf8.Valid(raw) {
		return false
	}
	for i := 0; i+1 < len(raw); i++ {
		if raw[i] != '\\' {
			continue
		}
		if raw[i+1] == 'u' && i+6 <= len(raw) {
			code, err := strconv.ParseUint(string(raw[i+2:i+6]), 16, 16)
			if err == nil && code >= 0xd800 && code <= 0xdfff {
				return false
			}
		}
		i++ // Consume one JSON escape: only odd backslash runs introduce \u.
	}
	return true
}
