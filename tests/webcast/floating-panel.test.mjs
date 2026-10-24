import {test} from 'node:test';
import assert from 'node:assert/strict';
import {Element} from './dom-fixture.mjs';
import {floatingPanelTransition as transition,mountFloatingPanel} from '../../cmd/internal/webcast/web/floating-panel.js';

const init=()=>transition(null,{type:'init',viewport:{width:800,height:600},size:{width:300,height:320},left:488,top:110});
test('pure state shares the anchor, owns one pointer and restores canceled movement',()=>{
 let s=init();assert.equal(s?.mode,'expanded');
 s=transition(s,{type:'press',id:7,x:10,y:20});
 s=transition(s,{type:'move',id:8,x:0,y:0});assert.equal(s.left,488);
 s=transition(s,{type:'press',id:8,x:0,y:0});assert.equal(s.gesture.id,7);
 s=transition(s,{type:'move',id:7,x:-100,y:100});assert.equal(s.left,378);assert.equal(s.top,190);
 s=transition(s,{type:'cancel'});assert.equal(s.left,488);assert.equal(s.top,110);assert.equal(s.gesture,null);
 s=transition(s,{type:'mode',mode:'minimized',size:{width:90,height:32}});assert.equal(s.left,488);
 s=transition(s,{type:'mode',mode:'expanded',size:{width:300,height:320}});assert.equal(s.left,488);
});
test('generated transitions keep finite bounded coordinates and terminal disposal',()=>{
 let s=init(),seed=8347;const random=()=>((seed=(seed*1664525+1013904223)>>>0)/2**32);
 for(let i=0;i<3000;i++){
  const type=['press','move','end','cancel','mode','geometry','keyboard'][Math.floor(random()*7)];
  const e={type,id:Math.floor(random()*3),x:(random()-.5)*4000,y:(random()-.5)*4000,dx:random()*200-100,dy:random()*200-100,mode:random()<.5?'minimized':'expanded',size:{width:random()*600,height:random()*600},viewport:{width:random()*1000,height:random()*700}};
  const prev=s;s=transition(s,e);assert.notEqual(s,undefined);assert.ok(Number.isFinite(s.left)&&Number.isFinite(s.top));
  const mx=Math.min(12,s.viewport.width/2),my=Math.min(12,s.viewport.height/2);
  assert.ok(s.left>=mx&&s.left<=Math.max(mx,s.viewport.width-mx-s.size.width));
  assert.ok(s.top>=my&&s.top<=Math.max(my,s.viewport.height-my-s.size.height));
  assert.ok(prev.gesture===null||e.type!=='press'||s.gesture.id===prev.gesture.id);
 }
 s=transition(s,{type:'geometry',viewport:{width:NaN,height:Infinity},size:{width:-1,height:NaN}});assert.ok(Number.isFinite(s.left)&&Number.isFinite(s.top));
 s=transition(s,{type:'dispose'});assert.equal(s.gesture,null);assert.equal(s.disposed,true);
 assert.equal(transition(s,{type:'press',id:1,x:1,y:1}),s);
});

class Node extends Element {
 constructor(tag,cost){super(tag);this.cost=cost;this.style=new Proxy({}, {set:(o,k,v)=>{cost.styles++;o[k]=v;return true;}});}
 append(...nodes){for(const n of nodes)n.parentNode=this;super.append(...nodes);}
 remove(){if(this.parentNode)this.parentNode.children=this.parentNode.children.filter(n=>n!==this);}
 removeEventListener(k,f){this.listeners[k]=(this.listeners[k]||[]).filter(x=>x!==f);}
 getBoundingClientRect(){this.cost.geometry++;return this.className.includes('minimized')?{width:90,height:32}:{width:300,height:320};}
 setPointerCapture(id){super.setPointerCapture(id);this.cost.capture.push(['set',id]);}
 releasePointerCapture(id){super.releasePointerCapture(id);this.cost.capture.push(['release',id]);this.fire('lostpointercapture',{pointerId:id});}
 fire(type,extra={}){const e={type,target:this,pointerId:1,clientX:10,clientY:20,button:0,isPrimary:true,preventDefault(){this.prevented=true;},stopPropagation(){this.stopped=true;},...extra};for(let node=this;node;node=node.parentNode){for(const f of [...(node.listeners[type]||[])])f(e);if(e.stopped)break;}return e;}
}
function fixture(n=1){
 const cost={styles:0,geometry:0,capture:[]},doc={createElement:tag=>new Node(tag,cost)};
 const win=new Node('window',cost);win.innerWidth=800;win.innerHeight=600;
 const parent=doc.createElement('main'),body=doc.createElement('div');for(let i=0;i<n;i++)body.append(doc.createElement('button'));
 const panel=mountFloatingPanel(doc,win,{parent,body});assert.equal(typeof panel.setCount,'function');panel.setCount(n);
 return {...panel,win,cost,parent,header:panel.root.querySelector('.floating-panel-header'),handle:panel.root.querySelector('.floating-panel-drag'),minimize:panel.root.querySelector('.floating-panel-minimize'),restore:panel.root.querySelector('.floating-panel-restore')};
}
function drag(f,target=f.header){target.fire('pointerdown');f.root.fire('pointermove',{clientX:-100,clientY:100});}
test('adapter preserves body and local mode across count updates; ordinary rows remain untouched',()=>{
 const f=fixture(4);assert.equal(f.body.parentNode,f.root);assert.equal(f.root.hidden,false);
 const rowEvent=f.body.fire('pointerdown');assert.equal(rowEvent.prevented,undefined);assert.equal(f.root.captured,undefined);
 f.minimize.fire('click');assert.equal(f.root.classList.contains('minimized'),true);assert.equal(f.restore.textContent,'Notes (4)');
 f.setCount(8);assert.equal(f.restore.textContent,'Notes (8)');assert.equal(f.root.classList.contains('minimized'),true);
 f.setCount(0);assert.equal(f.root.hidden,true);f.setCount(2);assert.equal(f.root.classList.contains('minimized'),true);
 f.restore.fire('click');assert.equal(f.root.classList.contains('minimized'),false);assert.equal(f.body.hidden,false);
});
test('adapter captures one pointer, cancels on lost capture, and rejects a dragged restore click',()=>{
 const f=fixture();const original=f.root.style.inset.split(' ')[3];drag(f);assert.equal(f.header.captured,1);assert.notEqual(f.root.style.inset.split(' ')[3],original);
 f.root.fire('pointermove',{pointerId:2,clientX:200});assert.equal(f.header.captured,1);
 f.root.fire('lostpointercapture');assert.equal(f.root.style.inset.split(' ')[3],original);assert.ok(f.header.captured==null && f.restore.captured==null);
 f.minimize.fire('click');drag(f,f.restore);f.root.fire('pointerup');f.restore.fire('click');assert.equal(f.root.classList.contains('minimized'),true);
 f.restore.fire('pointerdown');f.root.fire('pointerup');f.restore.fire('click');assert.equal(f.root.classList.contains('minimized'),false);
});
test('tiny viewports, resize cancellation, keyboard movement and disposal own their resources',()=>{
 const f=fixture();drag(f);f.win.innerWidth=30;f.win.innerHeight=20;f.win.fire('resize');assert.ok(f.header.captured==null && f.restore.captured==null);assert.equal(f.root.style.inset.split(' ')[3],'12px');assert.equal(f.root.style.inset.split(' ')[0],'10px');
 f.win.innerWidth=800;f.win.innerHeight=600;f.win.fire('resize');const before=f.root.style.inset.split(' ')[3];
 const key=f.handle.fire('keydown',{key:'ArrowRight'});assert.equal(key.prevented,true);assert.notEqual(f.root.style.inset.split(' ')[3],before);
 drag(f);f.dispose();const cost={...f.cost};f.root.fire('pointermove');f.win.fire('resize');f.setCount(99);f.dispose();
 assert.deepEqual(f.cost,cost);assert.ok(f.header.captured==null && f.restore.captured==null);assert.equal(f.parent.children.length,0);
 for(const node of [f.root,f.win,f.header,f.handle,f.minimize,f.restore])assert.equal(Object.values(node.listeners).flat().length,0);
});
test('drag work and geometry measurements stay constant at N and 2N notes',()=>{
 function count(n){const f=fixture(n);const start={...f.cost};drag(f);for(let i=0;i<100;i++)f.root.fire('pointermove',{clientX:i,clientY:i});f.root.fire('pointerup');return {geometry:f.cost.geometry-start.geometry,styles:f.cost.styles-start.styles};}
 const a=count(100),b=count(200);assert.deepEqual(a,b);assert.equal(a.geometry,1);assert.ok(a.styles<=103);
});
test('restore owns its capture so native pointer clicks retain the restore target',()=>{
 const f=fixture();f.minimize.fire('click');f.restore.fire('pointerdown');
 assert.equal(f.restore.hasPointerCapture(1),true);
 f.root.fire('pointerup');assert.equal(f.restore.hasPointerCapture(1),false);
 f.restore.fire('click');assert.equal(f.root.classList.contains('minimized'),false);
});
test('header gestures stay local while body input and scrolling remain native',()=>{
 const f=fixture();const click=f.header.fire('click');assert.equal(click.stopped,true);
 const down=f.header.fire('pointerdown',{target:f.minimize});assert.equal(down.stopped,true);assert.equal(f.header.captured,undefined);
 for(const type of ['pointerdown','pointermove','pointerup','wheel','click']){
  const event=f.body.fire(type);assert.equal(event.prevented,undefined);assert.equal(event.stopped,undefined);
 }
});
