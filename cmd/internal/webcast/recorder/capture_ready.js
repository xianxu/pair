function captureReady(box, inspect = true) {
  // A synchronous observation only: the Go owner owns polling and deadlines.
  // Enumerate each accessible animation root once: document.getAnimations does
  // not include open-shadow animations in Chrome. Lists stay local to the page.
  const result = {viewport: [scrollX, scrollY, innerWidth, innerHeight],
    dpr: devicePixelRatio, pending: false, reason: ''};
  if (!inspect) return result;
  const fail = reason => {result.pending = true; result.reason = reason; return result;};
  const left = Math.max(0, box[0] - scrollX), top = Math.max(0, box[1] - scrollY);
  const right = Math.min(innerWidth, box[0] + box[2] - scrollX);
  const bottom = Math.min(innerHeight, box[1] + box[3] - scrollY);
  if (right <= left || bottom <= top) return result;
  let visited = 0;
  try {
    const animations = [], roots = new Set();
    function enumerate(root) {
      if (roots.has(root)) return '';
      const list = root.getAnimations();
      if (!Array.isArray(list)) return 'inspection unavailable';
      if (list.length > 512 - animations.length) return 'inspection limit';
      roots.add(root);
      for (const animation of list) animations.push(animation);
      return '';
    }
    let reason = enumerate(document);
    if (reason) return fail(reason);
    if (!document.documentElement) return fail('inspection unavailable');
    // Cursor DFS visits light DOM and open shadow roots once; no selector query,
    // child-array copy, per-animation DOM scan, or serialized animation list.
    const pending = [document.documentElement];
    while (pending.length) {
      if (++visited > 20000) return fail('inspection limit');
      const node = pending.pop();
      if (node.shadowRoot) {
        if (node.shadowRoot.mode !== 'open') return fail('inspection unavailable');
        reason = enumerate(node.shadowRoot);
        if (reason) return fail(reason);
        pending.push(node.shadowRoot);
      }
      if (node.nextSibling) pending.push(node.nextSibling);
      if (node.firstChild) pending.push(node.firstChild);
      if (visited + pending.length > 20000) return fail('inspection limit');
    }
    for (const animation of animations) {
      const state = animation.playState;
      if (state === 'finished' || state === 'idle') continue;
      if (state !== 'running' && state !== 'paused') return fail('inspection unavailable');
      const effect = animation.effect;
      if (!effect) return fail('inspection unavailable');
      // CSS animations/transitions expose their host as effect.target, while
      // implementations exposing CSSPseudoElement objects supply .element.
      let target = effect.target;
      const pseudo = effect.pseudoElement || (target && target.nodeType !== 1);
      if (target && target.nodeType !== 1) target = target.element;
      if (!target || target.nodeType !== 1) return fail('inspection unavailable');
      if (target.isConnected === false) continue;
      const rect = target.getBoundingClientRect();
      if (![rect.left, rect.top, rect.right, rect.bottom, rect.width, rect.height].every(Number.isFinite)) return fail('inspection unavailable');
      if (rect.width <= 0 || rect.height <= 0 || rect.right <= left || rect.left >= right || rect.bottom <= top || rect.top >= bottom) {
        // Target bounds do not bound inherited opacity/transform on overflowing
        // descendants or generated content. Include these conservatively rather
        // than walking a subtree per animation (or claiming false settlement).
        let mayOverflow = pseudo || target.firstChild || target.shadowRoot;
        if (!mayOverflow) {
          for (const kind of ['::before', '::after']) {
            const content = getComputedStyle(target, kind).content;
            if (typeof content !== 'string') return fail('inspection unavailable');
            if (content !== 'none' && content !== 'normal') mayOverflow = true;
          }
        }
        if (!mayOverflow) continue;
      }
      // An animated ancestor's enclosing box affects crops inside it; target
      // ownership must belong to a document/open shadow root observed above.
      if (!roots.has(target.getRootNode())) return fail('inspection unavailable');
      const timing = effect.getComputedTiming();
      if (!timing || typeof timing.endTime !== 'number' || Number.isNaN(timing.endTime) || typeof timing.iterations !== 'number' || Number.isNaN(timing.iterations)) return fail('inspection unavailable');
      if (state === 'paused' || animation.playbackRate === 0) return fail('paused animation');
      if (timing.endTime === Infinity || timing.iterations === Infinity) return fail('infinite animation');
      if (!Number.isFinite(timing.endTime) || !Number.isFinite(timing.iterations)) return fail('inspection unavailable');
      result.pending = true;
    }
  } catch (_) {
    return fail('inspection unavailable');
  }
  return result;
}
