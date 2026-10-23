import { test } from 'node:test';
import assert from 'node:assert/strict';
import { Model } from '../../cmd/internal/webcast/web/model.js';
import * as render from '../../cmd/internal/webcast/web/sketch-render.js';
function setup() {
  const m = new Model();
  let t = 0,
    seq = 0;
  const apply = (r) => m.apply({ t: t++, ...r });
  apply({ k: 'head', format: 'couch-webcast', v: 1, id: 'x', started: 'now' });
  apply({ k: 'seg', seg: 1, why: 'live', tab: { id: 'tab', url: 'u' }, vp: { w: 100, h: 100, dpr: 1 } });
  apply({ k: 'doc', doc: 1, url: 'u', tree: { id: 1, y: 'doc', c: [{ id: 7, y: 'el', n: 'body' }] } });
  const adapter = render.createSketchRecordDisplay(m);
  const record = (k, body = {}) => {
    const r = { k, t: t++, operation: 'op' + seq, sequence: seq++, digest: 'a'.repeat(64), ...body };
    m.apply(r);
    adapter.accept(r);
    return r;
  };
  record('sketch-begin', {
    id: 's',
    author: { id: 'operator', color: '#ff0000' },
    owner: 'operator',
    doc: 1,
    tab: 'tab',
    vp: { w: 100, h: 100, dpr: 1 },
  });
  record('sketch-stroke-begin', { sketch: 's', id: 'p', at: { node: 7, box: [0, 0, 100, 100] }, width: 4 });
  return { m, adapter, record, apply };
}
test('accepted display appends only new points; full bootstrap occurs once for N and 2N', () => {
  for (const n of [8, 16]) {
    let snapshots = 0;
    const original = Model.prototype.sketchProjection;
    Model.prototype.sketchProjection = function () {
      snapshots++;
      return original.call(this);
    };
    try {
      const { m, adapter, record } = setup();
      let last;
      for (let i = 0; i < n; i++)
        last = record('sketch-points', { sketch: 's', id: 'p', chunk: i, pts: [[i, 2, 0.5]] });
      adapter.accept(last);
      assert.equal(adapter.paths()[0].pts.length, n);
      assert.equal(adapter.stats().points, n);
      assert.equal(snapshots, 1);
      const info = m.sketchMetadata('s', 'p');
      info.begin.author.color = '#000000';
      assert.equal(m.sketchMetadata('s', 'p').begin.author.color, '#ff0000');
      record('sketch-stroke-end', { sketch: 's', id: 'p' });
      record('sketch-undo', { sketch: 's', id: 'p' });
      assert.equal(adapter.paths().length, 0);
      record('sketch-redo', { sketch: 's', id: 'p' });
      assert.equal(adapter.paths().length, 1);
    } finally {
      Model.prototype.sketchProjection = original;
    }
  }
});
test('actual pause and annotation clear apply accepted seal/visibility facts', () => {
  const { m, adapter, record } = setup();
  record('sketch-points', { sketch: 's', id: 'p', chunk: 0, pts: [[1, 2, 0.5]] });
  const pause = { k: 'stop', t: m.t + 1, why: 'pause' };
  m.apply(pause);
  adapter.accept(pause);
  assert.equal(adapter.paths()[0].sealed, true);
  assert.equal(adapter.paths()[0].truncated, true);
  const clear = { k: 'annotation-clear', t: m.t + 1, who: { id: 'operator' } };
  m.apply(clear);
  adapter.accept(clear);
  assert.equal(adapter.paths().length, 0);
});

test('constant-size accepted version metadata survives display bootstrap and incremental commits',()=>{
 const {m,adapter,record}=setup();
 const last=record('sketch-points',{sketch:'s',id:'p',chunk:0,pts:[[1,2,.5]],digest:'b'.repeat(64)});
 assert.equal(m.sketchMetadata('s').digest,last.digest);
 assert.equal(adapter.metadata('s').digest,last.digest);
 assert.equal(adapter.metadata('s').next_sequence,3);
 const boot=render.createSketchRecordDisplay(m);
 assert.deepEqual(boot.metadata('s'),adapter.metadata('s'));
 const display=render.createSketchDisplay();display.install(adapter.snapshot());
 assert.deepEqual(display.metadata('s'),adapter.metadata('s'));
 record('sketch-stroke-end',{sketch:'s',id:'p',digest:'c'.repeat(64)});
 assert.equal(adapter.metadata('s').digest,'c'.repeat(64));assert.equal(adapter.metadata('s').next_sequence,4);
});

test('restored accepted digest is constant-size metadata without changing persisted projection',async()=>{
 const {restoreSketchState,sketchMetadata,sketchProjection}=await import('../../cmd/internal/webcast/web/sketch-model.js');
 const {m,record}=setup();record('sketch-points',{sketch:'s',id:'p',chunk:0,pts:[[1,2,.5]],digest:'b'.repeat(64)});
 const projection=m.sketchProjection(),restored=restoreSketchState(projection);
 assert.deepEqual(sketchProjection(restored),projection);
 restored.sketches.get('s').receipts.values=()=>assert.fail('metadata cannot enumerate receipt history');
 assert.equal(sketchMetadata(restored,'s').digest,'b'.repeat(64));
 assert.equal(sketchMetadata(restored,'s').next_sequence,3);
});
