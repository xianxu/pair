package sessioninventory_test

import (
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

func TestCodexLargeRecordsPreserveIdentityAndEvidence(t *testing.T) {
	const id = "019d1111-1111-7111-8111-111111111111"
	meta := `{"type":"session_meta","payload":{"id":"` + id + `","source":"cli"}}`
	for _, size := range []int{1<<20 + 1, 8<<20 + 1} {
		padding := `{"type":"response_item","payload":{"type":"function_call_output","output":"` + strings.Repeat("x", size) + `"}}`
		body := meta + "\n" + padding + "\n" +
			`{"type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"after large record"}]}}` + "\n" +
			`{"type":"event_msg","payload":{"type":"token_count","info":{"last_token_usage":{"input_tokens":123}}}}`
		runtime := codexRuntimeWithRecord(t, id, body)
		inventory := inventoryFromScan(sessioninventory.ScanCodex(runtime))
		if len(inventory.Forests) != 1 || len(inventory.Forests[0].Roots) != 1 {
			t.Fatalf("size=%d: valid root lost: diagnostics=%v", size, inventory.Diagnostics)
		}
		root := inventory.Forests[0].Roots[0]
		root.Agent = sessioninventory.AgentCodex
		if !root.Resumable || root.NativeID != id {
			t.Fatalf("size=%d: root=%+v", size, root)
		}
		events, diagnostics := sessioninventory.NativeEventsWithRuntime(runtime, inventory, sessioninventory.AgentCodex)
		if len(diagnostics) != 0 || len(events) != 2 || events[0].Event.Kind != sessioninventory.EventToolResult || events[1].Event.Text != "after large record" {
			t.Fatalf("size=%d: events=%v diagnostics=%v", size, events, diagnostics)
		}
		usage, found, err := sessioninventory.TokenUsageForRoot(runtime, root)
		if err != nil || !found || usage.InputTokens != 123 {
			t.Fatalf("size=%d: usage=%v found=%v err=%v", size, usage, found, err)
		}
	}
}

func TestCodexLargeRecordsDoNotHideInvalidIdentity(t *testing.T) {
	const id = "019d1111-1111-7111-8111-111111111111"
	meta := `{"type":"session_meta","payload":{"id":"` + id + `","source":"cli"}}`
	padding := `{"type":"event_msg","payload":{"type":"item_completed","text":"` + strings.Repeat("x", 1<<20+1) + `"}}`
	for name, last := range map[string]string{
		"malformed":   `{"type":"event_msg","payload":!}`,
		"conflicting": `{"type":"session_meta","payload":{"id":"019d9999-9999-7999-8999-999999999999","source":"cli"}}`,
	} {
		t.Run(name, func(t *testing.T) {
			scan := sessioninventory.ScanCodex(codexRuntimeWithRecord(t, id, meta+"\n"+padding+"\n"+last))
			// The large record is valid: the later invalid record must still be
			// visited, retaining a disputed fact instead of losing the transcript.
			if len(scan.Facts) != 1 || len(scan.Diagnostics) == 0 {
				t.Fatalf("facts=%v diagnostics=%v", scan.Facts, scan.Diagnostics)
			}
			inventory := inventoryFromScan(scan)
			for _, forest := range inventory.Forests {
				for _, root := range forest.Roots {
					if root.Resumable {
						t.Fatal("invalid identity became resumable")
					}
				}
			}
		})
	}
}
