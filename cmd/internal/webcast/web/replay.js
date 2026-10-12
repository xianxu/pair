// Replaying a recording in the viewer (#471): a source that stands in for
// the live event stream behind the same surface connect() returns, so the
// model and the renderer are #460's, unchanged. The state at t is the
// catch-up of the records up to t, reduced (the format's seek rule: one
// segment at most is replayed); the timeline bar seeks, plays, pauses and
// changes speed, shows pauses as gaps, notes and strokes at their moments,
// and who drove.

import { Model } from './model.js';

// catchUp is recording.CatchUp (catchup.go), record for record: head and
// the assets before the latest seg, the bare seg the current document began
// in, the notes, note states and strokes since then, the latest ctl before
// the latest seg, and everything from the latest seg on. Go's is the rule;
// replay_live_test.go holds this port to it at every pin.
export function catchUp(recs) {
  let last = -1, doc = -1;
  recs.forEach((r, i) => {
    if (r.k !== 'seg') return;
    last = i;
    if (r.why !== 'resume') doc = i;
  });
  if (last < 0) return recs.slice();
  let ctl = -1;
  for (let i = last - 1; i >= 0 && ctl < 0; i--) if (recs[i].k === 'ctl') ctl = i;
  if (doc < 0) doc = 0;
  const out = [], kept = new Set();
  recs.forEach((r, i) => {
    if (i === doc && doc < last) { out.push(r); return; }
    switch (r.k) {
      case 'head': case 'asset':
        if (i < last) { out.push(r); return; }
        break;
      case 'note':
        if (i >= doc) { kept.add(r.id); out.push(r); }
        return;
      case 'note-state':
        if (i >= doc && kept.has(r.id)) out.push(r);
        return;
      case 'stroke':
        if (i >= doc) out.push(r);
        return;
      case 'ctl':
        if (i === ctl) { out.push(r); return; }
        break;
    }
    if (i >= last) out.push(r);
  });
  return out;
}

// timeline is what the bar draws, from the merged records: segments (a seg
// to the next stop or seg), gaps (a pause to the next seg), notes, strokes,
// who drove when (ctl records carry the control state after them), and the
// acts a driver caused (records with by).
export function timeline(recs) {
  const end = recs.length ? recs[recs.length - 1].t : 0;
  const tl = { end, segs: [], gaps: [], notes: [], strokes: [], drives: [], acts: [] };
  let seg = null, stopped = null, drive = null;
  const closeSeg = (t) => { if (seg) { seg.t1 = t; tl.segs.push(seg); seg = null; } };
  for (const r of recs) {
    switch (r.k) {
      case 'seg':
        closeSeg(r.t);
        if (stopped !== null) tl.gaps.push({ t0: stopped, t1: r.t });
        stopped = null;
        seg = { seg: r.seg, why: r.why, t0: r.t, t1: end };
        break;
      case 'stop':
        closeSeg(r.t);
        if (r.why === 'pause') stopped = r.t;
        break;
      case 'note':
        tl.notes.push({ t: r.t, id: r.id, author: r.author, text: r.text || '' });
        break;
      case 'stroke':
        tl.strokes.push({ t: r.t, id: r.id, color: r.color || (r.author && r.author.color) || '', dur: r.pts.length ? r.pts[r.pts.length - 1][2] || 0 : 0 });
        break;
      case 'ctl': {
        const d = r.driver && r.driver.id ? r.driver : null;
        if (drive && (!d || d.id !== drive.driver.id)) { drive.t1 = r.t; tl.drives.push(drive); drive = null; }
        if (d && !drive) drive = { t0: r.t, t1: end, driver: d };
        break;
      }
    }
    if (r.by) tl.acts.push({ t: r.t, by: r.by, k: r.k });
  }
  closeSeg(end);
  if (drive) tl.drives.push(drive);
  return tl;
}

// GAP_SHARE is the most of the bar's width one gap takes: an hour's pause
// must not flatten the segments around it.
export const GAP_SHARE = 0.03;

// placer maps time to x on a bar width px wide and back. Segments and the
// time outside any gap are linear; each gap is squeezed to at most
// GAP_SHARE of the width. A point inside a gap seeks to the gap's end
// (where the cast resumed).
export function placer(tl, width) {
  const spans = [];
  let at = 0;
  for (const g of tl.gaps) {
    if (g.t0 > at) spans.push({ t0: at, t1: g.t0, gap: false });
    spans.push({ t0: g.t0, t1: g.t1, gap: true });
    at = g.t1;
  }
  if (tl.end > at || spans.length === 0) spans.push({ t0: at, t1: Math.max(at, tl.end), gap: false });
  // A gap weighs its duration, capped so that it takes at most GAP_SHARE
  // of the width however many other gaps there are (c / (live + c) is
  // GAP_SHARE, and other gaps only widen the denominator). With nothing
  // but gaps, durations are the weights.
  const live = spans.reduce((n, s) => n + (s.gap ? 0 : s.t1 - s.t0), 0);
  const cap = live > 0 ? (GAP_SHARE * live) / (1 - GAP_SHARE) : Infinity;
  const weight = (s) => s.gap ? Math.min(s.t1 - s.t0, cap) : s.t1 - s.t0;
  const sum = spans.reduce((n, s) => n + weight(s), 0) || 1;
  let x = 0;
  for (const s of spans) { s.x0 = x; x += (weight(s) / sum) * width; s.x1 = x; }
  const xAt = (t) => {
    const s = spans.find((s) => t <= s.t1) || spans[spans.length - 1];
    const d = s.t1 - s.t0;
    return d > 0 ? s.x0 + ((Math.min(Math.max(t, s.t0), s.t1) - s.t0) / d) * (s.x1 - s.x0) : s.x0;
  };
  const timeAt = (px) => {
    const p = Math.min(Math.max(px, 0), width);
    const s = spans.find((s) => p <= s.x1) || spans[spans.length - 1];
    if (s.gap) return s.t1;
    const w = s.x1 - s.x0;
    return Math.round(w > 0 ? s.t0 + ((p - s.x0) / w) * (s.t1 - s.t0) : s.t0);
  };
  return { spans, xAt, timeAt };
}

// parseMoment reads a #t=<ms> link (a whole number of ms since the
// recording started); anything else is null. momentHash writes one.
export function parseMoment(hash) {
  const m = /^#t=(\d{1,15})$/.exec(hash || '');
  return m ? Number(m[1]) : null;
}
export const momentHash = (t) => `#t=${Math.max(0, Math.round(t))}`;

// SPEEDS are the playback rates the bar offers.
export const SPEEDS = [0.5, 1, 2, 4];

// Player is the replay's state: the playhead t, playing or not, the speed,
// and the model and renderer drawn to t. It holds no clock: tick(dt) is
// driven by the shell's animation frames, so tests drive it by hand.
// deps: model() a fresh Model; renderer(model) a renderer over it ({apply,
// flush, revokeAll?}); onRecord(r, model) after each applied record.
// A record the model refuses is a broken recording: the player stops in
// error, keeping the last state it drew.
export class Player {
  constructor(recs, deps) {
    this.recs = recs;
    this.deps = { model: () => new Model(), onRecord: () => {}, ...deps };
    this.tl = timeline(recs);
    this.end = this.tl.end;
    this.playing = false;
    this.speed = 1;
    this.error = null;
    this.rebuilds = 0;
    this.fresh();
  }

  fresh() {
    if (this.renderer && this.renderer.revokeAll) this.renderer.revokeAll();
    this.model = this.deps.model();
    this.renderer = this.deps.renderer(this.model);
    this.next = 0;
    this.t = 0;
  }

  // cut is the index of the first record after t.
  cut(t) {
    let lo = 0, hi = this.recs.length;
    while (lo < hi) { const mid = (lo + hi) >> 1; if (this.recs[mid].t <= t) lo = mid + 1; else hi = mid; }
    return lo;
  }

  apply(r) {
    this.model.apply(r);
    this.renderer.apply(r);
    this.deps.onRecord(r, this.model);
  }

  // seek draws the recording at t. Forward within the same stretch it
  // applies the records in between; across a seg, or backward, it rebuilds
  // from the catch-up at t.
  seek(t) {
    if (this.error) return;
    t = Math.min(Math.max(0, t), this.end);
    try {
      if (t >= this.t && this.next > 0) {
        const to = this.cut(t);
        let crossing = false;
        for (let i = this.next; i < to; i++) if (this.recs[i].k === 'seg') { crossing = true; break; }
        if (!crossing) {
          for (; this.next < to; this.next++) this.apply(this.recs[this.next]);
          this.t = t;
          return;
        }
      }
      this.rebuild(t);
    } catch (err) {
      this.error = err;
      this.playing = false;
    }
  }

  rebuild(t) {
    this.rebuilds++;
    const to = this.cut(t);
    this.fresh();
    for (const r of catchUp(this.recs.slice(0, to))) this.apply(r);
    this.next = to;
    this.t = t;
  }

  // gapAt is the pause t falls inside, if any.
  gapAt(t) { return this.tl.gaps.find((g) => t > g.t0 && t < g.t1) || null; }

  // tick advances a playing replay by dt ms of wall time, skipping pauses;
  // it stops at the end.
  tick(dt) {
    if (!this.playing || this.error) return;
    let t = this.t + dt * this.speed;
    const g = this.gapAt(t);
    if (g) t = g.t1;
    if (t >= this.end) { t = this.end; this.playing = false; }
    this.seek(t);
  }

  play() {
    if (this.error) return;
    if (this.t >= this.end) this.seek(0);
    this.playing = true;
  }

  pause() { this.playing = false; }

  setSpeed(s) { if (SPEEDS.includes(s)) this.speed = s; }
}
