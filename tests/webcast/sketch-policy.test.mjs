import { test } from 'node:test';
import assert from 'node:assert/strict';
const mod = () => import('../../cmd/internal/webcast/web/sketch-policy.js');
const ready = (policy) => {
  policy.dispatch({ type: 'sync', allowed: true, live: true, epoch: 1 });
  policy.dispatch({ type: 'enter', sketch: 's' });
  policy.dispatch({ type: 'prepared', sketch: 's' });
};
test('UI leave requires label, Escape resumes, and handoff preserves independent old label', async () => {
  const { createSketchPolicy } = await mod();
  const p = createSketchPolicy();
  ready(p);
  assert.equal(p.snapshot().phase, 'drawing');
  p.dispatch({ type: 'leave', tool: 'point' });
  assert.equal(p.snapshot().phase, 'label-required');
  p.dispatch({ type: 'escape' });
  assert.equal(p.snapshot().phase, 'drawing');
  p.dispatch({ type: 'sync', allowed: false, live: true, epoch: 2 });
  assert.equal(p.snapshot().phase, 'interrupted');
  p.dispatch({ type: 'sync', allowed: true, live: true, epoch: 3 });
  p.dispatch({ type: 'enter', sketch: 'new' });
  p.dispatch({ type: 'prepared', sketch: 'new' });
  const old = p
    .dispatch({ type: 'queue', sketch: 's', kind: 'label', body: { label: 'old intent' } })
    .find((e) => e.type === 'send');
  const next = p
    .dispatch({ type: 'queue', sketch: 'new', kind: 'stroke-begin', body: { id: 'p' } })
    .find((e) => e.type === 'send');
  assert.ok(old);
  assert.ok(next);
  assert.equal(old.command.epoch, 1);
  assert.equal(next.command.epoch, 3);
});
test('FIFO unknown outcome blocks new operations and retries exact immutable identity/body/epoch', async () => {
  const { createSketchPolicy } = await mod();
  let id = 0;
  const p = createSketchPolicy({ operationId: () => `op${id++}` });
  ready(p);
  const body = { pts: [[1, 2, 0.5]] };
  const first = p.dispatch({ type: 'queue', kind: 'points', body }).find((e) => e.type === 'send').command;
  body.pts[0][0] = 999;
  p.dispatch({ type: 'queue', kind: 'stroke-end', body: { id: 'p' } });
  assert.equal(p.snapshot().slots[0].queued, 1);
  p.dispatch({ type: 'reply', operation: first.operation, outcome: 'unknown' });
  assert.equal(p.snapshot().phase, 'reconciling');
  const retry = p.dispatch({ type: 'retry' }).find((e) => e.type === 'status');
  assert.deepEqual(retry.command, first);
  assert.equal(first.body.pts[0][0], 1);
  assert.equal(p.dispatch({ type: 'reply', operation: 'not-current', outcome: 'accepted' }).length, 0);
  const second = p
    .dispatch({ type: 'reply', operation: first.operation, outcome: 'accepted' })
    .find((e) => e.type === 'send').command;
  assert.equal(second.sequence, first.sequence + 1);
  assert.equal(second.kind, 'stroke-end');
  assert.equal(id, 2);
});
test('only barrier-proven refusal releases a slot; queue capacity fails visibly without growing', async () => {
  const { createSketchPolicy } = await mod();
  const p = createSketchPolicy({ maxQueuedChunks: 2 });
  ready(p);
  const first = p
    .dispatch({ type: 'queue', kind: 'points', body: { n: 0 } })
    .find((e) => e.type === 'send').command;
  for (let i = 1; i <= 2; i++) p.dispatch({ type: 'queue', kind: 'points', body: { n: i } });
  assert.ok(
    p.dispatch({ type: 'queue', kind: 'points', body: { n: 3 } }).some((e) => e.type === 'stop-gesture'),
  );
  assert.equal(p.snapshot().slots[0].queued, 2);
  p.dispatch({ type: 'reply', operation: first.operation, outcome: 'not-observed' });
  assert.equal(p.snapshot().slots[0].inflight.operation, first.operation);
  p.dispatch({ type: 'reply', operation: first.operation, outcome: 'refused', final: true });
  assert.equal(p.snapshot().slots[0].inflight, null);
});
test('async-anchor gesture buffers bounded samples and emits original pressure only after anchor', async () => {
  const { createSketchGesture } = await mod();
  const g = createSketchGesture({ maxPoints: 4 });
  assert.deepEqual(g.start('p', [[1, 2, 0.1]]), { type: 'hit-test', stroke: 'p', point: [1, 2, 0.1] });
  assert.equal(g.samples([[2, 3, 0.2]]).length, 0);
  assert.equal(g.end().length, 0);
  const effects = g.anchor({ node: 7, box: [0, 0, 10, 10] });
  assert.equal(effects[0].type, 'stroke-begin');
  assert.deepEqual(effects[1].points, [
    [1, 2, 0.1],
    [2, 3, 0.2],
  ]);
  assert.equal(effects[2].type, 'stroke-end');
  g.start('q', [[0, 0, 1]]);
  assert.equal(
    g.samples([
      [1, 1, 1],
      [2, 2, 1],
      [3, 3, 1],
      [4, 4, 1],
    ])[0].type,
    'stop-gesture',
  );
  assert.equal(g.snapshot().buffered, 1);
});
test('generated network/control sequences retain at most one command in each sketch slot', async () => {
  const { createSketchPolicy } = await mod();
  for (let seed = 0; seed < 24; seed++) {
    const p = createSketchPolicy({
      operationId: (() => {
        let n = 0;
        return () => `${seed}-${n++}`;
      })(),
    });
    ready(p);
    for (let i = 0; i < 40; i++) {
      const op = (seed * 17 + i * 13) % 7;
      const inflight = p.snapshot().slots[0]?.inflight;
      if (op < 3) p.dispatch({ type: 'queue', kind: 'points', body: { n: i } });
      else if (op === 3 && inflight)
        p.dispatch({ type: 'reply', operation: inflight.operation, outcome: 'unknown' });
      else if (op === 4) p.dispatch({ type: 'retry' });
      else if (op === 5 && inflight)
        p.dispatch({ type: 'reply', operation: inflight.operation, outcome: 'accepted' });
      else p.dispatch({ type: 'sync', allowed: true, live: i % 9 !== 0, epoch: 1 });
      for (const slot of p.snapshot().slots) {
        assert.ok(slot.queued <= 16);
        if (slot.inflight) assert.ok(slot.inflight.operation);
      }
    }
  }
});
test('old label acknowledgment never selects a tool over a new active sketch', async () => {
  const { createSketchPolicy } = await mod();
  const p = createSketchPolicy();
  ready(p);
  p.dispatch({ type: 'leave', tool: 'point' });
  const op = p
    .dispatch({ type: 'queue', kind: 'label', body: { label: 'old' } })
    .find((e) => e.type === 'send').command;
  p.dispatch({ type: 'sync', allowed: true, live: true, epoch: 2 });
  p.dispatch({ type: 'enter', sketch: 'new' });
  p.dispatch({ type: 'prepared', sketch: 'new' });
  const effects = p.dispatch({ type: 'reply', operation: op.operation, outcome: 'accepted' });
  assert.equal(
    effects.some((e) => e.type === 'select-tool'),
    false,
  );
  assert.equal(p.snapshot().phase, 'drawing');
});
test('repeated preparation and anchor callbacks cannot reset slots or create a second begin', async () => {
  const { createSketchPolicy, createSketchGesture } = await mod();
  const p = createSketchPolicy();
  ready(p);
  assert.ok(p.dispatch({ type: 'enter', sketch: 'competing' }).some((e) => e.type === 'refused'));
  const g = createSketchGesture();
  g.start('p', [[1, 1, 1]]);
  g.anchor({ node: 1, box: [0, 0, 10, 10] });
  assert.deepEqual(g.anchor({ node: 1, box: [0, 0, 10, 10] }), []);
});
test('counted async-anchor buffering scales with new samples and fences retired hit-tests', async () => {
  const { createSketchGesture } = await mod();
  for (const n of [64, 128]) {
    const g = createSketchGesture();
    g.start('p', [[0, 0, 1]]);
    g.samples(Array.from({ length: n - 1 }, (_, i) => [i, 1, 0.5]));
    assert.equal(g.snapshot().visits, n);
    const effects = g.anchor({ node: 1, box: [0, 0, 10, 10] });
    assert.equal(effects.filter((e) => e.type === 'points').flatMap((e) => e.points).length, n);
    g.cancel();
    g.start('q', [[1, 1, 1]]);
    assert.deepEqual(g.anchor({ node: 1, box: [0, 0, 10, 10] }, 'p'), []);
    assert.equal(g.snapshot().waitingAnchor, true);
  }
});

test('begin unknown reconciles by principal operation and epoch before server sketch ID exists', async () => {
  const { createSketchPolicy } = await mod();
  const p = createSketchPolicy({ operationId: () => 'begin-id' });
  p.dispatch({ type: 'sync', allowed: true, live: true, epoch: 7 });
  p.dispatch({ type: 'enter', sketch: 'local-slot' });
  const send = p
    .dispatch({ type: 'queue', kind: 'begin', body: { color: 0 } })
    .find((e) => e.type === 'send');
  assert.ok(send);
  assert.equal(p.dispatch({ type: 'queue', kind: 'points', body: { pts: [[1, 2, 1]] } })[0].type, 'refused');
  p.dispatch({ type: 'reply', operation: 'begin-id', outcome: 'unknown' });
  const check = p.dispatch({ type: 'retry' }).find((e) => e.type === 'status');
  assert.deepEqual(check.command, send.command);
  assert.deepEqual(check.lookup, { operation: 'begin-id', sequence: 0, epoch: 7 });
  const effects = p.dispatch({
    type: 'reply',
    operation: 'begin-id',
    outcome: 'accepted',
    sketch: 'server-sketch',
    next_sequence: 1,
    source: { handle: 'source' },
  });
  assert.equal(p.snapshot().active, 'server-sketch');
  assert.equal(p.snapshot().slots[0].nextSequence, 1);
  assert.ok(effects.some((e) => e.type === 'begin-accepted' && e.sketch === 'server-sketch'));
  p.dispatch({ type: 'prepared', sketch: 'server-sketch' });
  assert.equal(p.snapshot().phase, 'drawing');
});
test('unknown outcomes stop newly sampled drawing commands without consuming another operation', async () => {
  const { createSketchPolicy } = await mod();
  const p = createSketchPolicy();
  ready(p);
  const op = p
    .dispatch({ type: 'queue', kind: 'points', body: { pts: [[1, 2, 1]] } })
    .find((e) => e.type === 'send').command;
  p.dispatch({ type: 'reply', operation: op.operation, outcome: 'unknown' });
  const queued = p.snapshot().slots[0].queued;
  assert.equal(p.dispatch({ type: 'queue', kind: 'points', body: { pts: [[3, 4, 1]] } })[0].type, 'refused');
  assert.equal(p.snapshot().slots[0].queued, queued);
});

test('unlabelled UI slots stay bounded by the recording sketch count', async () => {
  const { createSketchPolicy } = await mod();
  const p = createSketchPolicy();
  for (let i = 0; i < 256; i++) {
    p.dispatch({ type: 'sync', allowed: true, live: true, epoch: i });
    p.dispatch({ type: 'enter', sketch: `s${i}` });
    p.dispatch({ type: 'prepared', sketch: `s${i}` });
  }
  p.dispatch({ type: 'sync', allowed: true, live: true, epoch: 256 });
  assert.equal(p.dispatch({ type: 'enter', sketch: 'overflow' })[0].type, 'refused');
  assert.equal(p.snapshot().slots.length, 256);
});

test('definitively refused begins release their provisional UI slot without a new product limit', async () => {
  const { createSketchPolicy } = await mod();
  const p = createSketchPolicy();
  p.dispatch({ type: 'sync', allowed: true, live: true, epoch: 1 });
  for (let i = 0; i < 257; i++) {
    p.dispatch({ type: 'enter', sketch: `pending${i}` });
    const command = p
      .dispatch({ type: 'queue', kind: 'begin', body: { color: 0 } })
      .find((e) => e.type === 'send').command;
    p.dispatch({ type: 'reply', operation: command.operation, outcome: 'refused', final: true });
    assert.equal(p.snapshot().slots.length, 0);
  }
  assert.equal(p.snapshot().phase, 'off');
});
