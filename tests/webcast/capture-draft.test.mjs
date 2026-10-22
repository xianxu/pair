import {test} from 'node:test';
import assert from 'node:assert/strict';
import {captureDraftController} from '../../cmd/internal/webcast/web/capture-draft.js';
const flush=async()=>{for(let i=0;i<12;i++)await Promise.resolve();};
function fixture(call){let at=0,serial=0;const timers=new Map(),updates=[],calls=[];const c=captureDraftController({call:(...a)=>{calls.push(a);return call(...a);},id:()=>`save${++serial}`,project:(...a)=>updates.push(a),now:()=>at,setTimeout:(fn,ms)=>{const id=++serial;timers.set(id,{fn,when:at+ms});return id;},clearTimeout:id=>timers.delete(id)});return {c,updates,calls,async advance(ms){at+=ms;for(const [id,t] of [...timers])if(t.when<=at){timers.delete(id);t.fn();}await flush();}};}
test('prepare projects pending immediately then confirmed tick or explicit unavailable without blocking Save',async()=>{
 let release;const f=fixture(()=>new Promise(r=>release=r));f.c.start({region:[1,2,3,4]},'p1');assert.equal(f.updates[0][1].pending,true);await flush();
 release({state:'ready',draft:'signed-token',captured_at:123});await flush();assert.equal(f.updates.at(-1)[1].pending,false);assert.equal(f.updates.at(-1)[1].capturedAt,123);
 const unavailable=fixture(async()=>({state:'ready',draft:'bound-token',unavailable:'crop unavailable: capture busy'}));unavailable.c.start({},'p2');await flush();assert.equal(unavailable.updates.at(-1)[1].pending,false);assert.equal(unavailable.updates.at(-1)[1].payload.draft,'bound-token');assert.equal(unavailable.updates.at(-1)[1].message,'crop unavailable: capture busy');
});
test('a newer gesture cancels pending prepare by request and ignores its late ready answer',async()=>{
 const releases=[];const f=fixture((op)=>op==='prepare-note'?new Promise(r=>releases.push(r)):Promise.resolve({state:'cancelled'}));f.c.start({},'old');await flush();f.c.start({},'new');await flush();
 assert.ok(f.calls.some(([op,b])=>op==='cancel-draft'&&b.request==='old'));
 releases[0]({state:'ready',draft:'oldtoken',captured_at:1});await flush();assert.equal(f.updates.at(-1)[0],'new');
 releases[1]({state:'ready',draft:'newtoken',captured_at:2});await flush();assert.equal(f.updates.at(-1)[1].payload.draft,'newtoken');
});
test('pending prepare is polled, actual failures expose retry and timeout is absolute',async()=>{
 const f=fixture(async(op)=>op==='prepare-note'?{state:'pending'}:{state:'ready',draft:'token',captured_at:1});f.c.start({},'p');await flush();await f.advance(1000);assert.ok(f.calls.some(([op])=>op==='draft-status'));assert.equal(f.updates.at(-1)[1].pending,false);
 const bad=fixture(async()=>({state:'failed',error:'Chrome disconnected'}));bad.c.start({},'bad');await flush();assert.match(bad.updates.at(-1)[1].message,/Chrome disconnected/);assert.equal(bad.updates.at(-1)[1].retry,true);
 const hung=fixture(()=>new Promise(()=>{}));hung.c.start({},'hung');await flush();await hung.advance(36000);assert.equal(hung.updates.at(-1)[1].pending,false);assert.match(hung.updates.at(-1)[1].message,/timed out/i);
});
test('save accepts only confirmed saved and repeated clicks share one operation; success is never cancelled',async()=>{
 let release;const f=fixture(async(op)=>op==='prepare-note'?{state:'ready',draft:'signed-token',captured_at:1}:new Promise(r=>release=r));f.c.start({},'p');await flush();const a=f.c.save({text:'one',color:0}),b=f.c.save({text:'one',color:0});assert.equal(a,b);await flush();assert.equal(f.calls.filter(([op])=>op==='save-note').length,1);
 release({state:'saved',id:'note1'});assert.equal((await a).status,204);f.c.cancel();assert.equal(f.calls.filter(([op])=>op==='cancel-draft').length,0);
});
test('retry rekeys only the capture projection, retires old token, and accepts a full opaque token',async()=>{
 const long='signed.'+'x'.repeat(32000);let attempts=0;
 const f=fixture(async op=>op==='prepare-note'?(++attempts===1?{state:'failed',error:'Capture failed'}:{state:'ready',draft:long,captured_at:'2026-10-13T12:00:00Z'}):{state:'cancelled'});
 f.c.start({},'old');await flush();f.c.retry();await flush();assert.ok(f.updates.some(([request,u])=>request==='old'&&u.payload?.request));assert.equal(f.updates.at(-1)[1].payload.draft,long);assert.ok(f.calls.some(([op,b])=>op==='cancel-draft'&&b.request==='old'));
});
test('operator capture adapter sends distinct status envelopes and the prepared request in its body',async()=>{
 const {mountOperator}=await import('../../cmd/internal/webcast/operator.js');let options;const sent=[];
 const win={scrollX:0,scrollY:0,setTimeout:()=>1,clearTimeout(){},addEventListener(){},__couchcastPath:()=>[1,2]};
 const hit={getBoundingClientRect:()=>({x:10,y:20,width:30,height:40})},doc={createElement:()=>({}),elementFromPoint:()=>hit},host={style:{}},root={appendChild(){}};
 const ui={strokes:{clear(){}},setAnnotationAllowed(){}};
 mountOperator({host,root,report:m=>{if(m.t==='operator')sent.push(m.request);}},{annotate:{mountAnnotation:(d,w,v,o)=>{options=o;return ui;}},palette:[],css:''},doc,win);
 win.__couchcastAnnotationAllowed(true,'1');
 const body=options.prepareDraft([10,20,5,5]);assert.deepEqual(body.path,[1,2]);assert.equal(sent.length,0);
 const prepare=options.captureCall('prepare-note',{...body,request:'prepare1'},{timeoutMs:4000});assert.equal(sent[0].request,'prepare1');win.__couchcastOperatorReply({request:'prepare1',outcome:'pending',state:'pending'});assert.equal((await prepare).state,'pending');
 const status=options.captureCall('draft-status',{request:'prepare1'},{timeoutMs:4000});assert.notEqual(sent[1].request,'prepare1');assert.equal(sent[1].body.draft,'prepare1');win.__couchcastOperatorReply({request:sent[1].request,outcome:'accepted',state:'ready',draft:'signed'});assert.equal((await status).draft,'signed');
});
test('queued writer pending errors keep polling; confirmed refusal allows edited retry',async()=>{
 let attempt=0;const f=fixture(async op=>op==='prepare-note'?{state:'ready',draft:'signed'}:op==='save-note'?(++attempt===1?{state:'pending',error:'save not confirmed; retry Save note to check'}:{state:'saved',id:'note'}):{state:'ready',error:'Writer refused'});
 f.c.start({},'p');await flush();const first=f.c.save({text:'old',color:0});await flush();await f.advance(1000);assert.match((await first).text,/Writer refused/);
 assert.equal((await f.c.save({text:'edited',color:0})).status,204);assert.equal(f.calls.filter(([op])=>op==='save-note').at(-1)[1].text,'edited');
});
test('uncertain save retry checks status and retains request identity until confirmed',async()=>{
 let saves=0;const f=fixture(async op=>op==='prepare-note'?{state:'ready',draft:'signed'}:op==='draft-status'?{state:'pending'}:(++saves===1?Promise.reject(Error('network lost')):{state:'saved',id:'note'}));
 f.c.start({},'p');await flush();assert.equal((await f.c.save({text:'same',color:0})).status,409);assert.equal((await f.c.save({text:'same',color:0})).status,204);
 const sent=f.calls.filter(([op])=>op==='save-note');assert.equal(sent[0][1].request,sent[1][1].request);
});
test('prepare and cancellation retain originating authority across deferred dispatch',async()=>{
 let generation='1';const sent=[];const call=async(op,body,opts)=>{sent.push([op,opts.context]);return {state:'pending'};};call.context=()=>generation;
 const c=captureDraftController({call,id:()=> 'id',project(){},setTimeout:()=>1,clearTimeout(){}});c.start({},'first');generation='2';await flush();assert.deepEqual(sent[0],['prepare-note','1']);c.cancel();await flush();assert.deepEqual(sent[1],['cancel-draft','1']);
});
test('repeated pending responses stop at the absolute budget with at most one status poll per second',async()=>{
 const f=fixture(async()=>({state:'pending'}));f.c.start({},'p');await flush();for(let i=0;i<36;i++)await f.advance(1000);
 assert.equal(f.updates.at(-1)[1].state,'failed');assert.match(f.updates.at(-1)[1].message,/timed out/);assert.ok(f.calls.length<=36);assert.ok(f.calls.every(([, ,opts])=>opts.timeoutMs>0&&opts.timeoutMs<=4000));
});
