// Package drive is remote driving's contract (#466): a viewer, granted
// control by the operator, uses the cast tab through the cast.
//
// The wire a viewer's shell sends (Batch, Act; ParseBatch), the confinement
// every act passes before it reaches the page (Confine), and two seams with
// stateful fakes and contract suites: Injector dispatches a driver's input
// into the cast tab, and Controller holds the viewers' seats, who drives,
// and routes input. Control changes are recording.Ctl records, and what
// injected input causes is attributed with recording.Source.
//
// Pure: Act, Batch, ParseBatch, Confine. Seams: Injector (real: CDP
// Input.dispatch*, #467), Controller (real: behind HTTP, #467).
// atlas/webcast-format.md, "Remote driving", is the prose.
package drive
