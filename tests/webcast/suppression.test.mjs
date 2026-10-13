import { test } from 'node:test';
import assert from 'node:assert/strict';
import { suppressionPolicy } from '../../cmd/internal/webcast/operator-policy.js';

test('capture leases compose and cannot release an indefinite input guard', () => {
  let s = suppressionPolicy(new Map(), {op:'acquire',token:'shot1',until:100}, 0);
  s = suppressionPolicy(s.tokens, {op:'acquire',token:'input',until:0}, 1);
  s = suppressionPolicy(s.tokens, {op:'acquire',token:'shot2',until:200}, 2);
  s = suppressionPolicy(s.tokens, {op:'release',token:'shot1'}, 3);
  assert.equal(s.hidden, true);
  s = suppressionPolicy(s.tokens, {op:'check',token:'shot2'}, 201);
  assert.equal(s.confirmed, false);
  assert.equal(s.hidden, true);
  s = suppressionPolicy(s.tokens, {op:'release',token:'input'}, 202);
  assert.equal(s.hidden, false);
});
