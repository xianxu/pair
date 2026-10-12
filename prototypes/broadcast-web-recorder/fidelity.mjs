// Replay a recording with A and/or B at each screenshot moment and pixel-diff against the real tab.
// usage: node fidelity.mjs <run-dir> [A|B|AB]
import { readFileSync, writeFileSync, mkdirSync, existsSync } from 'node:fs';
import { PNG } from 'pngjs';
import pixelmatch from 'pixelmatch';
import { launch, newPage } from './lib/chrome.mjs';
import { serve } from './lib/server.mjs';
import { lowres } from './lib/shot.mjs';

const [dir, which = 'AB'] = process.argv.slice(2);
const meta = JSON.parse(readFileSync(dir + '/meta.json'));
const [W, H] = meta.viewport;
const here = new URL('.', import.meta.url).pathname;
const servers = [await serve(8801, here, '127.0.0.1'), await serve(8802, here, '127.0.0.1'), await serve(8802, here, '::1')];
const chrome = await launch({ port: 9900 + Math.floor(Math.random() * 90) });
const OFFSETS = [-600, -400, -200, 0, 200]; // a screenshot shows the last frame painted before its timestamp, up to a few hundred ms earlier; take the best-aligned replay
mkdirSync(dir + '/diff', { recursive: true });

function diff(a, b, outPath) {
  const A = PNG.sync.read(a), B = PNG.sync.read(b);
  if (A.width !== B.width || A.height !== B.height) return 1;
  const d = new PNG({ width: A.width, height: A.height });
  const n = pixelmatch(A.data, B.data, d.data, A.width, A.height, { threshold: 0.1 });
  if (outPath) writeFileSync(outPath, PNG.sync.write(d));
  return n / (A.width * A.height);
}

async function run(kind) {
  const { page } = await newPage(chrome, { width: W, height: H });
  await page.Page.enable(); await page.Runtime.enable();
  page.Runtime.consoleAPICalled(p => { if (p.type === 'error') console.error(kind, 'console', p.args.map(a => a.value).join(' ').slice(0, 200)); });
  const loaded = page.Page.loadEventFired();
  await page.Page.navigate({ url: `http://127.0.0.1:8801/replay/${kind.toLowerCase()}.html` }); await loaded;
  const evalp = async (expression) => { const r = await page.Runtime.evaluate({ expression, awaitPromise: true, returnByValue: true });
    if (r.exceptionDetails) throw new Error(kind + ': ' + JSON.stringify(r.exceptionDetails).slice(0, 500)); return r.result.value; };
  if (kind === 'A') await evalp(`fetch('/${dir}/rrweb.json').then(r => r.json()).then(e => aLoad(e))`);
  else await evalp(`fetch('/${dir}/cdp.jsonl').then(r => r.text()).then(t => bwLoad(t.trim().split('\\n').map(JSON.parse)))`);
  const rows = [];
  for (const s of meta.shots) {
    let best = { score: 2 }, stats;
    for (const off of OFFSETS) {
      const t = s.t + off;
      if (kind === 'A') await evalp(`aSeek(${t})`); else stats = await evalp(`bwSeek(${t}, ${W}, ${H})`);
      await new Promise(r => setTimeout(r, 250)); // images and fonts from the asset origin
      const buf = await lowres(page);
      const score = diff(readFileSync(`${dir}/shots/${s.i}.png`), buf);
      if (score < best.score) best = { score, off, buf, stats };
    }
    writeFileSync(`${dir}/diff/${kind}-${s.i}.png`, best.buf);
    diff(readFileSync(`${dir}/shots/${s.i}.png`), best.buf, `${dir}/diff/${kind}-${s.i}-diff.png`);
    rows.push({ i: s.i, score: +(best.score * 100).toFixed(2), off: best.off, stats: best.stats && { missing: best.stats.missing } });
  }
  await page.close?.();
  return rows;
}

const result = {};
for (const k of which.split('')) result[k] = await run(k);
const summary = Object.fromEntries(Object.entries(result).map(([k, rows]) => {
  const s = rows.map(r => r.score).sort((a, b) => a - b);
  return [k, { median: s[Math.floor(s.length / 2)], max: s[s.length - 1], perShot: s.join(' ') }];
}));
writeFileSync(dir + '/fidelity.json', JSON.stringify({ summary, result }, null, 1));
console.log(dir, JSON.stringify(summary));
for (const [k, rows] of Object.entries(result)) { const miss = rows.map(r => r.stats && JSON.stringify(r.stats.missing)).filter(x => x && x !== '{}'); if (miss.length) console.log(k, 'missing', miss.slice(-2).join(' ')); }
await chrome.close(); servers.forEach(s => s.close()); process.exit(0);
