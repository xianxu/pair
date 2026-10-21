// Pure, shared confidence rule. Keep the Go notes model and shared fixture in sync.
export const MAX_ANCHOR_CANDIDATES = 2000;
export const normalizeAnchorText = s => Array.from(String(s || '').replace(/[\u0009-\u000d\u0020\u0085\u00a0\u1680\u2000-\u200a\u2028\u2029\u202f\u205f\u3000]+/gu, ' ').replace(/^ +| +$/g, '')).slice(0, 200).join('');
const common = new Set(['save','cancel','ok','yes','no','next','back','close','delete','edit','more','submit','continue','learn more','click here']);
const meaningful = s => Array.from(s).length >= 3 && !common.has(s.toLowerCase());
export function scoreAnchor(a, c) {
  if (a.tag && a.tag !== c.tag) return -1;
  let score = 0, identity = false;
  for (const k of ['id','data-testid','data-test','data-cy']) {
    if (a.stable?.[k]) { if (a.stable[k] !== c.stable?.[k]) return -1; identity = true; }
  }
  if (identity) score += 50;
  const text = normalizeAnchorText(a.text), useful = meaningful(text);
  if (useful) { if (text !== normalizeAnchorText(c.text)) return -1; score += 40; }
  if (!identity && !useful) return 0;
  if (a.path?.length > 1 && c.path?.length > 1 && a.path.at(-2) === c.path.at(-2)) score += 20;
  const classes = new Set(a.classes || []);
  const overlap = new Set((c.classes || []).filter(s => s && classes.has(s))).size;
  return score + Math.min(10, overlap * 5);
}
export function chooseAnchor(a, candidates, {truncated = false, route = ''} = {}) {
  if (truncated || candidates.length > MAX_ANCHOR_CANDIDATES || !a.tag || (a.route && a.route !== route)) return null;
  let best = -1, runner = -1, winner = null;
  for (const c of candidates) {
    const score = scoreAnchor(a, c);
    if (score > best) { runner = best; best = score; winner = c; }
    else if (score > runner) runner = score;
  }
  return best >= 60 && best - runner >= 20 ? winner : null;
}
export function projectRegion(region, originalBox, currentBox) {
  const valid = b => Array.isArray(b) && b.length === 4 && b.every(Number.isFinite) && b[2] > 0 && b[3] > 0;
  if (![region, originalBox, currentBox].every(valid)) return null;
  const [x,y,w,h] = originalBox, [nx,ny,nw,nh] = currentBox;
  const result = [nx+(region[0]-x)/w*nw, ny+(region[1]-y)/h*nh, region[2]/w*nw, region[3]/h*nh];
  return valid(result) ? result : null;
}

// Affine point projection retains the original path and its pressure. A caller
// may omit pressure (mouse/touch capture); persisted tuples remain strict triples.
export function projectPoint(point, originalBox, currentBox) {
  const box = b => Array.isArray(b) && b.length === 4 && b.every(Number.isFinite) && b[2] > 0 && b[3] > 0;
  if (!Array.isArray(point) || ![2,3].includes(point.length) || !point.every(Number.isFinite) || !box(originalBox) || !box(currentBox)) return null;
  const p = point.length === 2 ? 1 : point[2];
  if (p < 0 || p > 1) return null;
  const result = [currentBox[0]+(point[0]-originalBox[0])/originalBox[2]*currentBox[2], currentBox[1]+(point[1]-originalBox[1])/originalBox[3]*currentBox[3], p];
  return result.every(Number.isFinite) ? result : null;
}
export function pressureWidth(base, pressure) {
  return Number.isFinite(base) && base > 0 && Number.isFinite(pressure) && pressure >= 0 && pressure <= 1 ? base*(.25+.75*pressure) : NaN;
}
