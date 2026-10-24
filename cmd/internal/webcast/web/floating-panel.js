// Local presentation state only: no note data, persistence, or feedback authority.
const finite = (value, fallback = 0) => Number.isFinite(value) ? value : fallback;
const dimensions = value => ({width: Math.max(0, finite(value?.width)), height: Math.max(0, finite(value?.height)), ...(Number.isFinite(value?.left) ? {left:value.left,top:finite(value.top)} : {})});
function bounded(state) {
  const {viewport, size} = state;
  const custom = Number.isFinite(viewport.left);
  const mx = custom ? 0 : Math.min(12, viewport.width / 2), my = custom ? 0 : Math.min(12, viewport.height / 2);
  const x = viewport.left || 0, y = viewport.top || 0;
  return {...state,
    left: Math.min(Math.max(x + mx, finite(state.left)), Math.max(x + mx, x + viewport.width - mx - size.width)),
    top: Math.min(Math.max(y + my, finite(state.top)), Math.max(y + my, y + viewport.height - my - size.height))};
}

// Events carry measured geometry; the reducer never reads the DOM.
export function floatingPanelTransition(previous, event) {
  if (previous?.disposed) return previous;
  let state = previous || {mode: 'expanded', left: finite(event.left), top: finite(event.top),
    viewport: dimensions(event.viewport), size: dimensions(event.size), gesture: null, suppressClick: false, disposed: false};
  const gesture = state.gesture;
  switch (event.type) {
    case 'init': break;
    case 'geometry':
    case 'mode':
      state = {...state, ...(gesture ? {left: gesture.left, top: gesture.top} : {}), gesture: null,
        viewport: event.viewport ? dimensions(event.viewport) : state.viewport,
        size: event.size ? dimensions(event.size) : state.size,
        mode: event.type === 'mode' && event.mode === 'minimized' ? 'minimized' : event.type === 'mode' ? 'expanded' : state.mode};
      break;
    case 'press':
      if (!gesture && Number.isFinite(event.x) && Number.isFinite(event.y)) state = {...state, suppressClick: false,
        gesture: {phase: 'pressed', id: event.id, x: event.x, y: event.y, left: state.left, top: state.top}};
      break;
    case 'move':
      if (gesture && gesture.id === event.id && Number.isFinite(event.x) && Number.isFinite(event.y)) {
        const dx = event.x - gesture.x, dy = event.y - gesture.y;
        if (gesture.phase === 'dragging' || Math.hypot(dx, dy) >= 4) state = {...state,
          left: gesture.left + dx, top: gesture.top + dy, gesture: {...gesture, phase: 'dragging'}};
      }
      break;
    case 'end':
      if (gesture?.id === event.id) state = {...state, gesture: null, suppressClick: gesture.phase === 'dragging'};
      break;
    case 'cancel':
      if (gesture) state = {...state, left: gesture.left, top: gesture.top, gesture: null, suppressClick: true};
      break;
    case 'click': state = {...state, suppressClick: false}; break;
    case 'keyboard':
      if (!gesture) state = {...state, left: state.left + finite(event.dx), top: state.top + finite(event.dy)};
      break;
    case 'dispose': state = {...state, gesture: null, disposed: true}; break;
  }
  return bounded(state);
}

export function mountFloatingPanel(doc, win, {parent, body, title = 'Notes', className = 'annotation-notes', initialTop = 110, initialRight = 12, geometry, showEmpty = false, onPresentation = () => {}}) {
  const element = (tag, cls, text) => {
    const node = doc.createElement(tag); node.className = cls;
    if (text !== undefined) node.textContent = text;
    return node;
  };
  const root = element('section', `floating-notes ${className}`);
  const header = element('div', 'floating-panel-header');
  const handle = element('button', 'floating-panel-drag', title);
  const minimize = element('button', 'floating-panel-minimize', 'Minimize');
  const restore = element('button', 'floating-panel-restore');
  for (const button of [handle, minimize, restore]) button.type = 'button';
  handle.setAttribute('aria-label', `Move ${title}; use arrow keys`);
  minimize.setAttribute('aria-label', `Minimize ${title}`);
  root.setAttribute('aria-label', title);
  body ||= element('div', '');
  body.className = `${body.className || ''} floating-panel-body`.trim();
  header.append(handle, minimize); root.append(header, body, restore); parent.append(root);
  let customGeometry = geometry?.('expanded');
  const viewport = () => customGeometry?.viewport || ({width: finite(win.innerWidth), height: finite(win.innerHeight)});
  const measure = () => {
    if (customGeometry?.size) return customGeometry.size;
    const rect = root.getBoundingClientRect();
    const width = rect.width || Math.min(300, Math.max(0, win.innerWidth - 24));
    const height = rect.height || Math.min(320, Math.max(0, win.innerHeight - 24));
    return customGeometry ? {width:Math.min(width, customGeometry.viewport.width),height:Math.min(height, customGeometry.viewport.height)} : {width,height};
  };
  let state = floatingPanelTransition(null, {type: 'init', viewport: viewport(), size: measure(),
    left: customGeometry?.left ?? finite(win.innerWidth) - finite(initialRight, 12) - Math.min(300, Math.max(0, finite(win.innerWidth) - 24)), top: customGeometry?.top ?? initialTop});
  let count = 0, statusText = '', captureTarget = null;
  const removers = [];
  const listen = (node, type, fn) => {
    node.addEventListener(type, fn);
    removers.push(() => node.removeEventListener?.(type, fn));
  };
  const anchor = previous => {
    if (!previous || previous.left !== state.left || previous.top !== state.top) {
      root.style.inset = `${state.top}px auto auto ${state.left}px`;
    }
  };
  const send = event => { const previous = state; state = floatingPanelTransition(state, event); anchor(previous); };
  const release = id => {
    const target = captureTarget; captureTarget = null;
    if (id !== undefined && target?.hasPointerCapture?.(id)) target.releasePointerCapture(id);
  };
  const cancel = () => { const id = state.gesture?.id; send({type: 'cancel'}); release(id); };
  const presentation = () => {
    const minimized = state.mode === 'minimized';
    root.hidden = (!showEmpty && count === 0) || state.viewport.width === 0 || state.viewport.height === 0;
    if (customGeometry) {
      root.style.width = minimized ? 'auto' : `${state.size.width}px`;
      root.style.height = minimized ? 'auto' : `${state.size.height}px`;
      root.style.maxWidth = `${state.viewport.width}px`;
      root.style.maxHeight = `${state.viewport.height}px`;
    }
    root.classList.toggle('minimized', minimized);
    header.hidden = minimized; body.hidden = minimized; restore.hidden = !minimized;
    handle.textContent = `${title} (${count})`;
    restore.textContent = `${title} (${count})${statusText ? ' · Status' : ''}`;
    restore.title = statusText;
    restore.setAttribute('aria-label', `${title} (${count})${statusText ? ': ' + statusText : ''}`);
    onPresentation({minimized, hidden:root.hidden});
  };
  const refresh = () => {
    if (state.disposed) return;
    cancel(); customGeometry = geometry?.(state.mode);
    if (geometry && !customGeometry) {
      for (const property of ['width','height','maxWidth','maxHeight']) root.style[property] = '';
    }
    send({type:'geometry', size:measure(), viewport:viewport()}); presentation();
  };
  const mode = value => {
    cancel();
    // The reducer owns the mode; render that state before measuring its form.
    send({type: 'mode', mode: value}); presentation();
    if (geometry) refresh();
    else send({type: 'geometry', size: measure(), viewport: viewport()});
  };
  const stop = event => { event.preventDefault(); event.stopPropagation(); };
  const press = event => {
    if (state.disposed || root.hidden) return;
    event.stopPropagation();
    if (state.gesture || event.button !== 0 || event.isPrimary === false || event.target === minimize) return;
    send({type: 'geometry', size: measure(), viewport: viewport()});
    send({type: 'press', id: event.pointerId, x: event.clientX, y: event.clientY});
    if (!state.gesture) return;
    captureTarget = state.mode === 'minimized' ? restore : header;
    captureTarget.setPointerCapture(event.pointerId);
    // Keep native button focus/click behavior; prevent page gesture handling.
    event.stopPropagation();
  };
  listen(header, 'pointerdown', press); listen(restore, 'pointerdown', press);
  listen(header, 'click', event => event.stopPropagation());
  listen(root, 'pointermove', event => {
    if (!state.gesture || state.gesture.id !== event.pointerId) return;
    send({type: 'move', id: event.pointerId, x: event.clientX, y: event.clientY}); stop(event);
  });
  listen(root, 'pointerup', event => {
    if (!state.gesture || state.gesture.id !== event.pointerId) return;
    send({type: 'end', id: event.pointerId}); release(event.pointerId); event.stopPropagation();
  });
  for (const type of ['pointercancel', 'lostpointercapture']) listen(root, type, event => {
    if (state.gesture?.id !== event.pointerId) return;
    cancel(); event.stopPropagation();
  });
  listen(minimize, 'click', event => { stop(event); if (!state.disposed) mode('minimized'); });
  listen(restore, 'click', event => {
    stop(event); if (state.disposed) return;
    if (!state.suppressClick || event.detail === 0) mode('expanded');
    send({type: 'click'});
  });
  const keyboard = event => {
    const delta = {ArrowLeft: [-1,0], ArrowRight: [1,0], ArrowUp: [0,-1], ArrowDown: [0,1]}[event.key];
    if (state.disposed || !delta) return;
    stop(event); const step = event.shiftKey ? 40 : 10;
    send({type: 'keyboard', dx: delta[0] * step, dy: delta[1] * step});
  };
  listen(handle, 'keydown', keyboard); listen(restore, 'keydown', keyboard);
  listen(win, 'resize', refresh);
  presentation(); anchor();
  return {root, body, header, refresh,
    expand() { if (!state.disposed && state.mode !== 'expanded') mode('expanded'); },
    setStatus(value) {
      if (state.disposed || statusText === String(value || '')) return;
      cancel(); statusText = String(value || ''); presentation();
      if (state.mode === 'minimized') send({type:'geometry', size:measure(), viewport:viewport()});
    },
    setCount(value) {
      if (state.disposed) return;
      const nextCount = Math.max(0, Math.floor(finite(value)));
      if (nextCount === count) return;
      count = nextCount;
      // The minimized button has intrinsic width. Retire any tentative drag
      // before changing its text so rollback and clamping use one owned anchor.
      const remeasure = count > 0 && state.mode === 'minimized';
      if (!count || remeasure) cancel();
      root.hidden = count === 0; presentation();
      if (remeasure) send({type: 'geometry', size: measure(), viewport: viewport()});
    },
    dispose() {
      if (state.disposed) return;
      const id = state.gesture?.id;
      state = floatingPanelTransition(state, {type: 'dispose'});
      for (const remove of removers) remove();
      release(id); root.remove?.();
    }};
}
