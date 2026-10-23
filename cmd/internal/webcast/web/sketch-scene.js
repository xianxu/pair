import { Model } from './model.js';
import { Renderer, currentFrame, blankFrames } from './render.js';

const PREFIX_BYTES = 16 * 1024 * 1024,
  CHUNK_BYTES = 32 * 1024;
const need = (ok, why) => {
  if (!ok) throw Error(`sketch scene: ${why}`);
};
const point = (p) =>
  Array.isArray(p) && p.length === 2 && p.every(Number.isFinite) && p.every((n) => Math.abs(n) <= 1e7);

// Viewer adapter for an owned held scene. The console injects the same narrow
// scene interface through its Chrome-owned opaque child session instead.
export function createSketchScene({
  frame,
  container,
  document: doc = globalThis.document,
  readSource,
  assetURL = () => '',
  prepareAssets,
  rendererFactory = (f, m, resolve) => new Renderer(f, m, resolve),
  modelFactory = () => new Model({ allowSketchSnapshot: true }),
} = {}) {
  let host = null;
  if (!frame) {
    need(doc && container, 'missing scene frame');
    host = doc.createElement('div');
    host.className = 'sketch-held';
    frame = doc.createElement('iframe');
    frame.setAttribute('sandbox', 'allow-same-origin');
    frame.setAttribute('title', 'Sketching this moment');
    frame.tabIndex = -1;
    host.appendChild(frame);
    container.appendChild(host);
    host.hidden = true;
  }
  let generation = 0,
    ready = false,
    model = null,
    renderer = null,
    source = null,
    controller = null,
    assets = null,
    scale = 1;
  const stats = { reads: 0, bytes: 0, records: 0, preparations: 0 };
  function layout() {
    if (!model) return;
    const actual = renderer?.frame || currentFrame(frame),
      vp = model.vp;
    if (!(vp.w > 0 && vp.h > 0)) return;
    const area = container?.getBoundingClientRect?.();
    if (area && area.width > 0 && area.height > 0) {
      scale = Math.min(area.width / vp.w, area.height / vp.h);
      Object.assign(actual.style, {
        width: `${vp.w}px`,
        height: `${vp.h}px`,
        transformOrigin: '0 0',
        transform: `scale(${scale})`,
        left: `${(area.width - vp.w * scale) / 2}px`,
        top: `${(area.height - vp.h * scale) / 2}px`,
      });
    } else {
      const rect = actual.getBoundingClientRect?.();
      scale = rect?.width > 0 ? rect.width / vp.w : 1;
    }
    actual.style.pointerEvents = 'none';
  }
  function dispose() {
    generation++;
    controller?.abort();
    controller = null;
    ready = false;
    renderer?.revokeAll();
    renderer = null;
    assets?.dispose?.();
    assets = null;
    model = null;
    source = null;
    try {
      blankFrames(frame);
    } catch {}
    if (host) host.hidden = true;
  }
  function coordinates() {
    need(ready, 'scene is not ready');
    layout();
    const actual = renderer.frame || currentFrame(frame);
    return {
      rect: actual.getBoundingClientRect(),
      scroll: actual.contentWindow || { scrollX: 0, scrollY: 0 },
    };
  }
  return {
    get ready() {
      return ready;
    },
    get source() {
      return source ? structuredClone(source) : null;
    },
    async prepare(descriptor, context = {}) {
      dispose();
      const token = generation;
      controller = new AbortController();
      const callContext = { ...context, source: structuredClone(descriptor), signal: controller.signal };
      need(
        descriptor &&
          Number.isSafeInteger(descriptor.length) &&
          descriptor.length > 0 &&
          descriptor.length <= PREFIX_BYTES &&
          typeof descriptor.handle === 'string' &&
          descriptor.handle,
        'invalid source descriptor',
      );
      const bytes = new Uint8Array(descriptor.length);
      let offset = 0;
      stats.preparations++;
      while (offset < bytes.length) {
        const size = Math.min(CHUNK_BYTES, bytes.length - offset);
        const chunk = await readSource(descriptor, offset, size, callContext);
        need(token === generation, 'source retired');
        need(chunk instanceof Uint8Array && chunk.length === size, 'invalid source chunk');
        bytes.set(chunk, offset);
        offset += chunk.length;
        stats.reads++;
        stats.bytes += chunk.length;
      }
      const next = modelFactory();
      const records = new TextDecoder('utf-8', { fatal: true })
        .decode(bytes)
        .split('\n')
        .filter(Boolean)
        .map((line) => JSON.parse(line));
      for (const record of records) {
        next.apply(record);
        stats.records++;
      }
      const last = records.at(-1);
      need(
        last?.k === 'sketch-begin' &&
          last.t === descriptor.time &&
          next.t === descriptor.time &&
          next.docSeg === descriptor.doc &&
          next.tab.id === descriptor.tab &&
          ['w', 'h', 'dpr'].every(
            (k) => next.vp[k] === descriptor.vp?.[k] && last.vp[k] === descriptor.vp[k],
          ) &&
          (!context.sketch || last.id === context.sketch),
        'source identity differs',
      );
      let prepared = null;
      if (prepareAssets) {
        prepared = await prepareAssets(
          Array.from(next.assets.values(), (v) => structuredClone(v)),
          callContext,
        );
        if (token !== generation) {
          prepared?.dispose?.();
          throw Error('sketch scene: source retired');
        }
      }
      need(token === generation, 'source retired');
      model = next;
      source = structuredClone(descriptor);
      assets = prepared;
      const resolve = (url) => {
        const binding = next.assets.get(url);
        if (!binding) return '';
        return (
          prepared?.url?.(binding.hash, binding.mime) ||
          assetURL(binding.hash, binding.mime, callContext) ||
          ''
        );
      };
      try {
        // Final-prefix validation/asset admission stays separate from the
        // state each historical mutation needs while rebuilding the DOM.
        const renderModel = modelFactory();
        renderer = rendererFactory(currentFrame(frame), renderModel, resolve);
        if (host) host.hidden = false;
        for (const record of records) {
          renderModel.apply(record);
          renderer.apply(record);
          if (record.k === 'seg') layout();
        }
        renderer.flush();
        await renderer.ready();
        need(token === generation, 'source retired');
        renderer.flush();
        layout();
        const roots = new Set([
          renderer.topDoc,
          ...Array.from(renderer.dom?.values() || []).filter((n) => n.nodeType === 9 || n.nodeType === 11),
        ]);
        for (const root of roots)
          for (const animation of root?.getAnimations?.() || []) {
            try {
              animation.finish();
            } catch {}
            animation.pause();
          }
        ready = true;
        return { source: structuredClone(source), vp: structuredClone(next.vp) };
      } catch (error) {
        if (token === generation) dispose();
        throw error;
      }
    },
    async hitTest(at) {
      need(ready && point(at), 'invalid hit test');
      const node = renderer.nodeAt(at[0], at[1]);
      const box = node && renderer.boxOf(node);
      need(
        node > 0 && box && box.length === 4 && box.every(Number.isFinite) && box[2] > 0 && box[3] > 0,
        'No recorded element here.',
      );
      return { node, box: [...box] };
    },
    async placement(path) {
      need(ready, 'scene is not ready');
      if (path.source?.doc !== model.docSeg || path.source?.tab !== model.tab.id || path.source.t > model.t)
        return null;
      return renderer.placementOf(path);
    },
    toPage(x, y) {
      const { rect, scroll } = coordinates();
      return [
        (x - rect.left) / scale + (scroll.scrollX || 0),
        (y - rect.top) / scale + (scroll.scrollY || 0),
      ];
    },
    fromPage(x, y) {
      const { rect, scroll } = coordinates();
      return [
        rect.left + (x - (scroll.scrollX || 0)) * scale,
        rect.top + (y - (scroll.scrollY || 0)) * scale,
      ];
    },
    snapshot() {
      need(ready, 'scene is not ready');
      return structuredClone(model.canonical());
    },
    resize: layout,
    dispose,
    stats: () => ({ ...stats }),
  };
}
