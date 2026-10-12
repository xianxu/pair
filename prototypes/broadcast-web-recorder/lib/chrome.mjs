import { spawn } from 'node:child_process';
import { mkdtempSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import path from 'node:path';
import CDP from 'chrome-remote-interface';

const CHROME = '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome';

export async function launch({ port = 9333, width = 1280, height = 800, headless = true } = {}) {
  const dir = mkdtempSync(path.join(process.env.TMPDIR || tmpdir(), 'bw-profile-'));
  const args = [
    `--user-data-dir=${dir}`, `--remote-debugging-port=${port}`,
    '--no-first-run', '--no-default-browser-check', '--disable-background-timer-throttling',
    '--disable-renderer-backgrounding', '--disable-backgrounding-occluded-windows',
    '--site-per-process', `--window-size=${width},${height}`, '--hide-scrollbars',
    '--force-device-scale-factor=1',
    ...(headless ? ['--headless=new'] : []), 'about:blank',
  ];
  const proc = spawn(CHROME, args, { stdio: ['ignore', 'ignore', 'pipe'] });
  let err = '';
  proc.stderr.on('data', d => { err += d; });
  for (let i = 0; i < 100; i++) {
    try { await CDP.Version({ port }); break; } catch { await new Promise(r => setTimeout(r, 100)); }
    if (i === 99) throw new Error('chrome did not start: ' + err.slice(-2000));
  }
  const browser = await CDP({ port, target: (await CDP.Version({ port })).webSocketDebuggerUrl });
  return {
    port, browser, proc,
    async close() { try { await browser.close(); } catch {} proc.kill('SIGKILL'); await new Promise(r => setTimeout(r, 300)); rmSync(dir, { recursive: true, force: true }); },
  };
}

// Create a fresh page target and return a flat-session client for it.
export async function newPage(chrome, { width = 1280, height = 800 } = {}) {
  const { targetId } = await chrome.browser.Target.createTarget({ url: 'about:blank' });
  const page = await CDP({ port: chrome.port, target: targetId });
  await page.Emulation.setDeviceMetricsOverride({ width, height, deviceScaleFactor: 1, mobile: false });
  return { targetId, page };
}
