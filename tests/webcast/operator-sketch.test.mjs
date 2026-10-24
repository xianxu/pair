import { test } from 'node:test';
import assert from 'node:assert/strict';
import { createOperatorSketchTransport, createOperatorSketchReceiver } from '../../cmd/internal/webcast/operator-sketch.js';

test('live receiver gates old and delayed bootstraps by confirmed scope and delivery epoch',()=>{
 const a={target:'A',loader:'L',root:'9007199254740993'},b={...a,target:'B'};
 const model={page_load_ready:true,page_load:a,sketch_cursor:{epoch:'A1',seq:0}};
 const receiver=createOperatorSketchReceiver(()=>{},()=>model);let serial=0;
 const install=(epoch,id,doc)=>{
  const value={bootstrap:{epoch,seq:0,display:{sketches:[{begin:{id,author:{id:'operator',color:'#ff0000'},doc,tab:'tab'},phase:'complete',label:id,reason:'',strokes:[{begin:{id:'p',at:{node:7,box:[0,0,100,100]},width:3},pts:[[10,10,.5]],visible:true,sealed:true,truncated:false}]}]}}};
  const bytes=new TextEncoder().encode(JSON.stringify(value)),handle='scope-'+(++serial);
  receiver.receive({phase:'begin',handle,epoch,total:bytes.length});
  receiver.receive({phase:'chunk',handle,epoch,total:bytes.length,offset:0,base64:Buffer.from(bytes).toString('base64')});
 };
 install('A1','saved-A',1);assert.equal(receiver.paths()[0].sketch,'saved-A');assert.equal(receiver.hasCurrentAnchor(7),true);
 model.page_load_ready=false;model.page_load=null;model.sketch_cursor.epoch='pending';
 assert.deepEqual(receiver.paths(),[]);assert.equal(receiver.hasAnchor(1,7),false);assert.equal(receiver.hasCurrentAnchor(7),false);
 model.page_load_ready=true;model.page_load=b;model.sketch_cursor.epoch='B1';
 assert.deepEqual(receiver.paths(),[]);assert.equal(receiver.hasCurrentAnchor(7),false);install('A1','delayed-A',1);assert.deepEqual(receiver.paths(),[]);
 install('B1','saved-B',3);assert.equal(receiver.paths()[0].sketch,'saved-B');
 model.page_load=a;model.sketch_cursor.epoch='A2';assert.deepEqual(receiver.paths(),[]);
 install('A2','saved-A',1);assert.equal(receiver.paths()[0].source.doc,1);
 model.paused=true;assert.equal(receiver.paths()[0].sketch,'saved-A');assert.equal(receiver.hasCurrentAnchor(7),true);receiver.retire();
});

test('console sketch transport retains original permission on unknown retry and joins close',async()=>{
 const sent=[],timers=new Map();let serial=0,generation='1';
 const win={setTimeout(fn,ms){timers.set(++serial,{fn,ms});return serial;},clearTimeout(id){timers.delete(id);}};
 const t=createOperatorSketchTransport(m=>sent.push(m),win,()=>generation);
 const body={operation:'begin',sequence:0,epoch:1,seg:1,color:0};
 const first=t.ask('sketch-begin',body);
 timers.values().next().value.fn();
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
 return {adapter,win,view,root,timers,listeners,sent,previous};
}

test('console disabled Sketch explains readiness and distinguishes driving holder',async()=>{
 const f=await lifecycleFixture();
 const toolbar=f.root.querySelector('.sketch-toolbar');
 const button=name=>toolbar.children.find(n=>n.textContent===name);
 f.win.__couchcastSketchAuthority({revision:'2',epoch:1,holder:'operator',ready:false,active:true,generation:'1'});
 assert.equal(toolbar.hidden,false);
 assert.equal(button('Sketch').disabled,true);
 assert.equal(button('Done').disabled,true);
 assert.match(f.root.querySelector('.feedback-status').textContent,/getting ready.*Wait/);
 f.win.__couchcastSketchAuthority({revision:'3',epoch:1,holder:'driver',ready:true,active:true,generation:'1'});
 assert.match(f.root.querySelector('.feedback-status').textContent,/viewer.*driving.*release/i);
 f.win.__couchcastSketchAuthority({revision:'4',epoch:1,holder:'operator',ready:true,active:true,generation:'1'});
 assert.equal(button('Sketch').disabled,false);
 assert.equal(button('Done').disabled,true);
 assert.equal(f.root.querySelector('.feedback-status').textContent,'');
 f.adapter.dispose();
});

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

// Real shared controller/receiver over a stateful recording reducer. Its view
// deliberately has only the console's synthetic fields, not Model APIs.
async function drawingConsole({empty=false,deferredAnchor=false,startContact=true}={}){
 const {mountOperatorSketch}=await import('../../cmd/internal/webcast/operator-sketch.js');
 const {Model}=await import('../../cmd/internal/webcast/web/model.js');
 const {Element}=await import('./dom-fixture.mjs');
 class E extends Element{appendChild(n){this.append(n);n.parentNode=this;return n;}insertBefore(n){this.appendChild(n);queueMicrotask(()=>n.onload?.());return n;}remove(){if(this.parentNode)this.parentNode.children=this.parentNode.children.filter(v=>v!==this);}}
 const root=new E('root'),box=new E('box');root.appendChild(box);
 const model=new Model();let time=0,serial=0,seq=0,begins=0,revision=0,pendingPoint=null,statusOverride=null,pendingAnchor=null,hitTests=0,replyFilter=null,clock=0;
 for(const r of [{k:'head',format:'couch-webcast',v:1,id:'console',started:'now'},{k:'seg',seg:1,why:'live',tab:{id:'tab',url:'u'},vp:{w:100,h:100,dpr:1}},{k:'doc',doc:1,url:'u',tree:{id:1,y:'doc',c:[{id:7,y:'el',n:'body'}]}}])model.apply({...r,t:time++});
 const subscribers=new Set(),sent=[],modes=[],timers=new Map();
 const view={model:{seg:1,docSeg:1,paused:false,live:true,ended:false},onRecord(fn){subscribers.add(fn);return()=>subscribers.delete(fn);},fromPage:(x,y)=>[x,y],placementOf:p=>p.at.box};
 const win={crypto:{randomUUID:()=>`op-${++serial}`},requestAnimationFrame:()=>1,setTimeout(fn,ms){timers.set(++serial,{fn,ms,at:clock+ms});return serial;},clearTimeout(id){timers.delete(id);},listeners:{},addEventListener(k,fn){(this.listeners[k]||=[]).push(fn);},removeEventListener(){}};
 let adapter;
 function transfer(payload){const bytes=Buffer.from(JSON.stringify(payload)),message={handle:`delivery-${++serial}`,epoch:'cast',total:bytes.length};adapter.receiver.receive({...message,phase:'begin'});return adapter.receiver.receive({...message,phase:'chunk',offset:0,base64:bytes.toString('base64')});}
 function facts(record,beforeOpen=''){
  if(record.k==='annotation-clear'){transfer({batch:{epoch:'cast',from:++seq,to:seq,patches:[{clear:true}]}});return;}
  const sketch=record.k==='sketch-begin'?record.id:record.sketch||'s1';
  const info=model.sketchMetadata(sketch,record.id&&record.k!=='sketch-begin'?record.id:beforeOpen);
  if(!info.begin)return;
  const patch={sketch,header:{...(record.k==='sketch-begin'?{begin:info.begin}:{}),phase:info.phase,reason:info.reason,label:info.label}};
  if(info.stroke){const p=info.stroke;patch.stroke={...(record.k==='sketch-stroke-begin'?{begin:p.begin}:{id:p.begin.id}),sealed:p.sealed,truncated:p.truncated,visible:p.visible};if(record.k==='sketch-points')patch.points={id:p.begin.id,offset:p.length-record.pts.length,pts:record.pts};}
  transfer({batch:{epoch:'cast',from:++seq,to:seq,patches:[patch]}});
 }
 function apply(record){const open=model.sketchMetadata(record.sketch||'s1').open_stroke;model.apply({...record,t:time++});facts(record,open);}
 const status=sketch=>{const s=model.sketchMetadata(sketch);return {phase:s.phase,reason:s.reason,label:s.label,open_stroke:s.open_stroke||'',visible_top:s.visible_top||'',redo_top:s.redo_top||''};};
 function answer(request,value){if(replyFilter?.(request,value)===false)return;adapter.transport.reply({request:request.request,ok:true,value});}
 function mutation(request){const b=request.body,k=request.kind;sent.push(structuredClone(request));
  if(k==='sketch-status'){const result={outcome:'accepted',operation:b.operation,sketch:b.sketch||'s1',sequence:b.sequence,next_sequence:model.sketchMetadata(b.sketch||'s1').next_sequence,status:status(b.sketch||'s1')};const value=statusOverride?statusOverride(result,value=>answer(request,value)):result;if(value!==undefined)answer(request,value);return;}
  if(k==='sketch-points'&&empty){pendingPoint=request;return;}
  const sketch=k==='sketch-begin'?`s${++begins}`:b.sketch;
  const record={k,operation:b.operation,sequence:b.sequence,digest:'a'.repeat(64)};
  if(k==='sketch-begin')Object.assign(record,{id:sketch,author:{id:'operator',color:'#ff0000'},owner:'operator',doc:1,tab:'tab',vp:{w:100,h:100,dpr:1}});
  else{record.sketch=sketch;if(b.id)record.id=b.id;if(k==='sketch-stroke-begin')Object.assign(record,{at:{node:b.node,box:b.box},width:3});if(k==='sketch-points')Object.assign(record,{chunk:b.chunk,pts:b.pts});if(k==='sketch-stroke-end'&&b.truncated)record.truncated=true;if(k==='sketch-label')record.label=b.label;}
  apply(record);answer(request,{outcome:'accepted',operation:b.operation,sketch,sequence:b.sequence,next_sequence:model.sketchMetadata(sketch).next_sequence,status:status(sketch),...(k==='sketch-begin'?{source:{handle:'h',time:time-1,length:100,doc:1,tab:'tab',vp:{w:100,h:100,dpr:1}}}:{})});
 }
 adapter=mountOperatorSketch({host:new E('host'),root,box,view,annotation:{setMode(mode){modes.push(mode);}},palette:['#ff0000'],css:'',shellURL:'data:text/html,private',report:({request})=>queueMicrotask(()=>{try{if(request.kind==='scene-open'||request.kind==='scene-dispose')answer(request,{});else if(request.kind==='scene-prepare')answer(request,{ready:true,vp:{w:100,h:100,dpr:1},scroll:[0,0]});else if(request.kind==='scene-hit'){hitTests++;if(deferredAnchor){pendingAnchor=()=>answer(request,{node:7,box:[0,0,100,100]});pendingAnchor.reject=()=>adapter.transport.reply({request:request.request,ok:false,error:'Retired anchor.'});}else answer(request,{node:7,box:[0,0,100,100]});}else if(request.kind==='scene-placement')answer(request,[0,0,100,100]);else mutation(request);}catch(error){adapter.transport.reply({request:request.request,ok:false,error:error.message});}})},{body:root,createElement:tag=>new E(tag)},win);
 transfer({bootstrap:{epoch:'cast',seq:0,display:{sketches:[]}}});
 const authority=(allowed=true,epoch=1)=>win.__couchcastSketchAuthority({revision:String(++revision),generation:'1',epoch,holder:allowed?'operator':'driver',ready:true,active:true});authority();
 const flush=async()=>{for(let i=0;i<120;i++)await Promise.resolve();};
 const event=(kind,extra={})=>{for(const fn of adapter.tool.layer.listeners[kind]||[])fn({pointerId:1,pointerType:'pen',isPrimary:true,button:0,buttons:1,pressure:.5,clientX:20,clientY:20,preventDefault(){},...extra});};
 function synthetic(){view.model={seg:model.seg,docSeg:model.docSeg,paused:!model.live,live:model.live,ended:model.ended,driver:model.control.driver?.id||''};for(const fn of subscribers)fn({k:'view',t:model.t},view.model);}
 async function context(kind){if(kind==='handoff'){apply({k:'ctl',op:'request',who:{id:'driver'},pending:[{id:'driver'}]});apply({k:'ctl',op:'grant',who:{id:'driver'},driver:{id:'driver'},pending:[]});authority(false,2);}else if(kind==='expiry'){const next=model.sketchMetadata('s1').next_sequence;apply({k:'sketch-interrupt',sketch:'s1',reason:'expiry',operation:'expiry',sequence:next,digest:'b'.repeat(64)});}else if(kind==='clear')apply({k:'annotation-clear',who:{id:'operator'}});else apply({k:'stop',why:kind==='pause'?'pause':'end'});synthetic();if(pendingPoint){const p=pendingPoint;pendingPoint=null;answer(p,{outcome:'refused',operation:p.body.operation,sketch:p.body.sketch,sequence:p.body.sequence,next_sequence:model.sketchMetadata(p.body.sketch).next_sequence,status:status(p.body.sketch)});}await flush();}
 await adapter.tool.enter();if(startContact)event('pointerdown');await flush();
 return {adapter,model,root,sent,modes,context,flush,event,authority,apply,synthetic,stats:()=>({hitTests,timers:timers.size}),deferAnchors(){deferredAnchor=true;},anchorCallbacks:()=>({resolve:pendingAnchor,reject:pendingAnchor?.reject}),resolveAnchor(){pendingAnchor?.();},tick(ms){clock+=ms;for(const [id,t]of [...timers])if(t.at<=clock){timers.delete(id);t.fn();}},exit(kind){if(kind==='Escape'){for(const fn of win.listeners.keydown||[])fn({key:'Escape',preventDefault(){},stopImmediatePropagation(){}});}else if(kind==='Done'){const toolbar=root.children.find(n=>n.className==='sketch-toolbar');for(const fn of toolbar.children.find(n=>n.textContent==='Done').listeners.click||[])fn();}else view.beforeAnnotationMode(kind);},setReply(fn){replyFilter=fn;},setStatus(fn){statusOverride=fn;}};
}
for(const kind of ['handoff','expiry','pause','clear','end'])for(const empty of [false,true])test(`console ${kind} reconciles authoritative ${empty?'empty':'nonempty'} open contact without Model helpers`,async()=>{const f=await drawingConsole({empty});await f.context(kind);assert.equal(f.adapter.tool.layer.hasPointerCapture(1),false);const retired=kind==='clear'||kind==='end';if(retired){assert.equal(f.adapter.tool.state().phase,'off');assert.deepEqual(f.adapter.tool.state().pendingLabels,[]);}else{assert.equal(f.adapter.tool.state().phase,kind==='pause'?'paused':'interrupted');f.adapter.tool.openLabel('s1');f.adapter.tool.labelInput.value='Late partial intent';await f.adapter.tool.saveLabel();assert.equal(f.model.sketchProjection().sketches[0].label,'Late partial intent');assert.equal(f.sent.filter(r=>r.kind==='sketch-label').length,1);}f.adapter.dispose();});

test('late old console label cannot retire a newly prepared drawer scene', async () => {
 const f=await drawingConsole();
 await f.context('expiry');
 await f.adapter.tool.enter();
 assert.equal(f.adapter.tool.state().active,'s2');
 assert.equal(f.adapter.scene.ready,true);
 f.adapter.tool.openLabel('s1');f.adapter.tool.labelInput.value='Old intent';
 await f.adapter.tool.saveLabel();
 assert.equal(f.model.sketchProjection().sketches[0].label,'Old intent');
 assert.equal(f.adapter.tool.state().active,'s2');
 assert.equal(f.adapter.scene.ready,true);
 assert.equal(f.adapter.tool.state().phase,'drawing');
 f.adapter.dispose();
});
test('stale status open-stroke cannot reopen the journal sealed contact', async () => {
 const f=await drawingConsole();
 const stroke=f.model.sketchMetadata('s1').open_stroke;
 await f.context('expiry');
 f.setStatus(result=>({...result,status:{...result.status,phase:'drawing',open_stroke:stroke}}));
 f.adapter.tool.openLabel('s1');f.adapter.tool.labelInput.value='Sealed intent';
 await f.adapter.tool.saveLabel();
 assert.equal(f.model.sketchProjection().sketches[0].label,'Sealed intent');
 assert.equal(f.sent.filter(r=>r.kind==='sketch-label').length,1);
 f.adapter.dispose();
});

test('already open label draft survives handoff and repeated synthetic refresh', async () => {
 const f=await drawingConsole();f.adapter.tool.openLabel('s1');f.adapter.tool.labelInput.value='Draft before handoff';
 await f.context('handoff');f.synthetic();f.adapter.tool.reconcileSketches();f.synthetic();
 assert.equal(f.adapter.tool.labelInput.value,'Draft before handoff');
 assert.equal(f.adapter.tool.labelInput.parentNode.hidden,false);
 await f.adapter.tool.saveLabel();
 assert.equal(f.model.sketchProjection().sketches[0].label,'Draft before handoff');f.adapter.dispose();
});

for(const change of ['clear','end','new-target'])test(`held label status cannot submit after ${change}`,async()=>{
 const f=await drawingConsole();await f.context('expiry');
 f.adapter.tool.openLabel('s1');f.adapter.tool.labelInput.value='Old draft';
 let complete;f.setStatus((result,answer)=>{complete=()=>answer(result);});
 const saving=f.adapter.tool.saveLabel();await f.flush();assert.equal(typeof complete,'function');
 if(change==='new-target'){f.adapter.tool.openLabel('other');f.adapter.tool.labelInput.value='Other draft';}
 else await f.context(change);
 complete();await saving;
 assert.equal(f.sent.filter(r=>r.kind==='sketch-label').length,0);
 assert.equal(f.model.sketchProjection().sketches[0].label,'');f.adapter.dispose();
});

test('repeated clear/bootstrap cycles retain no historical receiver identities',()=>{
 const receiver=createOperatorSketchReceiver(()=>{});let serial=0;
 const transfer=payload=>{const raw=Buffer.from(JSON.stringify(payload)),base={handle:`cycle-${++serial}`,epoch:'cast',total:raw.length};receiver.receive({...base,phase:'begin'});receiver.receive({...base,phase:'chunk',offset:0,base64:raw.toString('base64')});};
 for(let n=0;n<512;n++){
  const id=`s${n}`;
  transfer({bootstrap:{epoch:'cast',seq:n*2,display:{sketches:[{begin:{id,author:{id:'operator',color:'#ff0000'},doc:1,tab:'tab'},phase:'drawing',reason:'',label:'',strokes:[]}]}}});
  assert.equal(receiver.metadata(id).phase,'drawing');
  transfer({batch:{epoch:'cast',from:n*2+1,to:n*2+1,patches:[{clear:true}]}});
  assert.equal(receiver.metadata(id),null);assert.deepEqual(receiver.paths(),[]);
 }
 receiver.retire();
});

for(const exit of ['Done','Escape','point','note'])for(const released of [false,true])
 test(`console ${exit} preserves ${released?'released':'open'} contact through delayed anchor`,async()=>{
  const f=await drawingConsole({deferredAnchor:true});f.event('pointermove',{clientX:30,pressure:.8});
  if(released)f.event('pointerup',{buttons:0,clientX:30});await f.flush();
  try {
   f.exit(exit);assert.equal(f.adapter.tool.state().phase,'finishing');
   f.adapter.tool.openLabel('s1');f.adapter.tool.labelInput.value='Console preserved';const saving=f.adapter.tool.saveLabel();
   await f.flush();assert.equal(f.sent.filter(r=>r.kind==='sketch-label').length,0);
   f.resolveAnchor();await f.flush();await saving;
   assert.equal(f.adapter.tool.state().phase,'label-required');
   const stroke=f.model.sketchProjection().sketches[0].strokes[0];assert.ok(stroke);assert.equal(stroke.sealed,true);assert.deepEqual(stroke.pts,[[20,20,.5],[30,20,.8]]);
   f.adapter.tool.labelInput.value='Console preserved';await f.adapter.tool.saveLabel();assert.equal(f.model.sketchProjection().sketches[0].label,'Console preserved');
  }finally{f.resolveAnchor();await f.flush();f.adapter.dispose();}
 });

for(const kind of ['handoff','pause','clear','end'])test(`console ${kind} visibly retires pending-anchor finishing and fences late callback`,async()=>{
 const f=await drawingConsole({deferredAnchor:true});const leaving=f.adapter.tool.leave('point');
 await f.context(kind);await leaving;
 assert.match(f.root.querySelector('.feedback-status').textContent,/Couldn't save; accepted strokes kept/);
 f.resolveAnchor();await f.flush();assert.equal(f.sent.filter(r=>r.kind==='sketch-stroke-begin').length,0);assert.equal(f.adapter.tool.layer.hasPointerCapture(1),false);f.adapter.dispose();
});
test('console finishing deadline bounds anchor wait and retains explicit failure after late response',async()=>{
 const f=await drawingConsole({deferredAnchor:true});const leaving=f.adapter.tool.leave('watch');f.tick(5000);await leaving;await f.flush();
 assert.match(f.root.querySelector('.feedback-status').textContent,/Couldn't save; accepted strokes kept/);
 f.resolveAnchor();await f.flush();assert.equal(f.sent.filter(r=>r.kind==='sketch-stroke-begin').length,0);assert.match(f.root.querySelector('.feedback-status').textContent,/Couldn't save/);f.adapter.dispose();
});

for(const ending of ['pointerup','pointercancel','lostpointercapture','buffer-limit'])
 test(`console logical owner survives ${ending} and consumes successor until fresh down`,async()=>{
  const f=await drawingConsole({deferredAnchor:true});
  if(ending==='buffer-limit'){
   f.event('pointermove',{getCoalescedEvents:()=>Array.from({length:1023},(_,i)=>({clientX:20+i/100,clientY:20,pressure:.4}))});f.tick(50);
   f.event('pointermove',{clientX:40});f.tick(50);
  }else f.event(ending,{buttons:0});
  try{
   assert.equal(f.adapter.tool.layer.hasPointerCapture(1),false);let prevented=0,stopped=0;
   assert.doesNotThrow(()=>f.event('pointerdown',{pointerId:2,preventDefault(){prevented++;},stopPropagation(){stopped++;}}));
   assert.equal(prevented,1);assert.ok(stopped>0);assert.equal(f.adapter.tool.layer.hasPointerCapture(2),false);
   assert.match(f.root.querySelector('.feedback-status').textContent,/previous stroke.*wait|wait.*previous stroke/i);
   f.resolveAnchor();await f.flush();assert.doesNotThrow(()=>f.event('pointermove',{pointerId:2,clientX:80}));assert.doesNotThrow(()=>f.event('pointerup',{pointerId:2,buttons:0,clientX:80}));
   assert.equal(f.model.sketchProjection().sketches[0].strokes.length,1);
   f.event('pointerdown',{pointerId:3});f.event('pointerup',{pointerId:3,buttons:0});await f.flush();f.resolveAnchor();await f.flush();assert.equal(f.model.sketchProjection().sketches[0].strokes.length,2);
  }finally{f.resolveAnchor();await f.flush();f.adapter.dispose();}
 });

for(const failure of ['coordinates','capture'])test(`console ${failure} admission failure never publishes half-contact`,async()=>{
 const f=await drawingConsole({startContact:false}),layer=f.adapter.tool.layer,capture=layer.setPointerCapture.bind(layer),toPage=f.adapter.scene.toPage;
 if(failure==='coordinates')f.adapter.scene.toPage=()=>[NaN,1];else layer.setPointerCapture=id=>{capture(id);throw Error('capture failed');};
 assert.doesNotThrow(()=>f.event('pointerdown'));assert.doesNotThrow(()=>f.event('pointerup',{buttons:0}));assert.equal(layer.hasPointerCapture(1),false);assert.equal(f.sent.length,1);
 f.adapter.scene.toPage=toPage;layer.setPointerCapture=capture;f.event('pointerdown');f.event('pointerup',{buttons:0});await f.flush();assert.equal(f.model.sketchProjection().sketches[0].strokes.length,1);f.adapter.dispose();
});
test('console rejected N/2N downs allocate no IDs capture hit-tests timers or points',async()=>{
 const randomUUID=globalThis.crypto.randomUUID;let ids=0;globalThis.crypto.randomUUID=()=>{ids++;return randomUUID.call(globalThis.crypto);};
 try{for(const n of [8,16]){const f=await drawingConsole({deferredAnchor:true}),layer=f.adapter.tool.layer;let captures=0;const capture=layer.setPointerCapture.bind(layer);layer.setPointerCapture=id=>{captures++;capture(id);};f.event('pointerup',{buttons:0});const before=f.stats(),sent=f.sent.length,beforeIDs=ids;
 for(let i=0;i<n;i++){f.event('pointerdown',{pointerId:i%2?1:2});f.event('pointermove',{pointerId:i%2?1:2,clientX:80});f.event('pointerup',{pointerId:i%2?1:2,buttons:0});}
 assert.equal(ids,beforeIDs);assert.equal(captures,0);assert.deepEqual(f.stats(),before);assert.equal(f.sent.length,sent);f.resolveAnchor();await f.flush();assert.deepEqual(f.model.sketchProjection().sketches[0].strokes[0].pts,[[20,20,.5]]);f.adapter.dispose();}}finally{globalThis.crypto.randomUUID=randomUUID;}
});
test('console ordinary anchor timeout releases logical owner and fences late callback',async()=>{
 const f=await drawingConsole({deferredAnchor:true});f.event('pointerup',{buttons:0});f.tick(5000);await f.flush();assert.match(f.root.querySelector('.feedback-status').textContent,/no usable anchor; accepted strokes kept/);f.resolveAnchor();await f.flush();assert.equal(f.model.sketchProjection().sketches[0].strokes.length,0);
 f.event('pointerdown',{pointerId:2});f.event('pointerup',{pointerId:2,buttons:0});await f.flush();f.resolveAnchor();await f.flush();assert.equal(f.model.sketchProjection().sketches[0].strokes.length,1);f.adapter.dispose();
});

test('console Undo/Redo wait for logical owner with existing accepted top',async()=>{
 const f=await drawingConsole({deferredAnchor:true});f.event('pointerup',{buttons:0});f.resolveAnchor();await f.flush();f.event('pointerdown',{pointerId:2});f.event('pointerup',{pointerId:2,buttons:0});await f.flush();const count=f.sent.length;await f.adapter.tool.undo();await f.adapter.tool.redo();assert.equal(f.sent.length,count);f.resolveAnchor();await f.flush();await f.adapter.tool.undo();assert.equal(f.sent.at(-1).kind,'sketch-undo');f.adapter.dispose();
});
test('console 4096 point end transfers owner while FIFO keeps terminal before successor begin',async()=>{
 const f=await drawingConsole();let held=false,endACK;f.setReply((request,value)=>{if(request.kind==='sketch-stroke-end'&&!held){held=true;endACK=()=>f.adapter.transport.reply({request:request.request,ok:true,value});return false;}});
 for(let at=0;at<4095;at+=64){f.event('pointermove',{getCoalescedEvents:()=>Array.from({length:Math.min(64,4095-at)},(_,i)=>({clientX:20+(at+i)/100,clientY:20,pressure:.5}))});f.tick(50);await f.flush();}
 f.event('pointermove',{clientX:90});await f.flush();assert.equal(f.adapter.tool.layer.hasPointerCapture(1),false);f.event('pointerdown',{pointerId:2});f.event('pointerup',{pointerId:2,buttons:0});await f.flush();assert.equal(f.sent.filter(r=>r.kind==='sketch-stroke-begin').length,1);endACK();await f.flush();assert.equal(f.model.sketchProjection().sketches[0].strokes.length,2);assert.equal(f.model.sketchProjection().sketches[0].strokes[0].pts.length,4096);const order=f.sent.map(r=>r.kind);assert.ok(order.indexOf('sketch-stroke-end')<order.lastIndexOf('sketch-stroke-begin'));f.adapter.dispose();
});

for(const completion of ['reject','timeout'])for(const released of [false,true])test(`console retired ${released?'released':'open'} anchor ${completion} cannot cancel fresh owner after prior terminal confirmation`,async()=>{
 const f=await drawingConsole();let held=false,oldTerminal;
 f.setReply((request,value)=>{if(request.kind==='sketch-stroke-end'&&!held){held=true;oldTerminal=()=>f.adapter.transport.reply({request:request.request,ok:true,value});return false;}});
 f.event('pointerup',{buttons:0});await f.flush();f.tick(1000);f.deferAnchors();f.event('pointerdown',{pointerId:2});if(released)f.event('pointerup',{pointerId:2,buttons:0});await f.flush();const old=f.anchorCallbacks();
 f.tick(4000);await f.flush();assert.equal(f.adapter.tool.state().phase,'reconciling');assert.match(f.root.querySelector('.feedback-status').textContent,/Couldn't save this stroke; accepted strokes kept/);f.adapter.tool.retry();await f.flush();assert.equal(f.adapter.tool.state().phase,'drawing');
 f.event('pointerdown',{pointerId:3});await f.flush();const fresh=f.anchorCallbacks();if(completion==='reject')old.reject();else f.tick(1000);await f.flush();assert.equal(f.adapter.tool.layer.hasPointerCapture(3),true);
 f.event('pointerup',{pointerId:3,buttons:0});fresh.resolve();await f.flush();assert.equal(f.model.sketchProjection().sketches[0].strokes.length,2);old.resolve();oldTerminal();await f.flush();assert.equal(f.model.sketchProjection().sketches[0].strokes.length,2);f.adapter.dispose();
});

test('console required label Enter saves once and waits for accepted reply before switching tools',async()=>{
 const f=await drawingConsole();f.event('pointerup',{buttons:0});await f.flush();f.exit('point');await f.flush();const ui=f.adapter.tool;
 const key=extra=>{for(const fn of ui.labelInput.listeners.keydown||[])fn({key:'Enter',preventDefault(){},stopPropagation(){},...extra});};
 key();await f.flush();assert.equal(f.sent.filter(r=>r.kind==='sketch-label').length,0);ui.labelInput.value='Console Enter intent';key({isComposing:true});await f.flush();assert.equal(f.sent.filter(r=>r.kind==='sketch-label').length,0);
 let accept;f.setReply((request,value)=>{if(request.kind==='sketch-label'){accept=()=>f.adapter.transport.reply({request:request.request,ok:true,value});return false;}});
 key();await f.flush();assert.equal(f.sent.filter(r=>r.kind==='sketch-label').length,1);assert.equal(f.modes.filter(m=>m==='point').length,0);assert.equal(f.adapter.scene.ready,true);
 key({repeat:true});await f.flush();assert.equal(f.sent.filter(r=>r.kind==='sketch-label').length,1);accept();await f.flush();assert.equal(f.modes.filter(m=>m==='point').length,1);assert.equal(f.adapter.scene.ready,false);assert.equal(f.model.sketchProjection().sketches[0].label,'Console Enter intent');key();await f.flush();assert.equal(f.modes.filter(m=>m==='point').length,1);f.adapter.dispose();
});
