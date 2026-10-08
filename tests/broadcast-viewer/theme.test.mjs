// The viewer maps the operator's palette onto xterm.js's theme (#395 M4).
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { xtermTheme, FONT_FACES } from '../../cmd/internal/broadcast/web/viewer.js';

test('maps default colours and the 16 ANSI colours by name', () => {
  const ansi = Array.from({ length: 16 }, (_, i) => `#0000${i.toString(16).padStart(2, '0')}`);
  const t = xtermTheme({ foreground: '#eeeeee', background: '#101010', ansi });
  assert.equal(t.foreground, '#eeeeee');
  assert.equal(t.background, '#101010');
  assert.equal(t.black, '#000000');
  assert.equal(t.red, '#000001');
  assert.equal(t.white, '#000007');
  assert.equal(t.brightBlack, '#000008');
  assert.equal(t.brightWhite, '#00000f');
});

test('unknown colours are left to xterm.js defaults', () => {
  const t = xtermTheme({ foreground: '', background: '#101010', ansi: Array(16).fill('') });
  assert.deepEqual(Object.keys(t), ['background']);
});

test('rejects anything that is not a #rrggbb colour', () => {
  const t = xtermTheme({ foreground: 'red; background: url(x)', background: '#12345', ansi: ['javascript:1'] });
  assert.deepEqual(t, {});
});

// Every face the CSS declares is waited for before the first frame; a face
// left to load lazily draws in a fallback and shifts the rest of its row.
test('the viewer waits for all four font faces', () => {
  assert.deepEqual(FONT_FACES, ['', 'bold ', 'italic ', 'italic bold ']);
});
