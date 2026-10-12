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

// replayPath says whether the viewer was opened on a recording
// (<token>/r/<id>/) rather than a live cast.
export const replayPath = (pathname) => /\/r\/[^/]+\/$/.test(pathname || '');

// replaySource plays a recording behind the surface connect() returns
// (model, renderer, onRecord), so the shell lays out and draws it as it
// does a live cast. deps:
//   fetchRecords()    the recording's merged records (a promise)
//   renderer(model)   the renderer drawing that model
//   show(view)        renders the bar's status, title and URL
//   frame()           called after the page is drawn (once per frame)
//   schedule(fn)      runs fn before the next paint
//   now()             a clock in ms (performance.now)
//   hash(), setHash(h) the location's #t= (setHash replaces, no history)
//   ui(player, on)    builds the timeline bar; on.seek(t), on.toggle(),
//                     on.speed(s); returns {update()}
export function replaySource(deps) {
  const hooks = [];
  let player = null, ui = null, last = null, queued = false;
  const v = {
    get model() { return player ? player.model : null; },
    get renderer() { return player ? player.renderer : { flush() {}, canonical: () => null, ready: async () => {}, boxOf: () => null, nodeAt: () => 0 }; },
    get player() { return player; },
    onRecord: (fn) => hooks.push(fn),
  };
  const show = () => {
    if (!player) return;
    const m = player.model;
    const page = { title: m.title || m.url || '', url: m.url || '', page: m.seg > 0 };
    if (player.error) return deps.show({ ...page, status: 'Recording error', live: false });
    // live is the recorded moment's: a pause in the recording looks as it
    // did to live viewers.
    deps.show({ ...page, status: m.seg > 0 && !m.live ? 'Replay · paused' : 'Replay', live: m.live });
  };
  const draw = () => {
    if (queued) return;
    queued = true;
    deps.schedule(() => {
      queued = false;
      if (player.playing) {
        const now = deps.now();
        player.tick(last === null ? 0 : now - last);
        last = now;
        if (!player.playing) mark();
      }
      player.renderer.flush();
      show();
      ui.update();
      deps.frame();
      if (player.playing) draw();
      else last = null;
    });
  };
  const mark = () => deps.setHash(momentHash(player.t));
  const seek = (t) => { player.seek(t); if (player.error) console.warn('couch viewer: recording refused:', player.error); mark(); draw(); };
  v.loaded = (async () => {
    let recs;
    try { recs = await deps.fetchRecords(); } catch (err) {
      deps.show({ status: 'Recording unavailable', live: false, title: '', url: '', page: false });
      throw err;
    }
    player = new Player(recs, { renderer: deps.renderer, onRecord: (r, m) => { for (const fn of hooks) fn(r, m); } });
    ui = deps.ui(player, {
      seek,
      toggle: () => { if (player.playing) { player.pause(); mark(); } else { player.play(); last = null; } draw(); },
      speed: (s) => { player.setSpeed(s); draw(); },
    });
    const at = parseMoment(deps.hash());
    player.seek(at === null ? 0 : at);
    draw();
    return player;
  })();
  v.onHash = () => { const at = parseMoment(deps.hash()); if (player && at !== null && at !== Math.round(player.t)) seek(at); };
  return v;
}

// ---- the page: the timeline bar and the marks over the page ----

// clock formats ms as m:ss.d.
export function clock(ms) {
  const s = Math.max(0, ms) / 1000, m = Math.floor(s / 60);
  return `${m}:${(s - m * 60).toFixed(1).padStart(4, '0')}`;
}

// safeColor keeps a viewer-chosen color only when it is a plain hex color.
const safeColor = (c, dflt) => (/^#[0-9a-fA-F]{3,8}$/.test(c || '') ? c : dflt);

// STROKE_FADE is how long a replayed stroke stays after its last point.
export const STROKE_FADE = 1500;

// timelineBar builds the bar into el: play/pause, speed, the track (live
// segments, pauses as gaps, who drove, notes and strokes as ticks, the
// playhead) and the time. Names and texts are recorded input: only ever
// text and titles. It returns {update()} for each frame.
export function timelineBar(doc, el, player, on) {
  const mk = (tag, cls, parent) => { const e = doc.createElement(tag); if (cls) e.className = cls; if (parent) parent.appendChild(e); return e; };
  el.replaceChildren();
  el.hidden = false;
  const play = mk('button', 'replay-play', el);
  play.type = 'button';
  play.addEventListener('click', () => on.toggle());
  const speed = mk('select', 'replay-speed', el);
  speed.setAttribute('aria-label', 'Speed');
  for (const s of SPEEDS) { const o = mk('option', '', speed); o.value = String(s); o.textContent = `${s}×`; }
  speed.value = String(player.speed);
  speed.addEventListener('change', () => on.speed(Number(speed.value)));
  const track = mk('div', 'replay-track', el);
  track.tabIndex = 0;
  track.setAttribute('role', 'slider');
  track.setAttribute('aria-label', 'Recording time');
  const time = mk('span', 'replay-time', el);
  const tl = player.tl;
  let place = null, width = 0;
  const marks = [];
  const at = (e, t0, t1) => marks.push({ e, t0, t1 });
  for (const s of tl.segs) { const e = mk('div', 'replay-seg', track); e.title = `Segment ${s.seg} (${s.why})`; at(e, s.t0, s.t1); }
  for (const g of tl.gaps) { const e = mk('div', 'replay-gap', track); e.title = `Paused ${clock(g.t1 - g.t0)}`; at(e, g.t0, g.t1); }
  for (const d of tl.drives) {
    const e = mk('div', 'replay-drive', track);
    e.style.background = safeColor(d.driver.color, '#8e8e93');
    e.title = `${d.driver.name || d.driver.id} drove`;
    at(e, d.t0, d.t1);
  }
  for (const n of tl.notes) {
    const e = mk('div', 'replay-tick replay-tick-note', track);
    e.style.background = safeColor(n.author && n.author.color, '#ffb000');
    e.title = `${clock(n.t)} ${(n.author && n.author.name) || ''}: ${n.text}`;
    e.addEventListener('pointerdown', (ev) => { ev.stopPropagation(); on.seek(n.t); });
    at(e, n.t, n.t);
  }
  for (const s of tl.strokes) {
    const e = mk('div', 'replay-tick replay-tick-stroke', track);
    e.style.background = safeColor(s.color, '#ff3b30');
    e.title = `${clock(s.t)} stroke`;
    at(e, s.t, s.t);
  }
  const head = mk('div', 'replay-head', track);
  const relayout = () => {
    width = track.clientWidth;
    place = placer(tl, width);
    for (const m of marks) {
      const x0 = place.xAt(m.t0), x1 = place.xAt(m.t1);
      m.e.style.left = `${x0}px`;
      if (m.t1 > m.t0) m.e.style.width = `${Math.max(1, x1 - x0)}px`;
    }
  };
  const seekAt = (e) => on.seek(place.timeAt(e.clientX - track.getBoundingClientRect().left));
  track.addEventListener('pointerdown', (e) => { track.setPointerCapture(e.pointerId); seekAt(e); });
  track.addEventListener('pointermove', (e) => { if (track.hasPointerCapture(e.pointerId)) seekAt(e); });
  track.addEventListener('keydown', (e) => {
    const step = { ArrowLeft: -1000, ArrowRight: 1000, Home: -Infinity, End: Infinity }[e.key];
    if (step !== undefined) { e.preventDefault(); on.seek(player.t + step); }
  });
  doc.addEventListener('keydown', (e) => {
    if (e.key === ' ' && e.target === doc.body) { e.preventDefault(); on.toggle(); }
  });
  doc.defaultView.addEventListener('resize', relayout);
  relayout();
  return {
    update() {
      if (track.clientWidth !== width) relayout();
      play.textContent = player.playing ? 'Pause' : 'Play';
      head.style.left = `${place.xAt(player.t)}px`;
      track.setAttribute('aria-valuenow', String(Math.round(player.t)));
      time.textContent = `${clock(player.t)} / ${clock(player.end)}`;
    },
  };
}

// drawMarks draws the notes open on the current document at their element
// (its box as drawn, else the box the note carried) and the strokes being
// drawn at the playhead, into layer over the scaled page. fromPage maps a
// top-page point to the layer's px. #470's live notes and pointer UI is to
// replace this once both have landed.
export function drawMarks(doc, layer, player, fromPage) {
  layer.replaceChildren();
  const m = player.model, t = player.t;
  if (!m || !(m.seg > 0)) return;
  for (const n of m.notes) {
    if (n.seg < m.docSeg || n.state !== 'open') continue;
    const at = n.note.at || {};
    const box = (at.node && player.renderer.boxOf(at.node)) || at.box || n.note.region;
    if (!box) continue;
    const [x, y] = fromPage(box[0], box[1]);
    const pin = doc.createElement('div');
    pin.className = 'replay-note';
    pin.style.setProperty('--c', safeColor(n.note.author && n.note.author.color, '#ffb000'));
    pin.style.transform = `translate(${x}px, ${y}px)`;
    pin.textContent = n.note.text || '';
    pin.title = `${(n.note.author && n.note.author.name) || ''}: ${n.note.text || ''}`;
    layer.appendChild(pin);
  }
  const NS = 'http://www.w3.org/2000/svg';
  const svg = doc.createElementNS(NS, 'svg');
  svg.setAttribute('class', 'replay-strokes');
  for (const s of m.strokes) {
    if (s.seg < m.docSeg) continue;
    const r = s.stroke, since = t - r.t, dur = r.pts.length ? r.pts[r.pts.length - 1][2] || 0 : 0;
    if (since < 0 || since > dur + STROKE_FADE) continue;
    const pts = r.pts.filter((p) => (p[2] || 0) <= since).map((p) => fromPage(p[0], p[1]).join(',')).join(' ');
    if (!pts) continue;
    const line = doc.createElementNS(NS, 'polyline');
    line.setAttribute('points', pts);
    line.setAttribute('stroke', safeColor(r.color || (r.author && r.author.color), '#ff3b30'));
    line.setAttribute('opacity', String(since > dur ? 1 - (since - dur) / STROKE_FADE : 1));
    svg.appendChild(line);
  }
  layer.appendChild(svg);
}

// mountReplay is the shell's replay branch: the source, its bar in
// #timeline, its marks in #overlay, and #t= kept in the location. deps are
// the shell's (renderer, show, frame, schedule); fromPage maps a top-page
// point to the overlay's px at the current scale and scroll.
export function mountReplay(doc, win, deps, fromPage) {
  const layer = doc.createElement('div');
  layer.className = 'replay-layer';
  doc.getElementById('overlay').appendChild(layer);
  const v = replaySource({
    ...deps,
    fetchRecords: async () => {
      const res = await win.fetch('records');
      if (!res.ok) throw new Error(`records: ${res.status}`);
      return (await res.text()).split('\n').filter(Boolean).map((l) => JSON.parse(l));
    },
    frame: () => { deps.frame(); if (v.player) drawMarks(doc, layer, v.player, fromPage); },
    now: () => win.performance.now(),
    hash: () => win.location.hash,
    setHash: (h) => { if (win.location.hash !== h) win.history.replaceState(null, '', h); },
    ui: (player, on) => timelineBar(doc, doc.getElementById('timeline'), player, on),
  });
  win.addEventListener('hashchange', v.onHash);
  return v;
}
