import { test } from 'node:test';
import assert from 'node:assert/strict';
import { mountOperator, operatorTransport } from '../../cmd/internal/webcast/operator.js';

function harness() {
  const sent = [], timers = new Map();
  let serial = 0;
  const win = {
    setTimeout(fn) { timers.set(++serial, fn); return serial; },
    clearTimeout(id) { timers.delete(id); },
    addEventListener() {}, scrollX: 0, scrollY: 0,
  };
  const report = (message) => sent.push(message.request);
  const expire = () => { const [id, fn] = timers.entries().next().value; timers.delete(id); fn(); };
  return { sent, timers, win, report, expire };
}
const note = (draft) => ({ draft, color: 0, text: draft });

for (const replacement of [0, 2, 'ended']) {
  test(`scope ${replacement} retires pending requests without accepting or resending writes`, async () => {
    const h = harness(), transport = operatorTransport(h.report, h.win);
  transport.setAnnotationGeneration('1');
    transport.setScope(1);
    const old = transport.mutate('save-note', note('old'));
    const prepare = transport.ask('prepare-note', { seg: 1 });
    transport.setScope(replacement === 'ended' ? 1 : replacement, replacement === 'ended');
    assert.equal(h.timers.size, 0);
    assert.equal((await old).status, 409);
    assert.match((await prepare).error, /document|stale/i);
    assert.equal(h.sent.length, 2, 'retirement submitted a write');
    transport.setScope(1);
    const fresh = transport.mutate('save-note', note('fresh'));
    const freshID = h.sent.at(-1).request;
    assert.equal(h.sent.at(-1).op, 'save-note');
    assert.notEqual(freshID, h.sent[0].request);
    transport.reply({ request: h.sent[0].request, outcome: 'accepted' });
    assert.equal(h.timers.size, 1, 'old reply settled the fresh request');
    transport.reply({ request: freshID, outcome: 'accepted' });
    assert.equal((await fresh).status, 204);
  });
}

test('retired accepted completion cannot accept a stale caller or clear the new unsettled write', async () => {
  const h = harness(), transport = operatorTransport(h.report, h.win);
  transport.setAnnotationGeneration('1');
  transport.setScope(1);
  const old = transport.mutate('save-note', note('old'));
  transport.reply({ request: h.sent[0].request, outcome: 'accepted' });
  transport.setScope(0);
  transport.setScope(1);
  const fresh = transport.mutate('save-note', note('fresh'));
  assert.equal((await old).status, 409);
  h.expire();
  assert.equal((await fresh).status, 409);
  const retry = transport.mutate('save-note', note('fresh'));
  assert.equal(h.sent.at(-1).op, 'status');
  assert.equal(h.sent.at(-1).request, h.sent[1].request);
  transport.reply({ request: h.sent.at(-1).request, outcome: 'accepted' });
  assert.equal((await retry).status, 204);
});

function mounted() {
  const h = harness();
  let options, receive = () => {};
  const node = () => ({ style: {}, appendChild() {} });
  mountOperator({ host: node(), root: node(), report: h.report }, {
    annotate: { mountAnnotation(doc, win, view, opts) {
      options = opts;
      view.onRecord((...args) => receive(...args));
      return { strokes:{clear(){}}, setAnnotationAllowed() {}, setMode() {} };
    } }, palette: [], css: '',
  }, { createElement: node }, h.win);
  h.win.__couchcastAnnotationAllowed(true, '1');
  const snapshot = (revision, doc, extra = {}) => h.win.__couchcastOperatorView({feedback_mask:{scope:"cast",revision:"0",sketches:[],notes:[]},
    revision, doc, seg: revision, t: revision, notes: [], strokes: [], ...extra,
  });
  return { ...h, options, snapshot, onView(fn) { receive = fn; } };
}

test('adapter retires a lost save reply on switch and return before delivering the new view', async () => {
  const h = mounted();
  h.snapshot(1, 10);
  const old = h.options.post('note', note('old'));
  h.expire();
  assert.equal((await old).status, 409);
  h.snapshot(2, 0);
  let fresh;
  h.onView(() => { fresh = h.options.post('note', note('fresh')); });
  h.snapshot(3, 10);
  assert.equal(h.sent.at(-1).op, 'save-note', 'fresh document retried the dead operation');
  assert.notEqual(h.sent.at(-1).request, h.sent[0].request);
  h.win.__couchcastOperatorReply({ request: h.sent[0].request, outcome: 'accepted' });
  assert.equal(h.timers.size, 1);
  h.win.__couchcastOperatorReply({ request: h.sent.at(-1).request, outcome: 'accepted' });
  assert.equal((await fresh).status, 204);
});

test('adapter retains reconciliation through pause, driving and rejected older snapshots', async () => {
  const h = mounted();
  h.snapshot(1, 10);
  const old = h.options.post('note', note('old'));
  h.expire();
  assert.equal((await old).status, 409);
  h.snapshot(2, 10, { paused: true });
  h.snapshot(3, 10, { driver: 'Ana' });
  h.snapshot(2, 0);
  h.snapshot(4, 10);
  const retry = h.options.post('note', note('old'));
  assert.equal(h.sent.at(-1).op, 'status');
  assert.equal(h.sent.at(-1).request, h.sent[0].request);
  h.win.__couchcastOperatorReply({ request: h.sent[0].request, outcome: 'accepted' });
  assert.equal((await retry).status, 204);
});

test('adapter end retires requests even when the document number is retained', async () => {
  const h = mounted();
  h.snapshot(1, 10);
  const old = h.options.post('note', note('old'));
  h.snapshot(2, 10, { ended: true });
  assert.equal(h.timers.size, 0);
  assert.equal((await old).status, 409);
});

test('operator edit adapter preserves uncertain edit identity and submits only edit fields',async()=>{
 const h=mounted();h.snapshot(1,10);
 const body={seg:1,id:'note',text:'revised'};
 const first=h.options.post('note-edit',body);
 assert.equal(h.sent.at(-1).op,'edit-note');assert.deepEqual(h.sent.at(-1).body,body);
 const id=h.sent.at(-1).request;h.expire();await first;
 const retry=h.options.post('note-edit',body);assert.equal(h.sent.at(-1).op,'status');assert.equal(h.sent.at(-1).request,id);
 h.win.__couchcastOperatorReply({request:id,outcome:'accepted'});assert.equal((await retry).status,204);
});

test('operator shared sketch and annotation record subscribers coexist',()=>{
 const h=harness();const counts=[0,0];const node=()=>({style:{},appendChild(){}});
 mountOperator({host:node(),root:node(),report:h.report},{
  annotate:{mountAnnotation(doc,win,view){view.onRecord(()=>counts[0]++);return {strokes:{clear(){}},setAnnotationAllowed(){},setMode(){}};}},
  palette:[],css:'',sketch:{css:'',shellURL:'data:text/html,private',mount({view}){view.onRecord(()=>counts[1]++);return {tool:{}};}},
 },{createElement:node},h.win);
 h.win.__couchcastOperatorView({feedback_mask:{scope:"cast",revision:"0",sketches:[],notes:[]},revision:1,doc:1,seg:1,t:1,notes:[],strokes:[]});
 assert.deepEqual(counts,[1,1]);
});
