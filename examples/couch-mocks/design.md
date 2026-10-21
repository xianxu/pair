# Echo control board mock

Run `couch mock serve examples/couch-mocks`, then open the printed loopback URL. The highest numbered version is current by default. Compare with `couch mock use examples/couch-mocks mock-1`; switch back with `mock-2`. Both directions use one root fixtures.json.

| Screen ID | Content | Actions |
|---|---|---|
| index | Project overview and run count | Runs → runs; Run detail → detail |
| runs | Linked run list | Open a run → detail?id=run-N; Overview → index |
| detail | Selected run name and status | Runs → runs; Overview → index |

Every screen reads `/fixtures.json`. The scenario links preserve the screen identity. On runs, `?state=empty` shows the empty array and `?state=long-list` shows 80 runs. On every screen, `?state=error` shows the fixture error message. A missing run ID is an explicit not-found detail state. The normal list/detail share IDs and names.

The fixture shape is the future API contract: project.name, arrays of {id,name,status}, empty array, and error.message. Changing a field here is a contract decision for the future backend, not independent screen-local fake data.

Each feedback round is a git commit to the current version; undo by reverting that commit. Create mock-N+1 for a new direction worth comparing. The server does not invoke git or create versions. Keep versions committed with the project; sketch scoping uses version and screen marks supplied by the server.
