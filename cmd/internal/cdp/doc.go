// Package cdp is a minimal Chrome DevTools Protocol client over Chrome's
// --remote-debugging-pipe transport: Chrome reads commands on fd 3 and writes
// replies and events on fd 4, each message one JSON object terminated by a
// NUL byte. Sessions use flatten mode (a top-level sessionId on the message).
// This is the only place CDP wire types live; consumers keep them behind
// their own seam so the rest of the program never speaks CDP.
package cdp
