// Record one page with prototype A (rrweb injected over CDP), prototype B (own on CDP), both, or neither.
// usage: node record.mjs <out-dir> <url> <scenario> <mode: none|A|B|AB> [seconds]
import { readFileSync, writeFileSync, mkdirSync, createWriteStream } from 'node:fs';
import { launch, newPage } from './lib/chrome.mjs';
import { serve } from './lib/server.mjs';
import { scenarios } from './scenarios.mjs';
import { lowres } from './lib/shot.mjs';

const [out, url, scenarioName, mode, secs = '20'] = process.argv.slice(2);
const A = mode.includes('A'), B = mode.includes('B');
const W = 1280, H = 800;
mkdirSync(out + '/shots', { recursive: true });

const servers = [await serve(8801, undefined, '127.0.0.1'), await serve(8802, undefined, '127.0.0.1'), await serve(8802, undefined, '::1')];
const chrome = await launch({ port: 9333 + Math.floor(Math.random() * 500) });
const { page, targetId } = await newPage(chrome, { width: W, height: H });

const RRWEB = readFileSync(new URL('./node_modules/rrweb/dist/rrweb.umd.min.cjs', import.meta.url), 'utf8');
// rrweb in every frame; the top frame hands events to the driver through a binding, child frames post to the
// parent (rrweb's recordCrossOriginIframes protocol).
const A_INJECT = `(function(){ if (window.__bwAStarted) return; window.__bwAStarted = true;
  var module = { exports: {} }; var exports = module.exports; ${RRWEB}; var rr = module.exports.record ? module.exports : window.rrweb;
  var top = window === window.top; var buf = [];
  function flush(){ if (buf.length && typeof window.__bwA === 'function') { window.__bwA(JSON.stringify(buf)); buf = []; } }
  if (top) setInterval(flush, 50);
  rr.record({ emit: function(e){ if (top) buf.push(e); }, recordCrossOriginIframes: true, collectFonts: true,
    inlineStylesheet: true, inlineImages: false, recordCanvas: false, sampling: { scroll: 50, input: 'last' } });
})();`;

// Observer for state CDP does not report: form values set by script (no event fires), scroll offsets. Runs in an
// isolated world, so the page cannot see it; elements reach CDP as remote objects and become nodeIds.
const B_WORLD = '__bw_observer';
const B_INJECT = `(function(){ if (window.__bwObs) return; window.__bwObs = true;
  var last = new WeakMap(), dirty = [];
  function val(el){ return el.type === 'checkbox' || el.type === 'radio' ? (el.checked ? '1' : '0') : el.tagName === 'SELECT' ? String(el.selectedIndex) : el.value; }
  function scan(root){ root.querySelectorAll('input,textarea,select').forEach(function(el){ var v = val(el); if (last.get(el) !== v) { last.set(el, v); dirty.push({ el: el, k: 'value', v: v }); } });
    root.querySelectorAll('*').forEach(function(el){ if (el.shadowRoot) scan(el.shadowRoot); }); }
  var lastScroll = new WeakMap();
  document.addEventListener('scroll', function(e){ var t = e.target === document ? document.scrollingElement : e.target; if (!t) return;
    var s = t.scrollLeft + ',' + t.scrollTop; if (lastScroll.get(t) !== s) { lastScroll.set(t, s); dirty.push({ el: t, k: 'scroll', v: s }); } }, true);
  setInterval(function(){ if (document.documentElement) scan(document); if (dirty.length && typeof __bwB === 'function') __bwB(String(dirty.length)); }, 100);
  window.__bwTake = function(){ var d = dirty; dirty = []; return d; };
})();`;

const aEvents = [];
const bLog = createWriteStream(out + '/cdp.jsonl');
let bCount = 0, bBytes = 0;
const bEmit = (o) => { const s = JSON.stringify(o); bCount++; bBytes += s.length + 1; bLog.write(s + '\n'); };
const sessions = new Map(); // sessionId ('' for top) -> { frameId, isolatedCtx }
const send = (method, params = {}, sessionId) => page.send(method, params, sessionId || undefined);
const bNeeds = { mainWorldHooks: [], cdpOnly: [] };

async function setupTarget(sessionId, frameId) {
  sessions.set(sessionId, { frameId });
  await send('Page.enable', {}, sessionId);
  await send('Runtime.enable', {}, sessionId);
  if (A) {
    if (!sessionId) await send('Runtime.addBinding', { name: '__bwA' });
    await send('Page.addScriptToEvaluateOnNewDocument', { source: A_INJECT, runImmediately: true }, sessionId);
  }
  if (B) {
    await send('Runtime.addBinding', { name: '__bwB', executionContextName: B_WORLD }, sessionId);
    await send('Page.addScriptToEvaluateOnNewDocument', { source: B_INJECT, worldName: B_WORLD, runImmediately: true }, sessionId);
    await send('DOM.enable', { includeWhitespace: 'all' }, sessionId);
    await send('CSS.enable', {}, sessionId);
    await send('Network.enable', {}, sessionId);
  }
  await send('Target.setAutoAttach', { autoAttach: true, waitForDebuggerOnStart: true, flatten: true }, sessionId);
  if (sessionId) await send('Runtime.runIfWaitingForDebugger', {}, sessionId);
  if (B) await bSnapshot(sessionId);
}

async function bSnapshot(sessionId) {
  const { root } = await send('DOM.getDocument', { depth: -1, pierce: true }, sessionId);
  bEmit({ t: Date.now(), s: sessionId, frameId: sessions.get(sessionId).frameId, type: 'snapshot', root });
}

// B: changed sheets get their text fetched; assets are fetched from the network log for volume accounting.
const sheetHeaders = new Map(); // styleSheetId -> header
const isoCtx = new Map(); // sid|frameId -> isolated-world context id
// CSSOM edits (insertRule) fire styleSheetChanged, but getStyleSheetText returns the source text, not the rules: the
// rules are read from the owner element in the isolated world.
async function bSheetRules(sid, header) {
  const ctx = isoCtx.get(sid + '|' + header.frameId);
  if (!ctx || !header.ownerNode) return;
  try {
    const { object } = await send('DOM.resolveNode', { backendNodeId: header.ownerNode, executionContextId: ctx }, sid);
    const r = await send('Runtime.callFunctionOn', { objectId: object.objectId, returnByValue: true,
      functionDeclaration: 'function(){ return this.sheet ? Array.from(this.sheet.cssRules, r => r.cssText) : null }' }, sid);
    bEmit({ t: Date.now(), s: sid, type: 'rules', ownerNode: header.ownerNode, rules: r.result.value });
    send('Runtime.releaseObject', { objectId: object.objectId }, sid).catch(() => {});
  } catch (e) { bEmit({ t: Date.now(), s: sid, type: 'rules-error', err: String(e.message) }); }
}
async function bSheetText(sessionId, styleSheetId, header) {
  try {
    const { text } = await send('CSS.getStyleSheetText', { styleSheetId }, sessionId);
    bEmit({ t: Date.now(), s: sessionId, type: 'sheet', styleSheetId, text, header });
  } catch (e) { bEmit({ t: Date.now(), s: sessionId, type: 'sheet-error', styleSheetId, err: String(e.message) }); }
}
const assets = { count: 0, bytes: 0, byType: {} };
const requests = new Map();

page.on('event', async (msg) => {
  const { method, params } = msg; const sid = msg.sessionId || '';
  try {
    if (method === 'Target.attachedToTarget') {
      const ti = params.targetInfo;
      if (ti.type === 'iframe') { setupTarget(params.sessionId, ti.targetId).catch(e => console.error('attach', e.message)); }
      else send('Runtime.runIfWaitingForDebugger', {}, params.sessionId).catch(() => {});
      return;
    }
    if (method === 'Runtime.bindingCalled') {
      if (params.name === '__bwA') { for (const e of JSON.parse(params.payload)) aEvents.push(e); return; }
      if (params.name === '__bwB') { bTakeObserved(sid, params.executionContextId); return; }
    }
    if (!B) return;
    if (method.startsWith('DOM.')) {
      if (method === 'DOM.documentUpdated') { await bSnapshot(sid); return; }
      // style changes through el.style arrive as an invalidation, not an attributeModified: fetch the attribute.
      if (method === 'DOM.inlineStyleInvalidated') {
        for (const nodeId of params.nodeIds) send('DOM.getAttributes', { nodeId }, sid).then(({ attributes }) => {
          const i = attributes.indexOf('style');
          bEmit({ t: Date.now(), s: sid, type: i >= 0 ? 'attributeModified' : 'attributeRemoved', nodeId, name: 'style', value: i >= 0 ? attributes[i + 1] : undefined });
        }).catch(() => {});
        return;
      }
      bEmit({ t: Date.now(), s: sid, type: method.slice(4), ...params });
      if (method === 'DOM.childNodeInserted' && params.node.childNodeCount > 0 && !params.node.children)
        send('DOM.requestChildNodes', { nodeId: params.node.nodeId, depth: -1, pierce: true }, sid).catch(() => {});
      if (method === 'DOM.adoptedStyleSheetsModified') for (const id of params.adoptedStyleSheets) bSheetText(sid, id);
      return;
    }
    if (method === 'CSS.styleSheetAdded') { const h = params.header; sheetHeaders.set(h.styleSheetId, h); bEmit({ t: Date.now(), s: sid, type: 'sheetAdded', header: h });
      if (h.isConstructed || h.isMutable) bSheetText(sid, h.styleSheetId, h);
      // CDP cuts text node values at 10k characters; a big <style> (Tailwind's runtime output) needs its full source.
      else if (h.ownerNode && h.length > 9000) send('CSS.getStyleSheetText', { styleSheetId: h.styleSheetId }, sid)
        .then(({ text }) => bEmit({ t: Date.now(), s: sid, type: 'styleText', ownerNode: h.ownerNode, text })).catch(() => {});
      return; }
    if (method === 'CSS.styleSheetChanged') { const h = sheetHeaders.get(params.styleSheetId);
      if (h && h.isConstructed) bSheetText(sid, params.styleSheetId); else if (h) bSheetRules(sid, h); return; }
    if (method === 'Runtime.executionContextCreated') { const c = params.context;
      if (c.name === B_WORLD) isoCtx.set(sid + '|' + c.auxData.frameId, c.id); return; }
    if (method === 'CSS.fontsUpdated') { if (params.font) bEmit({ t: Date.now(), s: sid, type: 'font', font: params.font }); return; }
    if (method === 'Network.responseReceived') { requests.set(params.requestId, { type: params.type, url: params.response.url }); return; }
    if (method === 'Network.loadingFinished') {
      const { type, url: aurl } = requests.get(params.requestId) || {};
      if (['Stylesheet', 'Image', 'Font'].includes(type)) {
        try { const b = await send('Network.getResponseBody', { requestId: params.requestId }, sid);
          const n = b.base64Encoded ? Math.floor(b.body.length * 3 / 4) : Buffer.byteLength(b.body);
          assets.count++; assets.bytes += n; assets.byType[type] = (assets.byType[type] || 0) + n;
          // Stylesheets go into the stream (the asset proxy's job in the product), url() made absolute.
          if (type === 'Stylesheet') { const text = b.base64Encoded ? Buffer.from(b.body, 'base64').toString() : b.body;
            bEmit({ t: Date.now(), s: sid, type: 'asset', url: aurl, text: text.replace(/url\(\s*(['"]?)([^'")]+)\1\s*\)/g,
              (m, q, u) => /^(data:|#)/.test(u) ? m : `url(${q}${new URL(u, aurl).href}${q})`) }); } } catch {}
      }
    }
  } catch (e) { if (!/No node with given id|Could not find node/.test(e.message)) console.error(method, e.message); }
});

async function bTakeObserved(sid, contextId) {
  const r = await send('Runtime.evaluate', { expression: '__bwTake()', contextId }, sid);
  const len = (await send('Runtime.callFunctionOn', { functionDeclaration: 'function(){return this.length}', objectId: r.result.objectId, returnByValue: true }, sid)).result.value;
  for (let i = 0; i < len; i++) {
    const it = await send('Runtime.callFunctionOn', { functionDeclaration: 'function(i){return this[i]}', objectId: r.result.objectId, arguments: [{ value: i }] }, sid);
    const props = (await send('Runtime.getProperties', { objectId: it.result.objectId, ownProperties: true }, sid)).result;
    const el = props.find(p => p.name === 'el').value, k = props.find(p => p.name === 'k').value.value, v = props.find(p => p.name === 'v').value.value;
    try { const { nodeId } = await send('DOM.requestNode', { objectId: el.objectId }, sid);
      bEmit({ t: Date.now(), s: sid, type: k, nodeId, v }); } catch {}
  }
  send('Runtime.releaseObjectGroup', { objectGroup: '' }, sid).catch(() => {});
}

// Fidelity reference: low-res screenshots of the real tab, timestamped by the page clock.
const shots = [];
async function shot() {
  const t0 = Date.now();
  const buf = await lowres(page);
  const t1 = Date.now(); const i = shots.length;
  writeFileSync(`${out}/shots/${i}.png`, buf);
  shots.push({ i, t: Math.round((t0 + t1) / 2), took: t1 - t0 });
}

await page.Page.enable();
await setupTarget('', targetId);
const pm0 = await page.Performance.enable().then(() => page.Performance.getMetrics());
const navStart = Date.now();
await page.Page.navigate({ url });
await new Promise(r => setTimeout(r, 1500));
let shooting = true;
const shooter = (async () => { while (shooting) { await shot(); await new Promise(r => setTimeout(r, 2000)); } })();
await scenarios[scenarioName](page, Number(secs) * 1000);
shooting = false; await shooter;
const frames = (await page.Runtime.evaluate({ expression: 'JSON.stringify(window.__frames || [])', returnByValue: true })).result.value;
const pm1 = await page.Performance.getMetrics();
const metric = (m, n) => (m.metrics.find(x => x.name === n) || {}).value || 0;
const elapsed = (Date.now() - navStart) / 1000;
writeFileSync(out + '/rrweb.json', JSON.stringify(aEvents));
await new Promise(r => bLog.end(r));
const f = JSON.parse(frames || '[]').slice(5).sort((a, b) => a - b);
const pct = q => f.length ? +f[Math.floor(q * (f.length - 1))].toFixed(1) : null;
const meta = { url, scenario: scenarioName, mode, elapsed, viewport: [W, H], navStart, shots,
  A: A ? { events: aEvents.length, bytes: Buffer.byteLength(JSON.stringify(aEvents)), byType: aEvents.reduce((m, e) => (m[e.type] = (m[e.type] || 0) + 1, m), {}) } : null,
  B: B ? { events: bCount, bytes: bBytes, assets } : null,
  page: { frames: f.length, p50: pct(0.5), p95: pct(0.95), p99: pct(0.99), long: f.filter(x => x > 33.4).length,
    taskSec: +(metric(pm1, 'TaskDuration') - metric(pm0, 'TaskDuration')).toFixed(3), scriptSec: +(metric(pm1, 'ScriptDuration') - metric(pm0, 'ScriptDuration')).toFixed(3),
    layoutSec: +(metric(pm1, 'LayoutDuration') - metric(pm0, 'LayoutDuration')).toFixed(3) } };
writeFileSync(out + '/meta.json', JSON.stringify(meta, null, 1));
console.log(JSON.stringify({ mode, scenario: scenarioName, elapsed, A: meta.A && { events: meta.A.events, bytes: meta.A.bytes }, B: meta.B && { events: meta.B.events, bytes: meta.B.bytes }, page: meta.page }));
await chrome.close(); servers.forEach(s => s.close()); process.exit(0);
