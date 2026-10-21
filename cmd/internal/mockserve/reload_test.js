const assert = require("node:assert/strict");
const fs = require("node:fs");
const test = require("node:test");
const vm = require("node:vm");
const source = fs.readFileSync(__dirname + "/reload.js", "utf8");

function client(bootstrap, fetch) {
  const timers = new Map();
  const listeners = {};
  const listenerOptions = {};
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
    addEventListener: (name, callback, options) => {listeners[name] = callback; listenerOptions[name] = options;},
    location: {search: "?view=wide", hash: "#details", assign(target) {navigations.push(target);}, reload() {reloads++;}},
  };
  const context = vm.createContext({window, document, fetch, AbortController,
    setTimeout(callback, delay) {const id = ++timerID; timers.set(id, {callback, delay}); return id;},
    clearTimeout(id) {timers.delete(id);},
  });
  vm.runInContext(source, context);
  return {window, timers, nodes, navigations, listeners, get reloads() {return reloads;},
    dispatch(name, event = {}) {
      const callback = listeners[name];
      if (callback) {
        if (listenerOptions[name]?.once) delete listeners[name];
        callback(event);
      }
    },
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

test("repeated BFCache hide/show resumes one loop and retains the document baseline", async () => {
  let calls = 0;
  let observation = {version: "v1", revision: "life:1"};
  const app = client({version: "v1", screen: "index", revision: "life:1"}, async () => {
    calls++;
    return {ok: true, json: async () => observation};
  });
  app.dispatch("pageshow", {persisted: false});
  app.dispatch("pageshow", {persisted: true});
  assert.equal(app.timers.size, 1);
  await app.tick(500);
  for (let i = 0; i < 3; i++) {
    app.dispatch("pagehide", {persisted: true});
    assert.equal(app.timers.size, 0);
    app.dispatch("pageshow", {persisted: true});
    app.dispatch("pageshow", {persisted: true});
    assert.equal(app.timers.size, 1);
    await app.tick(500);
  }
  assert.equal(calls, 4);
  app.dispatch("pagehide", {persisted: true});
  observation = {version: "v1", revision: "life:2"};
  app.dispatch("pageshow", {persisted: true});
  await app.tick(500);
  assert.equal(app.reloads, 1);
  assert.equal(app.timers.size, 0);
  app.dispatch("pageshow", {persisted: true});
  assert.equal(app.timers.size, 0, "departing document must not start another poll");
});

test("version changed while paused navigates on BFCache restoration", async () => {
  const app = client({version: "v1", screen: "runs/my detail", revision: "life:1"},
    async () => ({ok: true, json: async () => ({version: "v2", revision: "life:2"})}));
  app.dispatch("pagehide", {persisted: true});
  app.dispatch("pageshow", {persisted: true});
  await app.tick(500);
  assert.deepEqual(app.navigations, ["/v2/runs/my%20detail.html?view=wide#details"]);
});

test("a queued timer from before suspension cannot consume the restored timer", async () => {
  let calls = 0;
  const app = client({version: "v1", screen: "index", revision: "life:1"}, async () => {
    calls++;
    return {ok: true, json: async () => ({version: "v1", revision: "life:1"})};
  });
  const queuedCallback = [...app.timers.values()][0].callback;
  app.dispatch("pagehide", {persisted: true});
  app.dispatch("pageshow", {persisted: false});
  assert.equal(app.timers.size, 0);
  app.dispatch("pageshow", {persisted: true});
  const restoredTimers = [...app.timers.keys()];
  await queuedCallback();
  assert.equal(calls, 0);
  assert.deepEqual([...app.timers.keys()], restoredTimers);
  await app.tick(500);
  assert.equal(calls, 1);
  assert.equal(app.timers.size, 1);
});

for (const completion of ["response", "rejection", "body"]) {
  test("stale " + completion + " after restore cannot affect the resumed request", async () => {
    const requests = [];
    const app = client({version: "v1", screen: "index", revision: "life:1"}, (_url, options) => {
      const request = {signal: options.signal};
      requests.push(request);
      return new Promise((resolve, reject) => Object.assign(request, {resolve, reject}));
    });
    const oldPoll = app.tick(500);
    let resolveBody;
    if (completion === "body") {
      let enteredBody;
      const bodyReady = new Promise(resolve => {enteredBody = resolve;});
      requests[0].resolve({ok: true, json: () => new Promise(resolve => {resolveBody = resolve; enteredBody();})});
      await bodyReady;
    }
    app.dispatch("pagehide", {persisted: true});
    assert.equal(requests[0].signal.aborted, true);
    assert.equal(app.timers.size, 0, "hide cancels request timeout immediately");
    app.dispatch("pageshow", {persisted: true});
    const newPoll = app.tick(500);
    assert.equal(requests.length, 2);
    const newTimers = [...app.timers.keys()];
    if (completion === "rejection") requests[0].reject(new Error("stale error"));
    else if (completion === "body") resolveBody({version: "v2", revision: "life:2"});
    else requests[0].resolve({ok: true, json: async () => ({error: "stale error"})});
    await oldPoll;
    assert.equal(app.nodes.length, 0);
    assert.equal(app.reloads, 0);
    assert.equal(app.navigations.length, 0);
    assert.deepEqual([...app.timers.keys()], newTimers, "stale finally must not change current timers");
    assert.equal(requests[1].signal.aborted, false);
    app.dispatch("pagehide", {persisted: true});
    assert.equal(requests[1].signal.aborted, true, "subsequent hide still owns the resumed request");
    requests[1].reject(new Error("cancelled"));
    await newPoll;
    assert.equal(app.timers.size, 0);
    assert.equal(app.nodes.length, 0);
  });
}
