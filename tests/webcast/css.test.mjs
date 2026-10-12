// The viewer's URL and stylesheet rewriting (#460).
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { rewriteCSS, rewriteSrcset, resolveURL, BLOCKED } from '../../cmd/internal/webcast/web/css.js';

const table = { 'http://x/a.png': 'assets/aa', 'http://x/f/b.woff2': 'assets/bb', 'http://x/i.css': 'assets/cc' };
const resolve = (abs) => table[abs];

test('url()s go through the asset table; misses load nothing', () => {
  assert.equal(rewriteCSS('a{background:url(a.png)}', 'http://x/p/', resolve), `a{background:url("${BLOCKED}")}`);
  assert.equal(rewriteCSS('a{background:url( "../a.png" )}', 'http://x/p/', resolve), 'a{background:url("assets/aa")}');
  assert.equal(rewriteCSS("@font-face{src:url('http://x/f/b.woff2')}", '', resolve), '@font-face{src:url("assets/bb")}');
  assert.equal(rewriteCSS('@import "i.css";', 'http://x/', resolve), '@import url("assets/cc");');
});

test('data: URLs and fragments are kept as they are', () => {
  const css = 'a{background:url(data:image/png;base64,AAAA)} b{filter:url(#f)}';
  assert.equal(rewriteCSS(css, 'http://x/', resolve), css);
});

test(':hover becomes the stand-in class, :hover-like names do not', () => {
  assert.equal(rewriteCSS('.b:hover>.c,.d:hover-x{}', '', resolve, 'h'), '.b.h>.c,.d:hover-x{}');
});

test('srcset candidates resolve one by one', () => {
  assert.equal(rewriteSrcset('a.png 1x, missing.png 2x', 'http://x/', resolve), 'assets/aa 1x');
  assert.equal(rewriteSrcset('a.png 100w,f/b.woff2 200w', 'http://x/', resolve), 'assets/aa 100w, assets/bb 200w');
});

test('only loadable URLs resolve', () => {
  assert.equal(resolveURL('javascript:alert(1)', 'http://x/'), '');
  assert.equal(resolveURL('/a', 'http://x/p'), 'http://x/a');
  assert.equal(resolveURL('a', ''), '');
});
