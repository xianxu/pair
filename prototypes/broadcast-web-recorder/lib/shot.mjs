import { PNG } from 'pngjs';
// Viewport screenshot halved with a 2x2 box filter. CDP's clip is in document coordinates, so a clipped capture of a
// scrolled page is offset; capture the viewport as is and scale here, identically for the tab and the replays.
export async function lowres(page) {
  const { data } = await page.Page.captureScreenshot({ format: 'png' });
  const im = PNG.sync.read(Buffer.from(data, 'base64'));
  const w = im.width >> 1, h = im.height >> 1, o = new PNG({ width: w, height: h });
  for (let y = 0; y < h; y++) for (let x = 0; x < w; x++) for (let c = 0; c < 4; c++) {
    const at = (yy, xx) => im.data[((yy) * im.width + xx) * 4 + c];
    o.data[(y * w + x) * 4 + c] = (at(2 * y, 2 * x) + at(2 * y, 2 * x + 1) + at(2 * y + 1, 2 * x) + at(2 * y + 1, 2 * x + 1)) >> 2;
  }
  return PNG.sync.write(o);
}
