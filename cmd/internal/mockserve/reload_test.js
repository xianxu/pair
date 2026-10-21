const assert = require("node:assert/strict");
const fs = require("node:fs");
const test = require("node:test");
const vm = require("node:vm");
const source = fs.readFileSync(__dirname + "/reload.js", "utf8");

function client(bootstrap, fetch) {
  const timers = new Map();
  const listeners = {};
  const nodes = [];
  const navigations = [];
  let timerID = 0;
  let reloads = 0;
  const document = {
    getElementById: () => bootstrap && {textContent: JSON.stringify(bootstrap)},
    createElement: () => ({setAttribute() {}, style: {}, remove() {nodes.splice(nodes.indexOf(this), 1);}}),
    body: {appendChild(node) {nodes.push(node);}},
  };
  const window = {
    addEventListener: (name, callback) => {listeners[name] = callback;},
    location: {search: "?view=wide", hash: "#details", assign(target) {navigations.push(target);}, reload() {reloads++;}},
  };
  const context = vm.createContext({window, document, fetch, AbortController,
    setTimeout(callback, delay) {const id = ++timerID; timers.set(id, {callback, delay}); return id;},
    clearTimeout(id) {timers.delete(id);},
  });
  vm.runInContext(source, context);
  return {window, timers, nodes, navigations, listeners, get reloads() {return reloads;},
    async tick(delay) {
      const found = [...timers].find(([,timer]) => timer.delay === delay);
      assert.ok(found, "expected timer for " + delay);
      timers.delete(found[0]);
      await found[1].callback();
    },
  };
}

test("decisions compare the first observation to document bootstrap", () => {
  const decide = client(null).window.couchMockReloadDecision;
  const baseline = {version: "v1", screen: "runs/my detail", revision: "old:1"};
  assert.equal(decide(baseline, {version: "v1", revision: "old:1"}).kind, "none");
  assert.equal(decide(baseline, {version: "v1", revision: "old:2"}).kind, "reload");
  assert.equal(decide(baseline, {version: "v1", revision: "restart:1"}).kind, "reload");
  assert.equal(decide(baseline, {version: "v2", revision: "old:1"}).target, "/v2/runs/my%20detail.html");
  assert.equal(decide(baseline, {version: "v/2", revision: "old:1"}).target, "/v%2F2/runs/my%20detail.html");
  assert.equal(decide(baseline, {error: "broken"}).kind, "error");
  assert.equal(decide(baseline, {}).kind, "error");
});

test("error preserves page; repaired observation reloads from original baseline", async () => {
  const replies = [{ok: false, status: 503, json: async () => ({error: "<broken>"})},
    {ok: true, json: async () => ({version: "v1", revision: "life:3"})}];
  const app = client({version: "v1", screen: "index", revision: "life:1"}, async () => replies.shift());
  await app.tick(500);
  assert.equal(app.reloads, 0);
  assert.equal(app.nodes.length, 1);
  assert.equal(app.nodes[0].textContent, "Mock preview: <broken>");
  await app.tick(500);
  assert.equal(app.nodes.length, 0);
  assert.equal(app.reloads, 1);
  assert.equal(app.timers.size, 0);
});

test("version change preserves query/hash and uses encoded screen", async () => {
  const app = client({version: "v1", screen: "runs/my detail", revision: "life:1"},
    async () => ({ok: true, json: async () => ({version: "v2", revision: "life:2"})}));
  await app.tick(500);
  assert.deepEqual(app.navigations, ["/v2/runs/my%20detail.html?view=wide#details"]);
  assert.equal(app.timers.size, 0);
});

test("pending request has no overlapping polls; timeout permits one retry", async () => {
  let calls = 0;
  const app = client({version: "v1", screen: "index", revision: "life:1"}, (_url, options) => {
    calls++;
    return new Promise((_resolve, reject) => options.signal.addEventListener("abort", () => reject(new Error("aborted"))));
  });
  const polling = app.tick(500);
  assert.equal(calls, 1);
  assert.deepEqual([...app.timers.values()].map(x => x.delay), [3000]);
  await app.tick(3000);
  await polling;
  assert.equal(calls, 1);
  assert.deepEqual([...app.timers.values()].map(x => x.delay), [500]);
  app.listeners.pagehide();
  assert.equal(app.timers.size, 0);
});

test("pagehide aborts pending fetch and schedules no retry", async () => {
  let signal;
  const app = client({version: "v1", screen: "index", revision: "life:1"}, (_url, options) => {
    signal = options.signal;
    return new Promise((_resolve, reject) => signal.addEventListener("abort", () => reject(new Error("aborted"))));
  });
  const polling = app.tick(500);
  app.listeners.pagehide();
  assert.equal(signal.aborted, true);
  await polling;
  assert.equal(app.timers.size, 0);
  assert.equal(app.nodes.length, 0);
});
