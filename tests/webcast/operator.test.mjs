import { operatorTransport, mountOperator } from '../../cmd/internal/webcast/operator.js';
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { annotationPolicy } from '../../cmd/internal/webcast/web/annotation-policy.js';
import { mountAnnotation } from '../../cmd/internal/webcast/web/annotate.js';

import { Element } from './dom-fixture.mjs';

test('operator pending load immediately retires shared notes and unsaved editor without mutation',()=>{
 class E extends Element {appendChild(el){this.append(el);el.parentNode=this;return el;}remove(){if(this.parentNode)this.parentNode.children=this.parentNode.children.filter(x=>x!==this);}getContext(){return {setTransform(){},clearRect(){}};}}
 const root=new E('shadow'),host=new E('host'),frames=[],sent=[];
 const doc={body:new E('body'),createElement:tag=>new E(tag)};
 const win={innerWidth:800,innerHeight:600,scrollX:0,scrollY:0,crypto:{getRandomValues:b=>b.fill(1)},performance:{now:()=>0},requestAnimationFrame:fn=>frames.push(fn),addEventListener(){},setTimeout:()=>1,clearTimeout(){},clearInterval(){}};
 const ui=mountOperator({host,root,report:m=>sent.push(m)},{annotate:{mountAnnotation},palette:['#ff0000'],css:''},doc,win);
 const load={target:'tab',loader:'L',root:'1'},mask={scope:'cast',revision:'0',sketches:[],notes:[]};
 const note={id:'saved',author:{id:'operator'},text:'Saved',at:{node:7,box:[0,0,20,20]}};
 const snapshot=(revision,extra={})=>win.__couchcastOperatorView({feedback_mask:mask,revision,seg:1,doc:1,t:0,notes:[note],strokes:[],page_load_ready:true,page_load:load,sketch_cursor:{epoch:'A',seq:0},...extra});
 snapshot(1);for(let i=0;i<3;i++)for(const fn of frames.splice(0))fn();assert.equal(root.querySelector('.feedback-open').textContent,'Open note');
 win.__couchcastAnnotationAllowed(true,'1');ui.dialog.draft({draft:'unsaved'});
 root.querySelector('.note-input').value='Unsaved';for(const fn of root.querySelector('.note-input').listeners.input)fn();
 assert.deepEqual(win.__couchcastNoteUnsaved(),{draft:true});const count=sent.filter(m=>m.t==='mutate').length;
 snapshot(2,{page_load_ready:false,page_load:null});
 assert.equal(root.querySelector('.feedback-rows').children.length,0);assert.equal(ui.dialog.open,false);assert.equal(win.__couchcastNoteUnsaved(),null);assert.equal(ui.mode,'watch');assert.equal(root.querySelector('.mode').disabled,true);
 assert.equal(sent.filter(m=>m.t==='mutate').length,count);
 snapshot(3);for(let i=0;i<3;i++)for(const fn of frames.splice(0))fn();assert.equal(root.querySelector('.feedback-open').textContent,'Open note');assert.equal(ui.dialog.open,false);
 snapshot(4,{page_load_ready:true,page_load:null});assert.equal(root.querySelector('.feedback-rows').children.length,0);
});

test('shared mount keeps all UI in its supplied root and holds a driven editor', () => {
  const body = new Element('body'), root = new Element('shadow'), box = new Element('box'); root.append(box);
  const doc = {body, createElement:(tag)=>new Element(tag)};
  const win = {innerWidth:100, crypto:{getRandomValues:(b)=>b.fill(1)}, performance:{now:()=>0}, requestAnimationFrame:()=>{}, addEventListener:()=>{}, setTimeout:()=>1, clearTimeout:()=>{}, clearInterval:()=>{}};
  let receive;
  const v = {model:{seg:1,docSeg:1,notes:[]}, onRecord:(fn)=>{receive=fn;}, toPage:(x,y)=>[x,y], fromPage:(x,y)=>[x,y]};
  const ui = mountAnnotation(doc,win,v,{root,box,stage:box,palette:['#ff0000'],who:{id:()=> 'operator',name:()=> 'Operator',viewer:()=>({id:'operator',name:'Operator'})},colors:{get:()=>0,set:()=>{}},post:async()=>({status:204}),operator:true});
  assert.equal(body.children.length,0);
  ui.setMode('note');
  assert.equal(ui.mode,'note');
  ui.dialog.draft({draft:'d1'});
  const input = root.querySelector('.note-input'); input.value='keep this'; for(const fn of input.listeners.input)fn();
  ui.setAnnotationAllowed(false);
  assert.equal(ui.mode,'watch');
  assert.equal(root.querySelector('.note-dialog').hidden,true);
  assert.equal(input.value,'keep this');
  assert.equal(root.querySelector('.mode').disabled,true);
  assert.equal(root.querySelector('.mode').title,'Annotation is off.');
  ui.setMode('point');
  assert.equal(ui.mode,'watch');
  // A projection refresh cannot reset the capability input.
  receive({k:'view',t:0},v.model);
  assert.equal(root.querySelector('.mode').disabled,true);
  ui.setAnnotationAllowed(true);
  assert.equal(ui.mode,'watch');
  assert.equal(input.value,'keep this');
  assert.equal(root.querySelector('.note-dialog').hidden,false);
  assert.equal(root.querySelector('.mode').disabled,false);
  v.model.driver='Ana'; receive({k:'ctl',t:1},v.model);
  assert.equal(ui.mode,'watch');
  assert.equal(input.value,'keep this');
  assert.equal(input.focused,false);
  assert.equal(root.querySelector('.note-dialog').hidden,true);
  v.model.driver=''; receive({k:'ctl',t:2},v.model);
  assert.equal(ui.mode,'watch');
  assert.equal(input.value,'keep this');
  assert.equal(root.querySelector('.note-dialog').hidden,false);
  assert.equal(input.focused,false);
  v.model.docSeg=2; receive({k:'seg',t:3},v.model);
  assert.equal(ui.dialog.open,false);
});

test('annotation availability owns mode and interruption effects', () => {
  let s = annotationPolicy(null, { type: 'sync', doc: 1, ready: true }).state;
  assert.equal(s.mode, 'watch');
  s = annotationPolicy(s, { type: 'mode', mode: 'note' }).state;
  assert.equal(s.mode, 'note');
  const driven = annotationPolicy(s, { type: 'sync', doc: 1, ready: true, driver: 'Ana' });
  assert.equal(driven.state.phase, 'driving');
  assert.equal(driven.state.mode, 'watch');
  assert.deepEqual(driven.effects, ['cancel', 'hold']);
  assert.equal(annotationPolicy(driven.state, { type: 'mode', mode: 'point' }).state.mode, 'watch');
  const returned = annotationPolicy(driven.state, { type: 'sync', doc: 1, ready: true });
  assert.equal(returned.state.mode, 'watch');
  assert.deepEqual(returned.effects, ['release']);
  const next = annotationPolicy(returned.state, { type: 'sync', doc: 2, ready: true });
  assert.deepEqual(next.effects, ['cancel', 'discard']);
});

test('pause holds a draft while navigation and explicit Escape discard it', () => {
  const ready = annotationPolicy(null, { type: 'sync', doc: 1, ready: true }).state;
  const paused = annotationPolicy(ready, { type: 'sync', doc: 1, ready: true, paused: true });
  assert.equal(paused.state.phase, 'paused');
  assert.ok(paused.effects.includes('hold'));
  assert.deepEqual(annotationPolicy(paused.state, { type: 'escape' }).effects, ['cancel', 'discard']);
  assert.equal(annotationPolicy(ready, { type: 'mode', mode: 'bogus' }).state.mode, 'watch');
  const ended = annotationPolicy(ready, { type: 'sync', doc: 1, ended: true });
  assert.equal(ended.state.phase, 'ended');
  assert.ok(ended.effects.includes('discard'));
});

// A timeout is reconciliation, never permission to submit a duplicate note.
test('operator transport retries the same mutation by status even after draft edits', async () => {
  const sent = [], timers = new Map(); let n = 0;
  const win = { setTimeout(fn) { timers.set(++n, fn); return n; }, clearTimeout(id) { timers.delete(id); } };
  const t = operatorTransport((m) => sent.push(m.request), win);
  t.setAnnotationGeneration('1');
  const first = t.mutate('save-note', {draft:'d',text:'one',color:0});
  timers.values().next().value();
  assert.equal((await first).status, 409);
  const retry = t.mutate('save-note', {draft:'d',text:'edited',color:0});
  assert.equal(sent[1].op, 'status');
  assert.equal(sent[1].request, sent[0].request);
  t.reply({request:sent[1].request,outcome:'accepted'});
  assert.equal((await retry).status, 409); // the edited text was never submitted
  const next = t.mutate('save-note', {draft:'e',text:'two',color:0});
  assert.notEqual(sent[2].request, sent[0].request);
  t.reply({request:sent[2].request,outcome:'refused',error:'paused'});
  assert.equal((await next).text, 'paused');
});


test('annotation allowed input cancels gestures and cannot override driving', () => {
  let s = annotationPolicy(null,{type:'sync',ready:true,doc:1}).state;
  s = annotationPolicy(s,{type:'mode',mode:'point'}).state;
  const off = annotationPolicy(s,{type:'sync',ready:true,doc:1,allowed:false});
  assert.equal(off.state.phase,'disabled');
  assert.equal(off.state.mode,'watch');
  assert.deepEqual(off.effects,['cancel','hold']);
  assert.equal(annotationPolicy(off.state,{type:'sync',ready:true,doc:1,allowed:true,driver:'Ana'}).state.phase,'driving');
});

test('shared viewer mount keeps a view-only link read-only after mode requests', () => {
  const root = new Element('body'), box = new Element('box'); root.append(box);
  const doc = {body:root, createElement:(tag)=>new Element(tag)};
  let frame;
  const win = {innerWidth:100, crypto:{getRandomValues:(b)=>b.fill(1)}, performance:{now:()=>0}, requestAnimationFrame:(fn)=>{frame=fn;}, addEventListener:()=>{}, setTimeout:()=>1, clearTimeout:()=>{}, clearInterval:()=>{}};
  const v = {model:{seg:1,docSeg:1,notes:[]}, onRecord:()=>{}, toPage:(x,y)=>[x,y], fromPage:(x,y)=>[x,y]};
  const ui = mountAnnotation(doc,win,v,{root,box,stage:box,palette:['#ff0000'],who:{id:()=> 'viewer',name:()=> 'Viewer',viewer:()=>({id:'viewer',name:'Viewer'})},colors:{get:()=>0,set:()=>{}},post:()=>{throw Error('read-only posted');},annotate:false});
  ui.setMode('note');
  assert.equal(ui.mode,'watch');
  assert.equal(ui.annotate,false);
  assert.equal(root.querySelector('.mode').disabled,true);
  // The separate operator capability input cannot grant a view-only link.
  ui.setAnnotationAllowed(true);
  ui.setMode('point');
  assert.equal(ui.mode,'watch');
});

for (const operator of [false, true]) {
  for (const gesture of ['click', 'drag', 'drag-back']) {
    test(`${operator ? 'cast Chrome' : 'viewer'} note ${gesture}: preview waits for drag; click box waits for release`, async () => {
      const root = new Element(operator ? 'shadow' : 'body'), box = new Element('box'); root.append(box);
      const doc = {body:root, createElement:(tag)=>new Element(tag)};
      const win = {innerWidth:100, crypto:{getRandomValues:(b)=>b.fill(1)}, performance:{now:()=>0}, requestAnimationFrame:()=>{}, addEventListener:()=>{}, setTimeout:()=>1, clearTimeout:()=>{}, clearInterval:()=>{}};
      const regions = [];
      const v = {model:{seg:1,docSeg:1,notes:[]}, onRecord:()=>{}, toPage:(x,y)=>[x,y], fromPage:(x,y)=>[x,y], nodeAt:()=>7, boxOf:()=>[0,0,100,100]};
      const ui = mountAnnotation(doc,win,v,{root,box,stage:box,palette:['#ff0000'],who:{id:()=>operator?'operator':'viewer',name:()=> 'Named',viewer:()=>({id:'named',name:'Named'})},colors:{get:()=>0,set:()=>{}},post:async(route,payload)=>{regions.push(payload.region);return {status:204};},operator,prepareDraft:operator?async(region)=>({region}):undefined});
      ui.setMode('note');
      const layer = root.querySelector('.annotate'), outline = root.querySelector('.drag-region');
      const pointer = async (kind,x,y) => {
        for (const fn of layer.listeners[kind] || []) await fn({target:layer,pointerId:1,pointerType:'mouse',clientX:x,clientY:y,preventDefault(){}});
      };
      await pointer('pointerdown',40,40);
      assert.equal(outline.hidden,true,'mouse down drew a box');
      assert.equal(ui.dialog.open,false);
      await pointer('pointermove',43,42);
      assert.equal(outline.hidden,true,'sub-threshold wobble drew a box');
      if (gesture !== 'click') {
        await pointer('pointermove',60,70);
        assert.equal(outline.hidden,false);
        assert.deepEqual([outline.style.left,outline.style.top,outline.style.width,outline.style.height],['40px','40px','20px','30px']);
        if (gesture === 'drag-back') {
          await pointer('pointermove',42,43);
          assert.equal(outline.hidden,false,'started drag became a click again');
          assert.equal(outline.style.width,'2px');
        }
      }
      await pointer('pointerup',gesture==='click'?43:gesture==='drag-back'?42:60,gesture==='click'?42:gesture==='drag-back'?43:70);
      assert.equal(outline.hidden,false,'completed note box is visible while editing');
      assert.equal(outline.style.width,gesture==='click'?'10px':gesture==='drag-back'?'2px':'20px');
      assert.equal(ui.dialog.open,true);
      const input = root.querySelector('.note-input'); input.value='note';
      for (const fn of input.listeners.input) fn();
      const save = root.querySelector('.note-buttons').children.at(-1);
      for (const fn of save.listeners.click) fn();
      assert.deepEqual(regions,[gesture==='click'?[35,35,10,10]:gesture==='drag-back'?[40,40,2,3]:[40,40,20,30]]);
      await Promise.resolve();
      assert.equal(outline.hidden,true,'saved draft left a preview behind');
    });
  }
}


for (const previous of ['save-note', 'resolve']) {
  test(`late ${previous} acceptance cannot complete a different draft`, async () => {
    const sent=[], timers=new Map(); let serial=0;
    const win={setTimeout(fn){timers.set(++serial,fn);return serial;},clearTimeout(id){timers.delete(id);}};
    const transport=operatorTransport(m=>sent.push(m.request),win);
  transport.setAnnotationGeneration('1');
    const first=transport.mutate(previous,previous==='save-note'?{draft:'A',text:'old',color:0}:{seg:1,id:'old-note',state:'resolved'});
    timers.values().next().value();
    assert.equal((await first).status,409);
    // Cancel the old editor, prepare B, and save it while A's result is unknown.
    const nextBody={draft:'B',text:'new draft',color:0};
    const replacement=transport.mutate('save-note',nextBody);
    assert.equal(sent[1].op,'status');
    transport.reply({request:sent[1].request,outcome:'accepted'});
    assert.equal((await replacement).status,409,'an unsent draft inherited success');
    const retry=transport.mutate('save-note',nextBody);
    assert.equal(sent[2].op,'save-note');
    assert.deepEqual(sent[2].body,nextBody);
    assert.notEqual(sent[2].request,sent[0].request);
    transport.reply({request:sent[2].request,outcome:'accepted'});
    assert.equal((await retry).status,204);
  });
}

test('operator Escape with page focus cancels menu-selected modes without touching held drafts', () => {
  const root=new Element('shadow'), box=new Element('box');root.append(box);
  const doc={body:new Element('body'),createElement:tag=>new Element(tag)};
  const listeners={};
  const win={innerWidth:100,crypto:{getRandomValues:b=>b.fill(1)},performance:{now:()=>0},requestAnimationFrame:()=>{},addEventListener:(kind,fn)=>{(listeners[kind]||=[]).push(fn);},setTimeout:()=>1,clearTimeout:()=>{},clearInterval:()=>{}};
  let receive;
  const v={model:{seg:1,docSeg:1,notes:[]},onRecord:fn=>{receive=fn;},toPage:(x,y)=>[x,y],fromPage:(x,y)=>[x,y]};
  const ui=mountAnnotation(doc,win,v,{root,box,stage:box,palette:['#ff0000'],who:{id:()=> 'operator',name:()=> 'Operator',viewer:()=>({id:'operator',name:'Operator'})},colors:{get:()=>0,set:()=>{}},post:async()=>({status:204}),operator:true});
  let stopped=0;
  const escape=()=>{for(const fn of listeners.keydown||[])fn({key:'Escape',target:doc.body,stopPropagation(){stopped++;},preventDefault(){}});};
  for(const mode of ['point','note']){ui.setMode(mode);escape();assert.equal(ui.mode,'watch');}
  assert.equal(stopped,2);
  escape();assert.equal(stopped,2,'Off intercepted page Escape');
  ui.dialog.draft({draft:'held'});v.model.driver='Ana';receive({k:'ctl',t:1},v.model);
  escape();assert.equal(ui.dialog.open,true,'remote Escape discarded held editor');
  assert.equal(stopped,2);
});

for (const operator of [false,true]) {
  test(`${operator?'operator':'viewer'} keeps a pressed color button across unrelated records`, () => {
    const root=new Element(operator?'shadow':'body'),box=new Element('box');root.append(box);
    const doc={body:root,createElement:tag=>new Element(tag)};
    const win={innerWidth:100,crypto:{getRandomValues:b=>b.fill(1)},performance:{now:()=>0},requestAnimationFrame:()=>{},addEventListener:()=>{},setTimeout:()=>1,clearTimeout:()=>{},clearInterval:()=>{}};
    let receive, picked=0;
    const v={model:{seg:1,docSeg:1,notes:[]},onRecord:fn=>{receive=fn;},toPage:(x,y)=>[x,y],fromPage:(x,y)=>[x,y]};
    mountAnnotation(doc,win,v,{root,box,stage:box,palette:['#ff0000','#00ff00'],who:{id:()=> 'named',name:()=> 'Named',viewer:()=>({id:'named',name:'Named'})},colors:{get:()=>picked,set:n=>{picked=n;}},post:async()=>({status:204}),operator});
    const swatches=root.querySelector('.swatches');
    const pressed=swatches.children[1]; // pointerdown starts on this exact element
    for(const k of ['scroll','ptr','focus','view']) receive({k,t:1},v.model);
    assert.equal(swatches.children[1],pressed,'record detached the pressed button');
    for(const fn of pressed.listeners.click)fn(); // pointerup completes its click
    assert.equal(picked,1);
    const selected=swatches.children[1];
    receive({k:'scroll',t:2},v.model);
    assert.equal(swatches.children[1],selected);
    assert.ok(selected.className.split(' ').includes('on'));
  });
}

test('operator mount starts disabled and a new generation resets pen without dropping draft', () => {
  const root = new Element('shadow'), box = new Element('box'); root.append(box);
  const doc = {body:new Element('body'),createElement:(tag)=>new Element(tag)};
  const win = {innerWidth:100,crypto:{getRandomValues:(b)=>b.fill(1)},performance:{now:()=>0},requestAnimationFrame:()=>{},addEventListener:()=>{},setTimeout:()=>1,clearTimeout:()=>{},clearInterval:()=>{}};
  const view = {model:{seg:1,docSeg:1,notes:[]},onRecord:()=>{},toPage:(x,y)=>[x,y],fromPage:(x,y)=>[x,y]};
  const ui = mountAnnotation(doc,win,view,{root,box,stage:box,palette:['#f00'],who:{id:()=> 'operator',name:()=> 'Operator',viewer:()=>({id:'operator',name:'Operator'})},colors:{get:()=>0,set:()=>{}},post:async()=>({status:204}),operator:true,annotationAllowed:false});
  ui.setMode('note');
  assert.equal(ui.mode,'watch');
  assert.equal(root.querySelector('.mode').disabled,true);
  ui.setAnnotationAllowed(true);
  ui.setMode('note');
  ui.dialog.draft({draft:'held'});
  root.querySelector('.note-input').value='keep me'; for(const fn of root.querySelector('.note-input').listeners.input)fn();
  // The transport coalesced Off/On. This is a new delivered permission, even
  // though the previous delivered boolean was also true.
  ui.setAnnotationAllowed(true, true);
  assert.equal(ui.mode,'watch');
  assert.equal(ui.dialog.open,true);
  assert.equal(root.querySelector('.note-input').value,'keep me');
});

for (const coalesced of [false,true]) test(`operator permission clears browser strokes while preserving held draft (coalesced=${coalesced})`, () => {
 class CanvasElement extends Element {getContext(){return {setTransform(){},clearRect(){},beginPath(){},stroke(){},moveTo(){},lineTo(){},arc(){},fill(){}};}}
 const root=new Element('shadow'); root.appendChild=(el)=>root.append(el);
 const host=new Element('host');
 const doc={body:new Element('body'),createElement:(tag)=>new CanvasElement(tag)};
 const win={innerWidth:100,scrollX:0,scrollY:0,crypto:{getRandomValues:(b)=>b.fill(1)},performance:{now:()=>0},requestAnimationFrame:()=>{},addEventListener:()=>{},setTimeout:()=>1,clearTimeout:()=>{},clearInterval:()=>{}};
 const ui=mountOperator({host,root,report:()=>{}},{annotate:{mountAnnotation},palette:['#f00'],css:''},doc,win);
 win.__couchcastOperatorView({feedback_mask:{scope:"cast",revision:"0",sketches:[],notes:[]},page_load_ready:true,page_load:{target:'tab',loader:'A',root:'1'},revision:1,seg:1,doc:1,t:0,notes:[],strokes:[]});
 assert.equal(root.querySelector('.mode').disabled,true);
 win.__couchcastAnnotationAllowed(true);
 assert.equal(root.querySelector('.mode').disabled,true); // missing generation never enables
 win.__couchcastAnnotationAllowed(true,'1');
 ui.dialog.draft({draft:'held'});
 root.querySelector('.note-input').value='keep this draft'; for(const fn of root.querySelector('.note-input').listeners.input)fn();
 win.__couchcastOperatorView({feedback_mask:{scope:"cast",revision:"0",sketches:[],notes:[]},page_load_ready:true,page_load:{target:'tab',loader:'A',root:'1'},revision:2,seg:1,doc:1,t:0,notes:[],strokes:[{seq:1,id:'viewer.pen',t:0,color:'#f00',pts:[[1,2,0]]}]});
 assert.equal(ui.strokes.visible(100).length,1);
 if (!coalesced) win.__couchcastAnnotationAllowed(false,'2');
 win.__couchcastOperatorView({feedback_mask:{scope:"cast",revision:"0",sketches:[],notes:[]},page_load_ready:true,page_load:{target:'tab',loader:'A',root:'1'},revision:3,seg:1,doc:1,t:0,notes:[],strokes:[]});
 win.__couchcastAnnotationAllowed(true,'3');
 ui.setMode('point');
 assert.equal(ui.strokes.visible(100).length,0);
 assert.equal(ui.dialog.open,true);
 assert.equal(root.querySelector('.note-input').value,'keep this draft');
});

test('operator mutation retains originating generation across delayed delivery and reconciliation', async () => {
 const sent=[],timers=new Map(); let serial=0;
 const win={setTimeout(fn){timers.set(++serial,fn);return serial;},clearTimeout(id){timers.delete(id);}};
 const transport=operatorTransport(m=>sent.push(m.request),win);
 transport.setAnnotationGeneration('9007199254740993');
 const pending=transport.mutate('save-note',{draft:'held',color:0,text:'text'});
 assert.equal(sent[0].generation,'9007199254740993');
 transport.setAnnotationGeneration('9007199254740995');
 timers.values().next().value();
 await pending;
 const retry=transport.mutate('save-note',{draft:'held',color:0,text:'text'});
 assert.equal(sent[1].op,'status');
 assert.equal(sent[1].generation,'9007199254740993');
 transport.reply({request:sent[1].request,outcome:'refused'});
 await retry;
 const fresh=transport.mutate('save-note',{draft:'held',color:0,text:'text'});
 assert.equal(sent[2].generation,'9007199254740995');
 transport.reply({request:sent[2].request,outcome:'accepted'});
 await fresh;
});

for (const generation of [undefined,null,1,'01','-1','18446744073709551616','']) test(`operator transport fails closed for invalid originating generation ${generation}`,async()=>{
 const sent=[];
 const t=operatorTransport(m=>sent.push(m.request),{setTimeout(){throw Error('invalid request queued');},clearTimeout(){}});
 assert.equal(t.setAnnotationGeneration(generation),false);
 const result=await t.ask('stroke',{seg:1,id:'pen',color:0,pts:[[1,2,0]]});
 assert.equal(result.outcome,'refused');
 assert.equal(sent.length,0);
});

for (const operator of [false,true]) for (const access of ['own','other','legacy','off','view-only','paused','driving','off-driving','paused-driving','view-only-driving']) {
 test(`${operator?'operator':'viewer'} existing note ${access} obeys ownership in watch mode`,async()=>{
  const root=new Element('root'),box=new Element('box');root.append(box);
  const frames=[]; const posts=[];
  const doc={body:root,createElement:t=>new Element(t)};
  const win={innerWidth:100,crypto:{getRandomValues:b=>b.fill(1)},performance:{now:()=>0},requestAnimationFrame:fn=>frames.push(fn),addEventListener(){},setTimeout:()=>1,clearTimeout(){},clearInterval(){}};
  const owner=operator?'operator':'session';
  const note={id:'n',owner:access==='legacy'?'':access==='other'?'someone':owner,text:'original',region:[0,0,50,50],author:{name:'Ana',color:'#112233'}};
  const v={model:{seg:3,docSeg:1,notes:[{note,seg:1,state:'open'}],driver:access.includes('driving')?'Driver':'',paused:access.includes('paused')},onRecord(){},toPage:(x,y)=>[x,y],fromPage:(x,y)=>[x,y],boxOf:()=>null};
  const ui=mountAnnotation(doc,win,v,{root,box,stage:box,palette:['#112233'],owner,operator,annotate:!access.startsWith('view-only'),annotationAllowed:!access.startsWith('off'),who:{id:()=> 'attribution',name:()=>'',viewer:()=>({id:'attribution',name:''})},colors:{get:()=>0,set(){}},post:async(route,body)=>{posts.push({route,body});return {status:204};}});
  // Only pins are drawn; the canvas fixture need not implement painting.
  root.querySelector('.strokes').getContext=()=>({setTransform(){},clearRect(){}});
  while(frames.length)frames.shift()();
  const chip=root.querySelector('.chip');for(const fn of chip.listeners.click)fn({stopPropagation(){}});
  const canEdit=access==='own'||(!operator&&access==='driving');
  if(operator&&access.includes('driving')){
   assert.equal(ui.dialog.open,false);
   assert.equal(root.querySelector('.annotate').classList.contains('operator-driving'),true);
   assert.equal(root.querySelector('.annotate').classList.contains('operator-off'),true);
   assert.equal(root.querySelector('.note-dialog').hidden,true);
   assert.equal(posts.length,0);return;
  }
  assert.equal(ui.dialog.open,true);
  assert.equal(root.querySelector('.note-dialog').hidden,false);
  assert.equal(root.querySelector('.annotate').classList.contains('operator-off'),false);
  const input=root.querySelector('.note-input'), buttons=root.querySelector('.note-buttons'),save=buttons.children.at(-1);
  assert.equal(save.classList.contains('hidden'),!canEdit);
  if(!canEdit)return;
  assert.equal(input.value,'original');assert.equal(ui.mode,'watch');
  assert.equal(root.querySelector('.annotate').classList.contains('operator-off'),false);
  for(const fn of buttons.children[1].listeners.click)fn();assert.equal(posts.length,0);
  for(const fn of chip.listeners.click)fn({stopPropagation(){}});
  input.value='revised';for(const fn of input.listeners.input)fn();for(const fn of save.listeners.click)fn();
  assert.deepEqual(posts,[{route:'note-edit',body:{seg:3,id:'n',text:'revised'}}]);
  await Promise.resolve();assert.equal(ui.dialog.open,false);
 });
}

test('OFF pin inspection reveals the held draft for recovery without replacing its text',()=>{
 const root=new Element('root'),box=new Element('box');root.append(box);
 const frames=[];
 const doc={body:root,createElement:t=>new Element(t)};
 const win={innerWidth:100,crypto:{getRandomValues:b=>b.fill(1)},performance:{now:()=>0},requestAnimationFrame:fn=>frames.push(fn),addEventListener(){},setTimeout:()=>1,clearTimeout(){},clearInterval(){}};
 const note={id:'n',owner:'operator',text:'existing',region:[0,0,50,50],author:{name:'Operator',color:'#112233'}};
 const v={model:{seg:1,docSeg:1,notes:[{note,seg:1,state:'open'}]},onRecord(){},toPage:(x,y)=>[x,y],fromPage:(x,y)=>[x,y],boxOf:()=>null};
 const ui=mountAnnotation(doc,win,v,{root,box,stage:box,palette:['#112233'],operator:true,who:{id:()=> 'operator',name:()=> 'Operator',viewer:()=>({id:'operator',name:'Operator'})},colors:{get:()=>0,set(){}},post:()=>{throw Error('recovery must not post');}});
 root.querySelector('.strokes').getContext=()=>({setTransform(){},clearRect(){}});
 while(frames.length)frames.shift()();
 const payload={draft:'held'};ui.dialog.draft(payload);
 const input=root.querySelector('.note-input');input.value='unsaved draft';for(const fn of input.listeners.input)fn();
 ui.setAnnotationAllowed(false);
 const chip=root.querySelector('.chip');for(const fn of chip.listeners.click)fn({stopPropagation(){}});
 assert.equal(ui.dialog.draftPayload,payload);assert.equal(input.value,'unsaved draft');
 assert.equal(root.querySelector('.note-dialog').hidden,false);
 assert.match(root.querySelector('.note-msg').textContent,/cancel.*another/i);
 ui.setAnnotationAllowed(false);assert.equal(root.querySelector('.note-dialog').hidden,false,'projection refresh hid recovery');
 assert.ok(!root.querySelector('.note-author').textContent);
 ui.setAnnotationAllowed(true);assert.equal(input.value,'unsaved draft');assert.equal(ui.dialog.draftPayload,payload);
 const cancel=root.querySelector('.note-buttons').children[1];for(const fn of cancel.listeners.click)fn();
 for(const fn of chip.listeners.click)fn({stopPropagation(){}});
 assert.equal(input.value,'existing');assert.equal(ui.dialog.draftPayload,null);
});

test('annotation publishes one shared workspace and preserves unsaved draft authority',()=>{
 const root=new Element('shadow'),box=new Element('box');root.append(box);
 const doc={body:root,createElement:tag=>new Element(tag)};
 const win={innerWidth:800,innerHeight:600,crypto:{getRandomValues:b=>b.fill(1)},performance:{now:()=>0},requestAnimationFrame:()=>{},addEventListener(){},removeEventListener(){},setTimeout:()=>1,clearTimeout(){},clearInterval(){}};
 const view={model:{seg:1,docSeg:1,notes:[]},onRecord(){},toPage:(x,y)=>[x,y],fromPage:(x,y)=>[x,y]};
 const ui=mountAnnotation(doc,win,view,{root,box,stage:box,palette:['#ff0000'],who:{id:()=> 'operator',name:()=> 'Operator',viewer:()=>({id:'operator',name:'Operator'})},colors:{get:()=>0,set(){}},post:async()=>({status:204}),operator:true});
 assert.equal(typeof ui.notes?.setSketches,'function');
 assert.equal(ui.controls.pointer,root.querySelector('.mode'));
 assert.equal(ui.controls.note.textContent,'💭');
 assert.ok(ui.notes.body.contains(root.querySelector('.note-dialog')));
 ui.dialog.draft({draft:'held'});const input=root.querySelector('.note-input');input.value='still unsaved';for(const fn of input.listeners.input)fn();
 assert.deepEqual(ui.dialog.unsaved,{draft:true});
 ui.notes.showReadonly('finished sketch');assert.deepEqual(ui.dialog.unsaved,{draft:true});
 ui.setAnnotationAllowed(false);assert.equal(input.value,'still unsaved');
 ui.dispose();
});
