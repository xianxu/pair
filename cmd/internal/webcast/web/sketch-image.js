import { Replayer, inBrowser } from './replayer.js';
import { currentFrame } from './render.js';
import { paintSketchPaths } from './sketch-render.js';

async function start() {
  const [recordsResponse, sceneResponse] = await Promise.all(['records', 'scene'].map((url) => fetch(url, {cache: 'no-store'})));
  if (!recordsResponse.ok || !sceneResponse.ok) throw Error('historical sketch source unavailable');
  const records = (await recordsResponse.text()).split('\n').filter(Boolean).map(JSON.parse);
  const {plan, source, color} = await sceneResponse.json();
  const frame = document.getElementById('page');
  const replayer = new Replayer(records, (hash) => `assets/${hash}`, inBrowser(frame));
  // The server supplies an exact begin-event prefix, including only those
  // assets already bound there. Time alone must not select later same-t events.
  await replayer.replayAt(source.t, 0);
  const drawn = currentFrame(frame);
  drawn.style.left = `${-(plan.region[0] - source.scroll[0]) * plan.scale}px`;
  drawn.style.top = `${-(plan.region[1] - source.scroll[1]) * plan.scale}px`;
  drawn.style.transform = `scale(${plan.scale})`;
  const canvas = document.getElementById('ink');
  canvas.width = plan.width;
  canvas.height = plan.height;
  paintSketchPaths(canvas.getContext('2d'), plan.paths.map((p) => ({...p, color})), {
    scale: plan.scale,
    toClient: (x, y) => [(x - plan.region[0]) * plan.scale, (y - plan.region[1]) * plan.scale],
  });
  await new Promise((done) => requestAnimationFrame(() => requestAnimationFrame(done)));
  return {width:plan.width,height:plan.height};
}
window.sketchImage = {ready:start()};
