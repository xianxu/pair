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

// The lexical pieces both passes skip or read. A comment runs to its end
// (or the text's); a string stops at its quote or a newline. A name must
// start where no name character precedes it ("myurl(" is not url(). The
// asset proxy's scanner (assets.RewriteCSS) reads the same spans:
// tests/webcast/css_urls.json holds both to them.
const COMMENT = String.raw`\/\*[\s\S]*?(?:\*\/|$)`;
const STRING = String.raw`"(?:[^"\\\n]|\\.)*"|'(?:[^'\\\n]|\\.)*'`;
const NAME = String.raw`(?<![\w\u0080-￿-])`;
const refs = new RegExp(String.raw`(?<comment>${COMMENT})|@import\s+(?<imp>${STRING})|${NAME}url\(\s*(?:(?<quoted>${STRING})|(?<bare>[^)'"\s]*))\s*\)|(?<str>${STRING})`, 'gi');
const hover = /:hover(?![\w-])/g;
const imageSetScan = new RegExp(String.raw`(?<skip>${COMMENT}|${STRING})|${NAME}(?:-webkit-)?image-set\(`, 'gi');
const commentAt = new RegExp(COMMENT, 'y');

// unquote is a CSS string's value: quotes off, escapes of one character
// undone.
const unquote = (s) => s.slice(1, -1).replace(/\\(.)/gs, '$1');

// urlImageSets wraps each plain string directly inside an image-set() in
// url(), so the one url() pass below rewrites it: a string there is a URL,
// one nested deeper (type("image/avif")) is not.
function urlImageSets(text) {
  let out = '', from = 0, m;
  imageSetScan.lastIndex = 0;
  while ((m = imageSetScan.exec(text))) {
    if (m.groups.skip !== undefined) continue;
    let i = m.index + m[0].length, depth = 1;
    out += text.slice(from, i);
    while (i < text.length && depth > 0) {
      const c = text[i];
      commentAt.lastIndex = i;
      const comment = c === '/' && commentAt.exec(text);
      if (comment) {
        out += comment[0];
        i += comment[0].length;
        continue;
      }
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
    imageSetScan.lastIndex = i;
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
  // One pass over every form, so a rewritten URL is never rewritten again;
  // comments and other strings are passed over whole.
  let out = urlImageSets(String(text)).replace(refs, (m, ...args) => {
    const g = args.at(-1);
    if (g.comment !== undefined || g.str !== undefined) return m;
    if (g.imp !== undefined) return `@import url(${JSON.stringify(one(unquote(g.imp)))})`;
    const raw = g.quoted !== undefined ? unquote(g.quoted) : g.bare ?? '';
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
