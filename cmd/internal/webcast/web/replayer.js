// Replayer draws a recording's records into the page frame with the
// viewer's model and renderer, unscaled at the operator's viewport: the
// fidelity check's replay (#478) and the conformance harness's (#460).
// A replayAt after what is drawn continues from it: the animations paused
// at the last capture resume, so a moment shortly after is at its phase.
// One before what is drawn seeks: the catch-up of the records up to the
// window's start (the format's seek rule), so it replays one segment.
//
// replayAt(t, windowMs) draws the page as the real tab looked at t:
// records up to t - windowMs at once (their transitions finished), then the
// last window in real time, so a transition or animation running at t is
// at the phase it had in the tab; animations are paused at t, and the
// assets bound by t finish loading (#482), for a capture.
// A document a segment snapshots mid-cast (go-live, resume, a tab switch)
// was already running, so its animations are finished, not started.
import { Model, goJSON } from './model.js';
import { Renderer, currentFrame, blankFrames, HOLD_MS } from './render.js';
import { catchUp } from './replay.js';

// inBrowser is what a Replayer needs of a page that holds the frame.
// The renderer swaps the page frame with a sibling while it holds a paint
// (#475), so the frame drawn into is looked up each time.
export function inBrowser(frame) {
  const docs = () => {
    const top = currentFrame(frame).contentDocument;
    if (!top) return [];
    return [top, ...[...top.querySelectorAll('iframe')].map((f) => f.contentDocument).filter(Boolean)];
  };
  return {
    renderer: (model, resolve) => new Renderer(currentFrame(frame), model, resolve),
    blank: () => blankFrames(frame),
    now: () => performance.now(),
    sleep: (ms) => new Promise((ok) => setTimeout(ok, Math.max(0, ms))),
    paint: () => new Promise((ok) => requestAnimationFrame(() => requestAnimationFrame(ok))),
    animations: () => docs().flatMap((d) => d.getAnimations()),
    layout: (vp) => {
      const f = currentFrame(frame);
      f.style.width = `${vp.w}px`;
      f.style.height = `${vp.h}px`;
    },
  };
}

export class Replayer {
  // recs are the records in order; assetURL(hash) is where an asset's
  // body loads from; deps is inBrowser(frame), or a fake.
  constructor(recs, assetURL, deps) {
    if (recs.some(r => r.k === 'sketch-snapshot')) throw new Error('snapshot bootstrap in raw replay source');
    this.recs = recs;
    this.assetURL = assetURL;
    this.deps = deps;
    this.reset();
  }

  // reset starts a fresh viewer, which holds no previous paint.
  reset() {
    if (this.deps.blank) this.deps.blank();
    this.next = 0;
    this.at = -Infinity; // the time drawn so far
    this.model = new Model();
    this.renderer = this.deps.renderer(this.model, (abs) => {
      const a = this.model.assets.get(abs);
      return a ? this.assetURL(a.hash) : '';
    });
  }

  // seek draws the state at t afresh from the catch-up of the records up
  // to t, and goes on from the first record after it.
  seek(t) {
    let lo = 0, hi = this.recs.length; // the first record after t
    while (lo < hi) {
      const mid = (lo + hi) >> 1;
      if (this.recs[mid].t <= t) lo = mid + 1; else hi = mid;
    }
    this.reset();
    this.model.allowSketchSnapshot = true;
    try {
      for (const r of catchUp(this.recs.slice(0, lo))) this.feed(r);
    } finally {
      this.model.allowSketchSnapshot = false;
    }
    this.next = lo;
    this.at = t;
  }

  feed(r) {
    this.model.apply(r);
    this.renderer.apply(r);
    if ((r.k === 'seg' || r.k === 'vp') && this.model.vp.w > 0) this.deps.layout(this.model.vp);
  }

  // settle waits for the drawn page's stylesheets, fonts and images, and a
  // paint, and returns its canonical state.
  async settle() {
    this.renderer.flush();
    await this.renderer.ready();
    this.renderer.flush();
    await this.deps.paint();
    return goJSON(this.renderer.canonical());
  }

  // replayTo applies every record up to t at once.
  async replayTo(t) {
    while (this.next < this.recs.length && this.recs[this.next].t <= t) this.feed(this.recs[this.next++]);
    this.at = Math.max(this.at, t);
    return this.settle();
  }

  // releaseAt is when a viewer that drew the recording up to t shows that
  // moment's document (#475): t itself, or when the paint held over t is
  // released, its stylesheets bound or HOLD_MS after its segment began. It
  // draws from the start, applying records past t until then.
  async releaseAt(t) {
    this.reset();
    let began = 0;
    const step = async () => {
      const r = this.recs[this.next++];
      if (r.k === 'seg' || r.k === 'head') began = r.t;
      this.feed(r);
      this.renderer.flush();
      if (this.renderer.holding) await this.renderer.bound();
      return r.t;
    };
    while (this.next < this.recs.length && this.recs[this.next].t <= t) await step();
    let at = t;
    while (this.renderer.holding && this.next < this.recs.length && this.recs[this.next].t < began + HOLD_MS) at = await step();
    this.at = at;
    return this.renderer.holding ? Math.max(at, began + HOLD_MS) : at;
  }

  finishAnimations() {
    for (const a of this.deps.animations()) {
      try { a.finish(); } catch { /* an infinite animation has no end */ }
    }
  }

  // replayAt: see the file's comment. It returns what was drawn: whether
  // the cast was live at t, the viewport, and the canonical state.
  async replayAt(t, windowMs) {
    let start = t - windowMs;
    if (start <= this.at && t >= this.at) {
      // Continuing: the window starts at what is drawn, whose animations
      // were paused there and run on from their phase.
      start = this.at;
      await this.settle();
      for (const a of this.deps.animations()) if (a.playState === 'paused') a.play();
    } else {
      if (t < this.at) this.seek(start);
      while (this.next < this.recs.length && this.recs[this.next].t <= start) this.feed(this.recs[this.next++]);
      await this.settle();
      this.finishAnimations();
    }
    const t0 = this.deps.now();
    const until = (rt) => this.deps.sleep(rt - start - (this.deps.now() - t0));
    while (this.next < this.recs.length && this.recs[this.next].t <= t) {
      const rt = this.recs[this.next].t;
      await until(rt);
      let running = false;
      while (this.next < this.recs.length && this.recs[this.next].t === rt) {
        const r = this.recs[this.next++];
        this.feed(r);
        running = running || (r.k === 'seg' && r.why !== 'navigate');
      }
      this.renderer.flush();
      // A segment that isn't a navigation snapshots a document that was
      // already running: its animations have run in the tab (their phase
      // isn't recorded), so they are taken as finished, as before the window.
      if (running) this.finishAnimations();
    }
    await until(t);
    // A held paint (#475) is released as soon as the stylesheets whose
    // bytes have arrived apply; the viewer then redraws (onDirty).
    if (this.renderer.holding) { await this.renderer.bound(); this.renderer.flush(); }
    this.at = Math.max(this.at, t);
    // The tab painted t with the assets whose bytes it had (#482): a font
    // or image bound just before t is still loading in the viewer, so the
    // clocks stop at t, the loads finish (ready's bound), and anything the
    // loads started is paused too, a frame or two after its start.
    for (const a of this.deps.animations()) a.pause();
    const canonical = await this.settle();
    for (const a of this.deps.animations()) if (a.playState !== 'paused') a.pause();
    const m = this.model;
    return { live: m.seg > 0 && m.live, vp: { w: m.vp.w, h: m.vp.h }, canonical };
  }
}
