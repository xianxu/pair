// Package recording is couch broadcast-web's recording format (#457): the
// records a recorder emits, live and on disk, and what they mean.
//
// The format is specified in atlas/webcast-format.md; this package is its
// executable form. Record and its kinds are the wire shape (one JSON object
// per line), Reduce is the meaning (the state at time t), Validate is the
// conformance check, and CatchUp is the rule for joining a cast mid-way.
//
// Pure: the records, codec, State/Reduce, Canonical, Validate, CatchUp, Merge.
// IO: DirWriter and ReadDir (the on-disk layout). Seams for the wave-1
// components: Sink (the recorder's output), Live (the fan-out to viewers),
// with MemSink, FakeHub and FakeRecorder as their stateful doubles.
package recording
