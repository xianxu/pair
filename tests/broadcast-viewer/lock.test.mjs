// The lock page's pure parts (pair#495): reading a code from URL#NNNN,
// cleaning the viewer's name, and reading the gate's answers.
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { codeFromFragment, cleanName, unlock, MAX_NAME } from '../../cmd/internal/broadcast/web/lock.js';
import { sessionLost } from '../../cmd/internal/broadcast/web/collab.js';

test('a code rides the fragment; anything else is ignored', () => {
  assert.equal(codeFromFragment('#4821'), '4821');
  assert.equal(codeFromFragment('#903 114'), '903114');
  assert.equal(codeFromFragment('#90-31-14'), '903114');
  for (const h of ['', '#', '#ab12', '#12', '#<script>', '#4821&x=1', 'x4821']) {
    assert.equal(codeFromFragment(h), '', h);
  }
});

test('a name is trimmed and collapsed, and must pass the board’s rule', () => {
  assert.equal(cleanName('  Ada   Lovelace '), 'Ada Lovelace');
  assert.equal(cleanName(''), '');
  assert.equal(cleanName('x'.repeat(MAX_NAME)), 'x'.repeat(MAX_NAME));
  assert.equal(cleanName('x'.repeat(MAX_NAME + 1)), '', 'too long is refused, not cut');
  assert.equal(cleanName('A\nB'), '', 'one line');
  assert.equal(cleanName('A\u200bB'), '', 'no format characters');
});

function answering(status, body) {
  const calls = [];
  const fetchFn = async (url, init) => {
    calls.push({ url, init });
    return { ok: status >= 200 && status < 300, status, json: async () => body };
  };
  return { fetchFn, calls };
}

test('unlock posts the code alone, same-origin', async () => {
  const { fetchFn, calls } = answering(200, { ok: true });
  assert.deepEqual(await unlock(fetchFn, '4821'), { ok: true });
  assert.equal(calls[0].url, 'unlock');
  assert.equal(calls[0].init.method, 'POST');
  assert.equal(calls[0].init.credentials, 'same-origin');
  assert.deepEqual(JSON.parse(calls[0].init.body), { code: '4821' });
});

test('a refusal shows the gate’s sentence', async () => {
  const { fetchFn } = answering(403, { ok: false, error: 'That code isn’t right.' });
  assert.deepEqual(await unlock(fetchFn, '0000'), { ok: false, error: 'That code isn’t right.' });
});

test('a network failure is not a wrong code', async () => {
  const r = await unlock(async () => { throw new Error('offline'); }, '4821');
  assert.equal(r.ok, false);
  assert.match(r.error, /reach/);
});

test('only a 401 from the session probe is a lost session', async () => {
  for (const [status, lost] of [[401, true], [204, false], [404, false], [502, false]]) {
    assert.equal(await sessionLost(async () => ({ status })), lost, String(status));
  }
  assert.equal(await sessionLost(async () => { throw new Error('offline'); }), false);
});
