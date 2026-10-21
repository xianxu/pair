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
  let stopped = false;
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

  async function poll() {
    if (stopped) return;
    const controller = new AbortController();
    pending = controller;
    const timeout = setTimeout(() => controller.abort(), 3000);
    let departing = false;
    try {
      const response = await fetch("/__couch_mock/state", {cache: "no-store", signal: controller.signal});
      const observation = await response.json();
      if (stopped) return;
      if (!response.ok && !observation.error) throw new Error("Mock server returned HTTP " + response.status);
      const decision = reloadDecision(baseline, observation);
      if (decision.kind === "error") {
        showError(decision.message);
      } else {
        if (banner) { banner.remove(); banner = undefined; }
        if (decision.kind === "navigate") {
          departing = true;
          window.location.assign(decision.target + window.location.search + window.location.hash);
        } else if (decision.kind === "reload") {
          departing = true;
          window.location.reload();
        }
      }
    } catch (error) {
      if (!stopped) showError(error.message || "Cannot reach mock server.");
    } finally {
      clearTimeout(timeout);
      pending = undefined;
      if (!stopped && !departing) timer = setTimeout(poll, 500);
    }
  }

  window.addEventListener("pagehide", () => {
    stopped = true;
    clearTimeout(timer);
    if (pending) pending.abort();
  }, {once: true});
  timer = setTimeout(poll, 500);
})();
