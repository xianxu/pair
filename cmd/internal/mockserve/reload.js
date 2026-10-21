(() => {
  "use strict";

  function reloadDecision(baseline, observation) {
    if (observation.error) return {kind: "error", message: observation.error};
    if (!observation.version || !observation.revision) {
      return {kind: "error", message: "Mock server returned an invalid observation."};
    }
    if (observation.version !== baseline.version) {
      const screen = baseline.screen.split("/").map(encodeURIComponent).join("/");
      return {kind: "navigate", target: "/" + encodeURIComponent(observation.version) + "/" + screen + ".html"};
    }
    if (observation.revision !== baseline.revision) return {kind: "reload"};
    return {kind: "none"};
  }

  window.couchMockReloadDecision = reloadDecision;
  const element = document.getElementById("couch-mock-bootstrap");
  if (!element) return;
  const baseline = JSON.parse(element.textContent);
  let phase = "active";
  let generation = 0;
  let timer;
  let pending;
  let banner;

  function showError(message) {
    if (!banner) {
      banner = document.createElement("div");
      banner.id = "couch-mock-status";
      banner.setAttribute("role", "status");
      banner.style.cssText = "position:fixed;top:0;left:0;right:0;z-index:2147483647;padding:12px;background:#7f1d1d;color:white;font:14px sans-serif;white-space:pre-wrap";
      (document.body || document.documentElement).appendChild(banner);
    }
    banner.textContent = "Mock preview: " + message;
  }

  function schedulePoll() {
    if (phase !== "active" || timer !== undefined || pending) return;
    const owner = generation;
    timer = setTimeout(() => {
      if (phase !== "active" || owner !== generation) return;
      timer = undefined;
      return poll(owner);
    }, 500);
  }

  async function poll(owner) {
    if (phase !== "active" || owner !== generation || pending) return;
    const controller = new AbortController();
    const request = {controller};
    pending = request;
    const ownsRequest = () => owner === generation && pending === request;
    request.timeout = setTimeout(() => {
      if (ownsRequest() && phase === "active") controller.abort();
    }, 3000);
    try {
      const response = await fetch("/__couch_mock/state", {cache: "no-store", signal: controller.signal});
      if (!ownsRequest() || phase !== "active") return;
      const observation = await response.json();
      if (!ownsRequest() || phase !== "active") return;
      if (!response.ok && !observation.error) throw new Error("Mock server returned HTTP " + response.status);
      const decision = reloadDecision(baseline, observation);
      if (decision.kind === "error") {
        showError(decision.message);
      } else {
        if (banner) { banner.remove(); banner = undefined; }
        if (decision.kind === "navigate") {
          phase = "departing";
          window.location.assign(decision.target + window.location.search + window.location.hash);
        } else if (decision.kind === "reload") {
          phase = "departing";
          window.location.reload();
        }
      }
    } catch (error) {
      if (ownsRequest() && phase === "active") showError(error.message || "Cannot reach mock server.");
    } finally {
      // A suspended request can finish after restoration has started a new one.
      // Only its current owner may touch the shared request and polling timer.
      if (ownsRequest()) {
        clearTimeout(request.timeout);
        pending = undefined;
        schedulePoll();
      }
    }
  }

  window.addEventListener("pagehide", () => {
    phase = "paused";
    generation++;
    clearTimeout(timer);
    timer = undefined;
    if (pending) {
      const request = pending;
      pending = undefined;
      clearTimeout(request.timeout);
      request.controller.abort();
    }
  });
  window.addEventListener("pageshow", event => {
    if (!event.persisted || phase !== "paused") return;
    phase = "active";
    schedulePoll();
  });
  schedulePoll();
})();
