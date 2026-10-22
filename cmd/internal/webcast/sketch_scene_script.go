package webcast

import "strings"

// buildSketchSceneScript assembles the actual viewer Model and Renderer for a
// classic script in the child opaque-origin isolated world. It contains no
// binding, credentials, source data or public page messaging channel.
func buildSketchSceneScript() (string, error) {
	type part struct {
		name, file string
		imports    map[string]string
	}
	parts := []part{
		{"anchor", "annotation-anchor.js", nil},
		{"placement", "annotation-placement.js", map[string]string{"./annotation-anchor.js": "anchor"}},
		{"css", "css.js", nil},
		{"sketch", "sketch-model.js", nil},
		{"model", "model.js", map[string]string{"./sketch-model.js": "sketch"}},
		{"render", "render.js", map[string]string{"./annotation-placement.js": "placement", "./css.js": "css"}},
	}
	var code strings.Builder
	code.WriteString("(()=>{\n")
	for _, p := range parts {
		raw, err := webFS.ReadFile("web/" + p.file)
		if err != nil {
			return "", err
		}
		module, err := assembleOperatorModule(string(raw), p.imports)
		if err != nil {
			return "", err
		}
		code.WriteString("const " + p.name + "=" + module + ";\n")
	}
	code.WriteString(sketchSceneRuntime)
	code.WriteString("\n})();")
	return code.String(), nil
}

const sketchSceneRuntime = `
const need=(ok,why)=>{if(!ok)throw Error('private sketch scene: '+why);};
const safe=n=>Number.isSafeInteger(n)&&n>=0;
const hash=h=>typeof h==='string'&&/^[0-9a-f]{64}$/.test(h);
let transfer=null,state=null,records=null,descriptor=null,renderer=null,ready=false;
const assets=new Map();
function releaseScene(){ready=false;renderer?.revokeAll();renderer=null;for(const value of assets.values())URL.revokeObjectURL(value.url);assets.clear();state=null;records=null;descriptor=null;}
function consume(handle){need(transfer&&transfer.handle===handle&&transfer.offset===transfer.bytes.length,'incomplete transfer');const bytes=transfer.bytes;transfer=null;return bytes;}
globalThis.__couchcastSketchScene={
 beginTransfer(handle,total,mode='source'){need(!transfer&&typeof handle==='string'&&handle.length>0&&handle.length<=128&&safe(total)&&(mode==='source'||mode==='asset')&&total<=(mode==='asset'?32:16)*1024*1024,'invalid transfer');transfer={handle,offset:0,bytes:new Uint8Array(total)};return {offset:0,total};},
 appendTransfer(handle,offset,base64){need(transfer&&transfer.handle===handle&&safe(offset)&&typeof base64==='string'&&base64.length<=43692,'invalid chunk');const text=atob(base64);need(text.length>0&&text.length<=32768&&offset+text.length<=transfer.bytes.length,'chunk bound');if(offset<transfer.offset){need(offset+text.length<=transfer.offset,'overlapping chunk');for(let i=0;i<text.length;i++)need(transfer.bytes[offset+i]===text.charCodeAt(i),'conflicting chunk');}else{need(offset===transfer.offset,'chunk gap');for(let i=0;i<text.length;i++)transfer.bytes[offset+i]=text.charCodeAt(i);transfer.offset+=text.length;}return {offset:transfer.offset,total:transfer.bytes.length};},
 abortTransfer(){transfer=null;},
 prepareSource(handle,source){
  const bytes=consume(handle);need(bytes.length<=16*1024*1024,'source bound');releaseScene();
  need(source&&safe(source.time)&&safe(source.doc)&&source.doc>0&&typeof source.tab==='string'&&source.tab&&safe(source.length)&&source.length===bytes.length,'source descriptor');
  const text=new TextDecoder('utf-8',{fatal:true}).decode(bytes);need(text.endsWith('\n'),'torn source prefix');
  const nextRecords=text.split('\n').filter(Boolean).map(line=>JSON.parse(line));const next=new model.Model({allowSketchSnapshot:true});for(const record of nextRecords)next.apply(record);
  const last=nextRecords.at(-1);need(last?.k==='sketch-begin'&&last.t===source.time&&next.t===source.time&&next.docSeg===source.doc&&next.tab.id===source.tab&&(!source.sketch||last.id===source.sketch),'source identity');
  state=next;records=nextRecords;descriptor=structuredClone(source);
  return {assets:Array.from(next.assets.values(),a=>({hash:a.hash,mime:a.mime,size:a.size}))};
 },
 installAsset(handle,digest,mime){
  need(state&&hash(digest)&&typeof mime==='string','asset identity');const named=Array.from(state.assets.values()).some(a=>a.hash===digest&&a.mime===mime);need(named,'unnamed asset');const bytes=consume(handle);const previous=assets.get(digest);if(previous)URL.revokeObjectURL(previous.url);assets.set(digest,{url:URL.createObjectURL(new Blob([bytes],{type:mime})),mime});return {length:bytes.length,url:assets.get(digest).url};
 },
 async render(){
  need(state&&records,'source not prepared');const frame=document.getElementById('page');need(frame,'missing opaque scene frame');
  frame.style.width=state.vp.w+'px';frame.style.height=state.vp.h+'px';frame.style.border='0';
  renderer=new render.Renderer(frame,state,url=>{const binding=state.assets.get(url);return binding&&assets.get(binding.hash)?.url||'';});
  for(const record of records)renderer.apply(record);renderer.flush();await renderer.ready();renderer.flush();
  const roots=new Set([renderer.topDoc,...Array.from(renderer.dom.values()).filter(n=>n.nodeType===9||n.nodeType===11)]);for(const root of roots)for(const animation of root?.getAnimations?.()||[]){try{animation.finish();}catch{}animation.pause();}
  ready=true;return {ready:true,doc:state.docSeg,tab:state.tab.id,time:state.t,vp:structuredClone(state.vp),scroll:[...(state.scrolls.get(state.top.id)||[0,0])]};
 },
 hitTest(point){need(ready&&Array.isArray(point)&&point.length===2&&point.every(n=>Number.isFinite(n)&&Math.abs(n)<=1e7),'invalid hit test');const node=renderer.nodeAt(...point),box=node&&renderer.boxOf(node);need(node>0&&box&&box.length===4&&box.every(Number.isFinite)&&box[2]>0&&box[3]>0,'no recorded element');return {node,box:[...box]};},
 placement(path){need(ready,'scene not ready');if(path?.source?.doc!==state.docSeg||path.source.tab!==state.tab.id||path.source.t>state.t)return null;return renderer.placementOf(path);},
 inspect(ids){need(ready&&Array.isArray(ids)&&ids.length<=32&&ids.every(safe),'invalid inspection');return ids.map(id=>{const node=renderer.dom.get(id);if(!node)return null;const box=renderer.boxOf(id);return {id,text:(node.textContent||'').slice(0,200),name:node.name||'',width:box?.[2]||0,height:box?.[3]||0,imageWidth:node.naturalWidth||0,imageHeight:node.naturalHeight||0,src:node.src||'',nested:node.contentDocument?.body?.textContent?.slice(0,200)||''};});},
 dispose(){transfer=null;releaseScene();const frame=globalThis.document?.getElementById('page');if(frame)frame.src='about:blank';return true;}
};`
