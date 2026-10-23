package webcast

import (
	"strings"
	"testing"
)

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

func TestSketchSceneScriptRendersOrderedHistoricalMutationsBeforeReplacement(t *testing.T) {
	script, err := buildSketchSceneScript()
	if err != nil {
		t.Fatal(err)
	}
	// Keep the assembled real Model and production runtime, replacing only the
	// browser-facing renderer with a stateful contract observer.
	source := strings.TrimSuffix(script, "\n})();") + `;
 const fixtureRecords=[
 {k:'head',t:0,format:'couch-webcast',v:1,id:'scene',started:'now'},
 {k:'seg',t:1,seg:1,why:'live',tab:{id:'tab',url:'https://fixture/'},vp:{w:100,h:50,dpr:1}},
 {k:'doc',t:2,doc:1,url:'https://fixture/',tree:{id:1,y:'doc',c:[{id:2,y:'el',n:'html',c:[{id:7,y:'el',n:'body',c:[{id:8,y:'text',v:'old'}]}]}]}},
 {k:'attr',t:3,id:7,name:'class',v:'old-root'},
 {k:'attr',t:4,id:7,name:'style',v:'color:red'},
 {k:'text',t:5,id:8,v:'changed before replacement'},
 {k:'kids',t:6,p:2,c:[{id:20,y:'el',n:'body',c:[{id:21,y:'text',v:'new screen'}]}]},
 {k:'sketch-begin',t:7,id:'s',author:{id:'operator',color:'#ff0000'},owner:'operator',doc:1,tab:'tab',vp:{w:100,h:50,dpr:1},operation:'begin',sequence:0,digest:'a'.repeat(64)}];
 const frame={style:{}};globalThis.document={getElementById:id=>id==='page'?frame:null};
 let count=0,renderModel,disposed=0;
 render.Renderer=class {
  constructor(_,m){renderModel=m;this.dom=new Map();this.topDoc={getAnimations:()=>[]};}
  apply(r){
   if(renderModel.t!==r.t)throw Error('renderer received final model instead of ordered record '+r.k);
   if(r.k==='attr'&&renderModel.nodes.get(r.id)?.a?.[r.name]!==r.v)throw Error('historical attribute node missing');
   if(r.k==='text'&&renderModel.nodes.get(r.id)?.v!==r.v)throw Error('historical text node missing');
   if(r.k==='kids'&&(renderModel.nodes.has(7)||renderModel.nodes.get(21)?.v!=='new screen'))throw Error('replacement not applied before renderer');
   count++;
  }
  flush(){} async ready(){} revokeAll(){disposed++;}
  nodeAt(){return 20;} boxOf(){return [10,20,30,40];}
 };
 (async()=>{
  const api=globalThis.__couchcastSketchScene,bytes=new TextEncoder().encode(fixtureRecords.map(r=>JSON.stringify(r)).join('\n')+'\n');
  api.beginTransfer('source',bytes.length);api.appendTransfer('source',0,Buffer.from(bytes).toString('base64'));
  api.prepareSource('source',{time:7,doc:1,tab:'tab',length:bytes.length,sketch:'s'});
  const ready=await api.render();
  if(!ready.ready||ready.doc!==1||ready.time!==7||count!==fixtureRecords.length||renderModel.nodes.has(7))throw Error('final source state differs');
  if(api.hitTest([11,22]).node!==20)throw Error('final hit testing lost');
  api.dispose();if(disposed!==1)throw Error('scene cleanup lost');
 })().catch(error=>{console.error(error);process.exitCode=1;});
})();`
	runOperatorJS(t, source)
}
