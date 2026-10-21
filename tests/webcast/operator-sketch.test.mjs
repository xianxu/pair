import { test } from 'node:test';
import assert from 'node:assert/strict';
import { createOperatorSketchTransport, createOperatorSketchReceiver } from '../../cmd/internal/webcast/operator-sketch.js';

test('console sketch transport retains original permission on unknown retry and joins close',async()=>{
 const sent=[],timers=new Map();let serial=0,generation='1';
 const win={setTimeout(fn){timers.set(++serial,fn);return serial;},clearTimeout(id){timers.delete(id);}};
 const t=createOperatorSketchTransport(m=>sent.push(m),win,()=>generation);
 const body={operation:'begin',sequence:0,epoch:1,seg:1,color:0};
 const first=t.ask('sketch-begin',body);
 timers.values().next().value();
 await assert.rejects(first,/confirmation/);
 generation='3';
 const retry=t.ask('sketch-status',{operation:'begin',sequence:0,epoch:1});
 assert.equal(sent[1].generation,'1');
 t.reply({request:sent[1].request,ok:true,value:{outcome:'accepted',sketch:'server'}});
 assert.equal((await retry).sketch,'server');
 const pending=t.ask('scene-hit',{point:[1,2]},1);
 t.close();await assert.rejects(pending,/retired/);
 assert.equal(t.reply({request:sent.at(-1).request,ok:true,value:{}}),false);
});

test('console delivery ACK follows complete validated transfer and gap requests bootstrap',()=>{
 const receiver=createOperatorSketchReceiver(()=>{});
 const value={bootstrap:{epoch:'cast',seq:0,display:{sketches:[]}}};
 const raw=new TextEncoder().encode(JSON.stringify(value));
 const start=receiver.receive({phase:'begin',handle:'h',epoch:'cast',total:raw.length});
 assert.equal(start.offset,0);
 const ack=receiver.receive({phase:'chunk',handle:'h',epoch:'cast',total:raw.length,offset:0,base64:Buffer.from(raw).toString('base64')});
 assert.equal(ack.epoch,'cast');assert.equal(ack.seq,0);assert.equal(ack.needBootstrap,false);
 assert.throws(()=>receiver.receive({phase:'chunk',handle:'other',epoch:'cast',total:raw.length,offset:0,base64:Buffer.from(raw).toString('base64')}));
 receiver.retire();assert.equal(receiver.cursor().epoch,null);
});

test('late console scene preparation cannot restore a disposed source',async()=>{
 const {mountOperatorSketch}=await import('../../cmd/internal/webcast/operator-sketch.js');
 const {Element}=await import('./dom-fixture.mjs');
 class E extends Element{
  appendChild(n){this.append(n);n.parentNode=this;return n;}
  insertBefore(n){this.appendChild(n);queueMicrotask(()=>n.onload?.());return n;}
  remove(){if(this.parentNode)this.parentNode.children=this.parentNode.children.filter(x=>x!==this);}
 }
 const root=new E('root'),box=new E('box');root.appendChild(box);
 const doc={body:root,createElement:tag=>new E(tag)};
 let serial=0;const sent=[];
 const win={crypto:{randomUUID:()=>String(++serial)},requestAnimationFrame:()=>1,cancelAnimationFrame(){},setTimeout:()=>++serial,clearTimeout(){},addEventListener(){}};
 const view={model:{seg:1,docSeg:1,paused:false,ended:false},onRecord:()=>()=>{}};
 const adapter=mountOperatorSketch({host:new E('host'),root,box,report:message=>{sent.push(message.request);if(message.request.kind==='scene-open')queueMicrotask(()=>adapter.transport.reply({request:message.request.request,ok:true,value:{}}));},view,annotation:{setMode(){}},palette:['#ff0000'],css:'',shellURL:'data:text/html,private'},doc,win);
 win.__couchcastSketchAuthority({revision:'1',epoch:1,holder:'operator',ready:true,active:true,generation:'1'});
 await adapter.scene.preflight();
 const pending=adapter.scene.prepare({handle:'source'},{sketch:'sk',epoch:1});
 const request=sent.at(-1).request;
 adapter.scene.dispose();
 adapter.transport.reply({request,ok:true,value:{ready:true,vp:{w:100,h:100},scroll:[0,0]}});
 await assert.rejects(pending,/retired/);
 assert.equal(adapter.scene.ready,false);
 adapter.dispose();
});

async function lifecycleFixture(){
 const {mountOperatorSketch}=await import('../../cmd/internal/webcast/operator-sketch.js');
 const {Element}=await import('./dom-fixture.mjs');
 class E extends Element{
  appendChild(n){this.append(n);n.parentNode=this;return n;}
  insertBefore(n){return this.appendChild(n);}
  remove(){if(this.parentNode)this.parentNode.children=this.parentNode.children.filter(x=>x!==this);}
 }
 const root=new E('root'),box=new E('box');root.appendChild(box);
 const timers=new Map(),listeners=new Map(),sent=[];let serial=0;
 const win={crypto:{randomUUID:()=>String(++serial)},requestAnimationFrame:()=>1,cancelAnimationFrame(){},
  setTimeout(fn){const id=++serial;timers.set(id,fn);return id;},clearTimeout(id){timers.delete(id);},
  addEventListener(k,fn){listeners.set(k,fn);},removeEventListener(k,fn){if(listeners.get(k)===fn)listeners.delete(k);}};
 const previous=()=>true;
 const view={model:{seg:1,docSeg:1,paused:false,ended:false},onRecord:()=>()=>{},beforeAnnotationMode:previous};
 const adapter=mountOperatorSketch({host:new E('host'),root,box,report:m=>sent.push(m.request),view,annotation:{setMode(){}},palette:['#ff0000'],css:'',shellURL:'data:text/html;couch-sketch=1,private'},{body:root,createElement:tag=>new E(tag)},win);
 win.__couchcastSketchAuthority({revision:'1',epoch:1,holder:'operator',ready:true,active:true,generation:'1'});
 return {adapter,win,view,timers,listeners,sent,previous};
}

test('preflight timeout removes failed frame and permits a fresh attempt',async()=>{
 const f=await lifecycleFixture();
 const first=f.adapter.scene.preflight(),old=f.win.__couchcastPrivateSceneFrame;
 f.timers.values().next().value();
 await assert.rejects(first,/load/);
 assert.equal(f.win.__couchcastPrivateSceneFrame,undefined);
 const second=f.adapter.scene.preflight();const canceled=assert.rejects(second,/retired/);
 assert.notEqual(f.win.__couchcastPrivateSceneFrame,old);
 f.adapter.dispose();await canceled;
});

test('late old preflight cannot open or clear a replacement scene',async()=>{
 const f=await lifecycleFixture();
 const first=f.adapter.scene.preflight(),old=f.win.__couchcastPrivateSceneFrame;
 const retired=assert.rejects(first,/retired/);
 f.adapter.scene.dispose();
 const second=f.adapter.scene.preflight();const canceled=assert.rejects(second,/retired/);
 const replacement=f.win.__couchcastPrivateSceneFrame;
 old.onload?.();await Promise.resolve();await Promise.resolve();
 assert.equal(f.sent.filter(x=>x.kind==='scene-open').length,0);
 assert.equal(f.win.__couchcastPrivateSceneFrame,replacement);
 await retired;f.adapter.dispose();await canceled;
});

test('disposal removes only owned hooks and permanently fences receiver delivery',async()=>{
 const f=await lifecycleFixture(),oldDelivery=f.win.__couchcastSketchDelivery,oldAuthority=f.win.__couchcastSketchAuthority;
 const replacement=()=>{};f.win.__couchcastSketchReply=replacement;
 f.adapter.dispose();
 assert.equal(f.win.__couchcastSketchReply,replacement);
 assert.equal(f.win.__couchcastSketchDelivery,undefined);
 assert.equal(f.win.__couchcastSketchAuthority,undefined);
 assert.equal(f.view.beforeAnnotationMode,f.previous);
 assert.equal(f.listeners.size,0);
 assert.throws(()=>oldDelivery({phase:'begin',handle:'h',epoch:'cast',total:1}),/retired/);
 assert.equal(oldAuthority({epoch:2,holder:'operator',ready:true,generation:'2'}).ready,false);
});

test('old scene-open refusal preserves a replacement frame and shared preflight joins once',async()=>{
 const f=await lifecycleFixture();
 const first=f.adapter.scene.preflight();assert.equal(f.adapter.scene.preflight(),first);
 const old=f.win.__couchcastPrivateSceneFrame;old.onload();
 await Promise.resolve();await Promise.resolve();
 const request=f.sent.find(x=>x.kind==='scene-open');assert.ok(request);
 const retired=assert.rejects(first,/retired/);f.adapter.scene.dispose();
 const second=f.adapter.scene.preflight(),replacement=f.win.__couchcastPrivateSceneFrame;
 const canceled=assert.rejects(second,/retired/);
 f.adapter.transport.reply({request:request.request,ok:false,error:'old scene refused'});
 await retired;assert.equal(f.win.__couchcastPrivateSceneFrame,replacement);
 f.adapter.dispose();await canceled;
});

test('placement sends bounded anchor and source without a long strokes points',async()=>{
 const f=await lifecycleFixture();
 const path={source:{doc:1,tab:'tab',t:4},at:{node:8,box:[1,2,3,4]},pts:Array.from({length:4096},()=>[9999999.25,9999999.25,0.5555555])};
 const pending=f.adapter.scene.placement(path),request=f.sent.at(-1);
 assert.deepEqual(request.body,{path:{source:path.source,at:path.at}});
 assert.ok(JSON.stringify(request).length<1024);
 f.adapter.transport.reply({request:request.request,ok:true,value:[1,2,3,4]});await pending;f.adapter.dispose();
});

test('receiver repeats final ACK and requests bootstrap for a real batch gap',()=>{
 const receiver=createOperatorSketchReceiver(()=>{});
 function transfer(handle,payload){const body=Buffer.from(JSON.stringify(payload));const begin={phase:'begin',handle,epoch:'cast',total:body.length};receiver.receive(begin);const chunk={...begin,phase:'chunk',offset:0,base64:body.toString('base64')};const ack=receiver.receive(chunk);assert.deepEqual(receiver.receive(chunk),ack);return ack;}
 transfer('bootstrap',{bootstrap:{epoch:'cast',seq:0,display:{sketches:[]}}});
 const gap=transfer('gap',{batch:{epoch:'cast',from:2,to:2,patches:[{}]}});
 assert.equal(gap.needBootstrap,true);assert.equal(gap.seq,0);
 receiver.retire();
});

test('authority revision fences stale enable and duplicate updates without resetting UI',async()=>{
 const f=await lifecycleFixture();let updates=0;
 const original=f.adapter.tool.setAuthority;f.adapter.tool.setAuthority=(...args)=>{updates++;return original(...args);};
 const hook=f.win.__couchcastSketchAuthority;
 assert.deepEqual(hook({revision:'2',generation:'2',epoch:3,holder:'operator',ready:true,active:true}),{revision:'2',generation:'2',epoch:3,ready:true});
 const disabled={revision:'3',generation:'3',epoch:4,holder:'operator',ready:false,active:true};
 const receipt=hook(disabled);
 assert.deepEqual(receipt,{revision:'3',generation:'3',epoch:4,ready:false});
 assert.equal(f.adapter.tool.state().allowed,false);
 assert.deepEqual(hook({revision:'2',generation:'2',epoch:3,holder:'operator',ready:true,active:true}),receipt);
 assert.deepEqual(hook({...disabled,generation:'99',ready:true}),receipt);
 assert.equal(updates,2);assert.equal(f.adapter.tool.state().allowed,false);
 assert.deepEqual(hook({revision:'4',generation:'4',epoch:5,holder:'operator',ready:true,active:false}),{revision:'4',generation:'4',epoch:5,ready:false});
 assert.equal(f.adapter.tool.state().allowed,false);
 f.adapter.dispose();
});

test('authority validates canonical uint64 strings and safe epoch before advancing revision',async()=>{
 const f=await lifecycleFixture(),hook=f.win.__couchcastSketchAuthority;
 const valid={revision:'2',generation:'2',epoch:2,holder:'operator',ready:true,active:true};
 for(const patch of [{revision:'02'},{revision:'18446744073709551616'},{generation:'02'},{generation:'18446744073709551616'},{epoch:9007199254740992},{epoch:-1},{ready:'true'},{active:'true'}])
  assert.throws(()=>hook({...valid,...patch}),/authority/);
 assert.deepEqual(hook(valid),{revision:'2',generation:'2',epoch:2,ready:true});
 f.adapter.dispose();
});
