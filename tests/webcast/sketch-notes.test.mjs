import { test } from 'node:test';
import assert from 'node:assert/strict';
import { Element } from './dom-fixture.mjs';
const load=()=>import('../../cmd/internal/webcast/web/sketch-notes.js');
const options={metadata:()=>({phase:'complete',label:'final'}),editable:new Set(),viewport:{width:300,height:200}};
test('visible painted bounds keep a control when the first sample scrolled away, union multiple paths',async()=>{
 const {sketchNoteEntries}=await load();
 const paths=[{sketch:'a',get pts(){throw Error('must reuse painted bounds');}},{sketch:'a'},{sketch:'hidden'}];
 assert.deepEqual(sketchNoteEntries(paths,[{left:10,top:-400,right:60,bottom:50},{left:100,top:30,right:160,bottom:90},{left:400,top:1,right:420,bottom:8}],options),[
 {sketch:'a',bounds:{left:10,top:0,right:160,bottom:90},label:'final',editable:false}]);
});
test('bounds grouping visits P/2P unique identities linearly without walking points',async()=>{
 const {sketchNoteEntries}=await load(),costs=[];
 for(const n of [128,256]){
  const cost={array:0,identity:0,bounds:0,metadata:0};
  const raw=Array.from({length:n},(_,i)=>new Proxy({sketch:String(i)},{get(o,k){if(k==='pts')throw Error('point rescan');if(k==='sketch')cost.identity++;return o[k];}}));
  const paths=new Proxy(raw,{get(o,k){if(/^\d+$/.test(String(k)))cost.array++;return o[k];}});
  const bounds=new Proxy(raw.map(()=>({left:10,top:10,right:20,bottom:20})),{get(o,k){if(/^\d+$/.test(String(k)))cost.bounds++;return o[k];}});
  const entries=sketchNoteEntries(paths,bounds,{...options,metadata:()=>{cost.metadata++;return {label:'final'};}});
  assert.equal(entries.length,n);assert.equal(cost.metadata,n);assert.ok(cost.identity<=12*n);assert.ok(cost.array<=2*n);assert.ok(cost.bounds<=2*n);costs.push(cost);
 }
 for(const key of Object.keys(costs[0]))assert.ok(costs[1][key]<=2*costs[0][key],key);
});
test('generated malformed painted bounds and metadata follow independent visibility and identity oracle',async()=>{
 const {sketchNoteEntries}=await load();
 const samples=[null,{}, {left:NaN,top:1,right:2,bottom:3},{left:1,top:Infinity,right:2,bottom:3},{left:4,top:1,right:2,bottom:3},{left:1,top:4,right:2,bottom:3},{left:-20,top:-10,right:30,bottom:40},{left:290,top:190,right:340,bottom:210},{left:310,top:1,right:400,bottom:30},{left:10,top:20,right:30,bottom:40}];
 const infos=[null,{}, {label:'done'}, {phase:'drawing',label:9}, {phase:'interrupted'}, {phase:'cleared',label:''}];
 let seed=850;const next=n=>{seed=(1664525*seed+1013904223)>>>0;return seed%n;};
 for(let round=0;round<12;round++){
  const paths=[],bounds=[],meta=new Map(),editable=new Set(),expected=new Map();
  for(let i=0;i<80;i++){const id=i%13===0?null:`s${i>>1}`;paths.push({sketch:id});bounds.push(samples[next(samples.length)]);if(typeof id==='string'){meta.set(id,infos[next(infos.length)]);if(next(2))editable.add(id);}}
  for(let i=0;i<paths.length;i++){
   const id=paths[i].sketch,b=bounds[i];if(typeof id!=='string'||!b||![b.left,b.top,b.right,b.bottom].every(Number.isFinite)||b.left>b.right||b.top>b.bottom)continue;
   const clipped={left:Math.max(0,b.left),top:Math.max(0,b.top),right:Math.min(300,b.right),bottom:Math.min(200,b.bottom)};
   if(clipped.left>=clipped.right||clipped.top>=clipped.bottom)continue;
   const old=expected.get(id);if(old){for(const k of ['left','top'])old.bounds[k]=Math.min(old.bounds[k],clipped[k]);for(const k of ['right','bottom'])old.bounds[k]=Math.max(old.bounds[k],clipped[k]);continue;}
   const info=meta.get(id),label=typeof info?.label==='string'?info.label:'',add=!label&&editable.has(id)&&['drawing','interrupted','label-required'].includes(info?.phase);if(!label&&!add)continue;
   expected.set(id,{sketch:id,bounds:clipped,label,editable:add});
  }
  assert.deepEqual(sketchNoteEntries(paths,bounds,{...options,metadata:k=>meta.get(k),editable}),[...expected.values()]);
 }
});
function fixture(){
 class E extends Element{appendChild(e){this.children.push(e);e.parentNode=this;return e;}remove(){if(this.parentNode)this.parentNode.children=this.parentNode.children.filter(x=>x!==this);}getBoundingClientRect(){return this.className==='sketch-note-control'?{width:80,height:44}:{left:0,top:0,width:300,height:200};}}
 const root=new E('root'),doc={createElement:t=>new E(t)},click=e=>e.listeners.click[0]({stopPropagation(){},preventDefault(){}});return {root,doc,click};
}
test('keyed controls use latest identity and final view never mutates',async()=>{
 const {mountSketchNoteControls}=await load(),h=fixture(),sent=[],ui=mountSketchNoteControls(h.doc,{parent:h.root,onAdd:k=>sent.push(k)});
 ui.update([{sketch:'a',bounds:{left:10,top:140,right:130,bottom:150},label:'',editable:true}],options.viewport);
 const b=h.root.querySelector('.sketch-note-control');h.click(b);assert.deepEqual(sent,['a']);
 assert.ok(Number.parseFloat(b.style.top)+44<=132,'8px above ink; (110,140) remains reachable');
 ui.update([{sketch:'a',bounds:{left:1,top:1,right:2,bottom:2},label:'<script>text</script>',editable:false}],options.viewport);
 assert.equal(h.root.querySelector('.sketch-note-control'),b);h.click(b);assert.equal(h.root.querySelector('.sketch-note-text').textContent,'<script>text</script>');assert.deepEqual(sent,['a']);
 h.click(h.root.querySelector('.sketch-note-close'));assert.equal(h.root.querySelector('.sketch-note-view').hidden,true);ui.update([],options.viewport);h.click(b);assert.deepEqual(sent,['a']);ui.dispose();assert.equal(h.root.children.length,0);
});
test('complete button remains inside all viewport edges and separated from ink where space permits',async()=>{
 const {mountSketchNoteControls}=await load(),h=fixture(),ui=mountSketchNoteControls(h.doc,{parent:h.root,onAdd:()=>{}});
 for(const bounds of [{left:0,top:0,right:10,bottom:10},{left:290,top:0,right:300,bottom:10},{left:0,top:190,right:10,bottom:200},{left:290,top:190,right:300,bottom:200}]){
 ui.update([{sketch:'a',bounds,label:'done'}],options.viewport);const b=h.root.querySelector('.sketch-note-control'),x=Number.parseFloat(b.style.left),y=Number.parseFloat(b.style.top);
 assert.ok(x>=0&&y>=0&&x+80<=300&&y+44<=200);assert.ok(y+44<=bounds.top-8||y>=bounds.bottom+8||x+80<=bounds.left-8||x>=bounds.right+8);
 }
 ui.dispose();
});
