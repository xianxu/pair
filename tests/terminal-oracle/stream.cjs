'use strict';
// Bounded incremental screen-only oracle for the performance PTY driver.
const readline = require('node:readline');
const { Terminal } = require('@xterm/headless');
let terminal;
(async () => {
  for await (const line of readline.createInterface({ input: process.stdin })) {
    const q = JSON.parse(line);
    if (q.init) {
      if (terminal || !Number.isInteger(q.cols) || !Number.isInteger(q.rows) || q.cols*q.rows>262144 || q.cols<1 || q.rows<1) throw Error('invalid geometry');
      terminal = new Terminal({cols:q.cols, rows:q.rows, scrollback:0, allowProposedApi:true});
    } else {
      if (!terminal || typeof q.data!=='string' || q.data.length>1048576) throw Error('invalid write');
      await new Promise(resolve => terminal.write(Buffer.from(q.data,'base64'),resolve));
    }
    const b=terminal.buffer.active, lines=[];
    for(let y=0;y<terminal.rows;y++) lines.push(b.getLine(b.viewportY+y).translateToString(true));
    process.stdout.write(JSON.stringify({lines})+'\n');
  }
  if (terminal) terminal.dispose();
})().catch(e=>{process.stderr.write(String(e)+'\n');process.exitCode=1;});
