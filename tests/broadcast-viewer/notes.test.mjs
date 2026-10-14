// Sticky notes on the viewer page (#447): hit-testing, placement, the byte
// cap, list parsing, and the dialog. Run by TestViewerNode via `node --test`.
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { noteAt, boxHasPin, noteBytes, noteVisible, parseNotes, noteDialog, MAX_NOTE_BYTES } from '../../cmd/internal/broadcast/web/notes.js';

const list = { guard: 12, notes: [{ id: 1, col: 10, row: 5, color: 0, text: 'a' }, { id: 2, col: 78, row: 2, color: 1, text: 'b' }] };

test('a tap on either cell of a 💭 hits it', () => {
  assert.equal(noteAt(list, 10, 5, 80, 24).id, 1);
  assert.equal(noteAt(list, 11, 5, 80, 24).id, 1);
  assert.equal(noteAt(list, 12, 5, 80, 24), null);
  assert.equal(noteAt(list, 9, 5, 80, 24), null);
  assert.equal(noteAt(list, 10, 6, 80, 24), null);
});

test('a hidden note is not hit', () => {
  // The grid shrank: note 2's second cell is off the grid.
  assert.equal(noteAt(list, 78, 2, 79, 24), null);
  // Row 5 became the last row and col 10 is inside the guard.
  assert.equal(noteAt(list, 10, 5, 80, 6), null);
  assert.ok(noteVisible({ col: 12, row: 5 }, 80, 6, 12));
});

test('a box has a pin spot as Couch decides (the shared case table)', () => {
  const table = JSON.parse(readFileSync(new URL('./pinspots.json', import.meta.url), 'utf8'));
  for (const c of table.cases) {
    assert.equal(boxHasPin(c.box, c.cols, c.rows, table.guard), c.has, c.name);
  }
});

test('the byte cap counts UTF-8 like Couch', () => {
  assert.equal(noteBytes('é'), 2);
  assert.equal(noteBytes('é'.repeat(512)), MAX_NOTE_BYTES);
  assert.ok(noteBytes('é'.repeat(513)) > MAX_NOTE_BYTES);
});

test('parseNotes keeps only well-formed entries', () => {
  const box = [0, 1, 4, 2];
  const got = parseNotes({ guard: 12, notes: [{ id: 1, col: 1, row: 1, box, color: 0, text: 'ok', extra: 'x' }, { id: '2', col: 1, row: 1, box, color: 0, text: 'x' },
    { id: 3, col: 1.5, row: 1, box, color: 0, text: 'x' }, { id: 4, col: 1, row: 1, box, color: 0, text: 5 }, null,
    { id: 5, col: 1, row: 1, color: 0, text: 'no box' }, { id: 6, col: 1, row: 1, box: [0, 1, 0, 2], color: 0, text: 'empty box' },
    { id: 8, col: 1, row: 1, box: [0, 1, 4], color: 0, text: 'short box' }] });
  assert.deepEqual(got, { guard: 12, notes: [{ id: 1, col: 1, row: 1, box, color: 0, text: 'ok' }] });
  for (const junk of [null, 'x', {}, { guard: -1, notes: [] }, { guard: 1, notes: 'x' }]) {
    assert.deepEqual(parseNotes(junk), { guard: 0, notes: [] });
  }
});

// A fake element: enough of the DOM for the dialog, and a guard that fails
// if anything assigns innerHTML.
function el() {
  const classes = new Set();
  const listeners = {};
  const e = {
    textContent: '',
    value: '',
    disabled: false,
    classList: {
      add: (c) => classes.add(c),
      remove: (c) => classes.delete(c),
      toggle: (c, on) => (on ? classes.add(c) : classes.delete(c)),
      has: (c) => classes.has(c),
    },
    addEventListener: (n, f) => { listeners[n] = f; },
    fire: (n, ev = {}) => listeners[n](ev),
  };
  Object.defineProperty(e, 'innerHTML', { set() { throw new Error('innerHTML assigned'); } });
  return e;
}

function dialog(respond = () => ({ status: 204, text: '' })) {
  const els = { root: el(), text: el(), input: el(), count: el(), msg: el(), save: el(), cancel: el(), resolve: el() };
  els.root.classList.add('hidden');
  const posts = [];
  const d = noteDialog(els, {
    post: (route, body) => { posts.push({ route, body }); return Promise.resolve(respond(route, body)); },
    color: () => 3,
  });
  const type = (text) => { els.input.value = text; els.input.fire('input'); };
  const settle = () => new Promise((r) => setTimeout(r, 0));
  return { d, els, posts, type, settle };
}

test('a draft is saved with its cell, grid, color and text, then closes', async () => {
  const f = dialog();
  f.d.draft({ cols: 80, rows: 24, col: 4, row: 2 });
  assert.ok(!f.els.root.classList.has('hidden'));
  assert.ok(f.els.save.disabled, 'empty draft cannot be saved');
  f.type('look <b>here</b>');
  assert.equal(f.posts.length, 0, 'typing posts nothing');
  assert.equal(f.els.count.textContent, `16 / ${MAX_NOTE_BYTES}`);
  f.els.save.fire('click');
  await f.settle();
  assert.deepEqual(f.posts, [{ route: 'note', body: { cols: 80, rows: 24, col: 4, row: 2, color: 3, text: 'look <b>here</b>' } }]);
  assert.ok(f.els.root.classList.has('hidden'));
  assert.ok(!f.d.open);
});

test('Save is disabled for blank text and over the byte cap', () => {
  const f = dialog();
  f.d.draft({ cols: 80, rows: 24, col: 0, row: 0 });
  f.type('   ');
  assert.ok(f.els.save.disabled);
  f.type('é'.repeat(513));
  assert.ok(f.els.save.disabled);
  f.type('é'.repeat(512));
  assert.ok(!f.els.save.disabled);
});

test('a refusal shows Couch\'s sentence as text and keeps the draft', async () => {
  const f = dialog(() => ({ status: 409, text: 'This color already has 8 notes. Resolve one first.\n' }));
  f.d.draft({ cols: 80, rows: 24, col: 0, row: 0 });
  f.type('hi');
  f.els.save.fire('click');
  await f.settle();
  assert.equal(f.els.msg.textContent, 'This color already has 8 notes. Resolve one first.');
  assert.equal(f.els.input.value, 'hi');
  assert.ok(f.d.open);
});

test('Cancel discards a draft without a request', () => {
  const f = dialog();
  f.d.draft({ cols: 80, rows: 24, col: 0, row: 0 });
  f.type('never sent');
  f.els.cancel.fire('click');
  assert.equal(f.posts.length, 0);
  assert.equal(f.els.input.value, '');
  assert.ok(f.els.root.classList.has('hidden'));
});

test('viewing renders text literally; Resolve only when allowed', async () => {
  const f = dialog();
  f.d.view({ id: 7, text: '<b>x</b>' }, { resolvable: false });
  assert.equal(f.els.text.textContent, '<b>x</b>');
  assert.ok(f.els.resolve.classList.has('hidden'));
  assert.ok(f.els.save.classList.has('hidden'));
  f.d.view({ id: 7, text: 'x' }, { resolvable: true });
  assert.ok(!f.els.resolve.classList.has('hidden'));
  f.els.resolve.fire('click');
  await f.settle();
  assert.deepEqual(f.posts, [{ route: 'resolve', body: { id: 7 } }]);
  assert.ok(!f.d.open);
});

test('a viewed note resolved elsewhere closes its dialog', () => {
  const f = dialog();
  f.d.view({ id: 7, text: 'x' }, { resolvable: true });
  f.d.update({ guard: 12, notes: [{ id: 7, col: 0, row: 0, color: 0, text: 'x' }] });
  assert.ok(f.d.open);
  f.d.update({ guard: 12, notes: [] });
  assert.ok(!f.d.open);
});

test('Escape closes the dialog', () => {
  const f = dialog();
  f.d.draft({ cols: 80, rows: 24, col: 0, row: 0 });
  f.els.root.fire('keydown', { key: 'Escape' });
  assert.ok(!f.d.open);
});

test('an answer applies only to the dialog that sent it (#447 M2 review)', async () => {
  let release;
  const els = { root: el(), text: el(), input: el(), count: el(), msg: el(), save: el(), cancel: el(), resolve: el() };
  const d = noteDialog(els, { post: () => new Promise((r) => { release = r; }), color: () => 0 });
  d.draft({ cols: 80, rows: 24, col: 0, row: 0 });
  els.input.value = 'first';
  els.input.fire('input');
  els.save.fire('click');
  els.cancel.fire('click');
  d.draft({ cols: 80, rows: 24, col: 5, row: 5 });
  els.input.value = 'second draft';
  els.input.fire('input');
  release({ status: 204, text: '' });
  await new Promise((r) => setTimeout(r, 0));
  assert.ok(d.open, 'the old 204 must not close the new draft');
  assert.equal(els.input.value, 'second draft');
  assert.equal(els.msg.textContent, '');
});

test('viewing a note moves focus into the dialog so Escape reaches it', () => {
  const els = { root: el(), text: el(), input: el(), count: el(), msg: el(), save: el(), cancel: el(), resolve: el() };
  let focused = null;
  els.cancel.focus = () => { focused = 'cancel'; };
  const d = noteDialog(els, { post: () => Promise.resolve({ status: 204 }), color: () => 0 });
  d.view({ id: 1, text: 'x' }, { resolvable: false });
  assert.equal(focused, 'cancel');
  els.root.fire('keydown', { key: 'Escape' });
  assert.ok(!d.open);
});

// pair#499: a paused web cast holds the dialog. A held draft can't be saved,
// says why and keeps its text; a held view's Resolve is off and a click posts
// nothing; releasing the hold restores both and clears the reason.
test('hold disables Save and Resolve, keeps a draft, and releases', async () => {
  const f = dialog();
  f.d.draft({ cols: 80, rows: 24, col: 0, row: 0 });
  f.type('kept');
  f.d.hold('The cast is paused.');
  assert.ok(f.els.save.disabled, 'a held draft cannot be saved');
  assert.equal(f.els.msg.textContent, 'The cast is paused.');
  f.els.save.fire('click');
  await f.settle();
  assert.equal(f.posts.length, 0, 'a held Save posts nothing');
  assert.equal(f.els.input.value, 'kept', 'the draft keeps its text');
  f.d.hold('');
  assert.ok(!f.els.save.disabled, 'released, Save is back');
  assert.equal(f.els.msg.textContent, '');

  f.d.close(); // Explicitly discard the draft before checking a held read-only view.
  f.d.hold('The cast is paused.');
  f.d.view({ id: 7, text: 'x' }, { resolvable: true });
  assert.ok(f.els.resolve.disabled, 'a view opened while held has Resolve off');
  assert.equal(f.els.msg.textContent, 'The cast is paused.');
  f.els.resolve.fire('click');
  await f.settle();
  assert.equal(f.posts.length, 0, 'a held Resolve posts nothing');
  f.d.hold('');
  assert.ok(!f.els.resolve.disabled);
  f.els.resolve.fire('click');
  await f.settle();
  assert.equal(f.posts.length, 1, 'released, Resolve posts');
});

import * as collab from '../../cmd/internal/broadcast/web/collab.js';

test('dialog transition preserves unsent edits, pending identity and held results', () => {
  assert.equal(typeof collab.noteDialogTransition, 'function');
  const step = collab.noteDialogTransition;
  let s = step(null, { type: 'open-edit', note: { id: 7, text: 'old' } }).state;
  let out = step(s, { type: 'save' });
  assert.equal(out.effect.kind, 'save-edit');
  s = out.state;
  const pending = s.pending;
  s = step(s, { type: 'input', text: 'newer' }).state;
  assert.equal(step(s, { type: 'save' }).effect, null);
  s = step(s, { type: 'hold', reason: 'paused' }).state;
  assert.equal(s.pending, pending);
  s = step(s, { type: 'result', pending, status: 204 }).state;
  assert.equal(s.mode, 'edit');
  assert.equal(s.text, 'newer');
  assert.match(s.error, /earlier/i);
  s = step(s, { type: 'hold', reason: '' }).state;
  out = step(s, { type: 'save' });
  const stale = out.state.pending;
  s = step(out.state, { type: 'retire' }).state;
  s = step(s, { type: 'open-edit', note: { id: 8, text: 'another' } }).state;
  assert.equal(step(s, { type: 'result', pending: stale, status: 204 }).state, s);
});

test('edit save sends only id and text; pending input cannot send twice', async () => {
  let release;
  const f = dialog(() => new Promise(r => { release = r; }));
  assert.equal(typeof f.d.edit, 'function');
  f.d.edit({ id: 7, text: 'old', author: 'Ana', color: 4 });
  assert.equal(f.els.input.value, 'old');
  f.type('revised'); f.els.save.fire('click');
  assert.equal(f.els.input.readOnly,false,'existing edits retain input while pending');
  f.type('unsent'); f.els.save.fire('click');
  assert.deepEqual(f.posts, [{route:'note-edit',body:{id:7,text:'revised'}}]);
  release({ status:204 }); await f.settle();
  assert.equal(f.els.input.value, 'unsent'); assert.equal(f.d.open,true);
  f.d.update({notes:[]}); assert.equal(f.d.open,false);
});

test('readonly refresh shows edited text but edit refresh preserves typing', () => {
  const f = dialog();
  f.d.view({id:7,text:'old'}, {resolvable:false});
  f.d.update({notes:[{id:7,text:'changed',edited:true}]});
  assert.match(f.els.text.textContent, /changed.*\(edited\)/);
  f.d.edit({id:7,text:'changed'}); f.type('unsent');
  f.d.update({notes:[{id:7,text:'remote'}]});
  assert.equal(f.els.input.value,'unsent');
  f.els.cancel.fire('click'); assert.equal(f.posts.length,0);
});

test('session owner reads only a successful session header and fails closed', async () => {
  assert.equal(typeof collab.sessionNoteOwner,'function');
  const calls=[];
  assert.equal(await collab.sessionNoteOwner(async(...args)=>{calls.push(args);return {status:204,headers:{get:k=>k==='X-Couch-Note-Owner'?'owner':''}};}),'owner');
  assert.equal(calls[0][0],'session'); assert.equal(calls[0][1].credentials,'same-origin'); assert.equal(calls[0][1].cache,'no-store');
  assert.equal(await collab.sessionNoteOwner(async()=>{throw Error('offline');}),'');
  assert.equal(await collab.sessionNoteOwner(async()=>({status:401})),'');
});

test('terminal notes retain validated owner and edited metadata',()=>{
  const n={id:1,col:1,row:1,box:[0,0,3,3],color:0,text:'x',owner:'mine',edited:true};
  assert.deepEqual(parseNotes({guard:0,notes:[n]}).notes,[n]);
});

for (const status of [409, undefined]) test(`dialog result ${status} preserves text and permits a deliberate retry`,()=>{
 const step=collab.noteDialogTransition;
 let s=step(null,{type:'open-edit',note:{id:7,text:'old'}}).state;
 s=step(s,{type:'input',text:'submitted'}).state;
 s=step(s,{type:'save'}).state;const pending=s.pending;
 s=step(s,{type:'input',text:'newer'}).state;
 s=step(s,{type:'result',pending,status,text:'Not saved'}).state;
 assert.equal(s.pending,null);assert.equal(s.text,'newer');assert.equal(s.error,'Not saved');
 const next=step(s,{type:'save'});assert.equal(next.effect.text,'newer');assert.notEqual(next.state.pending,pending);
 assert.equal(step(next.state,{type:'result',pending,status:204}).state,next.state);
});

test('an edited existing note labels its editor and skips beforeSave',()=>{
 const els={root:el(),text:el(),input:el(),count:el(),msg:el(),save:el(),cancel:el(),resolve:el()};
 const posts=[];
 const d=noteDialog(els,{post:(...args)=>{posts.push(args);return Promise.resolve({status:204});},beforeSave:()=>{throw Error('editing asked for attribution');}});
 d.edit({id:1,text:'updated',edited:true});
 assert.match(els.count.textContent,/edited/);
 els.save.fire('click');assert.deepEqual(posts,[['note-edit',{id:1,text:'updated'}]]);
});

test('opening another note cannot replace a held draft or its pending request',async()=>{
 let release;
 const f=dialog(()=>new Promise(r=>{release=r;}));
 const payload={draft:'held'};
 f.d.draft(payload);f.type('unsaved draft');f.els.save.fire('click');
 f.d.hold('Annotation is off.');
 assert.equal(f.d.view({id:9,text:'existing'},{resolvable:true}),false);
 assert.equal(f.els.input.value,'unsaved draft');assert.equal(f.d.draftPayload,payload);
 assert.match(f.els.msg.textContent,/cancel.*another/i);
 f.d.hold('');assert.equal(f.els.save.disabled,true);
 release({status:409,text:'Off'});await f.settle();
 assert.equal(f.els.input.value,'unsaved draft');assert.equal(f.d.draftPayload,payload);
 f.els.cancel.fire('click');
 assert.equal(f.d.view({id:9,text:'existing'},{resolvable:true}),true);
 assert.equal(f.els.text.textContent,'existing');
});

test('pending creation ignores input and accepted result cannot create another note',()=>{
 const step=collab.noteDialogTransition;
 let s=step(null,{type:'open-draft',payload:{draft:'one-use'}}).state;
 s=step(s,{type:'input',text:'submitted'}).state;
 const sent=step(s,{type:'save'});assert.equal(sent.effect.kind,'save-new');
 s=step(sent.state,{type:'input',text:'typed during creation'}).state;
 assert.equal(s,sent.state,'pending creation input must be an identity transition');
 s=step(s,{type:'result',pending:sent.state.pending,status:204}).state;
 assert.equal(s.mode,'closed');assert.equal(step(s,{type:'save'}).effect,null);
});

test('mounted delayed creation locks text and sends only one POST after acceptance',async()=>{
 let release;
 const f=dialog(()=>new Promise(r=>{release=r;}));
 f.d.draft({draft:'one-use'});f.type('submitted');f.els.save.fire('click');
 assert.equal(f.els.input.readOnly,true);
 f.type('typed during creation');f.els.save.fire('click');
 assert.equal(f.els.input.value,'submitted');assert.equal(f.posts.length,1);
 release({status:204});await f.settle();
 assert.equal(f.d.open,false);assert.equal(f.els.input.readOnly,false);
 f.els.save.fire('click');assert.equal(f.posts.length,1);
});

for(const status of [409,undefined]) test(`creation ${status} releases text for correction and retry`,async()=>{
 let release;
 const f=dialog(()=>new Promise(r=>{release=r;}));
 f.d.draft({draft:'one-use'});f.type('submitted');f.els.save.fire('click');
 assert.equal(f.els.input.readOnly,true);
 release({status,text:'Retry'});await f.settle();
 assert.equal(f.els.input.readOnly,false);assert.equal(f.els.input.value,'submitted');
 f.type('corrected');f.els.save.fire('click');
 assert.equal(f.posts.length,2);assert.equal(f.posts[1].body.text,'corrected');
});
