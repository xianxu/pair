package sessioninventory

import "testing"

func TestScannerStateValidationRejectsIncompleteRoleIdentity(t *testing.T) {
	t.Parallel()
	valid := ScannerState{Version: 1, Agent: AgentClaude, NativeID: "child", IdentityAnchor: "root", Role: RoleSubagent, ParentID: ptrTestString("root"), ScannerSchema: "claude-v1", FirstRecordValidated: true}
	if err := ValidateScannerState(valid); err != nil {
		t.Fatal(err)
	}
	for _, state := range []ScannerState{
		{},
		{Version: 1, Agent: AgentClaude, NativeID: "child", IdentityAnchor: "root", Role: RoleSubagent, ScannerSchema: "claude-v1"},
		{Version: 1, Agent: AgentClaude, NativeID: "root", IdentityAnchor: "root", Role: RoleRoot, ParentID: ptrTestString("parent"), ScannerSchema: "claude-v1"},
		{Version: 1, Agent: AgentClaude, NativeID: "root", IdentityAnchor: "root", Role: RoleUnknown, ScannerSchema: "claude-v1"},
	} {
		if err := ValidateScannerState(state); err == nil {
			t.Fatalf("invalid state accepted: %#v", state)
		}
	}
}

func ptrTestString(value string) *string { return &value }
