function captureReady(box, inspect = true) {
  // A synchronous observation only: the Go owner owns polling and deadlines.
  // Document.getAnimations includes CSS transitions, pseudo-elements and shadow
  // descendants; keep the browser's list local and never serialize it over CDP.
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
    const animations = document.getAnimations();
    if (!Array.isArray(animations)) return fail('inspection unavailable');
    if (animations.length > 512) return fail('inspection limit');
    for (const animation of animations) {
      const state = animation.playState;
      if (state === 'finished' || state === 'idle') continue;
      if (state !== 'running' && state !== 'paused') return fail('inspection unavailable');
      const effect = animation.effect;
      if (!effect) return fail('inspection unavailable');
      // CSS animations/transitions expose their host as effect.target, while
      // implementations exposing CSSPseudoElement objects supply .element.
      let target = effect.target;
      if (target && target.nodeType !== 1) target = target.element;
      if (!target || target.nodeType !== 1) return fail('inspection unavailable');
      if (target.isConnected === false) continue;
      const rect = target.getBoundingClientRect();
      if (![rect.left, rect.top, rect.right, rect.bottom, rect.width, rect.height].every(Number.isFinite)) return fail('inspection unavailable');
      if (rect.width <= 0 || rect.height <= 0 || rect.right <= left || rect.left >= right || rect.bottom <= top || rect.top >= bottom) continue;
      // A target ancestor's enclosing box affects every crop within it. Follow
      // only its shadow-root ownership chain to enforce the declared scope.
      let node = target;
      while (node) {
        if (++visited > 20000) return fail('inspection limit');
        const root = node.getRootNode();
        if (root === document) break;
        if (!root || root.mode !== 'open' || !root.host) return fail('inspection unavailable');
        node = root.host;
      }
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
