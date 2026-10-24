import {test} from 'node:test';
import assert from 'node:assert/strict';
import {mountFeedbackGlow} from '../../cmd/internal/webcast/web/feedback-glow.js';
test('hover changes pixels only and owns resize/teardown',()=>{
 let writes=0,clears=0,draws=0,removed=0;const listeners=new Map();
 const g={setTransform(){},clearRect(){clears++},save(){},restore(){}};
 const canvas={style:{},setAttribute(){writes++},getContext:()=>g,remove(){removed++}};
 for(const key of ['width','height'])Object.defineProperty(canvas,key,{get(){return this['_'+key]},set(v){writes++;this['_'+key]=v}});
 const doc={createElement:()=>canvas},win={devicePixelRatio:2,addEventListener(k,f){listeners.set(k,f)},removeEventListener(k){listeners.delete(k)}};
 const parent={append(){}},box={getBoundingClientRect:()=>({left:4,top:8,width:300,height:200})};
 const glow=mountFeedbackGlow(doc,win,{parent,box});const initial=writes;
 assert.equal(glow.draw((ctx,r)=>{assert.equal(ctx,g);assert.equal(r.left,4);draws++}),true);
 glow.clear();glow.draw(()=>draws++);assert.equal(writes,initial);assert.equal(draws,2);assert.ok(clears>=3);
 listeners.get('resize')();glow.dispose();glow.dispose();assert.equal(removed,1);assert.equal(listeners.size,0);
 assert.equal(glow.draw(()=>assert.fail('disposed paint')),false);
});
test('missing canvas context and collapsed viewport are unavailable, never inferred',()=>{
 const canvas={style:{},getContext:()=>null,remove(){}};
 const win={addEventListener(){},removeEventListener(){}},doc={createElement:()=>canvas};
 const glow=mountFeedbackGlow(doc,win,{parent:{append(){}},box:{getBoundingClientRect:()=>({width:0,height:0,left:0,top:0})}});
 assert.equal(glow.draw(()=>assert.fail()),false);glow.dispose();
});
