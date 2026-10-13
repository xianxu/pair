// A document owns its suppression tokens. Capture leases expire only to restore
// UI after a lost reply; the caller must discard pixels if its lease expired.
// Input guards never expire: an uncertain CDP effect outlives a caller timeout.
export function suppressionPolicy(tokens, event, now) {
  const next = new Map([...tokens].filter(([, until]) => until === 0 || until > now));
  if (event.op === 'acquire' && event.token && (event.until === 0 || event.until > now)) next.set(event.token, event.until);
  if (event.op === 'release') next.delete(event.token);
  return { tokens: next, hidden: next.size > 0, confirmed: next.has(event.token) };
}
