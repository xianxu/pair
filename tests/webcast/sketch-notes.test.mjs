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
 const cost={create:0,measure:0,replace:0};
 class E extends Element{appendChild(e){this.children.push(e);e.parentNode=this;return e;}remove(){if(this.parentNode)this.parentNode.children=this.parentNode.children.filter(x=>x!==this);}replaceChildren(...es){cost.replace+=es.length;this.children=es;}getBoundingClientRect(){cost.measure++;return this.className==='sketch-notes-summary'?{width:140,height:44}:{left:0,top:0,width:300,height:200};}}
 const root=new E('root'),doc={createElement:t=>{cost.create++;return new E(t);}},click=e=>e.listeners.click[0]({stopPropagation(){},preventDefault(){}});return {root,doc,click,cost};
}
const entry=(sketch,bounds={left:10,top:140,right:130,bottom:150})=>({sketch,bounds,label:'',editable:true});
test('identical overlapping and edge-clamped bounds expose every sketch through one group',async()=>{
 const {mountSketchNoteControls}=await load(),h=fixture(),sent=[],ui=mountSketchNoteControls(h.doc,{parent:h.root,onAdd:k=>sent.push(k)});
 for(const bounds of [{left:10,top:140,right:130,bottom:150},{left:290,top:190,right:300,bottom:200}]){
  ui.update([entry('a',bounds),entry('b',bounds),entry('c',{...bounds,left:bounds.left-2})],options.viewport);
  const summary=h.root.querySelector('.sketch-notes-summary');assert.ok(summary,'one grouped affordance replaces overlapping per-sketch buttons');assert.equal(summary.textContent,'Sketch notes (3)');h.click(summary);
  const body=h.root.querySelector('.sketch-notes-list-body');assert.equal(body.children.length,3);
  assert.deepEqual(body.children.map(b=>b.dataset.sketch),['a','b','c']);
  for(const b of body.children){assert.match(b.textContent,/Add note.*[abc]/);h.click(b);}
 }
 assert.deepEqual(sent,['a','b','c','a','b','c']);ui.dispose();
});
test('keyed rows re-read latest identity and close list before existing Add or final inspection',async()=>{
 const {mountSketchNoteControls}=await load(),h=fixture(),sent=[],ui=mountSketchNoteControls(h.doc,{parent:h.root,onAdd:k=>sent.push(k)});
 ui.update([entry('a')],options.viewport);const summary=h.root.querySelector('.sketch-notes-summary');h.click(summary);
 const row=h.root.querySelector('.sketch-note-control');h.click(row);assert.deepEqual(sent,['a']);assert.equal(h.root.querySelector('.sketch-notes-list').hidden,true);
 ui.update([entry('b'),entry('a')],options.viewport);assert.equal(summary.textContent,'Sketch notes (2)');assert.equal(h.root.querySelector('.sketch-notes-list-body').children[1],row);assert.match(row.textContent,/^2\. Add note/);
 ui.update([{...entry('a'),label:'<script>text</script>',editable:false}],options.viewport);assert.equal(h.root.querySelector('.sketch-note-control'),row);assert.match(row.textContent,/View note/);h.click(summary);h.click(row);
 assert.equal(h.root.querySelector('.sketch-note-text').textContent,'<script>text</script>');assert.deepEqual(sent,['a']);assert.equal(h.root.querySelector('.sketch-notes-list').hidden,true);
 h.click(h.root.querySelector('.sketch-note-close'));assert.equal(h.root.querySelector('.sketch-note-view').hidden,true);
 h.click(summary);h.click(h.root.querySelector('.sketch-notes-list-close'));assert.equal(h.root.querySelector('.sketch-notes-list').hidden,true);
 ui.update([],options.viewport);h.click(row);h.click(summary);assert.deepEqual(sent,['a']);assert.equal(summary.hidden,true);assert.equal(h.root.querySelector('.sketch-notes-list').hidden,true);
 ui.dispose();h.click(row);assert.equal(h.root.children.length,0);
});
test('complete summary stays inside all edges and one bounded scrolling list retains all rows',async()=>{
 const {mountSketchNoteControls}=await load(),h=fixture(),ui=mountSketchNoteControls(h.doc,{parent:h.root,onAdd:()=>{}});
 for(const bounds of [{left:0,top:0,right:10,bottom:10},{left:290,top:0,right:300,bottom:10},{left:0,top:190,right:10,bottom:200},{left:290,top:190,right:300,bottom:200}]){
  ui.update(Array.from({length:80},(_,i)=>entry(String(i),bounds)),options.viewport);const b=h.root.querySelector('.sketch-notes-summary'),x=parseFloat(b.style.left),y=parseFloat(b.style.top);
  assert.ok(x>=0&&y>=0&&x+140<=300&&y+44<=200);h.click(b);
  const panel=h.root.querySelector('.sketch-notes-list');assert.ok(parseFloat(panel.style.width)<=300&&parseFloat(panel.style.height)<=200);assert.equal(h.root.querySelector('.sketch-notes-list-body').children.length,80);
 }
 ui.dispose();
});
test('group updates scale linearly and measure only one summary at N and 2N',async()=>{
 const {mountSketchNoteControls}=await load(),costs=[];
 for(const n of [128,256]){const h=fixture(),ui=mountSketchNoteControls(h.doc,{parent:h.root,onAdd:()=>{}});let visits=0;const entries=Array.from({length:n},(_,i)=>new Proxy(entry(String(i)),{get(o,k){visits++;return o[k];}}));ui.update(entries,options.viewport);assert.equal(h.cost.measure,1);assert.ok(h.cost.create<=n+12);assert.ok(visits<=20*n);costs.push({...h.cost,visits});const before=h.cost.replace;ui.update(entries,options.viewport);assert.equal(h.cost.replace,before,'unchanged keyed order does not detach focused rows');ui.dispose();}
 assert.ok(costs[1].create<=2*costs[0].create);assert.ok(costs[1].visits<=2*costs[0].visits);
});
