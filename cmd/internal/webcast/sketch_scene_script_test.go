package webcast

import "testing"

func TestSketchSceneScriptAssemblyAndTransferBounds(t *testing.T) {
	script, err := buildSketchSceneScript()
	if err != nil {
		t.Fatal(err)
	}
	runOperatorJS(t, script+`;
 const api=globalThis.__couchcastSketchScene;
 if(!api||typeof api.prepareSource!=='function'||typeof api.render!=='function'||typeof api.hitTest!=='function')throw Error('actual scene API missing');
 const mustFail=fn=>{let failed=false;try{fn()}catch{failed=true}if(!failed)throw Error('malformed transfer admitted');};
 mustFail(()=>api.beginTransfer('h',16777217));
 api.beginTransfer('asset',16777217,'asset');api.abortTransfer();mustFail(()=>api.beginTransfer('asset',33554433,'asset'));
 api.beginTransfer('h',4);mustFail(()=>api.beginTransfer('other',1));
 mustFail(()=>api.appendTransfer('h',1,'YQ=='));api.appendTransfer('h',0,'YWJjZA==');
 mustFail(()=>api.appendTransfer('other',0,'YQ=='));api.abortTransfer();mustFail(()=>api.appendTransfer('h',0,'YQ=='));
 `)
}
