package sessioninventory_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessioninventorytest"
)

func TestInventoryWithRuntimeObservesStateAcrossCalls(t *testing.T) {
	t.Parallel()

	runtime := sessioninventorytest.NewFakeRuntime()
	root := sessioninventory.StorageRoot{Agent: sessioninventory.AgentClaude, Name: "claude", Path: "/native/claude"}
	runtime.AddRoot(root)
	runtime.PutFile(sessioninventory.FileEntry{Artifact: sessioninventory.Artifact{StorageRoot: root.Name, RelativePath: "root"}}, []byte("root:root-1"))

	first := sessioninventory.InventoryWithRuntime(runtime, fixtureScanner{agent: sessioninventory.AgentClaude})
	if got := first.Forests[0].Roots[0]; got.NativeID != "root-1" || len(got.Children) != 0 {
		t.Fatalf("first inventory root = %#v", got)
	}

	runtime.PutFile(sessioninventory.FileEntry{Artifact: sessioninventory.Artifact{StorageRoot: root.Name, RelativePath: "child"}}, []byte("child:child-1:root-1"))
	second := sessioninventory.InventoryWithRuntime(runtime, fixtureScanner{agent: sessioninventory.AgentClaude})
	if got := second.Forests[0].Roots[0]; len(got.Children) != 1 || got.Children[0].NativeID != "child-1" {
		t.Fatalf("second inventory root = %#v, want newly persisted child", got)
	}
}

type fixtureScanner struct {
	agent sessioninventory.Agent
}

func (scanner fixtureScanner) Scan(runtime sessioninventory.Runtime) sessioninventory.ScanResult {
	var result sessioninventory.ScanResult
	for _, root := range runtime.NativeRoots(scanner.agent) {
		files, err := runtime.ListFiles(root)
		if err != nil {
			panic(err)
		}
		for _, file := range files {
			content, err := runtime.ReadFile(file.Artifact, 1024)
			if err != nil {
				panic(err)
			}
			fields := strings.Split(string(content), ":")
			switch fields[0] {
			case "root":
				result.Facts = append(result.Facts, sessioninventory.Fact{Agent: scanner.agent, NativeID: fields[1], Role: sessioninventory.RoleRoot, Artifacts: []sessioninventory.Artifact{file.Artifact}})
			case "child":
				parent := fields[2]
				result.Facts = append(result.Facts, sessioninventory.Fact{Agent: scanner.agent, NativeID: fields[1], Role: sessioninventory.RoleSubagent, ParentID: &parent, Artifacts: []sessioninventory.Artifact{file.Artifact}})
			default:
				panic(fmt.Sprintf("unknown fixture record %q", content))
			}
		}
	}
	return result
}
