// URL and stylesheet rewriting for the viewer's frame (#460), pure so node
// tests can hold it. Every URL a drawn page would load goes through
// resolve(absURL), which returns the asset proxy's URL or nothing: nothing
// is ever fetched from the product's origin (Principle 4).

// BLOCKED stands in for a URL with no asset: it loads nothing.
export const BLOCKED = 'about:invalid';

// resolveURL is v resolved against base, or '' when it isn't a URL a page
// could load (javascript:, garbage).
export function resolveURL(v, base) {
  try {
    const u = new URL(String(v).trim(), base || undefined);
    return /^(https?|data|blob):$/.test(u.protocol) ? u.href : '';
  } catch {
    return '';
  }
}

const refs = /@import\s+(["'])((?:(?!\1).)*)\1|url\(\s*(?:"((?:[^"\\]|\\.)*)"|'((?:[^'\\]|\\.)*)'|([^)'"\s]*))\s*\)/gi;
const hover = /:hover(?![\w-])/g;

// rewriteCSS points every url() (and string @import) through resolve, and
// rewrites :hover to the class that stands in for it.
export function rewriteCSS(text, base, resolve, hoverClass) {
  const one = (raw) => {
    if (raw === '' || raw.startsWith('#') || /^data:/i.test(raw)) return raw;
    const abs = resolveURL(raw, base);
    return (abs && resolve(abs)) || BLOCKED;
  };
  // One pass over both forms, so a rewritten URL is never rewritten again.
  let out = String(text).replace(refs, (m, q, imp, dq, sq, bare) => {
    if (imp !== undefined) return `@import url(${JSON.stringify(one(imp))})`;
    const raw = dq ?? sq ?? bare ?? '';
    const to = one(raw);
    return to === raw ? m : `url(${JSON.stringify(to)})`;
  });
  if (hoverClass) out = out.replace(hover, `.${hoverClass}`);
  return out;
}

// rewriteSrcset resolves each candidate of a srcset; candidates with no
// asset are dropped.
export function rewriteSrcset(v, base, resolve) {
  return String(v).split(/,\s+|,(?=\S)/).map((part) => part.trim()).filter(Boolean).flatMap((part) => {
    const [u, ...desc] = part.split(/\s+/);
    const abs = resolveURL(u, base);
    const to = abs && (/^data:/i.test(abs) ? abs : resolve(abs));
    return to ? [[to, ...desc].join(' ')] : [];
  }).join(', ');
}
