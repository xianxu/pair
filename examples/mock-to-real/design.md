# The boundary belongs to the development host

A screen catalogue lets teams replace mock screens one route at a time while
keeping one navigable application. `mocks/screens.json` is the authority for
screen state; neither the app nor its production builder maintains a second
status list. A real framework registers handlers by screen ID. In this sample,
`internal/app.RealHandlers` is that registration map.

Couch reads the catalogue and serves the development origin. It handles mock
screens and forwards real screens and APIs to an independently running HTTP
application. The application's language, framework, and build system are its
own choices. Go's `net/http` here demonstrates the seam without imposing a Go
adapter, reverse proxy package, or production dependency on Pair internals.

Canonical links (`/`, `/runs`, `/runs/detail`) remain on the development origin.
Crossing from a real screen to a mock or back requires a full document request;
a framework's SPA router must opt out of intercepting that navigation. The host
does not infer or patch client routing. API failures and real-handler failures
never select mock fallback content.

The development catalogue reader is read-only. It accepts unknown fields for
additive evolution while rejecting invalid known types, ambiguous duplicates,
duplicate IDs/routes/mock paths, unsafe paths, and unsupported versions/states.
The catalogue is bounded to 256 KiB and 1–256 screens, with one root `/` screen
using `index.html`. Canonical routes reject reserved host namespaces, encoding,
controls, whitespace, and dynamic route punctuation. Normalized relative mock
paths end in `.html`. These validation rules follow the shared host convention;
the independent sample has no Pair import.

The only production projection contains screen IDs, routes, and states. Keeping
that projection allows the app to render explicit unimplemented placeholders
and mark links without embedding development paths or data. The app prepares a
route lookup map and escaped navigation markup once, keeping request routing
constant-time and independent of later changes to caller-owned catalogue slices.

The application build treats completeness as a report, not a release veto.
A mock-state detail screen builds successfully and reports its ID/route; its
production representation is “Not implemented yet” plus a home link. A screen
marked real without a registered handler is a configuration failure. Registering
the real handler and changing the one catalogue row removes the missing report
and renders actual application content.

NEVER SHIP mock content: source references and illustrative fixtures stay in the
development tree. The Go builder stages only real application source and its
minimal projection, validates the compiled candidate, and publishes to a new
output path. Staging and child processes belong to the calling context and are
joined/removed before the build returns. The private-copy production tests
exercise both partial and completed builds and inspect the actual binary and
HTTP surface, so a real-only claim is backed by artifact evidence.

A fixture is an example of a response shape, not the real app's data store.
The pure contract helper derives required recursive fields and types from one
representative fixture subtree. It permits extra real fields and any number of
homogeneous real items, including zero; no mock value is needed by the real
server. Shape contracts complement behavioral API tests, and the deliberate
field/type mutation is a control against vacuous contract checks.
