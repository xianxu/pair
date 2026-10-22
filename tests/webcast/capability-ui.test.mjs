import {test} from 'node:test';
import assert from 'node:assert/strict';
import {Element} from './dom-fixture.mjs';
import {mountAnnotate} from '../../cmd/internal/webcast/web/annotate.js';
import {mountDrive} from '../../cmd/internal/webcast/web/drive.js';

function harness() {
 const root=new Element('body'), box=new Element('box'),bar=new Element('bar');bar.className='bar';root.append(box,bar);
 const doc={body:root,createElement:tag=>new Element(tag),getElementById:()=>box,querySelector:s=>root.querySelector(s)};
 const timers=new Map(), posts=[];let serial=0, caps={annotation:{allowed:false},drive:{allowed:false}}, receiveCaps=[],receive=[];
 const win={innerWidth:100,crypto:{getRandomValues:b=>b.fill(1)},performance:{now:()=>100},requestAnimationFrame:()=>{},addEventListener:()=>{},setTimeout:fn=>{timers.set(++serial,fn);return serial;},clearTimeout:id=>timers.delete(id),setInterval:fn=>{timers.set(++serial,fn);return serial;},clearInterval:id=>timers.delete(id),fetch:async(url,init)=>{posts.push({url,body:init?.body,headers:init?.headers});return {ok:true,status:204,json:async()=>url==='palette.json'?({palette:['#ff0000'],annotate:true}):url==='save-note'?({state:'saved',id:'saved'}):({state:'ready',draft:'signed-capture',captured_at:'2026-10-13T12:00:00Z'}),text:async()=>''};}};
 const v={model:{seg:1,docSeg:1,live:true,notes:[]},onCapabilities:fn=>{receiveCaps.push(fn);fn(caps);},onRecord:fn=>receive.push(fn),toPage:(x,y)=>[x,y],fromPage:(x,y)=>[x,y],nodeAt:()=>7,boxOf:()=>[0,0,100,100]};
 return {root,box,doc,win,v,posts,timers,caps(a,d,g=1){caps={annotation:{allowed:a,state:a?'on':'off',generation:g},drive:{allowed:d,state:d?'on':'off',generation:g}};for(const fn of receiveCaps)fn(caps);},rec(r){for(const fn of receive)fn(r,v.model);},async event(el,k,extra={}){for(const fn of el.listeners[k]||[])await fn({target:el,pointerId:1,pointerType:'mouse',clientX:30,clientY:30,preventDefault(){},...extra});}};
}
test('late annotation mount takes latest permission; off cancels stroke and holds draft',async()=>{
 const h=harness();let palette;
 const original=h.win.fetch;h.win.fetch=(url,init)=>url==='palette.json'?new Promise(r=>palette=r):original(url,init);
 h.caps(true,false);const mounting=mountAnnotate(h.doc,h.win,h.v);h.caps(false,false,2);
 palette({ok:true,json:async()=>({palette:['#ff0000'],annotate:true})});const ui=await mounting;
 ui.setMode('point');assert.equal(ui.mode,'watch');assert.match(h.root.querySelector('.annotation-hint').textContent,/off/i);
 h.caps(true,false,3);ui.setMode('point');const layer=h.root.querySelector('.annotate');
 await h.event(layer,'pointerdown');await h.event(layer,'pointermove',{clientX:40});
 const posted=h.posts.length;h.caps(false,false,4);await h.event(layer,'pointerup');for(const fn of [...h.timers.values()])fn();
 assert.equal(ui.mode,'watch');assert.equal(h.posts.length,posted);assert.equal(layer.captured,null);
 h.caps(true,false,5);ui.setMode('note');await h.event(layer,'pointerdown');await h.event(layer,'pointerup');for(let i=0;i<12;i++)await Promise.resolve();const heldPosts=h.posts.length;
 const input=h.root.querySelector('.note-input');input.value='held text';await h.event(input,'input');
 h.caps(false,false,6);const save=h.root.querySelector('.note-buttons').children.at(-1);await h.event(save,'click');assert.equal(h.posts.length,heldPosts);
 h.caps(true,false,7);assert.equal(input.value,'held text');assert.equal(ui.mode,'watch');
 ui.who.setName('Viewer');await h.event(save,'click');for(let i=0;i<12;i++)await Promise.resolve();assert.equal(h.posts.length,heldPosts+1);assert.equal(h.posts.at(-1).url,'save-note');
 h.rec({k:'stroke',t:8,id:'other.s',color:'#ff0000',pts:[[1,2,0]]});assert.ok(ui.strokes.size>0);
 h.rec({k:'annotation-clear',t:9});assert.equal(ui.strokes.size,0);
});
test('drive off drops queued input and requires fresh consent after enable',async()=>{
 const h=harness();let release;
 h.win.fetch=async(url,init)=>{h.posts.push({url,body:JSON.parse(init.body),headers:init.headers});if(url==='drive/input')return new Promise(r=>release=r);return {status:204,text:async()=>''};};
 const ui=mountDrive(h.doc,h.win,h.v,{viewer:'me',key:'key'});
 await ui.control('request');assert.equal(h.posts.length,0,'default off posted');
 h.caps(false,true);await ui.control('request');h.rec({k:'ctl',op:'grant',driver:{id:'me'},pending:[]});assert.equal(ui.view().capture,true);
 const keys=h.root.querySelector('.drive-keys');keys.value='first';await h.event(keys,'input');keys.value='queued';await h.event(keys,'input');
 const old=h.posts.slice();assert.ok(old.length>=2);for(const p of old)assert.equal(p.headers['X-Couch-Drive-Generation'],'1');
 h.caps(false,false,2);assert.equal(ui.view().capture,false);assert.equal(keys.focused,false);
 h.caps(false,true,3);assert.equal(ui.view().capture,false,'old grant revived');
 release({status:403,text:async()=> 'old refusal'});await Promise.resolve();await Promise.resolve();for(const fn of [...h.timers.values()])fn();
 assert.equal(h.posts.filter(p=>p.url==='drive/input').length,1,'queued input escaped');
 await ui.control('request');assert.equal(h.posts.at(-1).headers['X-Couch-Drive-Generation'],'3');for(const p of old)assert.equal(p.headers['X-Couch-Drive-Generation'],'1');
 h.rec({k:'ctl',op:'grant',driver:{id:'me'},pending:[]});assert.equal(ui.view().capture,true);
 await ui.control('release');assert.equal(h.posts.at(-1).headers['X-Couch-Drive-Generation'],'3');
});

test('drive disconnect at the same generation can resume its existing grant',async()=>{
 const h=harness();h.caps(false,true,3);
 const ui=mountDrive(h.doc,h.win,h.v,{viewer:'me',key:'key'});
 h.rec({k:'ctl',driver:{id:'me'},pending:[]});assert.equal(ui.view().capture,true);
 const layer=h.root.querySelector('.drive');await h.event(layer,'pointerdown');
 h.caps(false,false,3);assert.equal(layer.captured,null);assert.equal(ui.view().capture,false);
 h.caps(false,true,3);h.rec({k:'ctl',driver:{id:'me'},pending:[]});assert.equal(ui.view().capture,true);
 h.caps(false,true,5);h.rec({k:'ctl',driver:{id:'me'},pending:[]});assert.equal(ui.view().capture,false,'coalesced off/on restored old consent');
});

test('current grant reaches initial and reconnect viewers in either SSE order',()=>{
 const h=harness();const ui=mountDrive(h.doc,h.win,h.v,{viewer:'me',key:'key'});
 h.rec({k:'ctl',driver:{id:'me'},pending:[]});assert.equal(ui.view().capture,false);
 h.caps(false,true,3);assert.equal(ui.view().capture,true,'initial grant before capability lost');
 h.caps(false,false,3);h.rec({k:'ctl',driver:{id:'me'},pending:[]});assert.equal(ui.view().capture,false);
 h.caps(false,true,3);assert.equal(ui.view().capture,true,'reconnect grant before capability lost');
 h.caps(false,false,4);h.rec({k:'ctl',driver:{id:'me'},pending:[]});h.caps(false,true,5);
 assert.equal(ui.view().capture,false,'prior generation revived');
});

test('annotation request carries originating generation and clear cancels held gesture before capability arrives',async()=>{
 const h=harness();h.caps(true,false,11);const ui=await mountAnnotate(h.doc,h.win,h.v);
 ui.setMode('point');const layer=h.root.querySelector('.annotate');
 await h.event(layer,'pointerdown');await h.event(layer,'pointermove',{clientX:40});
 for(const fn of [...h.timers.values()])fn();
 const request=h.posts.find(p=>p.url==='stroke');assert.ok(request);
 assert.equal(request.headers['X-Couch-Annotation-Generation'],'11');
 const before=h.posts.length;h.rec({k:'annotation-clear',t:10});
 assert.equal(ui.mode,'watch');assert.equal(layer.captured,null);
 await h.event(layer,'pointermove',{clientX:50});await h.event(layer,'pointerup');
 for(const fn of [...h.timers.values()])fn();assert.equal(h.posts.length,before);
 h.caps(true,false,13);assert.equal(request.headers['X-Couch-Annotation-Generation'],'11');
 ui.who.setName('Viewer');ui.setMode('note');await h.event(layer,'pointerdown');await h.event(layer,'pointerup');for(let i=0;i<12;i++)await Promise.resolve();
 const input=h.root.querySelector('.note-input');input.value='fresh save';await h.event(input,'input');
 await h.event(h.root.querySelector('.note-buttons').children.at(-1),'click');
 assert.equal(h.posts.at(-1).headers['X-Couch-Annotation-Generation'],'13');
});

test('sketch bootstrap current control reaches driving UI without historical ctl chatter',()=>{
 const h=harness();h.caps(false,true,3);
 const ui=mountDrive(h.doc,h.win,h.v,{viewer:'me',key:'key'});
 h.rec({k:'sketch-snapshot',control:{driver:{id:'me'},pending:[]}});
 assert.equal(ui.view().capture,true);
 h.rec({k:'sketch-snapshot',control:{pending:[{id:'me'}]}});
 assert.equal(ui.view().capture,false);assert.equal(ui.view().state,'asking');
 h.rec({k:'ctl',driver:{id:'other',name:'Other'},pending:[]});
 assert.equal(ui.view().state,'other');
});
test('sketch suspension stops driving input without surrendering the seat', async () => {
  const h = harness();
  h.caps(false, true, 3);
  const ui = mountDrive(h.doc, h.win, h.v, { viewer: 'me', key: 'key' });
  h.rec({ k: 'ctl', driver: { id: 'me' }, pending: [] });
  const layer = h.root.querySelector('.drive'),
    keys = h.root.querySelector('.drive-keys');
  ui.setSketching(true);
  assert.equal(layer.classList.contains('on'), false);
  keys.value = 'must not drive';
  await h.event(keys, 'input');
  assert.equal(h.posts.length, 0);
  assert.equal(ui.view().state, 'driving');
  ui.setSketching(false);
  assert.equal(layer.classList.contains('on'), true);
  keys.value = 'now drive';
  await h.event(keys, 'input');
  assert.equal(h.posts.at(-1).url, 'drive/input');
});
test('annotation semantic mode gate defers both toolbar and API changes', async () => {
  const h = harness();
  h.caps(true, false);
  const modes = [];
  h.v.beforeAnnotationMode = (mode) => {
    modes.push(mode);
    return false;
  };
  const ui = await mountAnnotate(h.doc, h.win, h.v);
  ui.setMode('point');
  assert.equal(ui.mode, 'watch');
  await h.event(h.root.querySelector('.mode'), 'click');
  assert.equal(ui.mode, 'watch');
  assert.ok(modes.includes('point'));
  h.v.beforeAnnotationMode = () => true;
  ui.setMode('note');
  assert.equal(ui.mode, 'note');
});
