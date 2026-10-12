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

const imageSet = /(?<![\w-])(?:-webkit-)?image-set\(/gi;

// urlImageSets wraps each plain string directly inside an image-set() in
// url(), so the one url() pass below rewrites it: a string there is a URL,
// one nested deeper (type("image/avif")) is not.
function urlImageSets(text) {
  let out = '', from = 0, m;
  imageSet.lastIndex = 0;
  while ((m = imageSet.exec(text))) {
    let i = m.index + m[0].length, depth = 1;
    out += text.slice(from, i);
    while (i < text.length && depth > 0) {
      const c = text[i];
      if (c === '"' || c === "'") {
        let j = i + 1;
        while (j < text.length && text[j] !== c && text[j] !== '\n') j += text[j] === '\\' ? 2 : 1;
        const str = text.slice(i, j + 1);
        out += depth === 1 && text[j] === c ? `url(${str})` : str;
        i = j + 1;
        continue;
      }
      if (c === '(') depth++;
      if (c === ')') depth--;
      out += c;
      i++;
    }
    from = i;
    imageSet.lastIndex = i;
  }
  return out + text.slice(from);
}

// rewriteCSS points every url() (and string @import, and image-set()
// string) through resolve, and rewrites :hover to the class that stands in
// for it. The spans it treats as URLs are the asset proxy's too
// (tests/webcast/css_urls.json).
export function rewriteCSS(text, base, resolve, hoverClass) {
  const one = (raw) => {
    if (raw === '' || raw.startsWith('#') || /^data:/i.test(raw)) return raw;
    const abs = resolveURL(raw, base);
    return (abs && resolve(abs)) || BLOCKED;
  };
  // One pass over both forms, so a rewritten URL is never rewritten again.
  let out = urlImageSets(String(text)).replace(refs, (m, q, imp, dq, sq, bare) => {
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
