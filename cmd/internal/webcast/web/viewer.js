// The web cast's stub viewer (#454): the cast's live state and the operator's
// page title and URL. Title and URL come from whatever app is cast, so they
// are only ever set as text.

// view maps a state event (and whether the cast ended) to what is shown.
export function view(state, ended) {
  if (ended) return { status: 'Cast ended', live: false, title: '', url: '' };
  if (!state) return { status: 'Connecting…', live: false, title: '', url: '' };
  if (!state.live) return { status: 'Not live', live: false, title: '', url: '' };
  return { status: 'Live', live: true, title: state.title || state.url || '', url: state.url || '' };
}

// connect follows the cast's event stream; deps.open returns an EventSource,
// deps.show renders a view.
export function connect(deps) {
  const es = deps.open();
  es.addEventListener('state', (e) => deps.show(view(JSON.parse(e.data), false)));
  es.addEventListener('end', () => { es.close(); deps.show(view(null, true)); });
  es.onerror = () => {
    if (es.readyState === 2) deps.show({ ...view(null, true), status: 'Disconnected' });
    else deps.show(view(null, false)); // EventSource reconnects by itself
  };
  return es;
}

export function render(doc, v) {
  const status = doc.getElementById('status');
  status.textContent = v.status;
  status.className = v.live ? 'status live' : 'status';
  doc.getElementById('title').textContent = v.title;
  doc.getElementById('url').textContent = v.url;
  doc.title = v.title ? `${v.title} · Couch web cast` : 'Couch web cast';
}

if (typeof document !== 'undefined') {
  connect({ open: () => new EventSource('events'), show: (v) => render(document, v) });
}
