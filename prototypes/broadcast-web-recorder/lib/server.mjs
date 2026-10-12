import http from 'node:http';
import { readFile } from 'node:fs/promises';
import path from 'node:path';
const ROOT = new URL('../pages/', import.meta.url).pathname;
const TYPES = { '.html': 'text/html', '.js': 'text/javascript', '.css': 'text/css', '.png': 'image/png', '.ttf': 'font/ttf', '.json': 'application/json' };
export function serve(port, extraRoot, host = '127.0.0.1') {
  const srv = http.createServer(async (req, res) => {
    const u = new URL(req.url, 'http://x');
    for (const root of [ROOT, extraRoot].filter(Boolean)) {
      try {
        const body = await readFile(path.join(root, decodeURIComponent(u.pathname)));
        res.writeHead(200, { 'content-type': TYPES[path.extname(u.pathname)] || 'application/octet-stream', 'access-control-allow-origin': '*', 'cache-control': 'no-store' });
        return res.end(body);
      } catch {}
    }
    res.writeHead(404); res.end('nf');
  });
  return new Promise(r => srv.listen(port, host, () => r(srv)));
}
