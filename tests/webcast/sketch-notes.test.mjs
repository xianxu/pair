import { test } from 'node:test';
import assert from 'node:assert/strict';
const load=()=>import('../../cmd/internal/webcast/web/sketch-notes.js');
const options={metadata:()=>({phase:'complete',label:'final'}),editable:new Set(),viewport:{width:300,height:200}};
test('visible painted bounds keep a control when the first sample scrolled away, union multiple paths',async()=>{
 const {sketchNoteEntries}=await load();
 const paths=[{sketch:'a',get pts(){throw Error('must reuse painted bounds');}},{sketch:'a'},{sketch:'hidden'}];
 assert.deepEqual(sketchNoteEntries(paths,[{left:10,top:-400,right:60,bottom:50},{left:100,top:30,right:160,bottom:90},{left:400,top:1,right:420,bottom:8}],options),[
 {sketch:'a',bounds:{left:10,top:0,right:160,bottom:90},label:'final',editable:false},
 {sketch:'hidden',bounds:null,label:'final',editable:false}]);
});
test('accepted sketch identity survives scroll-away, missing and malformed placement',async()=>{
 const {sketchNoteEntries}=await load();
 for(const bounds of [null,undefined,{left:0,top:220,right:40,bottom:250},{left:NaN,top:0,right:4,bottom:5}]){
  assert.deepEqual(sketchNoteEntries([{sketch:'finished'},{sketch:'owned'},{sketch:'not-owned'}],[bounds,bounds,bounds],{
   ...options,metadata:id=>id==='finished'?{phase:'complete',label:'Immutable label'}:{phase:'interrupted'},editable:new Set(['owned']),
  }),[{sketch:'finished',bounds:null,label:'Immutable label',editable:false},{sketch:'owned',bounds:null,label:'',editable:true}]);
 }
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
test('repeated unplaced identities read metadata once, including ineligible sketches',async()=>{
 const {sketchNoteEntries}=await load(),reads=new Map();
 const paths=Array.from({length:256},(_,i)=>({sketch:i%2?'labeled':'ineligible'}));
 const result=sketchNoteEntries(paths,[],{...options,metadata:id=>{
  reads.set(id,(reads.get(id)||0)+1);return id==='labeled'?{phase:'complete',label:'Kept'}:{};
 }});
 assert.deepEqual(result,[{sketch:'labeled',bounds:null,label:'Kept',editable:false}]);
 assert.deepEqual([...reads],[['ineligible',1],['labeled',1]]);
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
   const id=paths[i].sketch,b=bounds[i];if(typeof id!=='string')continue;
   const info=meta.get(id),label=typeof info?.label==='string'?info.label:'',add=!label&&editable.has(id)&&['drawing','interrupted','label-required'].includes(info?.phase);if(!label&&!add)continue;
   if(!expected.has(id))expected.set(id,{sketch:id,bounds:null,label,editable:add});
   if(!b||![b.left,b.top,b.right,b.bottom].every(Number.isFinite)||b.left>b.right||b.top>b.bottom)continue;
   const clipped={left:Math.max(0,b.left),top:Math.max(0,b.top),right:Math.min(300,b.right),bottom:Math.min(200,b.bottom)};
   if(clipped.left>=clipped.right||clipped.top>=clipped.bottom)continue;
   const old=expected.get(id);if(old.bounds){for(const k of ['left','top'])old.bounds[k]=Math.min(old.bounds[k],clipped[k]);for(const k of ['right','bottom'])old.bounds[k]=Math.max(old.bounds[k],clipped[k]);}else old.bounds=clipped;
  }
  assert.deepEqual(sketchNoteEntries(paths,bounds,{...options,metadata:k=>meta.get(k),editable}),[...expected.values()]);
 }
});
