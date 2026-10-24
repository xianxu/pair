import test from 'node:test';
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import vm from 'node:vm';
import { feedbackLayout } from '../../cmd/internal/webcast/web/feedback-layout.js';

const nonnegative = value => Number.isFinite(value) ? Math.max(0, value) : 0;

function checkBounds(input) {
  const result = feedbackLayout(input);
  const width = nonnegative(input.width);
  const height = nonnegative(input.height);
  const { toolbar, content, panel, initial } = result;
  for (const group of Object.values(result)) {
    for (const value of Object.values(group)) {
      assert.ok(Number.isFinite(value) && value >= 0, `invalid ${value}`);
    }
  }
  // Scale-aware tolerance handles cancellation at extreme finite dimensions.
  const tolerance = Math.max(1, width, height) * Number.EPSILON * 4;
  for (const rect of [toolbar, content]) {
    assert.ok(rect.left + rect.width <= width + tolerance);
    assert.ok(rect.top + rect.height <= height + tolerance);
  }
  assert.ok(content.left + content.width <= toolbar.left + tolerance);
  assert.ok(panel.width <= content.width);
  assert.ok(panel.height <= content.height);
  assert.ok(initial.left >= content.left);
  assert.ok(initial.top >= content.top);
  assert.ok(initial.left + panel.width <= toolbar.left + tolerance);
  assert.ok(initial.left + panel.width <= content.left + content.width + tolerance);
  assert.ok(initial.top + panel.height <= content.top + content.height + tolerance);
  return result;
}

test('320px viewport fits panel beside the protected toolbar', () => {
  assert.deepEqual(feedbackLayout({ width: 320, height: 640 }), {
    toolbar: { left: 252, top: 12, width: 56, height: 616 },
    content: { left: 12, top: 12, width: 228, height: 616 },
    panel: { width: 228, height: 320 },
    initial: { left: 12, top: 110 },
  });
});

test('768px and 1280px anchor the panel at the usable right edge', () => {
  for (const width of [768, 1280]) {
    const result = checkBounds({ width, height: 800 });
    assert.deepEqual(result.panel, { width: 300, height: 320 });
    assert.deepEqual(result.initial, { left: width - 380, top: 110 });
    assert.equal(result.initial.left + result.panel.width + 12, result.toolbar.left);
  }
});

test('short viewport clamps the preferred top and panel height', () => {
  assert.deepEqual(checkBounds({ width: 768, height: 100 }).initial, { left: 388, top: 12 });
  assert.equal(checkBounds({ width: 768, height: 100 }).panel.height, 76);
  assert.equal(checkBounds({ width: 768, height: 400 }).initial.top, 68);
});

test('custom sizes and gap reserve exactly the requested toolbar', () => {
  assert.deepEqual(feedbackLayout({ width: 500, height: 500, toolbarWidth: 80, gap: 20, panelWidth: 100, panelHeight: 50 }), {
    toolbar: { left: 400, top: 20, width: 80, height: 460 },
    content: { left: 20, top: 20, width: 360, height: 460 },
    panel: { width: 100, height: 50 },
    initial: { left: 280, top: 110 },
  });
});

test('tiny viewports never force a positive panel width over the toolbar', () => {
  for (const width of [0, 0.1, 1, 12, 24, 30, 56, 68, 80]) {
    const result = checkBounds({ width, height: 10 });
    assert.equal(result.panel.width, 0);
    assert.equal(result.panel.height, 0);
  }
});

test('normalizes invalid, negative and extreme dimensions without overflow', () => {
  const cases = [undefined, null, NaN, Infinity, -Infinity, -1, -Number.MAX_VALUE, 0, Number.MIN_VALUE, Number.MAX_VALUE, '320', {}, 0.5, 24, 100];
  for (const width of cases) {
    for (const height of cases) checkBounds({ width, height });
  }
  for (const key of ['toolbarWidth', 'gap', 'panelWidth', 'panelHeight']) {
    for (const value of cases) checkBounds({ width: 320, height: 640, [key]: value });
  }
});

test('generated viewports and preferences preserve containment and toolbar separation', () => {
  let seed = 856;
  const next = () => ((seed = (Math.imul(seed, 1664525) + 1013904223) >>> 0) / 2 ** 32);
  for (let i = 0; i < 10000; i += 1) {
    checkBounds({ width: next() * 2000, height: next() * 2000, gap: next() * 150,
      toolbarWidth: next() * 400, panelWidth: next() * 900, panelHeight: next() * 900 });
  }
});

test('fixed arithmetic budget and input reads are independent of row count, without DOM IO', async () => {
  const source = await readFile(new URL('../../cmd/internal/webcast/web/feedback-layout.js', import.meta.url), 'utf8');
  let arithmeticCalls = 0;
  const context = vm.createContext({
    Math: new Proxy(Math, { get(target, key) {
      const value = target[key];
      return typeof value === 'function' ? (...args) => { arithmeticCalls += 1; return value(...args); } : value;
    } }),
    Number,
  });
  vm.runInContext(source.replace('export function', 'function'), context);
  const counts = [];
  for (const rowCount of [0, 1, 1000000]) {
    let reads = 0;
    const input = new Proxy({ width: 768, height: 800 }, { get(target, key) {
      reads += 1;
      assert.notEqual(key, 'rows', `must not inspect ${rowCount} rows`);
      return target[key];
    } });
    arithmeticCalls = 0;
    context.feedbackLayout(input);
    counts.push({ arithmeticCalls, reads });
  }
  assert.ok(counts[0].arithmeticCalls > 0 && counts[0].arithmeticCalls <= 32);
  assert.equal(counts[0].reads, 6);
  assert.deepEqual(counts[1], counts[0]);
  assert.deepEqual(counts[2], counts[0]);
});
