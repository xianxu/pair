import { test } from 'node:test';
import assert from 'node:assert/strict';
import { preparePageInk } from '../../cmd/internal/webcast/web/sketch-page-image.js';
const plan = {scene:{doc:1,tab:'tab',t:50,scroll:[0,0]},region:[0,0,300,200],scale:1,width:300,height:200};
const shape = (index=1,x=40) => ({index,id:`s${index}`,color:'#ff0000',source:{doc:1,tab:'tab',t:10},paths:[{id:'p',at:{node:1,box:[0,0,100,100]},width:8,pts:[[x,40,0],[x+30,40,1]]}]});
const place = () => ({box:[20,10,100,100],clip:[0,0,300,200]});
test('all chosen paths are preflighted and affine pressure remains original',()=>{
 const input=[shape(1),shape(2,170)];const original=structuredClone(input);
 const out=preparePageInk(plan,input,place);
 assert.equal(out.shapes.length,2);assert.deepEqual(out.shapes.map(s=>s.index),[1,2]);
 assert.deepEqual(out.shapes[0].paths[0].projected[0],[60,50,0]);
 assert.deepEqual(out.shapes[0].paths[0].projected[1],[90,50,1]);
 assert.deepEqual(input,original);assert.equal(out.clipped,false);
});
test('missing, ambiguous, hidden and out-of-view paths fail the entire selected image',()=>{
 for(const target of [null,{box:[400,0,100,100],clip:[0,0,300,200]},{box:[0,0,100,100],clip:[0,0,5,5]}])
  assert.throws(()=>preparePageInk(plan,[shape()],()=>target),/index 1/);
 const invalid=shape();invalid.paths.push({...invalid.paths[0],id:'hidden'});
 assert.throws(()=>preparePageInk(plan,[invalid],p=>p.id==='hidden'?null:place()),/hidden/);
});
test('ordinary pressure fringe clipping is declared while centerline offscreen is refused',()=>{
 const edge=shape();edge.paths[0].pts=[[0,40,1],[20,40,1]];
 const out=preparePageInk(plan,[edge],()=>({box:[0,0,100,100],clip:[0,0,300,200]}));assert.equal(out.clipped,true);
 edge.paths[0].pts[0][0]=-1;assert.throws(()=>preparePageInk(plan,[edge],()=>({box:[0,0,100,100],clip:[0,0,300,200]})),/outside/);
});
test('badges are bounded, ordered and never silently omitted or packed without limit',()=>{
 const out=preparePageInk(plan,[shape(1)],place);const b=out.shapes[0].badge;
 assert.ok(b.x>=0&&b.y>=0&&b.x+b.w<=plan.width&&b.y+b.h<=plan.height);
 assert.throws(()=>preparePageInk({...plan,width:10},[shape()],place),/badge/);
 assert.throws(()=>preparePageInk({...plan,width:20,height:20},[shape(1),shape(2)],place),/index 2.*badge/);
 assert.throws(()=>preparePageInk(plan,[shape(2)],place),/index/);
});

test('shared starting points choose deterministic nearby distinct readable badges',()=>{
 const input=Array.from({length:8},(_,i)=>shape(i+1));
 const badges=preparePageInk(plan,input,place).shapes.map(s=>s.badge);
 assert.deepEqual(preparePageInk(plan,input,place).shapes.map(s=>s.badge),badges);
 assert.equal(new Set(badges.map(b=>`${b.x},${b.y}`)).size,8);
 for(let i=0;i<badges.length;i++){
  const a=badges[i];assert.equal(a.w,20);assert.equal(a.h,20);
  assert.ok(a.x>=0&&a.y>=0&&a.x+a.w<=plan.width&&a.y+a.h<=plan.height);
  for(const b of badges.slice(0,i))assert.ok(a.x>=b.x+b.w||b.x>=a.x+a.w||a.y>=b.y+b.h||b.y>=a.y+a.h);
 }
});
