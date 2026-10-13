import { operatorTransport } from '../../cmd/internal/webcast/operator.js';
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { annotationPolicy } from '../../cmd/internal/webcast/web/annotation-policy.js';
import { mountAnnotation } from '../../cmd/internal/webcast/web/annotate.js';

class Element {
  constructor(tag) {
    this.tagName = tag; this.children = []; this.listeners = {}; this.dataset = {};
    this.style = { setProperty() {} }; this.value = ''; this.className = '';
    this.classList = { toggle: (v,on) => { const set = new Set(this.className.split(' ')); on ? set.add(v) : set.delete(v); this.className = [...set].join(' '); }, add: (v) => this.classList.toggle(v,true), remove: (v) => this.classList.toggle(v,false) };
  }
  append(...es) { this.children.push(...es); }
  replaceChildren(...es) { this.children = es; }
  setAttribute() {}
  addEventListener(k,f) { (this.listeners[k] ||= []).push(f); }
  querySelectorAll(s) { return this.children.flatMap((e) => [...(s === 'button' && e.tagName === 'button' ? [e] : []), ...e.querySelectorAll(s)]); }
  querySelector(s) { return this.children.find((e) => e.className.split(' ').includes(s.slice(1))) || this.children.map((e) => e.querySelector(s)).find(Boolean); }
  getBoundingClientRect() { return {left:0,top:0,right:100,bottom:100,width:100,height:100}; }
  focus() { this.focused=true; }
  blur() { this.focused=false; }
}

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
  const input = root.querySelector('.note-input'); input.value='keep this';
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
  const first = t.mutate('save-note', {draft:'d',text:'one',color:0});
  timers.values().next().value();
  assert.equal((await first).status, 409);
  const retry = t.mutate('save-note', {draft:'d',text:'edited',color:0});
  assert.equal(sent[1].op, 'status');
  assert.equal(sent[1].request, sent[0].request);
  t.reply({request:sent[1].request,outcome:'accepted'});
  assert.equal((await retry).status, 204);
  const next = t.mutate('save-note', {draft:'e',text:'two',color:0});
  assert.notEqual(sent[2].request, sent[0].request);
  t.reply({request:sent[2].request,outcome:'refused',error:'paused'});
  assert.equal((await next).text, 'paused');
});
