// Scripted interactions, identical for every recorder mode.
const sleep = ms => new Promise(r => setTimeout(r, ms));
async function center(page, sel) {
  const r = await page.Runtime.evaluate({ expression: `(() => { const e = document.querySelector(${JSON.stringify(sel)}); if (!e) return null; e.scrollIntoView({block:'nearest'}); const b = e.getBoundingClientRect(); return [b.x + b.width/2, b.y + b.height/2]; })()`, returnByValue: true });
  return r.result.value;
}
async function click(page, sel) {
  const p = await center(page, sel); if (!p) { console.error('no element', sel); return; }
  const [x, y] = p;
  await page.Input.dispatchMouseEvent({ type: 'mouseMoved', x, y });
  await page.Input.dispatchMouseEvent({ type: 'mousePressed', x, y, button: 'left', clickCount: 1 });
  await page.Input.dispatchMouseEvent({ type: 'mouseReleased', x, y, button: 'left', clickCount: 1 });
}
async function type(page, text) { for (const ch of text) { await page.Input.insertText({ text: ch }); await sleep(60); } }
async function key(page, k, code) { await page.Input.dispatchKeyEvent({ type: 'keyDown', key: k, code, windowsVirtualKeyCode: k === 'Enter' ? 13 : 8 });
  await page.Input.dispatchKeyEvent({ type: 'keyUp', key: k, code, windowsVirtualKeyCode: k === 'Enter' ? 13 : 8 }); }
async function wheel(page, dy, x = 640, y = 400) { await page.Input.dispatchMouseEvent({ type: 'mouseWheel', x, y, deltaX: 0, deltaY: dy }); }
async function until(t0, ms) { const left = ms - (Date.now() - t0); if (left > 0) await sleep(left); }

export const scenarios = {
  async idle(page, ms) { await sleep(ms); },
  async app(page, ms) {
    const t0 = Date.now();
    await sleep(2000); await click(page, '#nav-tasks'); await sleep(800);
    await click(page, '#draft'); await type(page, 'Review PR 42'); await key(page, 'Enter', 'Enter'); await sleep(1200);
    await click(page, '#search'); await type(page, 'fix'); await sleep(1500);
    for (let i = 0; i < 3; i++) { await key(page, 'Backspace', 'Backspace'); await sleep(100); }
    await click(page, 'li input[type=checkbox]'); await sleep(800);
    await click(page, '#nav-settings'); await sleep(800); await wheel(page, 400); await sleep(800);
    await click(page, '#open-modal'); await sleep(1500); await click(page, '#close-modal'); await sleep(800);
    await click(page, '#nav-dashboard'); await sleep(2500);
    await click(page, '#next-doc'); await sleep(2500);
    await until(t0, ms);
  },
  async browse(page, ms) {
    const t0 = Date.now();
    await sleep(1500);
    for (let i = 0; i < 6; i++) { await wheel(page, 500); await sleep(900); }
    await wheel(page, -1500); await sleep(1000);
    await until(t0, ms);
  },
};
