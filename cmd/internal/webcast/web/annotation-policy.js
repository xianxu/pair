// Availability and mode are one policy for the shared annotation mount.
// A caller supplies driver only when driving forbids its annotations (the
// operator); a remote viewer's annotation rules remain unchanged.
export function annotationPolicy(previous, event) {
  const old = previous || { phase: 'idle', mode: 'watch', doc: 0, driver: '' };
  let state = { ...old };
  const effects = [];
  if (event.type === 'sync') {
    const phase = event.ended ? 'ended' : !event.ready ? 'idle' : event.driver ? 'driving' : event.paused ? 'paused' : 'ready';
    state = { ...state, phase, doc: event.doc || 0, driver: event.driver || '' };
    const changed = old.doc && state.doc && old.doc !== state.doc;
    if (changed || phase !== 'ready') state.mode = 'watch';
    if (changed || (phase !== old.phase && phase !== 'ready')) effects.push('cancel');
    if (changed || phase === 'ended' || phase === 'idle') effects.push('discard');
    else if (phase !== 'ready' && phase !== old.phase) effects.push('hold');
    else if (phase === 'ready' && phase !== old.phase) effects.push('release');
  } else if (event.type === 'escape') {
    state.mode = 'watch';
    effects.push('cancel', 'discard');
  } else if (event.type === 'mode') {
    const mode = old.phase === 'ready' && ['watch', 'point', 'note'].includes(event.mode) ? event.mode : 'watch';
    if (mode !== old.mode) effects.push('cancel');
    state.mode = mode;
  }
  return { state, effects };
}
