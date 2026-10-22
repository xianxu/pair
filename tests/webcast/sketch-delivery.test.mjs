import { test } from 'node:test';
import assert from 'node:assert/strict';
const mod = () => import('../../cmd/internal/webcast/web/sketch-delivery.js');
test('bootstrap highwater, duplicates and gaps advance ACK only after accepted batch', async () => {
  const { createSketchDelivery } = await mod();
  const calls = [];
  const d = createSketchDelivery({
    installBootstrap: (b) => calls.push(b),
    applyBatch: (p) => calls.push(p),
  });
  assert.equal(d.receiveBatch({ epoch: 'e', from: 1, to: 1, patches: [{}] }).needBootstrap, true);
  d.receiveBootstrap({ epoch: 'e', seq: 4, display: { sketches: [] } });
  assert.deepEqual(d.receiveBatch({ epoch: 'e', from: 5, to: 6, patches: [{ n: 5 }, { n: 6 }] }), {
    epoch: 'e',
    seq: 6,
    needBootstrap: false,
  });
  const n = calls.length;
  d.receiveBatch({ epoch: 'e', from: 5, to: 6, patches: [{ n: 5 }, { n: 6 }] });
  assert.equal(calls.length, n);
  assert.equal(d.receiveBatch({ epoch: 'e', from: 8, to: 8, patches: [{}] }).needBootstrap, true);
  assert.deepEqual(d.cursor(), { epoch: 'e', seq: 6 });
  assert.equal(d.receiveBatch({ epoch: 'new', from: 1, to: 1, patches: [{}] }).needBootstrap, true);
});
test('failed callback, oversized batch and forged repeated body cannot mutate acknowledged cursor', async () => {
  const { createSketchDelivery } = await mod();
  const d = createSketchDelivery({
    installBootstrap() {},
    applyBatch() {
      throw Error('bad patch');
    },
  });
  d.receiveBootstrap({ epoch: 'e', seq: 0, display: { sketches: [] } });
  assert.throws(() => d.receiveBatch({ epoch: 'e', from: 1, to: 1, patches: [{}] }));
  assert.deepEqual(d.cursor(), { epoch: 'e', seq: 0 });
  assert.throws(() =>
    d.receiveBatch({ epoch: 'e', from: 1, to: 33, patches: Array.from({ length: 33 }, () => ({})) }),
  );
  assert.throws(() =>
    d.receiveBatch({ epoch: 'e', from: 1, to: 1, patches: [{ label: 'x'.repeat(65536) }] }),
  );
});
test('bounded chunk assembly checks handle/epoch/offset/total and retirement fences late chunks', async () => {
  const { createSketchTransfer } = await mod();
  const t = createSketchTransfer({ maxBytes: 12, maxChunk: 4 });
  t.begin({ handle: 'h', epoch: 'e', total: 6 });
  assert.deepEqual(
    t.append({ handle: 'h', epoch: 'e', total: 6, offset: 0, bytes: new Uint8Array([1, 2, 3, 4]) }),
    { offset: 4, complete: false },
  );
  assert.throws(() => t.append({ handle: 'h', epoch: 'e', total: 6, offset: 5, bytes: new Uint8Array([5]) }));
  assert.equal(
    t.append({ handle: 'h', epoch: 'e', total: 6, offset: 0, bytes: new Uint8Array([1, 2, 3, 4]) }).offset,
    4,
  );
  const done = t.append({ handle: 'h', epoch: 'e', total: 6, offset: 4, bytes: new Uint8Array([5, 6]) });
  assert.deepEqual([...done.value], [1, 2, 3, 4, 5, 6]);
  t.retire();
  assert.throws(() =>
    t.append({ handle: 'h', epoch: 'e', total: 6, offset: 4, bytes: new Uint8Array([5, 6]) }),
  );
  assert.throws(() => t.begin({ handle: 'too-big', epoch: 'e', total: 13 }));
});
test('conflicting duplicate batch and transfer bytes are refused without another apply or ACK', async () => {
  const { createSketchDelivery, createSketchTransfer } = await mod();
  let count = 0;
  const d = createSketchDelivery({
    installBootstrap() {},
    applyBatch() {
      count++;
    },
  });
  d.receiveBootstrap({ epoch: 'e', seq: 0, display: { sketches: [] } });
  d.receiveBatch({ epoch: 'e', from: 1, to: 1, patches: [{ label: 'a' }] });
  assert.throws(() => d.receiveBatch({ epoch: 'e', from: 1, to: 1, patches: [{ label: 'b' }] }));
  assert.equal(count, 1);
  const t = createSketchTransfer();
  t.begin({ handle: 'h', epoch: 'e', total: 3 });
  t.append({ handle: 'h', epoch: 'e', total: 3, offset: 0, bytes: new Uint8Array([1, 2]) });
  assert.throws(() =>
    t.append({ handle: 'h', epoch: 'e', total: 3, offset: 0, bytes: new Uint8Array([1, 9]) }),
  );
  assert.equal(t.cursor().offset, 2);
});

test('8/16 batches deliver only their new patches and duplicate retry visits zero points', async () => {
  const { createSketchDelivery } = await mod();
  for (const n of [8, 16]) {
    let count = 0,
      points = 0;
    const d = createSketchDelivery({
      installBootstrap() {},
      applyBatch: (p) => {
        count += p.length;
        for (const patch of p) points += patch.points.pts.length;
      },
    });
    d.receiveBootstrap({ epoch: 'e', seq: 0, display: { sketches: [] } });
    for (let i = 1; i <= n; i++) {
      const batch = {
        epoch: 'e',
        from: i,
        to: i,
        patches: [
          {
            sketch: 's',
            points: { id: 'p', offset: (i - 1) * 64, pts: Array.from({ length: 64 }, () => [1, 2, 1]) },
          },
        ],
      };
      d.receiveBatch(batch);
      d.receiveBatch(batch);
    }
    assert.equal(count, n);
    assert.equal(points, n * 64);
  }
});

test('transfer configuration cannot enlarge protocol resource bounds', async () => {
  const { createSketchTransfer, SKETCH_BOOTSTRAP_BYTES, SKETCH_TRANSFER_BYTES } = await mod();
  for (const options of [
    { maxBytes: SKETCH_BOOTSTRAP_BYTES + 1 },
    { maxBytes: Infinity },
    { maxChunk: SKETCH_TRANSFER_BYTES + 1 },
    { maxChunk: 0 },
  ])
    assert.throws(() => createSketchTransfer(options));
});
