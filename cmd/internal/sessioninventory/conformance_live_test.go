package sessioninventory

import (
	"os"
	"strings"
	"testing"
)

func TestLiveNativeSessionShapeConformance(t *testing.T) {
	if os.Getenv("PAIR_LIVE_NATIVE_SESSIONS") != "1" {
		t.Skip("set PAIR_LIVE_NATIVE_SESSIONS=1 to inspect installed native session shapes")
	}
	pairData := os.Getenv("PAIR_DATA_DIR")
	if pairData == "" {
		pairData = os.Getenv("HOME") + "/.local/share/pair"
	}
	runtime, err := DefaultOSRuntime(pairData)
	if err != nil {
		t.Fatal(err)
	}
	report, conformanceErr := RunConformance(runtime, AgentClaude, AgentCodex, AgentAgy, AgentMuse)
	rendered, err := RenderConformance(report)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(rendered), runtime.PairDataRoot().Path) || strings.Contains(string(rendered), "/Users/") || strings.Contains(string(rendered), "/home/") {
		t.Fatalf("conformance output leaked an absolute path: %s", rendered)
	}
	t.Logf("native session conformance (redacted): %s", rendered)
	codexScan := ScanCodex(runtime)
	codexInventory := BuildForest(codexScan.Facts)
	codexEvents, _ := NativeEventsWithRuntime(runtime, codexInventory, AgentCodex)
	if err := ValidateCodexLifecycleConformance(codexEvents); err != nil {
		t.Fatalf("Codex lifecycle envelope drift: %v", err)
	}
	t.Log("Codex lifecycle envelopes: keyed start/completion order conforms")
	if conformanceErr != nil {
		t.Fatal(conformanceErr)
	}
}
