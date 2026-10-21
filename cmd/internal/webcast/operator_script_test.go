package webcast

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/broadcast"
)

func TestOperatorModuleAssembly(t *testing.T) {
	for _, source := range []string{
		"export default 1;", "import x from './unknown.js';", "export * from './unknown.js';",
		"import { x } from './unknown.js';", "export class {}",
	} {
		if _, err := assembleOperatorModule(source, nil); err == nil {
			t.Errorf("unsupported module admitted: %s", source)
		}
	}
	js, err := assembleOperatorModule("import { value } from './dep.js';\nconst result = value + 1;\nexport { result };", map[string]string{"./dep.js": "dep"})
	if err != nil {
		t.Fatal(err)
	}
	runOperatorJS(t, "const dep={value:41}; const m="+js+"; if(m.result!==42) throw Error('imports');")
}

func TestOperatorSharedModulesExecute(t *testing.T) {
	collab, err := assembleOperatorModule(string(broadcast.CollabJS()), nil)
	if err != nil {
		t.Fatal(err)
	}
	bytes, err := webFS.ReadFile("web/annotate.js")
	if err != nil {
		t.Fatal(err)
	}
	policyBytes, err := webFS.ReadFile("web/annotation-policy.js")
	if err != nil {
		t.Fatal(err)
	}
	policy, err := assembleOperatorModule(string(policyBytes), nil)
	if err != nil {
		t.Fatal(err)
	}
	annotate, err := assembleOperatorModule(string(bytes), map[string]string{"./collab.js": "collab", "./annotation-policy.js": "policy", "./capture-draft.js": "capture"})
	if err != nil {
		t.Fatal(err)
	}
	runOperatorJS(t, "const collab="+collab+"; const policy="+policy+"; const capture={}; const annotate="+annotate+`;const b=annotate.regionOf([50,50],[50,50],{x:0,y:0,w:100,h:100},'mouse'); if(JSON.stringify(b)!=='[45,45,10,10]') throw Error('box'); const c=annotate.chunker(); c.add(1,2,0); if(c.take()[0][0][0]!==1) throw Error('stroke');`)
}

func runOperatorJS(t *testing.T, source string) {
	t.Helper()
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not installed")
	}
	cmd := exec.Command(node, "--input-type=commonjs")
	cmd.Stdin = strings.NewReader(source)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("assembled JavaScript: %v\n%s", err, out)
	}
}

func TestOperatorNamedClassModuleAssembly(t *testing.T) {
	js, err := assembleOperatorModule("export class Scene { value(){return 42;} }", nil)
	if err != nil {
		t.Fatal(err)
	}
	runOperatorJS(t, "const m="+js+"; if(new m.Scene().value()!==42)throw Error('named class');")
}
