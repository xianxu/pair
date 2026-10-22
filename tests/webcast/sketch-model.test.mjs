import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { projectPoint, pressureWidth } from '../../cmd/internal/webcast/web/annotation-anchor.js';
const geometry = JSON.parse(readFileSync(new URL('./sketch-geometry.json', import.meta.url), 'utf8'));
for (const c of geometry.points)
  test(`geometry ${c.name}`, () => assert.deepEqual(projectPoint(c.point, c.from, c.to), c.want));
for (const c of geometry.widths)
  test(`width ${c.base}/${c.pressure}`, () => {
    const got = pressureWidth(c.base, c.pressure);
    if (c.want === null) assert.ok(Number.isNaN(got));
    else assert.equal(got, c.want);
  });
test('point projection preserves original paths and rejects malformed/nonfinite values', () => {
  const p = [3, 4, 0.2];
  assert.deepEqual(projectPoint(p, [0, 0, 10, 10], [0, 0, 20, 30]), [6, 12, 0.2]);
  assert.deepEqual(p, [3, 4, 0.2]);
  for (const p of [[1], [1, 2, 1, 4], [NaN, 2, 1], [1, Infinity, 1], [1, 2, null]])
    assert.equal(projectPoint(p, [0, 0, 10, 10], [0, 0, 10, 10]), null);
});

const sketchModule = () => import('../../cmd/internal/webcast/web/sketch-model.js');
const begin = () => ({
  t: 3,
  k: 'sketch-begin',
  id: 's',
  author: { id: 'operator', color: '#123456' },
  owner: 'operator',
  doc: 1,
  tab: 'a',
  vp: { w: 100, h: 100, dpr: 1 },
  operation: 'op0',
  sequence: 0,
  digest: 'a'.repeat(64),
});
test('known sketch validation rejects malformed browser counters, tuples, identities and anchors', async () => {
  const { checkSketchRecord } = await sketchModule();
  assert.doesNotThrow(() => checkSketchRecord(begin()));
  for (const changed of [
    { sequence: 2 ** 53 },
    { sequence: -1 },
    { operation: 'bad\u200b' },
    { owner: 'cookie' },
    { author: { id: 'operator', color: 'red' } },
    { vp: { w: 0, h: 100, dpr: 1 } },
    { digest: 'A'.repeat(64) },
  ])
    assert.throws(() => checkSketchRecord({ ...begin(), ...changed }));
  const points = { ...begin(), k: 'sketch-points', sketch: 's', id: 'p', chunk: 0, pts: [[1, 2, 0.5]] };
  assert.doesNotThrow(() => checkSketchRecord(points));
  for (const pts of [
    [[1, 2]],
    [[1, 2, 1, 4]],
    [[1, 2, -0.1]],
    [[1e8, 2, 1]],
    Array.from({ length: 65 }, () => [1, 2, 1]),
  ])
    assert.throws(() => checkSketchRecord({ ...points, pts }));
});
test('canonical semantic identity strips additive unknowns but preserves known nested fields', async () => {
  const { sketchCanonical, sketchDigest } = await sketchModule();
  const r = begin(),
    unknown = {
      ...r,
      future: { effect: 'ignored' },
      author: { ...r.author, future: 1 },
      vp: { ...r.vp, future: 1 },
    };
  assert.equal(sketchCanonical(r), sketchCanonical(unknown));
  assert.equal(sketchCanonical({ ...r, t: 100, k: 'sketch-begin' }), sketchCanonical(r));
  assert.notEqual(sketchCanonical({ ...r, owner: 'b'.repeat(64) }), sketchCanonical(r));
  const { createHash } = await import('node:crypto');
  assert.equal(await sketchDigest(r), createHash('sha256').update(sketchCanonical(r)).digest('hex'));
});
test('canonical maps order every key by UTF8 bytes including numeric and supplementary names', async () => {
  const { sketchCanonical } = await sketchModule();
  const r = {
    ...begin(),
    k: 'sketch-stroke-begin',
    sketch: 's',
    id: 'p',
    width: 2,
    at: {
      node: 1,
      box: [0, 0, 10, 10],
      stable: { 2: 'two', 10: 'ten', '\u{10000}': 'astral', '\ue000': 'private' },
    },
  };
  const s = sketchCanonical(r);
  assert.ok(s.indexOf('"10"') < s.indexOf('"2"'));
  assert.ok(s.indexOf('"\ue000"') < s.indexOf('"\u{10000}"'));
});
test('wire validation covers safe timestamp, bounded author and existing anchor kind', async () => {
  const { checkSketchRecord } = await sketchModule();
  for (const changed of [
    { t: -1 },
    { t: 2 ** 53 },
    { author: { id: 'operator', color: '#123456', name: 'x'.repeat(129) } },
    { vp: { w: 100, h: 100, dpr: 1e8 } },
  ])
    assert.throws(() => checkSketchRecord({ ...begin(), ...changed }));
  assert.throws(() =>
    checkSketchRecord({
      ...begin(),
      k: 'sketch-stroke-begin',
      sketch: 's',
      id: 'p',
      width: 2,
      at: { node: 1, box: [0, 0, 10, 10], kind: 'selector' },
    }),
  );
});
test('pure core preserves receipt identity, empty branches and interruption evidence', async () => {
  const { createSketchState, nextSketch, sketchProjection } = await sketchModule();
  const s = createSketchState();
  const ctx = { live: true, ended: false, doc: 1, tab: 'a', driver: '' };
  const apply = (r) => {
    const d = nextSketch(s, r, ctx);
    if (!d.duplicate) d.commit();
    return d;
  };
  const op = (n, kind, more = {}) => ({ ...begin(), k: kind, operation: `op${n}`, sequence: n, ...more });
  apply(begin());
  apply(
    op(1, 'sketch-stroke-begin', { sketch: 's', id: 'p', at: { node: 9, box: [0, 0, 10, 10] }, width: 4 }),
  );
  const points = op(2, 'sketch-points', { sketch: 's', id: 'p', chunk: 0, pts: [[1, 2, 0.5]] });
  apply(points);
  const before = sketchProjection(s);
  assert.equal(apply({ ...points, t: 100 }).duplicate, true);
  assert.deepEqual(sketchProjection(s), before);
  assert.throws(() => apply({ ...points, pts: [[2, 2, 0.5]] }));
  assert.deepEqual(sketchProjection(s), before);
  apply(op(3, 'sketch-stroke-end', { sketch: 's', id: 'p' }));
  apply(op(4, 'sketch-undo', { sketch: 's', id: 'p' }));
  assert.deepEqual(sketchProjection(s).sketches[0].redo, ['p']);
  apply(
    op(5, 'sketch-stroke-begin', {
      sketch: 's',
      id: 'empty',
      at: { node: 9, box: [0, 0, 10, 10] },
      width: 4,
    }),
  );
  apply(op(6, 'sketch-stroke-end', { sketch: 's', id: 'empty' }));
  assert.deepEqual(sketchProjection(s).sketches[0].visible, []);
  assert.deepEqual(sketchProjection(s).sketches[0].redo, []);
  apply(op(7, 'sketch-interrupt', { sketch: 's', reason: 'expiry' }));
  apply(op(8, 'sketch-label', { sketch: 's', label: 'partial' }));
  assert.equal(sketchProjection(s).sketches[0].phase, 'interrupted');
  assert.equal(sketchProjection(s).sketches[0].reason, 'expiry');
  assert.throws(() => apply(op(9, 'sketch-label', { sketch: 's', label: 'again' })));
});
test('Model routes sketches and context atomically and leaves DOM canonical unchanged', async () => {
  const { Model } = await import('../../cmd/internal/webcast/web/model.js');
  const m = new Model();
  m.apply({ k: 'head', t: 0, format: 'couch-webcast', v: 1, id: 'r', started: 'now' });
  m.apply({ k: 'seg', t: 1, seg: 1, why: 'live', tab: { id: 'a' }, vp: { w: 100, h: 100, dpr: 1 } });
  const canonical = m.canonical();
  m.apply(begin());
  assert.deepEqual(m.canonical(), canonical);
  assert.equal(m.sketchProjection().active, 's');
  const before = structuredClone({ t: m.t, control: m.control, sketch: m.sketchProjection() });
  assert.throws(() => m.apply({ k: 'ctl', t: 5, op: 'grant', who: { id: 'driver' }, driver: null }));
  assert.deepEqual({ t: m.t, control: m.control, sketch: m.sketchProjection() }, before);
  m.apply({ k: 'ctl', t: 6, op: 'reset', who: { id: 'operator' } });
  assert.equal(m.sketchProjection().sketches[0].reason, 'reset');
});
test('canonical JSON uses exact Go HTML-safe escaping', async () => {
  const { sketchCanonical } = await sketchModule();
  const r = { ...begin(), author: { ...begin().author, name: '<>&\u2028\u2029' } };
  const got = sketchCanonical(r);
  assert.ok(got.includes('"name":"\\u003c\\u003e\\u0026\\u2028\\u2029"'));
  assert.ok(got.includes('"sequence":0'));
});
const corpus = JSON.parse(readFileSync(new URL('./sketch-corpus.json', import.meta.url), 'utf8'));
for (const c of corpus.cases)
  test(`Go prefix contract ${c.name}`, async () => {
    const { Model } = await import('../../cmd/internal/webcast/web/model.js');
    const m = new Model();
    for (let i = 0; i < c.records.length; i++) {
      const before = { t: m.t, control: structuredClone(m.control), projection: m.sketchProjection() };
      let error = false;
      try {
        m.apply(c.records[i]);
      } catch {
        error = true;
      }
      assert.equal(error, c.prefixes[i].error, `error prefix ${i}`);
      if (error)
        assert.deepEqual(
          { t: m.t, control: m.control, projection: m.sketchProjection() },
          before,
          `atomic prefix ${i}`,
        );
      assert.equal(m.t, c.prefixes[i].t, `time prefix ${i}`);
      assert.deepEqual(m.sketchProjection(), c.prefixes[i].projection, `projection prefix ${i}`);
      const normalized = {};
      if (m.control.driver) normalized.driver = m.control.driver;
      if (m.control.pending.length) normalized.pending = m.control.pending;
      assert.deepEqual(normalized, c.prefixes[i].control, `control prefix ${i}`);
    }
  });
for (const [i, c] of corpus.canonical.entries())
  test(`Go canonical bytes and digest ${i}`, async () => {
    const { sketchCanonical, sketchDigest } = await sketchModule();
    assert.equal(sketchCanonical(c.record), c.canonical);
    assert.equal(await sketchDigest(c.record), c.digest);
  });
test('counted N/2N append and stack operations visit only newly accepted points', async () => {
  const { createSketchState, nextSketch } = await sketchModule();
  for (const n of [64, 128]) {
    const s = createSketchState(),
      ctx = { live: true, doc: 1, tab: 'a', driver: '' };
    const apply = (r) => {
      const d = nextSketch(s, r, ctx);
      d.commit();
    };
    apply(begin());
    apply({
      ...begin(),
      k: 'sketch-stroke-begin',
      sequence: 1,
      operation: 'op1',
      sketch: 's',
      id: 'p',
      at: { node: 1, box: [0, 0, 10, 10] },
      width: 4,
    });
    let seq = 2,
      last;
    for (let i = 0; i < n; i += 64) {
      last = {
        ...begin(),
        k: 'sketch-points',
        sketch: 's',
        id: 'p',
        sequence: seq,
        operation: `op${seq++}`,
        chunk: i / 64,
        pts: Array.from({ length: 64 }, () => [1, 2, 1]),
      };
      apply(last);
    }
    apply({ ...last, t: 200 });
    assert.equal(s.work.points, n);
    for (const k of ['sketch-stroke-end', 'sketch-undo', 'sketch-redo'])
      apply({ ...begin(), k, sketch: 's', id: 'p', sequence: seq, operation: `op${seq++}` });
    assert.deepEqual(s.work, { points: n, stacks: 2 });
  }
});
test('nested owned anchor values are typed while future structured stable evidence has no effect', async () => {
  const { checkSketchRecord, sketchCanonical } = await sketchModule();
  const r = {
    ...begin(),
    k: 'sketch-stroke-begin',
    sketch: 's',
    id: 'p',
    width: 4,
    at: { node: 1, box: [0, 0, 10, 10] },
  };
  for (const at of [
    { ...r.at, path: [null] },
    { ...r.at, classes: [1] },
    { ...r.at, text: null },
    { ...r.at, stable: { id: null } },
  ])
    assert.throws(() => checkSketchRecord({ ...r, at }));
  assert.equal(sketchCanonical({ ...r, at: { ...r.at, stable: { future: { x: 1 } } } }), sketchCanonical(r));
});
test('owned optional author fields reject null and nonstrings', async () => {
  const { checkSketchRecord } = await sketchModule();
  for (const name of [null, 0, false, {}, []])
    assert.throws(() => checkSketchRecord({ ...begin(), author: { ...begin().author, name } }));
});
function reduceModel(Model, recs, options) {
  const m = new Model(options);
  for (const r of recs) m.apply(r);
  return m;
}
test('catch-up snapshot installs owned history and strict current control; raw model refuses it', async () => {
  const { Model } = await import('../../cmd/internal/webcast/web/model.js');
  const c = corpus.cases.find((c) => c.name === 'history');
  const recs = c.records.filter((_, i) => !c.prefixes[i].error).slice(0, 6);
  const source = reduceModel(Model, recs);
  const frame = {
    k: 'sketch-snapshot',
    t: source.t,
    control: source.control,
    sketches: source.sketchProjection(),
  };
  const target = reduceModel(Model, recs.slice(0, 3), { allowSketchSnapshot: true });
  assert.throws(() => reduceModel(Model, [...recs.slice(0, 3), frame]));
  target.apply(frame);
  assert.deepEqual(target.sketchProjection(), source.sketchProjection());
  frame.sketches.sketches[0].begin.author.color = 'bad';
  assert.equal(target.sketchProjection().sketches[0].begin.author.color, '#ff0000');
  const before = target.sketchProjection();
  const bad = { ...frame, sketches: { ...frame.sketches, active: 'missing' } };
  assert.throws(() => target.apply(bad));
  assert.deepEqual(target.sketchProjection(), before);
});
test('cast end invalidates an outstanding late label and unknowns never enter authority', async () => {
  const { Model } = await import('../../cmd/internal/webcast/web/model.js');
  const c = corpus.cases.find((c) => c.name === 'history');
  const recs = c.records.slice(0, 4);
  const m = reduceModel(Model, recs);
  m.apply({ k: 'stop', t: 4, why: 'end' });
  assert.throws(() =>
    m.apply({
      ...begin(),
      k: 'sketch-label',
      t: 5,
      sketch: 'sk',
      sequence: 1,
      operation: 'op1',
      label: 'late',
    }),
  );
  const n = reduceModel(Model, recs.slice(0, 3));
  n.apply({
    ...c.records[3],
    extra: 'ignore',
    author: { ...c.records[3].author, extra: 1 },
    vp: { ...c.records[3].vp, extra: 1 },
  });
  assert.deepEqual(n.sketchProjection(), reduceModel(Model, recs).sketchProjection());
});
for (const c of corpus.cases)
  test(`snapshot catch-up and recompaction ${c.name}`, async () => {
    const { Model } = await import('../../cmd/internal/webcast/web/model.js');
    const { catchUp, Player } = await import('../../cmd/internal/webcast/web/replay.js');
    let recs = [];
    const whole = new Model();
    for (let i = 0; i < c.records.length; i++) {
      if (c.prefixes[i].error) continue;
      const r = c.records[i];
      whole.apply(r);
      recs.push(r);
      for (const caught of [catchUp(recs), catchUp(catchUp(recs))]) {
        const m = reduceModel(Model, caught, { allowSketchSnapshot: true });
        assert.deepEqual(m.sketchProjection(), whole.sketchProjection(), `prefix ${i}`);
        assert.deepEqual(m.control, whole.control);
        assert.deepEqual(m.canonical(), whole.canonical());
      }
      const player = new Player(recs, { renderer: () => ({ apply() {}, flush() {} }) });
      player.seek(whole.t);
      assert.equal(player.error, null);
      assert.deepEqual(player.model.sketchProjection(), whole.sketchProjection());
    }
    assert.throws(
      () =>
        new Player(
          [
            ...recs,
            { k: 'sketch-snapshot', t: whole.t, control: whole.control, sketches: whole.sketchProjection() },
          ],
          { renderer: () => ({ apply() {} }) },
        ),
      /bootstrap|snapshot/,
    );
  });
test('clear seals an open partial stroke so catch-up keeps valid cleared history', async () => {
  const { Model } = await import('../../cmd/internal/webcast/web/model.js');
  const { catchUp } = await import('../../cmd/internal/webcast/web/replay.js');
  const recs = corpus.cases.find((c) => c.name === 'history').records.slice(0, 6);
  recs.push({ k: 'annotation-clear', t: 20, who: { id: 'operator' } });
  const whole = reduceModel(Model, recs),
    caught = reduceModel(Model, catchUp(recs), { allowSketchSnapshot: true });
  assert.deepEqual(caught.sketchProjection(), whole.sketchProjection());
  assert.equal(whole.sketchProjection().sketches[0].strokes[0].sealed, true);
});
test('snapshot paths, chunks, labels and active source must agree with semantic receipts', async () => {
  const { Model } = await import('../../cmd/internal/webcast/web/model.js');
  const recs = corpus.cases.find((c) => c.name === 'history').records.slice(0, 6);
  const source = reduceModel(Model, recs);
  for (const mutate of [
    (f) => {
      f.control = { driver: { id: 'another-driver' } };
    },
    (f) => {
      f.sketches.sketches[0].begin.digest = 'b'.repeat(64);
    },
    (f) => {
      f.sketches.sketches[0].strokes[0].begin.digest = 'b'.repeat(64);
    },
    (f) => {
      f.sketches.sketches[0].strokes[0].pts[0][0] = 9;
    },
    (f) => {
      f.sketches.sketches[0].strokes[0].next_chunk = 0;
    },
    (f) => {
      f.sketches.sketches[0].label = 'forged';
      f.sketches.sketches[0].phase = 'complete';
      f.sketches.active = '';
    },
    (f) => {
      f.sketches.sketches[0].receipts[0].semantic = 'x'.repeat(10000);
    },
  ]) {
    const frame = {
      t: source.t,
      k: 'sketch-snapshot',
      control: source.control,
      sketches: source.sketchProjection(),
    };
    mutate(frame);
    const target = reduceModel(Model, recs.slice(0, 3), { allowSketchSnapshot: true }),
      before = target.sketchProjection();
    assert.throws(() => target.apply(frame));
    assert.deepEqual(target.sketchProjection(), before);
  }
  const target = reduceModel(Model, recs.slice(0, 3), { allowSketchSnapshot: true });
  target.tab = { ...target.tab, id: 'different' };
  assert.throws(() =>
    target.apply({
      t: source.t,
      k: 'sketch-snapshot',
      control: source.control,
      sketches: source.sketchProjection(),
    }),
  );
});

test('snapshot receipt operations must form a valid history through the transition owner', async () => {
  const { Model } = await import('../../cmd/internal/webcast/web/model.js');
  const { sketchCanonical } = await sketchModule();
  const recs = corpus.cases.find((c) => c.name === 'history').records.slice(0, 4);
  const source = reduceModel(Model, recs),
    projection = source.sketchProjection(),
    view = projection.sketches[0];
  const invalid = {
    ...view.begin,
    k: 'sketch-undo',
    sketch: view.begin.id,
    id: 'missing',
    operation: 'missing-undo',
    sequence: view.next_sequence,
  };
  const semantic = 'sketch-undo:' + sketchCanonical(invalid);
  view.receipts.push({
    operation: invalid.operation,
    sequence: invalid.sequence,
    digest: invalid.digest,
    semantic,
  });
  view.next_sequence++;
  projection.history_bytes += 1000;
  const target = reduceModel(Model, recs.slice(0, 3), { allowSketchSnapshot: true });
  const before = target.sketchProjection();
  assert.throws(() =>
    target.apply({ k: 'sketch-snapshot', t: source.t, control: source.control, sketches: projection }),
  );
  assert.deepEqual(target.sketchProjection(), before);
});

test('sketch catch-up preserves the existing control baseline before driver-attributed DOM events', async () => {
  const { Model } = await import('../../cmd/internal/webcast/web/model.js');
  const { catchUp } = await import('../../cmd/internal/webcast/web/replay.js');
  const recs = structuredClone(corpus.cases.find((c) => c.name === 'history').records.slice(0, 4));
  const driver = { id: 'driver', color: '#00ff00' };
  recs.push(
    { k: 'ctl', t: 4, op: 'request', who: driver, pending: [driver] },
    { k: 'ctl', t: 5, op: 'grant', who: driver, driver },
    { k: 'click', t: 6, id: 2, by: driver.id },
  );
  const plain = catchUp(recs.filter((r) => !r.k.startsWith('sketch-'))),
    caught = catchUp(recs);
  assert.deepEqual(
    caught.filter((r) => r.k === 'ctl'),
    plain.filter((r) => r.k === 'ctl'),
  );
  assert.ok(
    caught.findIndex((r) => r.k === 'ctl' && r.op === 'grant') < caught.findIndex((r) => r.k === 'click'),
  );
  const source = reduceModel(Model, recs),
    joined = reduceModel(Model, caught, { allowSketchSnapshot: true });
  assert.deepEqual(joined.canonical(), source.canonical());
  assert.deepEqual(joined.control, source.control);
  let compacted = recs,
    t = 7;
  for (let seg = 2; seg <= 20; seg++) {
    const pending = { id: 'waiting' };
    compacted.push(
      { k: 'ctl', t: t++, op: 'request', who: pending, driver, pending: [pending] },
      { k: 'ctl', t: t++, op: 'deny', who: pending, driver },
      { k: 'stop', t: t++, why: 'pause' },
      { ...recs[1], t: t++, seg, why: 'resume' },
      { ...recs[2], t: t++ },
      { k: 'click', t: t++, id: 2, by: driver.id },
    );
    const baseline = catchUp(compacted.filter((r) => !r.k.startsWith('sketch-'))).filter(
      (r) => r.k === 'ctl',
    );
    compacted = catchUp(compacted);
    assert.deepEqual(
      compacted.filter((r) => r.k === 'ctl'),
      baseline,
    );
    assert.equal(compacted.filter((r) => r.k === 'ctl').length, 1, 'historical chatter remains compacted');
    assert.equal(compacted.filter((r) => r.k === 'sketch-snapshot').length, 1);
    assert.equal(reduceModel(Model, compacted, { allowSketchSnapshot: true }).control.driver.id, driver.id);
  }
});

test('Model exposes owned projections and keeps sketch and driving authority private', async () => {
  const { Model } = await import('../../cmd/internal/webcast/web/model.js');
  const recs = corpus.cases.find((c) => c.name === 'history').records.slice(0, 6);
  const m = reduceModel(Model, recs),
    before = m.sketchProjection();
  const leaked = m.sketches;
  if (leaked) {
    const history = leaked.sketches.get('sk');
    history.strokes.get('a').pts[0][0] = 999;
    history.next = 999;
    history.visible.push('forged');
    history.receipts.clear();
  }
  assert.deepEqual(
    m.sketchProjection(),
    before,
    'public handles cannot change accepted paths, sequence or stacks',
  );
  assert.equal(m.sketches, undefined);
  const projection = m.sketchProjection();
  projection.sketches[0].strokes[0].pts[0][0] = 999;
  projection.sketches[0].next_sequence = 999;
  projection.sketches[0].visible.push('forged');
  assert.deepEqual(m.sketchProjection(), before);
  const control = m.control;
  control.driver = { id: 'forged' };
  control.pending.push({ id: 'forged' });
  assert.deepEqual(m.control, { driver: null, pending: [] }, 'control getter owns its nested objects');
  assert.throws(() => {
    m.control = control;
  });
  assert.equal(m.ctlSeen, undefined);
  m.ctlSeen = false;
  m.apply({ k: 'ctl', t: 20, op: 'request', who: { id: 'driver' }, pending: [{ id: 'driver' }] });
  m.ctlSeen = false;
  assert.throws(() =>
    m.apply({ k: 'ctl', t: 21, op: 'grant', who: { id: 'other' }, driver: { id: 'other' } }),
  );
});

test('owned paths normalize signed zero without changing captured caller points or anchor boxes', async () => {
  const { Model } = await import('../../cmd/internal/webcast/web/model.js');
  const recs = structuredClone(corpus.cases.find((c) => c.name === 'history').records.slice(0, 6));
  recs[4].at.box = [-0, -0, 10, 10];
  recs[5].pts = [[-0, -0, -0]];
  const source = reduceModel(Model, recs),
    projection = source.sketchProjection();
  for (const value of projection.sketches[0].strokes[0].pts[0]) assert.equal(Object.is(value, -0), false);
  for (const value of projection.sketches[0].strokes[0].begin.at.box.slice(0, 2))
    assert.equal(Object.is(value, -0), false);
  assert.ok(recs[5].pts[0].every((value) => Object.is(value, -0)));
  assert.ok(recs[4].at.box.slice(0, 2).every((value) => Object.is(value, -0)));
  projection.sketches[0].strokes[0].pts[0] = [-0, -0, -0];
  const target = reduceModel(Model, recs.slice(0, 3), { allowSketchSnapshot: true });
  target.apply({ k: 'sketch-snapshot', t: source.t, control: source.control, sketches: projection });
  assert.ok(target.sketchProjection().sketches[0].strokes[0].pts[0].every((value) => !Object.is(value, -0)));
  assert.ok(projection.sketches[0].strokes[0].pts[0].every((value) => Object.is(value, -0)));
});

test('all semantic Unicode evidence rejects lone surrogates while unknown future values have no effect', async () => {
  const { checkSketchRecord, sketchCanonical } = await sketchModule();
  const r = {
    ...begin(),
    k: 'sketch-stroke-begin',
    sketch: 's',
    id: 'p',
    width: 4,
    at: { node: 1, box: [0, 0, 10, 10] },
  };
  for (const bad of ['\ud800', '\udfff']) {
    for (const key of ['tag', 'route', 'kind', 'sel', 'text'])
      assert.throws(() => checkSketchRecord({ ...r, at: { ...r.at, [key]: bad } }), key);
    for (const key of ['classes', 'path'])
      assert.throws(() => checkSketchRecord({ ...r, at: { ...r.at, [key]: [bad] } }), key);
    for (const stable of [{ futureString: bad }, { [bad]: 'semantic' }, { id: bad }])
      assert.throws(() => checkSketchRecord({ ...r, at: { ...r.at, stable } }));
    assert.throws(() => checkSketchRecord({ ...begin(), author: { ...begin().author, name: bad } }));
    assert.throws(() => checkSketchRecord({ ...r, k: 'sketch-label', label: bad }));
    const future = {
      ...r,
      at: { ...r.at, stable: { future: { bad }, [bad]: { future: true } }, future: bad },
      future: bad,
    };
    assert.doesNotThrow(() => checkSketchRecord(future));
    assert.equal(sketchCanonical(future), sketchCanonical(r));
  }
  assert.doesNotThrow(() =>
    checkSketchRecord({
      ...r,
      at: { ...r.at, text: '\ud83d\ude80\ufffd', stable: { '\ud83d\ude80': '\ufffd' } },
    }),
  );
});

test('explicit cancel end truncates accepted partial path and permits continuation', async () => {
 const {createSketchState,nextSketch,sketchProjection,sketchCanonical,checkSketchRecord}=await sketchModule();
 const s=createSketchState(),c={live:true,ended:false,doc:1,tab:'a',driver:''};let seq=0;
 const apply=(kind,fields={})=>{const r={...begin(),k:kind,operation:`cancel-${seq}`,sequence:seq++,sketch:'s',...fields};nextSketch(s,r,c).commit();return r;};
 apply('sketch-begin');apply('sketch-stroke-begin',{id:'p',at:{node:3,box:[0,0,10,10]},width:3});apply('sketch-points',{id:'p',chunk:0,pts:[[1,2,.5]]});
 const end=apply('sketch-stroke-end',{id:'p',truncated:true});let view=sketchProjection(s).sketches[0];assert.equal(view.phase,'drawing');assert.equal(view.strokes[0].truncated,true);assert.deepEqual(view.visible,['p']);
 assert.notEqual(sketchCanonical(end),sketchCanonical({...end,truncated:false}));assert.equal(sketchCanonical({...end,truncated:false}),sketchCanonical({...end,truncated:undefined}));
 for(const truncated of [null,1,'true'])assert.throws(()=>checkSketchRecord({...end,truncated}));
 apply('sketch-undo',{id:'p'});apply('sketch-redo',{id:'p'});apply('sketch-stroke-begin',{id:'q',at:{node:3,box:[0,0,10,10]},width:3});apply('sketch-stroke-end',{id:'q'});apply('sketch-label',{label:'partial'});
 view=sketchProjection(s).sketches[0];assert.equal(view.strokes[0].truncated,true);assert.equal(view.phase,'complete');
});
