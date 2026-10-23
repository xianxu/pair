import {test} from 'node:test';
import assert from 'node:assert/strict';
import {createFeedbackVisibility} from '../../cmd/internal/webcast/web/feedback-visibility.js';

function fixture(){
 const scene={recording_id:'rec',t:10,record_index:4,doc:1,tab:'A',url:'https://example.test/',title:'Page',vp:{w:100,h:80,dpr:1},scroll:[0,0]};
 const mask={scope:'cast',revision:'0'},notices=[];let busy=false;
 const v=createFeedbackVisibility({context:()=>structuredClone(scene),mask:()=>({...mask}),busy:()=>busy,notify:revision=>notices.push(revision)});
 const metadata=()=>({phase:'complete',label:'Move this',next_sequence:5,digest:'a'.repeat(64)});
 const path={sketch:'s',id:'p',at:{box:[0,0,100,80]},width:4,pts:[[10,10,.5],[20,20,1]]};
 const target={box:[0,0,100,80],clip:[0,0,100,80],locator:[0,1]};
 const paint=()=>{const token=v.token();v.notesPaint(token,[{id:'n',revision:'0',box:[1,1,20,20]}]);v.sketchesPaint(token,[path],[{...target,clip:[...scene.scroll,scene.vp.w,scene.vp.h]}],metadata);};
 return {v,scene,mask,notices,path,target,metadata,paint,busy(value){busy=value;}};
}
test('visibility requires both completed paints and returns owned stable wire values',()=>{
 const f=fixture(),token=f.v.invalidate();assert.throws(()=>f.v.read(),/unavailable|paint/i);
 f.v.notesPaint(token,[{id:'n',revision:'0',box:[1,1,20,20]}]);assert.throws(()=>f.v.read(),/unavailable|paint/i);
 f.v.sketchesPaint(token,[f.path],[f.target],f.metadata);
 const value=f.v.read();assert.equal(value.revision,token);assert.equal(value.mask_revision,'0');assert.equal(value.sketches[0].next_sequence,'5');assert.deepEqual(value.sketches[0].paths[0],{id:'p',locator:[0,1],box:[0,0,100,80],clip:[0,0,100,80]});
 value.sketches[0].paths[0].locator[0]=99;assert.deepEqual(f.v.read().sketches[0].paths[0].locator,[0,1]);
 assert.deepEqual(f.notices,[token,token],'completion must wake the same monotonic revision');
});
test('invalidation fences stale async paint, live geometry, mask and busy tools',()=>{
 const f=fixture(),old=f.v.invalidate();f.paint();const fresh=f.v.invalidate();
 f.v.sketchesPaint(old,[f.path],[f.target],f.metadata);f.v.notesPaint(fresh,[]);assert.throws(()=>f.v.read(),/unavailable|paint/i);
 f.v.sketchesPaint(fresh,[f.path],[f.target],f.metadata);assert.equal(f.v.read().revision,fresh);
 f.scene.scroll[1]=1;assert.throws(()=>f.v.read(),/stale|changed/i);f.paint();assert.equal(f.v.read().scene.scroll[1],1);
 f.mask.revision='1';assert.throws(()=>f.v.read(),/stale|changed/i);f.paint();f.busy(true);assert.throws(()=>f.v.read(),/busy|drawing/i);
});
test('partial shape, unsupported element clipping and absent locator never become eligible',()=>{
 for(const kind of ['partial','clip','locator']){
  const f=fixture();f.v.invalidate();f.v.notesPaint(f.v.token(),[]);
  const paths=kind==='partial'?[f.path,{...f.path,id:'missing'}]:[f.path];
  const targets=kind==='partial'?[f.target,null]:[{...f.target,...(kind==='clip'?{clip:[0,0,50,80]}:{locator:null})}];
  f.v.sketchesPaint(f.v.token(),paths,targets,f.metadata);assert.throws(()=>f.v.read(),/partial|clip|locator|unavailable/i,kind);
 }
});
test('offscreen and unfinished shapes are omitted without claiming their ink',()=>{
 const f=fixture();f.v.invalidate();f.v.notesPaint(f.v.token(),[{id:'off',revision:'0',box:[200,200,10,10]}]);
 f.v.sketchesPaint(f.v.token(),[f.path],[null],f.metadata);assert.deepEqual(f.v.read().sketches,[]);assert.deepEqual(f.v.read().notes,[]);
 f.v.invalidate();f.v.notesPaint(f.v.token(),[]);f.v.sketchesPaint(f.v.token(),[f.path],[f.target],()=>({...f.metadata(),phase:'drawing'}));assert.deepEqual(f.v.read().sketches,[]);
});

test('completed paint visits accepted paths and pressure points linearly at P and 2P',()=>{
 const numeric=key=>typeof key==='string'&&/^(0|[1-9][0-9]*)$/.test(key);
 function paint(size){
  const f=fixture(),counts={paths:0,properties:0,points:0,coordinates:0,targets:0,metadata:0};
  const expectedIDs=['accepted-even','accepted-odd'],expectedPaths=new Map(expectedIDs.map(id=>[id,[]]));
  const watch=(value,kind)=>new Proxy(value,{get(target,key,receiver){if(numeric(key))counts[kind]++;return Reflect.get(target,key,receiver);}});
  const paths=watch(Array.from({length:size},(_,i)=>{
   const sketch=expectedIDs[i%2],id=`path-${i}`;
   expectedPaths.get(sketch).push(id);
   const pts=watch([[10,10,.1],[20,20,.5],[30,30,1]].map(point=>watch(point,'coordinates')),'points');
   return new Proxy({...f.path,sketch,id,pts},{get(target,key,receiver){counts.properties++;return Reflect.get(target,key,receiver);}});
  }),'paths');
  const targets=watch(Array.from({length:size},()=>f.target),'targets');
  const token=f.v.invalidate();f.v.notesPaint(token,[]);
  assert.equal(f.v.sketchesPaint(token,paths,targets,id=>{
   counts.metadata++;assert.ok(expectedPaths.has(id));return f.metadata();
  }),true);
  const actual=f.v.read().sketches;
  assert.deepEqual(actual.map(sketch=>sketch.id),expectedIDs);
  for(const sketch of actual){
   assert.deepEqual(sketch.paths.map(path=>path.id),expectedPaths.get(sketch.id));
   assert.equal(sketch.next_sequence,'5');
   assert.equal(sketch.digest,'a'.repeat(64));
   assert.ok(sketch.paths.length<=64,'generated fixture respects the per-shape path bound');
  }
  assert.ok(actual.length<=256,'generated fixture respects the retained-shape bound');
  assert.equal(counts.paths,size,'one accepted path-array visit per path');
  assert.equal(counts.targets,size,'one corresponding paint target per path');
  assert.equal(counts.metadata,size,'accepted metadata lookup does not scan other paths');
  assert.equal(counts.points,size*3,'every pressure point is projected once');
  assert.ok(counts.coordinates>=size*3*3,'real production projection inspected the coordinates');
  assert.ok(counts.coordinates<=size*3*8,'coordinate reads stay bounded per pressure point');
  assert.ok(counts.properties<=size*16,'path property reads stay bounded per accepted path');
  return counts;
 }
 const p=paint(32),twice=paint(64);
 for(const key of Object.keys(p))assert.ok(twice[key]<=2*p[key]+4,`${key} grows at most linearly: ${p[key]} -> ${twice[key]}`);
});
