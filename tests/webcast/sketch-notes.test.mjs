import { test } from 'node:test';
import assert from 'node:assert/strict';
import { Element } from './dom-fixture.mjs';
const load = () => import('../../cmd/internal/webcast/web/sketch-notes.js');
const path = (sketch) => ({sketch, pts:[[20,30,0.5]], at:{box:[0,0,100,100]}});
test('entries are unique placed sketches with an existing final label or owned unfinished capability', async()=>{
  const {sketchNoteEntries}=await load();
  const meta=new Map([['a',{phase:'drawing'}],['b',{phase:'complete',label:'<b>final</b>'}],['c',{phase:'drawing'}]]);
  const entries=sketchNoteEntries([path('a'),path('a'),path('b'),path('c')],[[0,0,100,100],[0,0,100,100],[100,100,100,100],null],{metadata:k=>meta.get(k),editable:new Set(['a'])});
  assert.deepEqual(entries,[{sketch:'a',x:20,y:30,label:'',editable:true},{sketch:'b',x:120,y:130,label:'<b>final</b>',editable:false}]);
  assert.deepEqual(sketchNoteEntries([path('a')],[[NaN,0,1,1]],{metadata:k=>meta.get(k),editable:new Set(['a'])}),[]);
});
test('projection and membership visits grow linearly for P and 2P unique sketches', async()=>{
  const {sketchNoteEntries}=await load(), costs=[];
  for(const n of [128,256]) {
    const cost={paths:0,identity:0,points:0,metadata:0,projection:0};
    const history=Array.from({length:1024},()=>[20,30,.5]);
    const raw=Array.from({length:n},(_,i)=>new Proxy({...path(String(i)),pts:history},{
      get(target,key){
        if(key==='sketch')cost.identity++;
        if(key==='pts') {
          cost.points++;
          // The painter already visited every point. This projection may
          // inspect only its first point, regardless of history length.
          return new Proxy(target.pts,{get(points,k){
            if(/^\d+$/.test(String(k)))assert.equal(k,'0','no repeated history walk');
            return Reflect.get(points,k);
          }});
        }
        return Reflect.get(target,key);
      },
    }));
    const paths=new Proxy(raw,{get(target,key){if(/^\d+$/.test(String(key)))cost.paths++;return Reflect.get(target,key);}});
    const entries=sketchNoteEntries(paths,raw.map(()=>[0,0,100,100]),{
      metadata:()=>{cost.metadata++;return {phase:'complete',label:'done'};},editable:new Set(),
      toClient:(x,y)=>{cost.projection++;return [x,y];},
    });
    assert.equal(entries.length,n);
    assert.equal(cost.metadata,n);assert.equal(cost.projection,n);assert.equal(cost.points,n);
    assert.ok(cost.paths<=2*n,'grouping cannot repeatedly scan paths');
    assert.ok(cost.identity<=12*n,'grouping cannot repeatedly compare every identity');
    costs.push(cost);
  }
  for(const key of Object.keys(costs[0]))assert.ok(costs[1][key]<=2*costs[0][key],`${key} must scale linearly`);
});

test('generated malformed placement and metadata retain only finite positive unique identities',async()=>{
  const {sketchNoteEntries}=await load();
  const boxes=[null,[],[1,2,3],[0,0,0,1],[0,0,1,-1],[NaN,0,1,1],[0,Infinity,1,1],[0,0,100,100],[40,20,20,10],{box:[0,0,100,100]}];
  const points=[[[20,30,.5]],[],[[NaN,1,.5]],[[1,2,2]],[[1,2]],[[Infinity,1,.5]]];
  const infos=[null,{}, {phase:'complete',label:'final'}, {phase:'drawing',label:9}, {phase:'interrupted'}, {phase:'cleared',label:''}];
  const validBox=b=>Array.isArray(b)&&b.length===4&&b.every(Number.isFinite)&&b[2]>0&&b[3]>0;
  const validPoint=p=>Array.isArray(p)&&[2,3].includes(p.length)&&p.every(Number.isFinite)&&(p.length===2||(p[2]>=0&&p[2]<=1));
  let seed=850;const next=n=>{seed=(1664525*seed+1013904223)>>>0;return seed%n;};
  for(let round=0;round<12;round++) {
    const paths=[],targets=[],meta=new Map(),editable=new Set(),expected=new Map();
    for(let i=0;i<80;i++) {
      const identity=i%13===0?null:`s${Math.floor(i/2)}`;
      const p={sketch:identity,at:{box:boxes[next(boxes.length)]},pts:points[next(points.length)]};
      paths.push(p);targets.push(boxes[next(boxes.length)]);
      if(typeof identity==='string'){meta.set(identity,infos[next(infos.length)]);if(next(2))editable.add(identity);}
    }
    for(let i=0;i<paths.length;i++) {
      const p=paths[i],box=targets[i]?.box||targets[i],point=p.pts[0],info=meta.get(p.sketch);
      if(typeof p.sketch!=='string'||expected.has(p.sketch)||!validBox(box)||!validBox(p.at.box)||!validPoint(point))continue;
      const label=typeof info?.label==='string'?info.label:'';
      const add=!label&&editable.has(p.sketch)&&['drawing','interrupted','label-required'].includes(info?.phase);
      if(!label&&!add)continue;
      const x=box[0]+(point[0]-p.at.box[0])*box[2]/p.at.box[2];
      const y=box[1]+(point[1]-p.at.box[1])*box[3]/p.at.box[3];
      if(!Number.isFinite(x)||!Number.isFinite(y))continue;
      expected.set(p.sketch,{sketch:p.sketch,x,y,label,editable:add});
    }
    assert.deepEqual(sketchNoteEntries(paths,targets,{metadata:k=>meta.get(k),editable}),[...expected.values()],`seeded round ${round}`);
  }
});

function fixture(){
  class E extends Element {appendChild(e){this.children.push(e);e.parentNode=this;return e;} remove(){if(this.parentNode)this.parentNode.children=this.parentNode.children.filter(x=>x!==this);}}
  const root=new E('root'),doc={createElement:t=>new E(t)};
  const click=e=>e.listeners.click[0]({stopPropagation(){},preventDefault(){}});
  return {root,doc,click};
}
test('keyed controls read current identity and final inspection sends no command',async()=>{
  const {mountSketchNoteControls}=await load();const h=fixture(),sent=[];
  const ui=mountSketchNoteControls(h.doc,{parent:h.root,onAdd:k=>sent.push(k)});
  ui.update([{sketch:'a',x:1,y:2,label:'',editable:true}]);
  const b=h.root.querySelector('.sketch-note-control');h.click(b);assert.deepEqual(sent,['a']);
  ui.update([{sketch:'a',x:4,y:5,label:'<script>text</script>',editable:false}]);
  assert.equal(h.root.querySelector('.sketch-note-control'),b);assert.equal(b.textContent,'View note');assert.equal(b.style.top,'13px','top-edge control sits below ink');h.click(b);
  assert.equal(h.root.querySelector('.sketch-note-text').textContent,'<script>text</script>');assert.deepEqual(sent,['a']);
  h.click(h.root.querySelector('.sketch-note-close'));assert.equal(h.root.querySelector('.sketch-note-view').hidden,true);
  ui.update([{sketch:'a',x:4,y:140,label:'final',editable:false}]);assert.equal(b.style.top,'88px','44px control leaves an8px gap above ink');
  ui.update([]);h.click(b);assert.deepEqual(sent,['a']);ui.dispose();assert.equal(h.root.children.length,0);
});

test('unowned unfinished sketches never acquire a label action and malformed geometry is skipped',async()=>{
  const {sketchNoteEntries}=await load();
  const paths=[path('other'),path('bad'),path('mine'),path('mine')];
  assert.deepEqual(sketchNoteEntries(paths,[[0,0,10,10],[0,0,-1,10],null,[0,0,100,100]],{
    metadata:()=>({phase:'interrupted'}),editable:new Set(['mine','bad']),
  }),[{sketch:'mine',x:20,y:30,label:'',editable:true}]);
});
test('revoking a keyed Add note action makes its old click inert',async()=>{
  const {mountSketchNoteControls}=await load(),h=fixture(),sent=[];
  const ui=mountSketchNoteControls(h.doc,{parent:h.root,onAdd:k=>sent.push(k)});
  ui.update([{sketch:'a',x:1,y:2,label:'',editable:true}]);const b=h.root.querySelector('.sketch-note-control');
  ui.update([{sketch:'a',x:1,y:2,label:'',editable:false}]);h.click(b);assert.deepEqual(sent,[]);ui.dispose();h.click(b);assert.deepEqual(sent,[]);
});
