import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
const mod = () => import('../../cmd/internal/webcast/web/sketch-render.js');
const bootstrap = () => ({
  sketches: [
    {
      begin: { id: 's', author: { id: 'a', color: '#ff0000' }, doc: 1, tab: 't' },
      phase: 'drawing',
      reason: '',
      label: '',
      strokes: [
        {
          begin: { id: 'p', at: { node: 7, box: [0, 0, 10, 10] }, width: 4 },
          pts: [],
          visible: false,
          sealed: false,
          truncated: false,
        },
      ],
    },
  ],
});
test('authoritative display facts append only new points and keep caller/projected objects owned', async () => {
  const { createSketchDisplay } = await mod();
  for (const n of [64, 128]) {
    const d = createSketchDisplay();
    d.install(bootstrap());
    for (let at = 0; at < n; at += 64)
      d.apply({
        sketch: 's',
        points: { id: 'p', offset: at, pts: Array.from({ length: 64 }, (_, i) => [at + i, 2, 0.5]) },
      });
    assert.equal(d.stats().points, n);
    const before = d.snapshot();
    const patch = { sketch: 's', stroke: { id: 'p', visible: true, sealed: true } };
    d.apply(patch);
    assert.equal(d.stats().points, n);
    assert.equal(d.paths().length, 1);
    const publicView = d.snapshot();
    publicView.sketches[0].strokes[0].pts[0][0] = 999;
    assert.equal(d.snapshot().sketches[0].strokes[0].pts[0][0], 0);
    d.apply({ sketch: 's', stroke: { id: 'p', visible: false } });
    assert.equal(d.paths().length, 0);
    assert.equal(before.sketches[0].strokes[0].pts.length, n);
  }
});
test('display batch refuses gaps/conflicts atomically, exact echoed ranges cost zero', async () => {
  const { createSketchDisplay } = await mod();
  const d = createSketchDisplay();
  d.install(bootstrap());
  const patch = { sketch: 's', points: { id: 'p', offset: 0, pts: [[1, 2, 1]] } };
  d.apply(patch);
  d.apply(patch);
  assert.equal(d.stats().points, 1);
  const before = d.snapshot();
  assert.throws(() =>
    d.applyBatch([
      { sketch: 's', points: { id: 'p', offset: 1, pts: [[3, 4, 1]] } },
      { sketch: 's', points: { id: 'p', offset: 9, pts: [[1, 1, 1]] } },
    ]),
  );
  assert.deepEqual(d.snapshot(), before);
  assert.throws(() => d.apply({ ...patch, points: { ...patch.points, pts: [[8, 8, 1]] } }));
});
test('core projections and compact display select the same visible and live partial paths', async () => {
  const { displayFromSketchProjection, createSketchDisplay } = await mod();
  const corpus = JSON.parse(readFileSync(new URL('./sketch-corpus.json', import.meta.url), 'utf8'));
  for (const c of corpus.cases)
    for (const prefix of c.prefixes) {
      const view = displayFromSketchProjection(prefix.projection),
        d = createSketchDisplay();
      d.install(view);
      const want = prefix.projection.sketches.flatMap((s) =>
        s.phase === 'cleared'
          ? []
          : s.strokes
              .filter((p) => s.visible.includes(p.begin.id) || (!p.sealed && p.pts.length))
              .map((p) => [s.begin.id, p.begin.id, p.pts]),
      );
      assert.deepEqual(
        d.paths().map((p) => [p.sketch, p.id, p.pts]),
        want,
      );
    }
});
test('pressure painter shares affine fixtures and keeps width independent of anchor aspect ratio', async () => {
  const { projectSketchPoint, paintSketchPaths } = await mod();
  const f = JSON.parse(readFileSync(new URL('./sketch-geometry.json', import.meta.url), 'utf8'));
  for (const c of f.points) assert.deepEqual(projectSketchPoint(c.point, c.from, c.to), c.want);
  const calls = [];
  const g = {
    save() {},
    restore() {},
    beginPath() {},
    moveTo() {},
    lineTo() {},
    closePath() {},
    fill() {},
    arc(x, y, r) {
      calls.push([x, y, r]);
    },
  };
  paintSketchPaths(
    g,
    [
      {
        pts: [
          [1, 2, 0],
          [3, 4, 1],
        ],
        width: 4,
        color: '#ff0000',
      },
    ],
    { toClient: (x, y) => [x * 2, y * 2], scale: 2 },
  );
  assert.deepEqual(calls, [
    [2, 4, 1],
    [6, 8, 4],
  ]);
});
test('bootstrap counts participate in append admission, rejected growth leaves full display intact', async () => {
  const { createSketchDisplay } = await mod();
  const b = bootstrap();
  b.sketches[0].strokes = Array.from({ length: 4 }, (_, i) => ({
    ...structuredClone(b.sketches[0].strokes[0]),
    begin: { ...b.sketches[0].strokes[0].begin, id: `p${i}` },
    pts: Array.from({ length: 4096 }, () => [1, 1, 1]),
  }));
  const d = createSketchDisplay();
  d.install(b);
  const before = d.snapshot();
  assert.throws(() =>
    d.applyBatch([
      { sketch: 's', stroke: { begin: { id: 'new', at: { node: 7, box: [0, 0, 10, 10] }, width: 4 } } },
      { sketch: 's', points: { id: 'new', offset: 0, pts: [[1, 2, 1]] } },
    ]),
  );
  assert.deepEqual(d.snapshot(), before);
});

test('8/16 dirty chunks schedule one paint, disposal cancels late frame work', async () => {
  const { createSketchPaintScheduler } = await mod();
  for (const n of [8, 16]) {
    const callbacks = [];
    let paints = 0;
    const s = createSketchPaintScheduler({ schedule: (fn) => callbacks.push(fn), paint: () => paints++ });
    for (let i = 0; i < n; i++) s.invalidate();
    assert.equal(callbacks.length, 1);
    callbacks.shift()();
    assert.equal(paints, 1);
    s.invalidate();
    s.dispose();
    callbacks.shift()();
    assert.equal(paints, 1);
    assert.deepEqual(s.stats(), { scheduled: 2, painted: 1 });
  }
});

test('compact bootstrap shows accepted open partials and omits cleared historical sketches', async()=>{const {displayFromSketchProjection}=await mod();const p={sketches:[{begin:{id:'s',author:{id:'a',color:'#ff0000'},doc:1,tab:'t'},phase:'drawing',reason:'',label:'',visible:[],strokes:[{begin:{id:'p',at:{node:1,box:[0,0,10,10]},width:4},pts:[[1,2,.5]],sealed:false,truncated:false}]}]};assert.equal(displayFromSketchProjection(p).sketches[0].strokes[0].visible,true);p.sketches[0].phase='cleared';assert.deepEqual(displayFromSketchProjection(p),{sketches:[]});});
