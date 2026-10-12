// Reading #457's fixtures in node: the record files merged as
// recording.Merge does (by t; at equal t events, then notes, then pointer
// strokes), and the canonical pins.
import { readFileSync, existsSync, readdirSync } from 'node:fs';
import { join, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';

export const fixturesDir = join(dirname(fileURLToPath(import.meta.url)), '../../cmd/internal/webcast/recording/testdata/fixtures');
// Every fixture directory (one with pins), so a new fixture is never left
// out of the viewer's checks.
export const fixtures = readdirSync(fixturesDir, { withFileTypes: true })
  .filter((d) => d.isDirectory() && existsSync(join(fixturesDir, d.name, 'canonical.sha256')))
  .map((d) => d.name);

function lines(file) {
  if (!existsSync(file)) return [];
  return readFileSync(file, 'utf8').split('\n').filter(Boolean).map((l) => JSON.parse(l));
}

export function records(name) {
  const dir = join(fixturesDir, name);
  const tagged = [];
  ['events.jsonl', 'notes.jsonl', 'pointer.jsonl'].forEach((f, rank) =>
    lines(join(dir, f)).forEach((r, i) => tagged.push({ r, rank, i })));
  tagged.sort((a, b) => a.r.t - b.r.t || a.rank - b.rank || a.i - b.i);
  return tagged.map((x) => x.r);
}

export function pins(name) {
  return readFileSync(join(fixturesDir, name, 'canonical.sha256'), 'utf8').split('\n').filter(Boolean)
    .map((l) => { const [t, hash] = l.split(' '); return { t: Number(t), hash }; });
}
