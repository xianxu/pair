import { test } from 'node:test';
import assert from 'node:assert/strict';
const mod = () => import('../../cmd/internal/webcast/web/sketch-scene.js');
function fixture(text = 'frozen') {
  const records = [
    { k: 'head', t: 0, format: 'couch-webcast', v: 1, id: 'scene', started: 'now' },
    {
      k: 'seg',
      t: 1,
      seg: 1,
      why: 'live',
      tab: { id: 'tab', url: 'https://old/' },
      vp: { w: 100, h: 50, dpr: 1 },
    },
    {
      k: 'doc',
      t: 2,
      doc: 1,
      url: 'https://old/',
      tree: {
        id: 1,
        y: 'doc',
        c: [
          {
            id: 2,
            y: 'el',
            n: 'html',
            c: [{ id: 7, y: 'el', n: 'body', c: [{ id: 8, y: 'text', v: text }] }],
          },
        ],
      },
    },
    {
      k: 'sketch-begin',
      t: 3,
      id: 's',
      author: { id: 'operator', color: '#ff0000' },
      owner: 'operator',
      doc: 1,
      tab: 'tab',
      vp: { w: 100, h: 50, dpr: 1 },
      operation: 'begin',
      sequence: 0,
      digest: 'a'.repeat(64),
    },
  ];
  const bytes = new TextEncoder().encode(records.map((r) => JSON.stringify(r)).join('\n') + '\n');
  const source = {
    handle: 'h',
    time: 3,
    length: bytes.length,
    doc: 1,
    tab: 'tab',
    vp: { w: 100, h: 50, dpr: 1 },
  };
  let reads = 0,
    builds = 0,
    revoked = 0,
    pauses = 0,
    finished = 0;
  const applied = [];
  const frame = {
    style: {},
    contentWindow: { scrollX: 3, scrollY: 5 },
    getBoundingClientRect: () => ({ left: 20, top: 30, width: 200, height: 100 }),
  };
  const anim = {
    finish() {
      finished++;
    },
    pause() {
      pauses++;
    },
  };
  const document = {
    getAnimations: () => [anim],
    querySelectorAll: () => [],
    defaultView: frame.contentWindow,
  };
  return {
    records,
    bytes,
    source,
    frame,
    read: async (_, offset, length) => {
      reads++;
      return bytes.slice(offset, offset + length);
    },
    renderer: (_, model, resolve) => {
      builds++;
      return {
        dom: new Map(),
        topDoc: document,
        frame,
        apply: (r) => applied.push(r.k),
        flush() {},
        ready: async () => {},
        revokeAll() {
          revoked++;
        },
        nodeAt: () => 7,
        boxOf: () => [10, 20, 30, 40],
        placementOf: () => [10, 20, 30, 40],
        canonical: () => model.canonical(),
        resolve,
      };
    },
    stats: () => ({ reads, builds, revoked, pauses, finished, applied }),
  };
}
test('held source validates acknowledged prefix, freezes animations and maps its own coordinates', async () => {
  const { createSketchScene } = await mod();
  const f = fixture('x'.repeat(40000));
  const s = createSketchScene({
    frame: f.frame,
    readSource: f.read,
    rendererFactory: f.renderer,
    assetURL: () => '',
  });
  await s.prepare(f.source, { sketch: 's', epoch: 1 });
  assert.equal(s.ready, true);
  assert.equal(f.stats().reads, 2);
  assert.equal(f.stats().builds, 1);
  assert.equal(f.stats().pauses, 1);
  assert.equal(f.stats().finished, 1);
  assert.deepEqual(await s.hitTest([11, 22]), { node: 7, box: [10, 20, 30, 40] });
  assert.deepEqual(s.toPage(40, 50), [13, 15]);
  assert.deepEqual(s.fromPage(13, 15), [40, 50]);
  const snapshot = s.snapshot();
  snapshot.doc.c[0].c[0].c[0].v = 'mutated';
  assert.equal(s.snapshot().doc.c[0].c[0].c[0].v.length, 40000);
  s.dispose();
  assert.equal(f.stats().revoked, 1);
  assert.equal(s.ready, false);
  await assert.rejects(s.hitTest([1, 1]));
});
test('source identity and byte bounds refuse before scene publication; disposal fences a late read', async () => {
  const { createSketchScene } = await mod();
  const f = fixture();
  const bad = createSketchScene({ frame: f.frame, readSource: f.read, rendererFactory: f.renderer });
  await assert.rejects(bad.prepare({ ...f.source, time: 99 }, { sketch: 's', epoch: 1 }));
  assert.equal(bad.ready, false);
  assert.equal(f.stats().builds, 0);
  await assert.rejects(bad.prepare({ ...f.source, length: 16 * 1024 * 1024 + 1 }, { sketch: 's', epoch: 1 }));
  let resolve;
  const held = createSketchScene({
    frame: f.frame,
    readSource: () => new Promise((r) => (resolve = r)),
    rendererFactory: f.renderer,
  });
  const preparing = held.prepare(f.source, { sketch: 's', epoch: 1 });
  held.dispose();
  resolve(f.bytes);
  await assert.rejects(preparing);
  assert.equal(held.ready, false);
  assert.equal(f.stats().builds, 0);
});
test('read-only placement requires exact source identity and does not borrow live DOM state', async () => {
  const { createSketchScene } = await mod();
  const f = fixture();
  const scene = createSketchScene({ frame: f.frame, readSource: f.read, rendererFactory: f.renderer });
  await scene.prepare(f.source, { sketch: 's', epoch: 1 });
  assert.deepEqual(
    await scene.placement({ source: { doc: 1, tab: 'tab', t: 3 }, at: { node: 7 } }),
    [10, 20, 30, 40],
  );
  assert.equal(await scene.placement({ source: { doc: 2, tab: 'tab', t: 3 }, at: { node: 7 } }), null);
  assert.equal(await scene.placement({ source: { doc: 1, tab: 'other', t: 3 }, at: { node: 7 } }), null);
});

test('shared renderer original-node proof for sketches requires source document, tab and time', async () => {
  const { hasSketchSource } = await import('../../cmd/internal/webcast/web/render.js');
  const model = { docSeg: 2, tab: { id: 't' }, t: 10 };
  assert.equal(hasSketchSource({ source: { doc: 2, tab: 't', t: 3 } }, model), true);
  for (const source of [
    null,
    { doc: 1, tab: 't', t: 3 },
    { doc: 2, tab: 'other', t: 3 },
    { doc: 2, tab: 't', t: 11 },
    { doc: 2, tab: 't' },
    { doc: 2, tab: 't', t: -1 },
  ])
    assert.equal(hasSketchSource({ source }, model), false);
});
test('acknowledged viewport cannot differ from the prefix used to draw', async () => {
  const { createSketchScene } = await mod();
  const f = fixture();
  const scene = createSketchScene({ frame: f.frame, readSource: f.read, rendererFactory: f.renderer });
  await assert.rejects(
    () => scene.prepare({ ...f.source, vp: { w: 200, h: 50, dpr: 1 } }, { sketch: 's' }),
    /identity/,
  );
  assert.equal(f.stats().builds, 0);
});
test('renderer construction failure retires prepared private asset URLs', async () => {
  const { createSketchScene } = await mod();
  const f = fixture();
  let revoked = 0;
  const scene = createSketchScene({
    frame: f.frame,
    readSource: f.read,
    prepareAssets: async () => ({
      dispose() {
        revoked++;
      },
    }),
    rendererFactory() {
      throw Error('cannot render');
    },
  });
  await assert.rejects(() => scene.prepare(f.source, { sketch: 's' }), /cannot render/);
  assert.equal(revoked, 1);
  assert.equal(scene.source, null);
});

test('held scene renders historical mutations in order before later SPA root replacement', async () => {
  const { createSketchScene } = await mod();
  const f = fixture();
  const records = [...f.records.slice(0, -1),
    { k: 'attr', t: 3, id: 7, name: 'class', v: 'old-root' },
    { k: 'attr', t: 4, id: 7, name: 'style', v: 'color:red' },
    { k: 'text', t: 5, id: 8, v: 'changed before replacement' },
    { k: 'kids', t: 6, p: 2, c: [{ id: 20, y: 'el', n: 'body', c: [{ id: 21, y: 'text', v: 'new screen' }] }] },
    { ...f.records.at(-1), t: 7 },
  ];
  const bytes = new TextEncoder().encode(records.map(r => JSON.stringify(r)).join('\n') + '\n');
  let renderModel, applied = 0;
  const scene = createSketchScene({ frame: f.frame,
    readSource: async (_, offset, length) => bytes.slice(offset, offset + length),
    rendererFactory(frame, model, resolve) {
      renderModel = model;
      const renderer = f.renderer(frame, model, resolve);
      return { ...renderer, nodeAt: () => model.nodes.has(20) ? 20 : 0, apply(r) {
        assert.equal(model.t, r.t, 'renderer must see this record time, not the final begin time');
        if (r.k === 'attr') assert.equal(model.nodes.get(r.id)?.a?.[r.name], r.v, 'historical attribute node must still exist while rendered');
        if (r.k === 'text') assert.equal(model.nodes.get(r.id)?.v, r.v);
        if (r.k === 'kids') { assert.equal(model.nodes.has(7), false); assert.equal(model.nodes.get(21)?.v, 'new screen'); }
        applied++;
        renderer.apply(r);
      } };
    },
  });
  await scene.prepare({ ...f.source, time: 7, length: bytes.length }, { sketch: 's' });
  assert.equal(applied, records.length);
  assert.equal(renderModel.nodes.has(7), false);
  assert.equal(scene.ready, true);
  assert.equal(scene.snapshot().doc.c[0].c[0].c[0].v, 'new screen');
  assert.deepEqual(await scene.hitTest([11,22]), { node: 20, box: [10,20,30,40] });
  scene.dispose();
  assert.equal(f.stats().revoked, 1);
});
