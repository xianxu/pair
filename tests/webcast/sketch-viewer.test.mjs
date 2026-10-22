import { test } from 'node:test';
import assert from 'node:assert/strict';
import * as viewer from '../../cmd/internal/webcast/web/viewer.js';
test('drawing authority requires ready holder seat and annotation availability', () => {
  const caps = {
    annotation: { allowed: true, generation: 2 },
    drawing: { epoch: 5, holder: 'me', ready: true },
  };
  const model = { live: true, ended: false };
  assert.deepEqual(viewer.sketchAuthority(caps, { viewer: 'me' }, model), {
    allowed: true,
    live: true,
    ended: false,
    epoch: 5,
  });
  for (const seat of [null, { viewer: 'other' }])
    assert.equal(viewer.sketchAuthority(caps, seat, model).allowed, false);
  assert.equal(
    viewer.sketchAuthority({ ...caps, drawing: { ...caps.drawing, ready: false } }, { viewer: 'me' }, model)
      .allowed,
    false,
  );
  assert.equal(
    viewer.sketchAuthority({ ...caps, annotation: { allowed: false } }, { viewer: 'me' }, model).allowed,
    false,
  );
});
test('viewer sketch requests carry seat, generation and captured epoch; source chunks validate identity', async () => {
  const calls = [];
  let response = { outcome: 'accepted' };
  const transport = viewer.createViewerSketchTransport(
    async (url, init) => {
      calls.push({ url, init, body: JSON.parse(init.body) });
      return { ok: true, status: 200, json: async () => response };
    },
    { key: 'secret' },
    () => 9,
    { id: () => 'read-op', decode: (s) => Uint8Array.from(Buffer.from(s, 'base64')) },
  );
  const body = { operation: 'immutable', sequence: 0, epoch: 3, seg: 1, color: 0 };
  await transport.send('sketch-begin', body);
  assert.deepEqual(calls[0].body, body);
  assert.equal(calls[0].init.headers['X-Couch-Seat'], 'secret');
  assert.equal(calls[0].init.headers['X-Couch-Annotation-Generation'], '9');
  assert.equal(calls[0].init.headers['X-Couch-Drawing-Epoch'], '3');
  response = { source: 'h', offset: 0, total: 3, body: 'YWJj' };
  assert.deepEqual(
    await transport.readSource({ handle: 'h', length: 3 }, 0, 3, { sketch: 's', epoch: 3 }),
    new Uint8Array([97, 98, 99]),
  );
  response = { source: 'other', offset: 0, total: 3, body: 'YWJj' };
  await assert.rejects(
    () => transport.readSource({ handle: 'h', length: 3 }, 0, 3, { sketch: 's', epoch: 3 }),
    /chunk/,
  );
});
test('private asset chunks retain capture32MiB envelope independently of source16MiB', async () => {
  let total = 20 * 1024 * 1024;
  const transport = viewer.createViewerSketchTransport(
    async () => ({
      ok: true,
      status: 200,
      json: async () => ({ source: 'h', offset: 0, total, body: Buffer.alloc(32768).toString('base64') }),
    }),
    { key: 'seat' },
    () => 1,
    { id: () => 'chunk', decode: (s) => Uint8Array.from(Buffer.from(s, 'base64')) },
  );
  assert.equal(
    (await transport.readAsset({ handle: 'h' }, 'a'.repeat(64), 0, 32768, { sketch: 's', epoch: 2 })).total,
    total,
  );
  total = 32 * 1024 * 1024 + 1;
  await assert.rejects(
    () => transport.readAsset({ handle: 'h' }, 'a'.repeat(64), 0, 32768, { sketch: 's', epoch: 2 }),
    /chunk/,
  );
});
