// Replayer draws a recording's records into the page frame with the
// viewer's model and renderer, unscaled at the operator's viewport: the
// fidelity check's replay (#478) and the conformance harness's (#460).
// It moves forward; a replayAt before what is drawn starts over.
//
// replayAt(t, windowMs) draws the page as the real tab looked at t:
// records up to t - windowMs at once (their transitions finished), then the
// last window in real time, so a transition or animation running at t is
// at the phase it had in the tab; animations are paused at t for a capture.
// A document a segment snapshots mid-cast (go-live, resume, a tab switch)
// was already running, so its animations are finished, not started.
import { Model, goJSON } from './model.js';
import { Renderer } from './render.js';

// inBrowser is what a Replayer needs of a page that holds the frame.
export function inBrowser(frame) {
  const docs = () => {
    const top = frame.contentDocument;
    if (!top) return [];
    return [top, ...[...top.querySelectorAll('iframe')].map((f) => f.contentDocument).filter(Boolean)];
  };
  return {
    renderer: (model, resolve) => new Renderer(frame, model, resolve),
    now: () => performance.now(),
    sleep: (ms) => new Promise((ok) => setTimeout(ok, Math.max(0, ms))),
    paint: () => new Promise((ok) => requestAnimationFrame(() => requestAnimationFrame(ok))),
    animations: () => docs().flatMap((d) => d.getAnimations()),
    layout: (vp) => {
      frame.style.width = `${vp.w}px`;
      frame.style.height = `${vp.h}px`;
    },
  };
}

export class Replayer {
  // recs are the records in order; assetURL(hash) is where an asset's
  // body loads from; deps is inBrowser(frame), or a fake.
  constructor(recs, assetURL, deps) {
    this.recs = recs;
    this.assetURL = assetURL;
    this.deps = deps;
    this.reset();
  }

  reset() {
    this.next = 0;
    this.at = -Infinity; // the time drawn so far
    this.model = new Model();
    this.renderer = this.deps.renderer(this.model, (abs) => {
      const a = this.model.assets.get(abs);
      return a ? this.assetURL(a.hash) : '';
    });
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

  finishAnimations() {
    for (const a of this.deps.animations()) {
      try { a.finish(); } catch { /* an infinite animation has no end */ }
    }
  }

  // replayAt: see the file's comment. It returns what was drawn: whether
  // the cast was live at t, the viewport, and the canonical state.
  async replayAt(t, windowMs) {
    if (t < this.at) this.reset(); // backward: from the start
    // The window starts no earlier than what is drawn already.
    const start = Math.max(t - windowMs, this.at);
    while (this.next < this.recs.length && this.recs[this.next].t <= start) this.feed(this.recs[this.next++]);
    await this.settle();
    this.finishAnimations();
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
    this.at = Math.max(this.at, t);
    for (const a of this.deps.animations()) a.pause();
    const m = this.model;
    return { live: m.seg > 0 && m.live, vp: { w: m.vp.w, h: m.vp.h }, canonical: goJSON(this.renderer.canonical()) };
  }
}
