import { test } from 'node:test';
import assert from 'node:assert/strict';
import { Element } from './dom-fixture.mjs';
import { Model } from '../../cmd/internal/webcast/web/model.js';
const mod = () => import('../../cmd/internal/webcast/web/sketch.js');
function harness() {
  const context = {
    fills: [],
    globalAlpha: 1,
    save() {},
    restore() {},
    beginPath() {},
    paintedSegments: 0,
    moveTo() {
      this.paintedSegments++;
    },
    lineTo() {},
    closePath() {},
    fill() {
      this.fills.push(this.globalAlpha);
    },
    paintedPoints: 0,
    arcs: [],
    arc(x, y, radius) {
      this.arcs.push({ radius, alpha: this.globalAlpha });
      this.paintedPoints++;
    },
    clearRect() {},
    setTransform() {},
  };
  class E extends Element {
    appendChild(n) {
      this.append(n);
      n.parentNode = this;
      return n;
    }
    remove() {
      if (this.parentNode) this.parentNode.children = this.parentNode.children.filter((n) => n !== this);
    }
    getContext() {
      return context;
    }
  }
  const root = new E('root'),
    box = new E('box');
  root.appendChild(box);
  const doc = { body: root, createElement: (tag) => new E(tag) };
  let now = 0,
    serial = 0,
    sequenceTime = 3;
  const timers = new Map(),
    frames = [],
    subscriptions = [],
    sent = [];
  const m = new Model();
  for (const r of [
    { k: 'head', t: 0, format: 'couch-webcast', v: 1, id: 'ui', started: 'now' },
    {
      k: 'seg',
      t: 1,
      seg: 1,
      why: 'live',
      tab: { id: 'tab', url: 'https://old/' },
      vp: { w: 100, h: 100, dpr: 1 },
    },
    {
      k: 'doc',
      t: 2,
      doc: 1,
      url: 'https://old/',
      tree: { id: 1, y: 'doc', c: [{ id: 7, y: 'el', n: 'body' }] },
    },
  ])
    m.apply(r);
  const view = {
    get model() {
      return m;
    },
    onRecord: (fn) => subscriptions.push(fn),
    placementOf: (p) => p.at?.box || null,
    fromPage: (x, y) => [x, y],
    toPage: (x, y) => [x, y],
  };
  let authority = { allowed: true, live: true, ended: false, epoch: 1 },
    unknown = false;
  const prepared = [],
    modes = [],
    tools = [];
  const scene = {
    ready: false,
    source: null,
    async preflight() {},
    async prepare(source, ctx) {
      prepared.push([source, ctx]);
      this.source = source;
      this.ready = true;
    },
    async hitTest() {
      return { node: 7, box: [0, 0, 100, 100] };
    },
    async placement(p) {
      return p.at?.box || null;
    },
    toPage: (x, y) => [x, y],
    fromPage: (x, y) => [x, y],
    dispose() {
      this.ready = false;
    },
  };
  const emit = (r) => {
    m.apply(r);
    for (const fn of subscriptions) fn(r, m);
  };
  async function send(kind, body) {
    sent.push({ kind, body: structuredClone(body) });
    if (unknown && kind === 'sketch-points')
      return { operation: body.operation, sketch: body.sketch, sequence: body.sequence, outcome: 'unknown' };
    const op = { operation: body.operation, sequence: body.sequence, digest: 'a'.repeat(64) };
    sequenceTime = Math.max(sequenceTime, m.t + 1);
    let record = { k: kind, t: sequenceTime++, ...op };
    if (kind === 'sketch-begin')
      record = {
        ...record,
        id: 'server-s',
        author: { id: 'operator', color: '#ff0000' },
        owner: 'operator',
        doc: 1,
        tab: 'tab',
        vp: { w: 100, h: 100, dpr: 1 },
      };
    else {
      record.sketch = body.sketch;
      if (body.id) record.id = body.id;
      if (kind === 'sketch-stroke-begin')
        Object.assign(record, { at: { node: body.node, box: body.box }, width: 3 });
      if (kind === 'sketch-points') Object.assign(record, { chunk: body.chunk, pts: body.pts });
      if (kind === 'sketch-stroke-end' && body.truncated) record.truncated = true;
      if (kind === 'sketch-label') record.label = body.label;
    }
    emit(record);
    const p = m.sketchProjection().sketches.find((s) => s.begin.id === (body.sketch || 'server-s'));
    return {
      operation: body.operation,
      sketch: p.begin.id,
      sequence: body.sequence,
      next_sequence: p.next_sequence,
      outcome: 'accepted',
      source:
        kind === 'sketch-begin'
          ? { handle: 'h', time: record.t, length: 100, doc: 1, tab: 'tab', vp: { w: 100, h: 100, dpr: 1 } }
          : undefined,
      status: {
        phase: p.phase,
        reason: p.reason,
        label: p.label,
        open_stroke: p.strokes.find((s) => !s.sealed)?.begin.id || '',
        visible_top: p.visible.at(-1) || '',
        redo_top: p.redo.at(-1) || '',
      },
    };
  }
  const win = {
    devicePixelRatio: 1,
    requestAnimationFrame: (fn) => {
      frames.push(fn);
      return frames.length;
    },
    setTimeout: (fn, ms) => {
      const id = ++serial;
      timers.set(id, { fn, at: now + ms });
      return id;
    },
    clearTimeout: (id) => timers.delete(id),
    listeners: {},
    addEventListener(k, fn) {
      (this.listeners[k] ||= []).push(fn);
    },
    removeEventListener() {},
  };
  return {
    context,
    timerCount: () => timers.size,
    root,
    box,
    doc,
    win,
    view,
    scene,
    sent,
    prepared,
    modes,
    tools,
    emit,
    model: m,
    options: {
      root,
      box,
      scene,
      send,
      status: async () => ({ outcome: 'unknown' }),
      authority: () => authority,
      palette: ['#ff0000', '#00ff00'],
      id: (() => {
        let n = 0;
        return () => `id${++n}`;
      })(),
      onMode: (value) => modes.push(value),
      onTool: (t) => tools.push(t),
    },
    setAuthority: (a) => (authority = { ...authority, ...a }),
    setUnknown: (value) => (unknown = value),
    async flush() {
      for (let i = 0; i < 80; i++) await Promise.resolve();
      for (let i = 0; i < 4; i++) {
        for (const fn of frames.splice(0)) fn();
        for (let j = 0; j < 20; j++) await Promise.resolve();
      }
    },
    tick(ms) {
      now += ms;
      for (const [id, t] of [...timers])
        if (t.at <= now) {
          timers.delete(id);
          t.fn();
        }
    },
    event(el, type, extra = {}) {
      for (const fn of el.listeners[type] || [])
        fn({
          pointerId: 1,
          pointerType: 'pen',
          isPrimary: true,
          button: 0,
          buttons: 1,
          pressure: 0.25,
          clientX: 10,
          clientY: 10,
          preventDefault() {},
          stopPropagation() {},
          ...extra,
        });
    },
  };
}
test('unavailable drawing explains recovery and Done requires a sketch', async () => {
  const { mountSketch } = await mod();
  const h = harness();
  h.setAuthority({allowed:false});
  h.options.showUnavailable = true;
  h.options.unavailableHint = () => 'Turn on annotation (👆) in Couch’s Chrome status group to sketch.';
  const ui = mountSketch(h.doc, h.win, h.view, h.options);
  const toolbar = h.root.querySelector('.sketch-toolbar');
  const button = name => toolbar.children.find(n => n.textContent === name);
  const notice = h.root.querySelector('.sketch-status');
  assert.equal(toolbar.hidden, false);
  assert.equal(button('Sketch').disabled, true);
  assert.equal(button('Done').disabled, true);
  assert.match(notice.textContent, /Turn on annotation.*👆.*Chrome/);
  assert.equal(button('Sketch').title, notice.textContent);
  h.setAuthority({allowed:true});ui.setAuthority(h.options.authority());
  assert.equal(button('Sketch').disabled, false);
  assert.equal(button('Done').disabled, true);
  assert.equal(notice.textContent, '');
  await ui.enter();assert.equal(button('Done').disabled, false);
  await ui.leave();ui.labelInput.value='Availability smoke';await ui.saveLabel();
  assert.equal(button('Done').disabled, true);
  for (const [authority, expected] of [[{live:false},/Resume.*cast/],[{ended:true},/ended.*new cast/]]) {
    h.setAuthority(authority);ui.setAuthority(h.options.authority());
    assert.equal(button('Sketch').disabled,true);
    assert.match(notice.textContent,expected);
  }
  ui.dispose();
});

test('unfinished Note opens label once and idle authority preserves its controls and focus', async () => {
  const {mountSketch}=await mod(),h=harness();
  const ui=mountSketch(h.doc,h.win,h.view,h.options);
  await ui.enter();
  const note=h.root.querySelector('.sketch-toolbar').children.find(b=>b.textContent==='Add note');
  assert.ok(note,'unfinished sketch has a Note button');
  let focuses=0;ui.labelInput.focus=()=>focuses++;
  for(const fn of note.listeners.click)fn();await h.flush();
  assert.equal(h.root.querySelector('.sketch-label').hidden,false);
  ui.labelInput.value='Keep this note';
  const pending=h.root.querySelector('.sketch-pending-labels'),button=pending.children[0];
  for(let i=0;i<10;i++){ui.setAuthority(h.options.authority());ui.openLabel(ui.state().active);}
  assert.equal(focuses,1,'same editor must not be repeatedly focused');
  assert.equal(ui.labelInput.value,'Keep this note');
  assert.equal(pending.children[0],button,'unchanged unfinished Note control keeps its identity');
  ui.dispose();
});

test('shared mount streams pressure before pointerup, keeps one anchor, and Undo/Redo reach the model', async () => {
  const { mountSketch } = await mod();
  const h = harness(),
    ui = mountSketch(h.doc, h.win, h.view, h.options);
  await ui.enter();
  assert.equal(ui.state().phase, 'drawing');
  assert.equal(h.prepared.length, 1);
  h.event(ui.layer, 'pointerdown', { pressure: 0.2 });
  await h.flush();
  h.event(ui.layer, 'pointermove', {
    clientX: 30,
    pressure: 0.8,
    getCoalescedEvents: () => [
      { clientX: 20, clientY: 10, pressure: 0.4 },
      { clientX: 30, clientY: 10, pressure: 0.8 },
    ],
  });
  h.tick(50);
  await h.flush();
  const first = h.model.sketchProjection().sketches[0].strokes[0];
  assert.equal(first.sealed, false);
  assert.deepEqual(first.pts, [
    [10, 10, 0.2],
    [20, 10, 0.4],
    [30, 10, 0.8],
  ]);
  assert.equal(h.sent.filter((x) => x.kind === 'sketch-stroke-begin').length, 1);
  h.event(ui.layer, 'pointerup', { buttons: 0, pressure: 0, clientX: 30 });
  await h.flush();
  assert.equal(h.model.sketchProjection().sketches[0].strokes[0].sealed, true);
  assert.equal(h.model.sketchProjection().sketches[0].strokes[0].pts.at(-1)[2], 0.8);
  await ui.undo();
  assert.deepEqual(h.model.sketchProjection().sketches[0].visible, []);
  await ui.redo();
  assert.equal(h.model.sketchProjection().sketches[0].visible.length, 1);
  ui.dispose();
});
test('leave/Escape retain held scene; required label gates tool transition and spectators cannot draw', async () => {
  const { mountSketch } = await mod();
  const h = harness(),
    ui = mountSketch(h.doc, h.win, h.view, h.options);
  await ui.enter();
  await ui.leave('point');
  assert.equal(ui.state().phase, 'label-required');
  assert.deepEqual(h.tools, []);
  h.event(ui.labelInput, 'keydown', { key: 'Escape' });
  assert.equal(ui.state().phase, 'drawing');
  assert.equal(h.scene.ready, true);
  await ui.leave('note');
  ui.labelInput.value = '  ';
  await ui.saveLabel();
  assert.equal(h.sent.filter((x) => x.kind === 'sketch-label').length, 0);
  ui.labelInput.value = 'Move this control';
  await ui.saveLabel();
  assert.equal(h.model.sketchProjection().sketches[0].label, 'Move this control');
  assert.deepEqual(h.tools, ['note']);
  assert.equal(h.scene.ready, false);
  const other = harness();
  other.setAuthority({ allowed: false });
  const viewer = mountSketch(other.doc, other.win, other.view, other.options);
  await viewer.enter();
  other.event(viewer.layer, 'pointerdown');
  await other.flush();
  assert.equal(other.sent.length, 0);
});
test('pending async anchor buffers the contact, hover is ignored, cancel retains truncation and mouse fallback', async () => {
  const { mountSketch } = await mod();
  const h = harness();
  let resolve;
  h.scene.hitTest = () => new Promise((r) => (resolve = r));
  const ui = mountSketch(h.doc, h.win, h.view, h.options);
  await ui.enter();
  h.event(ui.layer, 'pointermove', { buttons: 0, pressure: 0 });
  assert.equal(h.sent.length, 1);
  h.event(ui.layer, 'pointerdown');
  h.event(ui.layer, 'pointermove', { clientX: 20, pressure: 0.7 });
  h.event(ui.layer, 'pointercancel', { buttons: 0, pressure: 0 });
  await h.flush();
  assert.equal(h.sent.length, 1);
  resolve({ node: 7, box: [0, 0, 100, 100] });
  await h.flush();
  const stroke = h.model.sketchProjection().sketches[0].strokes[0];
  assert.equal(stroke.truncated, true);
  assert.equal(ui.layer.hasPointerCapture(1), false);
  h.scene.hitTest = async () => ({ node: 7, box: [0, 0, 100, 100] });
  h.event(ui.layer, 'pointerdown', { pointerType: 'mouse', pressure: 0 });
  h.event(ui.layer, 'pointerup', { pointerType: 'mouse', pressure: 0, buttons: 0 });
  await h.flush();
  assert.equal(h.model.sketchProjection().sketches[0].strokes[1].pts[0][2], 1);
});
test('handoff fences queued local input but old-author label remains reachable', async () => {
  const { mountSketch } = await mod();
  const h = harness(),
    ui = mountSketch(h.doc, h.win, h.view, h.options);
  await ui.enter();
  const source = h.scene.source;
  ui.setAuthority({ allowed: false, live: true, ended: false, epoch: 2 });
  assert.equal(h.scene.ready, false);
  assert.equal(ui.state().phase, 'interrupted');
  h.event(ui.layer, 'pointerdown');
  await h.flush();
  assert.equal(h.sent.length, 1);
  h.emit({ k: 'ctl', t: 9, op: 'reset', who: { id: 'operator' } });
  ui.openLabel('server-s');
  ui.labelInput.value = 'Partial intent';
  await ui.saveLabel();
  assert.equal(h.model.sketchProjection().sketches[0].label, 'Partial intent');
  assert.equal(source.handle, 'h');
  assert.equal(h.sent.at(-1).body.epoch, 1);
});

test('failed scene preflight emits no begin and can be retried', async () => {
  const { mountSketch } = await mod();
  const h = harness();
  h.scene.preflight = async () => {
    throw Error('CSP blocked');
  };
  const ui = mountSketch(h.doc, h.win, h.view, h.options);
  await ui.enter();
  assert.equal(h.sent.length, 0);
  assert.equal(ui.state().active, null);
  h.scene.preflight = async () => {};
  await ui.enter();
  assert.equal(h.sent.length, 1);
  assert.equal(ui.state().phase, 'drawing');
});
test('unknown confirmation stops new input and retry queries the exact operation', async () => {
  const { mountSketch } = await mod();
  const h = harness(),
    lookups = [];
  h.options.status = async (b) => {
    lookups.push(b);
    return { outcome: 'unknown' };
  };
  const ui = mountSketch(h.doc, h.win, h.view, h.options);
  await ui.enter();
  h.setUnknown(true);
  h.event(ui.layer, 'pointerdown');
  await h.flush();
  assert.equal(ui.state().phase, 'reconciling');
  const count = h.sent.length;
  h.event(ui.layer, 'pointermove', { clientX: 20 });
  h.tick(50);
  await h.flush();
  assert.equal(h.sent.length, count);
  ui.retry();
  await h.flush();
  assert.deepEqual(lookups[0], {
    operation: h.sent.at(-1).body.operation,
    sequence: h.sent.at(-1).body.sequence,
    epoch: 1,
    sketch: 'server-s',
  });
  ui.dispose();
});

test('unconfirmed contact paints immediately and echo removes its translucent duplicate', async () => {
  const { mountSketch } = await mod();
  const h = harness();
  let anchor;
  h.scene.hitTest = () => new Promise((r) => (anchor = r));
  const ui = mountSketch(h.doc, h.win, h.view, h.options);
  await ui.enter();
  h.event(ui.layer, 'pointerdown');
  await h.flush();
  assert.ok(h.context.fills.some((a) => a < 1));
  anchor({ node: 7, box: [0, 0, 100, 100] });
  await h.flush();
  h.context.fills = [];
  h.event(ui.layer, 'pointerup', { buttons: 0 });
  await h.flush();
  assert.equal(h.context.fills.length, 0, 'sealing keeps already accepted ink without repainting it');
  ui.dispose();
});
test('retiring a pending hit test prevents any late stroke command', async () => {
  const { mountSketch } = await mod();
  const h = harness();
  let anchor;
  h.scene.hitTest = () => new Promise((r) => (anchor = r));
  const ui = mountSketch(h.doc, h.win, h.view, h.options);
  await ui.enter();
  h.event(ui.layer, 'pointerdown');
  await h.flush();
  ui.dispose();
  anchor({ node: 7, box: [0, 0, 100, 100] });
  await h.flush();
  assert.equal(h.sent.length, 1);
});
test('status not-observed resends the immutable operation once, then resumes', async () => {
  const { mountSketch } = await mod();
  const h = harness();
  h.options.status = async () => ({ outcome: 'not-observed' });
  const ui = mountSketch(h.doc, h.win, h.view, h.options);
  await ui.enter();
  h.setUnknown(true);
  h.event(ui.layer, 'pointerdown');
  await h.flush();
  const original = structuredClone(h.sent.at(-1));
  h.setUnknown(false);
  ui.retry();
  ui.retry();
  await h.flush();
  assert.deepEqual(h.sent.filter((s) => s.body.operation === original.body.operation).at(-1), original);
  assert.equal(h.sent.filter((s) => s.body.operation === original.body.operation).length, 2);
  assert.equal(ui.state().phase, 'drawing');
  ui.dispose();
});
test('unusable async anchor releases capture without sending a stroke', async () => {
  const { mountSketch } = await mod();
  const h = harness();
  h.scene.hitTest = async () => null;
  const ui = mountSketch(h.doc, h.win, h.view, h.options);
  await ui.enter();
  h.event(ui.layer, 'pointerdown');
  await h.flush();
  assert.equal(h.sent.length, 1);
  assert.equal(ui.layer.hasPointerCapture(1), false);
  ui.dispose();
});
test('interrupted accepted begin without a source keeps label access and never prepares a scene', async () => {
  const { mountSketch } = await mod();
  const h = harness(),
    send = h.options.send;
  h.options.send = async (k, b) => {
    const reply = await send(k, b);
    if (k === 'sketch-begin') {
      h.emit({ k: 'ctl', t: 5, op: 'reset', who: { id: 'operator' } });
      reply.status.phase = 'interrupted';
      reply.status.reason = 'reset';
      delete reply.source;
    }
    return reply;
  };
  const ui = mountSketch(h.doc, h.win, h.view, h.options);
  await ui.enter();
  assert.equal(h.prepared.length, 0);
  assert.equal(ui.state().phase, 'interrupted');
  assert.deepEqual(ui.state().pendingLabels, ['server-s']);
  ui.openLabel('server-s');
  ui.labelInput.value = 'Intent survived';
  await ui.saveLabel();
  assert.equal(h.model.sketchProjection().sketches[0].label, 'Intent survived');
});
test('actual annotation clear and stop end retire the scene and input', async () => {
  const { mountSketch } = await mod();
  for (const record of [
    { k: 'annotation-clear', who: { id: 'operator' } },
    { k: 'stop', why: 'end' },
  ]) {
    const h = harness(),
      ui = mountSketch(h.doc, h.win, h.view, h.options);
    await ui.enter();
    h.emit({ ...record, t: 9 });
    assert.equal(h.scene.ready, false);
    h.event(ui.layer, 'pointerdown');
    await h.flush();
    assert.equal(h.sent.length, 1);
    ui.dispose();
  }
});
test('unknown contact releases capture and seals its confirmed prefix before another stroke', async () => {
  const { mountSketch } = await mod();
  const h = harness();
  h.options.status = async () => ({ outcome: 'not-observed' });
  const ui = mountSketch(h.doc, h.win, h.view, h.options);
  await ui.enter();
  h.setUnknown(true);
  h.event(ui.layer, 'pointerdown');
  await h.flush();
  assert.equal(ui.layer.hasPointerCapture(1), false);
  h.event(ui.layer, 'pointerup', { buttons: 0 });
  h.setUnknown(false);
  ui.retry();
  await h.flush();
  const first = h.model.sketchProjection().sketches[0].strokes[0];
  assert.equal(first.sealed, true);
  assert.equal(first.truncated, true);
  h.event(ui.layer, 'pointerdown');
  h.event(ui.layer, 'pointerup', { buttons: 0 });
  await h.flush();
  assert.equal(h.model.sketchProjection().sketches[0].strokes.length, 2);
  ui.dispose();
});

test('global Escape leaves the tool, then resumes drawing without replacing the held scene', async () => {
  const { mountSketch } = await mod();
  const h = harness(),
    ui = mountSketch(h.doc, h.win, h.view, h.options);
  await ui.enter();
  h.event(h.win, 'keydown', { key: 'Escape' });
  assert.equal(ui.state().phase, 'label-required');
  h.event(h.win, 'keydown', { key: 'Escape' });
  assert.equal(ui.state().phase, 'drawing');
  assert.equal(h.prepared.length, 1);
  ui.dispose();
});
test('empty coalesced list falls back to contact and pointerup keeps final position with prior pressure', async () => {
  const { mountSketch } = await mod();
  const h = harness(),
    ui = mountSketch(h.doc, h.win, h.view, h.options);
  await ui.enter();
  h.event(ui.layer, 'pointerdown');
  await h.flush();
  h.event(ui.layer, 'pointermove', { clientX: 30, pressure: 0.6, getCoalescedEvents: () => [] });
  h.event(ui.layer, 'pointerup', { clientX: 40, pressure: 0, buttons: 0 });
  await h.flush();
  assert.deepEqual(h.model.sketchProjection().sketches[0].strokes[0].pts, [
    [10, 10, 0.25],
    [30, 10, 0.6],
    [40, 10, 0.6],
  ]);
  ui.dispose();
});
test('explicit recovery seals authoritative accepted prefix before Undo and further drawing', async () => {
  const { mountSketch } = await mod();
  const h = harness(),
    send = h.options.send;
  let refused;
  h.options.send = async (kind, body) => {
    if (kind === 'sketch-points' && body.chunk === 1 && !refused) {
      h.sent.push({ kind, body: structuredClone(body) });
      refused = body;
      const p = h.model.sketchProjection().sketches[0];
      return {
        operation: body.operation,
        sketch: body.sketch,
        sequence: body.sequence,
        next_sequence: p.next_sequence,
        outcome: 'refused',
        code: 'capacity',
        status: { phase: 'drawing', open_stroke: p.strokes[0].begin.id, visible_top: '', redo_top: '' },
      };
    }
    return send(kind, body);
  };
  const ui = mountSketch(h.doc, h.win, h.view, h.options);
  await ui.enter();
  h.event(ui.layer, 'pointerdown');
  await h.flush();
  h.event(ui.layer, 'pointermove', { clientX: 20 });
  h.tick(50);
  await h.flush();
  assert.equal(ui.state().phase, 'interrupted');
  assert.equal(ui.canContinue(), true);
  await ui.continueDrawing();
  const seal = h.sent.find((s) => s.kind === 'sketch-stroke-end');
  assert.equal(seal.body.sequence, refused.sequence);
  assert.notEqual(seal.body.operation, refused.operation);
  assert.equal(seal.body.truncated, true);
  assert.equal(ui.state().phase, 'drawing');
  assert.equal(h.model.sketchProjection().sketches[0].strokes[0].pts.length, 1);
  await ui.undo();
  assert.deepEqual(h.model.sketchProjection().sketches[0].visible, []);
  ui.setAuthority({ allowed: false, live: true, ended: false, epoch: 2 });
  const count = h.sent.length;
  await ui.continueDrawing();
  assert.equal(h.sent.length, count);
  ui.dispose();
});
test('acknowledged scene preparation suspends ordinary input before scene readiness', async () => {
  const { mountSketch } = await mod();
  const h = harness();
  let ready;
  h.scene.prepare = () => new Promise((r) => (ready = r));
  const ui = mountSketch(h.doc, h.win, h.view, h.options);
  const preparing = ui.enter();
  await h.flush();
  assert.deepEqual(h.modes, [true]);
  ui.setAuthority({ allowed: false, live: true, ended: false, epoch: 2 });
  assert.equal(h.modes.at(-1), false);
  ready();
  await preparing;
  assert.equal(ui.state().phase, 'interrupted');
  ui.dispose();
});
test('delayed accepted begin from a retired epoch cannot resurrect its source scene', async () => {
  const { mountSketch } = await mod();
  const h = harness(),
    send = h.options.send;
  let release;
  h.options.send = async (k, b) => {
    const reply = await send(k, b);
    return new Promise((r) => (release = () => r(reply)));
  };
  const ui = mountSketch(h.doc, h.win, h.view, h.options);
  const entering = ui.enter();
  await h.flush();
  ui.setAuthority({ allowed: false, live: true, ended: false, epoch: 2 });
  release();
  await entering;
  assert.equal(h.prepared.length, 0);
  assert.equal(h.modes.at(-1), false);
  assert.equal(ui.state().phase, 'interrupted');
  ui.dispose();
});
test('read-only replay mounts historical pressure paths without a drawing palette', async () => {
  const { mountSketch } = await mod();
  const h = harness();
  let sequence = 0;
  for (const r of [
    {
      k: 'sketch-begin',
      id: 'historical',
      author: { id: 'operator', color: '#00ff00' },
      owner: 'operator',
      doc: 1,
      tab: 'tab',
      vp: { w: 100, h: 100, dpr: 1 },
    },
    {
      k: 'sketch-stroke-begin',
      sketch: 'historical',
      id: 'p',
      at: { node: 7, box: [0, 0, 100, 100] },
      width: 4,
    },
    {
      k: 'sketch-points',
      sketch: 'historical',
      id: 'p',
      chunk: 0,
      pts: [
        [10, 10, 0.2],
        [30, 10, 0.8],
      ],
    },
    { k: 'sketch-stroke-end', sketch: 'historical', id: 'p' },
    { k: 'sketch-label', sketch: 'historical', label: 'Historical intent' },
  ])
    h.emit({ ...r, t: 3 + sequence, sequence, operation: 'old' + sequence++, digest: 'a'.repeat(64) });
  h.setAuthority({ allowed: false });
  h.options.palette = [];
  const ui = mountSketch(h.doc, h.win, h.view, h.options);
  await h.flush();
  assert.ok(h.context.fills.length > 0);
  await ui.enter();
  assert.equal(h.sent.length, 0);
  assert.equal(h.prepared.length, 0);
  ui.dispose();
});

test('actual accepted painter visits only new segments for 8/16 point chunks', async () => {
  const { mountSketch } = await mod();
  const work = [];
  for (const n of [8, 16]) {
    const h = harness();
    h.setAuthority({ allowed: false });
    const path = {
      sketch: 'historic',
      id: 'p',
      at: { node: 7, box: [0, 0, 100, 100] },
      width: 4,
      color: '#ff0000',
      pts: [],
      source: { doc: 1, tab: 'tab', t: 3 },
    };
    h.options.paths = () => (path.pts.length ? [path] : []);
    const ui = mountSketch(h.doc, h.win, h.view, h.options);
    await h.flush();
    for (let i = 0; i < n; i++) {
      path.pts.push([i + 10, 10, 0.5]);
      ui.refresh();
      await h.flush();
    }
    assert.ok(
      h.context.paintedPoints <= 2 * n,
      `painted ${h.context.paintedPoints} samples for ${n} new points`,
    );
    assert.equal(h.context.paintedSegments, n - 1);
    work.push(h.context.paintedPoints);
    const beforeGeometry = h.context.paintedPoints;
    ui.refresh({ geometry: true });
    await h.flush();
    assert.equal(
      h.context.paintedPoints - beforeGeometry,
      n,
      'geometry invalidation repaints the complete path',
    );
    const beforeBootstrap = h.context.paintedPoints;
    path.pts = path.pts.map((p) => [...p]);
    ui.refresh();
    await h.flush();
    assert.equal(
      h.context.paintedPoints - beforeBootstrap,
      n,
      'owned bootstrap replacement repaints the complete path',
    );
    path.pts = [];
    ui.refresh();
    await h.flush();
    ui.dispose();
  }
  assert.ok(work[1] <= 2 * work[0] + 1);
});
for(const during of ['move','pointerup'])test(`full point queue during ${during} retains one truncated terminal and releases capture safely`,async()=>{const {mountSketch}=await mod();const h=harness(),send=h.options.send;let release;h.options.send=async(kind,body)=>{const r=await send(kind,body);if(kind==='sketch-points'&&body.chunk===0)return new Promise(ok=>release=()=>ok(r));return r;};const ui=mountSketch(h.doc,h.win,h.view,h.options);await ui.enter();h.event(ui.layer,'pointerdown');await h.flush();for(let i=0;i<16;i++){h.event(ui.layer,'pointermove',{clientX:20+i});h.tick(50);await h.flush();}h.event(ui.layer,'pointermove',{clientX:50});if(during==='move')h.tick(50);else assert.doesNotThrow(()=>h.event(ui.layer,'pointerup',{clientX:50,buttons:0}));assert.equal(ui.layer.hasPointerCapture(1),false);assert.equal(ui.state().phase,'reconciling');assert.match(h.root.querySelector('.sketch-status').textContent,/queue|buffer|confirm/i);release();for(let i=0;i<8;i++)await h.flush();const s=h.model.sketchProjection().sketches[0].strokes[0];assert.equal(s.sealed,true);assert.equal(s.truncated,true);assert.equal(h.sent.filter(x=>x.kind==='sketch-stroke-end').length,1);assert.equal(ui.state().phase,'drawing');ui.dispose();});
for(const outcome of ['unknown','refused'])test(`saturated terminal survives ${outcome} via exact confirmation or explicit prefix recovery`,async()=>{const {mountSketch}=await mod();const h=harness(),send=h.options.send;let release,held=false;h.options.send=async(kind,body)=>{if(kind==='sketch-points'&&body.chunk===1&&!held){held=true;h.sent.push({kind,body:structuredClone(body)});return new Promise(ok=>release=()=>{const s=h.model.sketchProjection().sketches[0];ok({outcome,next_sequence:s.next_sequence,status:outcome==='refused'?{phase:s.phase,open_stroke:s.strokes[0].begin.id,visible_top:'',redo_top:''}:undefined});});}return send(kind,body);};h.options.status=async()=>({outcome:'not-observed'});const ui=mountSketch(h.doc,h.win,h.view,h.options);await ui.enter();h.event(ui.layer,'pointerdown');await h.flush();for(let i=0;i<18;i++){h.event(ui.layer,'pointermove',{clientX:20+i});h.tick(50);await h.flush();}assert.equal(ui.state().phase,'reconciling');release();await h.flush();if(outcome==='unknown')ui.retry();else{assert.equal(ui.canContinue(),true);await ui.continueDrawing();}for(let i=0;i<8;i++)await h.flush();const s=h.model.sketchProjection().sketches[0].strokes[0];assert.equal(s.sealed,true);assert.equal(s.truncated,true);assert.equal(h.sent.filter(r=>r.kind==='sketch-stroke-end').length,1);if(outcome==='refused')assert.equal(s.pts.length,1);else assert.ok(s.pts.length>1);assert.equal(ui.state().phase,'drawing');ui.dispose();});

test('optimistic contact uses approved 3px base and original pen pressure, mouse/touch constant width', async () => {
  const { mountSketch } = await mod();
  for (const [pointerType, pressure] of [['pen', 0], ['pen', .2], ['pen', .8], ['pen', 1], ['mouse', 0], ['touch', .4]]) {
    const h = harness();
    // Keep the original contact unacknowledged so this checks the local ink
    // contract independently of the fixture producer and persisted widths.
    h.scene.hitTest = () => new Promise(() => {});
    const ui = mountSketch(h.doc, h.win, h.view, h.options);
    await ui.enter();
    h.event(ui.layer, 'pointerdown', { pointerType, pressure });
    await h.flush();
    const local = h.context.arcs.filter(a => a.alpha < 1);
    assert.ok(local.length > 0);
    const p = pointerType === 'pen' ? pressure : 1;
    for (const arc of local) assert.equal(arc.radius * 2, 3 * (.25 + .75 * p), `${pointerType} pressure ${pressure}`);
    ui.dispose();
  }
});

for (const exit of ['Done', 'Escape', 'point', 'note']) for (const released of [false, true])
 test(`viewer ${exit} preserves ${released ? 'released' : 'open'} contact through delayed anchor`, async () => {
  const {mountSketch}=await mod();const h=harness();let anchor;
  h.scene.hitTest=()=>new Promise(resolve=>anchor=resolve);
  const ui=mountSketch(h.doc,h.win,h.view,h.options);await ui.enter();
  h.event(ui.layer,'pointerdown');h.event(ui.layer,'pointermove',{clientX:20,pressure:.8});
  if(released)h.event(ui.layer,'pointerup',{buttons:0,clientX:20});await h.flush();
  try {
   if(exit==='Escape')h.event(h.win,'keydown',{key:'Escape'});
   else if(exit==='Done'){const toolbar=h.root.children.find(n=>n.className==='sketch-toolbar');h.event(toolbar.children.find(n=>n.textContent==='Done'),'click');}
   else assert.equal(ui.beforeAnnotationMode(exit),false);
   assert.equal(ui.state().phase,'finishing');
   ui.openLabel('server-s');ui.labelInput.value='Preserved intent';const saving=ui.saveLabel();
   await h.flush();assert.equal(h.sent.filter(r=>r.kind==='sketch-label').length,0);
   anchor({node:7,box:[0,0,100,100]});await h.flush();await saving;
   assert.equal(ui.state().phase,'label-required');
   const strokes=h.model.sketchProjection().sketches[0].strokes;assert.equal(strokes.length,1);
   assert.deepEqual(strokes[0].pts,[[10,10,.25],[20,10,.8]]);assert.equal(strokes[0].sealed,true);
   ui.labelInput.value='Preserved intent';await ui.saveLabel();assert.equal(h.model.sketchProjection().sketches[0].label,'Preserved intent');
  } finally {anchor({node:7,box:[0,0,100,100]});await h.flush();ui.dispose();}
 });

test('unknown before voluntary exit keeps exact retry and seals prefix before label',async()=>{
 const {mountSketch}=await mod();const h=harness();const ui=mountSketch(h.doc,h.win,h.view,h.options);await ui.enter();
 h.setUnknown(true);h.event(ui.layer,'pointerdown');await h.flush();const original=structuredClone(h.sent.at(-1));
 await ui.leave('point');assert.equal(ui.state().phase,'reconciling');
 h.setUnknown(false);h.options.status=async()=>({outcome:'not-observed'});ui.retry();await h.flush();
 assert.deepEqual(h.sent.findLast(r=>r.kind==='sketch-points'),original);
 assert.equal(ui.state().phase,'label-required');assert.equal(h.model.sketchProjection().sketches[0].strokes[0].sealed,true);ui.dispose();
});
test('finishing pending anchor expires visibly, bounds leave wait and fences late anchor',async()=>{
 const {mountSketch}=await mod();const h=harness();let anchor;h.scene.hitTest=()=>new Promise(r=>anchor=r);
 const ui=mountSketch(h.doc,h.win,h.view,h.options);await ui.enter();h.event(ui.layer,'pointerdown');await h.flush();
 const leaving=ui.leave('watch');assert.equal(h.root.children.find(n=>n.className==='sketch-status').textContent,'Saving…');
 h.tick(5000);await leaving;await h.flush();
 const notice=h.root.children.find(n=>n.className==='sketch-status');assert.match(notice.textContent,/Couldn't save; accepted strokes kept/);
 anchor({node:7,box:[0,0,100,100]});await h.flush();assert.equal(h.sent.filter(r=>r.kind==='sketch-stroke-begin').length,0);assert.match(notice.textContent,/Couldn't save/);ui.dispose();
});
test('held mutation observer times out then confirms original receipt while late HTTP is ignored',async()=>{
 const {mountSketch}=await mod();const h=harness(),send=h.options.send;let response,late,command;
 h.options.send=async(k,b)=>{const result=await send(k,b);if(k==='sketch-points'){response=result;command=structuredClone(b);return new Promise(r=>late=r);}return result;};
 h.options.status=async b=>{assert.equal(b.operation,command.operation);return response;};
 const ui=mountSketch(h.doc,h.win,h.view,h.options);await ui.enter();h.event(ui.layer,'pointerdown');await h.flush();
 const leaving=ui.leave('point');h.tick(5000);await leaving;await h.flush();assert.equal(ui.state().phase,'reconciling');
 ui.retry();await h.flush();assert.equal(ui.state().phase,'label-required');
 const count=h.sent.length;late(response);await h.flush();assert.equal(h.sent.length,count);assert.equal(ui.state().phase,'label-required');
 assert.match(h.root.children.find(n=>n.className==='sketch-status').textContent,/Couldn't save/);ui.dispose();
});

for(const kind of ['handoff','pause','clear','end'])test(`viewer ${kind} visibly retires pending-anchor finishing and fences late callback`,async()=>{
 const {mountSketch}=await mod();const h=harness();let anchor;h.scene.hitTest=()=>new Promise(r=>anchor=r);const ui=mountSketch(h.doc,h.win,h.view,h.options);await ui.enter();h.event(ui.layer,'pointerdown');await h.flush();const leaving=ui.leave('point');
 if(kind==='handoff'){h.emit({k:'ctl',t:9,op:'request',who:{id:'driver'},pending:[{id:'driver'}]});h.emit({k:'ctl',t:10,op:'grant',who:{id:'driver'},driver:{id:'driver'},pending:[]});ui.setAuthority({allowed:false,live:true,ended:false,epoch:2});}
 else if(kind==='clear')h.emit({k:'annotation-clear',t:9,who:{id:'operator'}});else h.emit({k:'stop',t:9,why:kind});
 await leaving;assert.match(h.root.children.find(n=>n.className==='sketch-status').textContent,/Couldn't save; accepted strokes kept/);
 anchor({node:7,box:[0,0,100,100]});await h.flush();assert.equal(h.sent.filter(r=>r.kind==='sketch-stroke-begin').length,0);ui.dispose();
});
test('first deferred exit survives pending preflight and preparation without enabling capture',async()=>{
 const {mountSketch}=await mod();const h=harness();let preflight,prepare;h.scene.preflight=()=>new Promise(r=>preflight=r);h.scene.prepare=()=>new Promise(r=>prepare=()=>{h.scene.ready=true;r();});
 const ui=mountSketch(h.doc,h.win,h.view,h.options);const entering=ui.enter();await h.flush();assert.equal(ui.beforeAnnotationMode('point'),false);const leaving=ui.leave('note');ui.beforeAnnotationMode('note');
 assert.equal(h.root.children.find(n=>n.className==='sketch-status').textContent,'Saving…');preflight();await h.flush();assert.equal(ui.state().phase,'finishing');h.event(ui.layer,'pointerdown');assert.equal(h.sent.length,1);
 prepare();await entering;await leaving;assert.equal(ui.state().phase,'label-required');ui.labelInput.value='Prepared exit';await ui.saveLabel();assert.equal(h.tools.at(-1),'point');ui.dispose();
});

test('definite refusal while finishing keeps accepted prefix and requires explicit sealed recovery',async()=>{
 const {mountSketch}=await mod();const h=harness(),send=h.options.send;let points=0,refuse;
 h.options.send=(kind,body)=>{if(kind==='sketch-points'&&++points===2)return new Promise(resolve=>refuse=()=>resolve({outcome:'refused',next_sequence:body.sequence,status:{phase:'drawing',open_stroke:body.id}}));return send(kind,body);};
 const ui=mountSketch(h.doc,h.win,h.view,h.options);await ui.enter();h.event(ui.layer,'pointerdown');await h.flush();h.event(ui.layer,'pointermove',{clientX:20});h.tick(50);await h.flush();const leaving=ui.leave('point');refuse();await leaving;await h.flush();
 assert.equal(ui.state().phase,'interrupted');assert.equal(ui.canContinue(),true);assert.equal(h.model.sketchProjection().sketches[0].strokes[0].pts.length,1);
 await ui.continueDrawing();const stroke=h.model.sketchProjection().sketches[0].strokes[0];assert.equal(stroke.sealed,true);assert.equal(stroke.truncated,true);
 assert.equal(ui.state().phase,'label-required');ui.labelInput.value='Recovered exit';await ui.saveLabel();assert.equal(h.tools.at(-1),'point');ui.dispose();
});
test('unusable delayed anchor fails visibly before allowing label over accepted history',async()=>{
 const {mountSketch}=await mod();const h=harness();let fail;h.scene.hitTest=()=>new Promise((_,reject)=>fail=reject);
 const ui=mountSketch(h.doc,h.win,h.view,h.options);await ui.enter();h.event(ui.layer,'pointerdown');await h.flush();const leaving=ui.leave('watch');fail(Error('missing anchor'));await leaving;await h.flush();
 assert.equal(ui.state().phase,'label-required');assert.match(h.root.children.find(n=>n.className==='sketch-status').textContent,/no usable anchor/);assert.equal(h.model.sketchProjection().sketches[0].strokes.length,0);ui.dispose();
});

test('dispose retires finishing deadline and bounded request observer timers',async()=>{
 const {mountSketch}=await mod();const h=harness(),send=h.options.send;let late;
 h.options.send=(kind,body)=>kind==='sketch-points'?new Promise(r=>late=r):send(kind,body);
 const ui=mountSketch(h.doc,h.win,h.view,h.options);await ui.enter();h.event(ui.layer,'pointerdown');await h.flush();const leaving=ui.leave('watch');assert.ok(h.timerCount()>0);
 ui.dispose();assert.equal(h.timerCount(),0);await leaving;late({outcome:'unknown'});await h.flush();assert.equal(h.timerCount(),0);
});

// Canonical logical-owner table: physical release retains pending accumulator.
for (const ending of ['pointerup', 'pointercancel', 'lostpointercapture', 'buffer-limit'])
 test(`viewer logical owner survives ${ending} and consumes successor until fresh down`, async () => {
  const {mountSketch}=await mod();const h=harness();let anchor;
  h.scene.hitTest=()=>new Promise(resolve=>anchor=resolve);
  const ui=mountSketch(h.doc,h.win,h.view,h.options);await ui.enter();h.event(ui.layer,'pointerdown');await h.flush();
  if(ending==='buffer-limit'){
   h.event(ui.layer,'pointermove',{getCoalescedEvents:()=>Array.from({length:1023},(_,i)=>({clientX:10+i/100,clientY:10,pressure:.4}))});h.tick(50);
   h.event(ui.layer,'pointermove',{clientX:30});h.tick(50);
  }else h.event(ui.layer,ending,{buttons:0});
  try {
   assert.equal(ui.layer.hasPointerCapture(1),false);
   let prevented=0,stopped=0;
   assert.doesNotThrow(()=>h.event(ui.layer,'pointerdown',{pointerId:2,preventDefault(){prevented++;},stopPropagation(){stopped++;}}));
   assert.equal(prevented,1);assert.ok(stopped>0);assert.equal(ui.layer.hasPointerCapture(2),false);
   assert.match(h.root.children.find(n=>n.className==='sketch-status').textContent,/previous stroke.*wait|wait.*previous stroke/i);
   anchor({node:7,box:[0,0,100,100]});await h.flush();
   assert.doesNotThrow(()=>h.event(ui.layer,'pointermove',{pointerId:2,clientX:80}));
   assert.doesNotThrow(()=>h.event(ui.layer,'pointerup',{pointerId:2,buttons:0,clientX:80}));
   assert.equal(h.model.sketchProjection().sketches[0].strokes.length,1);
   h.scene.hitTest=async()=>({node:7,box:[0,0,100,100]});h.event(ui.layer,'pointerdown',{pointerId:3});h.event(ui.layer,'pointerup',{pointerId:3,buttons:0});await h.flush();
   assert.equal(h.model.sketchProjection().sketches[0].strokes.length,2);
  }finally{anchor({node:7,box:[0,0,100,100]});await h.flush();ui.dispose();}
 });

for(const failure of ['coordinates','capture'])test(`viewer ${failure} admission failure never publishes half-contact`,async()=>{
 const {mountSketch}=await mod();const h=harness();const ui=mountSketch(h.doc,h.win,h.view,h.options);await ui.enter();
 const capture=ui.layer.setPointerCapture.bind(ui.layer),toPage=h.scene.toPage;
 if(failure==='coordinates')h.scene.toPage=()=>[NaN,1];else ui.layer.setPointerCapture=id=>{capture(id);throw Error('capture failed');};
 assert.doesNotThrow(()=>h.event(ui.layer,'pointerdown'));assert.doesNotThrow(()=>h.event(ui.layer,'pointerup',{buttons:0}));assert.equal(ui.layer.hasPointerCapture(1),false);assert.equal(h.sent.length,1);
 h.scene.toPage=toPage;ui.layer.setPointerCapture=capture;h.event(ui.layer,'pointerdown');h.event(ui.layer,'pointerup',{buttons:0});await h.flush();assert.equal(h.model.sketchProjection().sketches[0].strokes.length,1);ui.dispose();
});
test('viewer rejected N/2N downs allocate no IDs capture hit-tests timers or points',async()=>{
 const {mountSketch}=await mod();
 for(const n of [8,16]){const h=harness();let anchor,ids=0,hits=0,captures=0;h.options.id=()=>`count-${++ids}`;h.scene.hitTest=()=>{hits++;return new Promise(r=>anchor=r);};
 const ui=mountSketch(h.doc,h.win,h.view,h.options);const capture=ui.layer.setPointerCapture.bind(ui.layer);ui.layer.setPointerCapture=id=>{captures++;capture(id);};await ui.enter();h.event(ui.layer,'pointerdown');h.event(ui.layer,'pointerup',{buttons:0});await h.flush();const before=[ids,hits,captures,h.timerCount(),h.sent.length];
 for(let i=0;i<n;i++){h.event(ui.layer,'pointerdown',{pointerId:i%2?1:2});h.event(ui.layer,'pointermove',{pointerId:i%2?1:2,clientX:80});h.event(ui.layer,'pointerup',{pointerId:i%2?1:2,buttons:0});}
 assert.deepEqual([ids,hits,captures,h.timerCount(),h.sent.length],before);anchor({node:7,box:[0,0,100,100]});await h.flush();assert.deepEqual(h.model.sketchProjection().sketches[0].strokes[0].pts,[[10,10,.25]]);ui.dispose();}
});
test('viewer ordinary anchor timeout releases logical owner and fences late callback',async()=>{
 const {mountSketch}=await mod();const h=harness();let anchor;h.scene.hitTest=()=>new Promise(r=>anchor=r);const ui=mountSketch(h.doc,h.win,h.view,h.options);await ui.enter();h.event(ui.layer,'pointerdown');h.event(ui.layer,'pointerup',{buttons:0});await h.flush();h.tick(5000);await h.flush();assert.match(h.root.children.find(n=>n.className==='sketch-status').textContent,/no usable anchor; accepted strokes kept/);
 h.scene.hitTest=async()=>({node:7,box:[0,0,100,100]});h.event(ui.layer,'pointerdown');h.event(ui.layer,'pointerup',{buttons:0});await h.flush();anchor({node:7,box:[0,0,100,100]});await h.flush();assert.equal(h.model.sketchProjection().sketches[0].strokes.length,1);ui.dispose();
});
test('viewer 4096 point end releases accumulator but FIFO keeps terminal before successor begin',async()=>{
 const {mountSketch}=await mod();const h=harness(),send=h.options.send;let endACK,held=false;
 h.options.send=async(k,b)=>{const reply=await send(k,b);if(k==='sketch-stroke-end'&&!held){held=true;return new Promise(r=>endACK=()=>r(reply));}return reply;};
 const ui=mountSketch(h.doc,h.win,h.view,h.options);await ui.enter();h.event(ui.layer,'pointerdown');await h.flush();
 for(let at=0;at<4095;at+=64){h.event(ui.layer,'pointermove',{getCoalescedEvents:()=>Array.from({length:Math.min(64,4095-at)},(_,i)=>({clientX:10+(at+i)/100,clientY:10,pressure:.5}))});h.tick(50);await h.flush();}
 h.event(ui.layer,'pointermove',{clientX:90});await h.flush();assert.equal(ui.layer.hasPointerCapture(1),false);h.event(ui.layer,'pointerdown',{pointerId:2});h.event(ui.layer,'pointerup',{pointerId:2,buttons:0});await h.flush();assert.equal(h.sent.filter(r=>r.kind==='sketch-stroke-begin').length,1);
 endACK();await h.flush();const strokes=h.model.sketchProjection().sketches[0].strokes;assert.equal(strokes.length,2);assert.equal(strokes[0].pts.length,4096);const order=h.sent.map(r=>r.kind);assert.ok(order.indexOf('sketch-stroke-end')<order.lastIndexOf('sketch-stroke-begin'));ui.dispose();
});

test('Undo and Redo cannot pass logical owner while new stroke awaits anchor',async()=>{
 const {mountSketch}=await mod();const h=harness();const ui=mountSketch(h.doc,h.win,h.view,h.options);await ui.enter();h.event(ui.layer,'pointerdown');h.event(ui.layer,'pointerup',{buttons:0});await h.flush();
 let anchor;h.scene.hitTest=()=>new Promise(r=>anchor=r);h.event(ui.layer,'pointerdown');h.event(ui.layer,'pointerup',{buttons:0});await h.flush();const count=h.sent.length;await ui.undo();await ui.redo();assert.equal(h.sent.length,count);
 const toolbar=h.root.children.find(n=>n.className==='sketch-toolbar');assert.equal(toolbar.children.find(n=>n.textContent==='Undo').disabled,true);assert.equal(toolbar.children.find(n=>n.textContent==='Redo').disabled,true);
 anchor({node:7,box:[0,0,100,100]});await h.flush();await ui.undo();assert.equal(h.sent.at(-1).kind,'sketch-undo');ui.dispose();
});
for(const completion of ['reject','timeout'])for(const released of [false,true])test(`retired ${released?'released':'open'} pending anchor ${completion} cannot cancel successor after prior terminal confirmation`,async()=>{
 const {mountSketch}=await mod();const h=harness(),send=h.options.send;let firstEnd=true,endResult,lateEnd,oldReject,oldResolve;
 h.options.send=async(k,b)=>{const value=await send(k,b);if(k==='sketch-stroke-end'&&firstEnd){firstEnd=false;endResult=value;return new Promise(r=>lateEnd=r);}return value;};
 h.options.status=async()=>endResult;
 const ui=mountSketch(h.doc,h.win,h.view,h.options);await ui.enter();h.event(ui.layer,'pointerdown');h.event(ui.layer,'pointerup',{buttons:0});await h.flush();h.tick(1000);
 h.scene.hitTest=()=>new Promise((resolve,reject)=>{oldResolve=resolve;oldReject=reject;});h.event(ui.layer,'pointerdown',{pointerId:2});if(released)h.event(ui.layer,'pointerup',{pointerId:2,buttons:0});await h.flush();h.context.fills=[];h.tick(4000);await h.flush();assert.equal(ui.state().phase,'reconciling');assert.equal(h.context.fills.some(a=>a<1),false);assert.match(h.root.children.find(n=>n.className==='sketch-status').textContent,/Couldn't save this stroke; accepted strokes kept/);ui.retry();await h.flush();assert.equal(ui.state().phase,'drawing');
 let freshResolve;h.scene.hitTest=()=>new Promise(r=>freshResolve=r);h.event(ui.layer,'pointerdown',{pointerId:3});await h.flush();
 if(completion==='reject')oldReject(Error('retired anchor'));else h.tick(1000);await h.flush();assert.equal(ui.layer.hasPointerCapture(3),true);
 h.event(ui.layer,'pointerup',{pointerId:3,buttons:0});freshResolve({node:7,box:[0,0,100,100]});await h.flush();assert.equal(h.model.sketchProjection().sketches[0].strokes.length,2);
 oldResolve({node:7,box:[0,0,100,100]});lateEnd(endResult);await h.flush();assert.equal(h.model.sketchProjection().sketches[0].strokes.length,2);ui.dispose();
});

test('viewer required label Enter saves once and waits for accepted reply before switching tools',async()=>{
 const {mountSketch}=await mod();const h=harness(),send=h.options.send;let accept;
 h.options.send=async(kind,body)=>{const result=await send(kind,body);return kind==='sketch-label'?new Promise(resolve=>accept=()=>resolve(result)):result;};
 const ui=mountSketch(h.doc,h.win,h.view,h.options);await ui.enter();await ui.leave('point');
 h.event(ui.labelInput,'keydown',{key:'Enter'});await h.flush();assert.equal(h.sent.filter(r=>r.kind==='sketch-label').length,0);
 ui.labelInput.value='Required intent';h.event(ui.labelInput,'keydown',{key:'Enter',isComposing:true});await h.flush();assert.equal(h.sent.filter(r=>r.kind==='sketch-label').length,0);
 h.event(ui.labelInput,'keydown',{key:'Enter'});await h.flush();assert.equal(h.sent.filter(r=>r.kind==='sketch-label').length,1);assert.deepEqual(h.tools,[]);assert.equal(h.scene.ready,true);
 h.event(ui.labelInput,'keydown',{key:'Enter',repeat:true});await h.flush();assert.equal(h.sent.filter(r=>r.kind==='sketch-label').length,1);
 accept();await h.flush();assert.deepEqual(h.tools,['point']);assert.equal(h.scene.ready,false);assert.equal(h.model.sketchProjection().sketches[0].label,'Required intent');h.event(ui.labelInput,'keydown',{key:'Enter'});await h.flush();assert.deepEqual(h.tools,['point']);ui.dispose();
});

test('live exact feedback mask clears accepted ink while preserving vectors and newer versions',async()=>{
 const {createFeedbackMask}=await import('../../cmd/internal/webcast/web/feedback-mask.js');
 const {mountSketch}=await mod(),h=harness();const hooks=new Set();
 h.view.onFeedback=fn=>{hooks.add(fn);return()=>hooks.delete(fn);};
 h.view.feedbackMask=createFeedbackMask({changed:()=>{for(const fn of hooks)fn();}});
 let clears=0;h.context.clearRect=()=>clears++;
 const ui=mountSketch(h.doc,h.win,h.view,h.options);await ui.enter();
 h.event(ui.layer,'pointerdown');await h.flush();h.event(ui.layer,'pointerup',{buttons:0});await h.flush();
 const before=h.model.sketchProjection(),meta=h.model.sketchMetadata('server-s');
 assert.ok(h.context.paintedPoints>0);h.context.paintedPoints=0;clears=0;
 h.view.feedbackMask.install({scope:'cast',revision:'1',sketches:[{id:'server-s',next_sequence:String(meta.next_sequence),digest:meta.digest}],notes:[]});await h.flush();
 assert.ok(clears>0,'mask must invalidate and erase old canvas pixels');assert.equal(h.context.paintedPoints,0);
 assert.deepEqual(h.model.sketchProjection(),before,'mask cannot alter accepted history');
 await ui.leave();ui.labelInput.value='new version';await ui.saveLabel();await h.flush();
 assert.ok(h.context.paintedPoints>0,'new accepted version must reappear');ui.dispose();
});

test('placed sketch Add note follows owned finish flow and final View note is read only',async()=>{
  const {mountSketch}=await mod(),h=harness(),ui=mountSketch(h.doc,h.win,h.view,h.options);
  await ui.enter();h.event(ui.layer,'pointerdown');await h.flush();h.event(ui.layer,'pointerup');await h.flush();
  const control=h.root.querySelector('.sketch-note-control');assert.ok(control,'placed sketch exposes note affordance');assert.equal(control.textContent,'Add note');
  h.event(control,'click');await h.flush();assert.equal(h.root.querySelector('.sketch-label').hidden,false);
  ui.labelInput.value='Final immutable note';await ui.saveLabel();await h.flush();
  const final=h.root.querySelector('.sketch-note-control');assert.equal(final,control);assert.equal(final.textContent,'View note');
  const sent=h.sent.length;h.event(final,'click');assert.equal(h.root.querySelector('.sketch-note-text').textContent,'Final immutable note');assert.equal(h.sent.length,sent);
  ui.dispose();assert.equal(h.root.querySelector('.sketch-note-controls'),undefined);
});

test('cached painted bounds retain note discovery after scroll hides the first point',async()=>{
 const {mountSketch}=await mod(),h=harness();
 h.box.getBoundingClientRect=()=>({left:0,top:0,width:300,height:200});
 const ui=mountSketch(h.doc,h.win,h.view,h.options);
 await ui.enter();h.event(ui.layer,'pointerdown',{clientX:20,clientY:10});await h.flush();
 h.event(ui.layer,'pointermove',{clientX:50,clientY:100});h.tick(50);await h.flush();h.event(ui.layer,'pointerup',{clientX:70,clientY:140});await h.flush();
 const button=h.root.querySelector('.sketch-note-control');assert.ok(button);
 button.getBoundingClientRect=()=>({width:80,height:44});
 // Existing transform invalidation must discard old bounds and repaint them
 // in current local coordinates, including later ink still in the viewport.
 h.scene.fromPage=(x,y)=>[x,y-100];ui.refresh({geometry:true});await h.flush();
 assert.equal(h.root.querySelector('.sketch-note-control'),button);
 assert.equal(button.textContent,'Add note');
 assert.ok(Number.parseFloat(button.style.top)>=48,'button sits below the visible tail with a gap');
 const count=h.context.paintedPoints;ui.refresh();await h.flush();
 assert.equal(h.context.paintedPoints,count,'same transform reuses painted bounds without another point walk');
 assert.equal(h.root.querySelector('.sketch-note-control'),button);
 ui.dispose();
});
