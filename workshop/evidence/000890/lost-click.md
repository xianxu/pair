# Native lost-click capture

Baseline source11fece5cfb3b7da9ab93e6e592ec4f940919887a; original owner test count10 PASS35.104s. Temporary Go overlay adds only event diagnostics and CDP Emulation.setCPUThrottlingRate(4) in the private headless browser, without machine stress. Count10:9 pass,1 fail at the same final Note wait; exit1,46.273s. Overlay retained at /Volumes/TestScratch/tmp/pair890-trace.k_uvge4k/; raw log throttled.log SHA256 df19807e575110e16ea02b21ecef1eb07fbafe9da127eb027276707927af3df8.

Save label was measured at center(427.7421875,260.09375), rect top248,height24.1875. Before pointerdown, the panel's resize clamp moves its top110->68, shifting Save top248->206. Native events hit the panel body at the old coordinates. Label remains label-required with no queued/inflight commands, generation0 allowed and host visible. This distinguishes stale geometry from save refusal or authority interruption. The original toolbar-only resize check admits the old panel geometry because the toolbar already fits below400 before its resize callback.

Final native event excerpt:

```json
[
  {
    "type": "focusin",
    "target": "sketch-label-input",
    "text": "",
    "phase": "label-required",
    "save": {
      "x": 389,
      "y": 248,
      "width": 77.484375,
      "height": 24.1875,
      "top": 248,
      "right": 466.484375,
      "bottom": 272.1875,
      "left": 389
    },
    "hidden": false
  },
  {
    "type": "pointerdown",
    "target": "feedback-body floating-panel-body",
    "text": "\u25a4Owner note while viewers OFFOpe",
    "x": 427.7421875,
    "y": 260.09375,
    "button": 0,
    "buttons": 1,
    "phase": "label-required",
    "save": {
      "x": 389,
      "y": 206,
      "width": 77.484375,
      "height": 24.1875,
      "top": 206,
      "right": 466.484375,
      "bottom": 230.1875,
      "left": 389
    },
    "hidden": false
  },
  {
    "type": "mousedown",
    "target": "feedback-body floating-panel-body",
    "text": "\u25a4Owner note while viewers OFFOpe",
    "x": 427,
    "y": 260,
    "button": 0,
    "buttons": 1,
    "phase": "label-required",
    "save": {
      "x": 389,
      "y": 206,
      "width": 77.484375,
      "height": 24.1875,
      "top": 206,
      "right": 466.484375,
      "bottom": 230.1875,
      "left": 389
    },
    "hidden": false
  },
  {
    "type": "pointerup",
    "target": "feedback-body floating-panel-body",
    "text": "\u25a4Owner note while viewers OFFOpe",
    "x": 427.7421875,
    "y": 260.09375,
    "button": 0,
    "buttons": 0,
    "phase": "label-required",
    "save": {
      "x": 389,
      "y": 206,
      "width": 77.484375,
      "height": 24.1875,
      "top": 206,
      "right": 466.484375,
      "bottom": 230.1875,
      "left": 389
    },
    "hidden": false
  },
  {
    "type": "mouseup",
    "target": "feedback-body floating-panel-body",
    "text": "\u25a4Owner note while viewers OFFOpe",
    "x": 427,
    "y": 260,
    "button": 0,
    "buttons": 0,
    "phase": "label-required",
    "save": {
      "x": 389,
      "y": 206,
      "width": 77.484375,
      "height": 24.1875,
      "top": 206,
      "right": 466.484375,
      "bottom": 230.1875,
      "left": 389
    },
    "hidden": false
  },
  {
    "type": "click",
    "target": "feedback-body floating-panel-body",
    "text": "\u25a4Owner note while viewers OFFOpe",
    "x": 427,
    "y": 260,
    "button": 0,
    "buttons": 0,
    "phase": "label-required",
    "save": {
      "x": 389,
      "y": 206,
      "width": 77.484375,
      "height": 24.1875,
      "top": 206,
      "right": 466.484375,
      "bottom": 230.1875,
      "left": 389
    },
    "hidden": false
  }
]
```

## Corrected fixture

The native held-event regression fails before the barrier (resize returned before page handlers,1.011s) and passes with it. Same geometry and metrics/wait error listener cleanup are checked with actual Chrome. Owner+regression count10 PASS61.602s; focused race PASS7.223s. Corrected owner with the same diagnostic trace and browser CPU throttle4 passed10/10 in46.813s; every Save click used y218 after rect top206. Corrected trace retained at /Volumes/TestScratch/tmp/pair890-fixedtrace.zgs8ozvg/run.log; SHA256 dcba0d71ac25d52aa4078125a96e4c3a2006e51e60066ea1ea492fb53c2b8433.
