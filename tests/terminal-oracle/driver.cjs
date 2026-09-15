'use strict';
// The oracle deliberately imports no Pair code or Go backend. One JSON request
// is interpreted as ordered terminal writes; each completed write is observed.
const fs = require('node:fs');
const { Terminal } = require('@xterm/headless');

async function main() {
  const request = JSON.parse(fs.readFileSync(0, 'utf8'));
  const { Cols: cols, Rows: rows, Chunks: chunks } = request;
  if (!Number.isInteger(cols) || !Number.isInteger(rows) || cols < 1 || rows < 1 || cols * rows > 262144 || !Array.isArray(chunks)) {
    throw new Error('invalid oracle dimensions or chunks');
  }
  const terminal = new Terminal({ cols, rows, allowProposedApi: true, scrollback: Number.isInteger(request.Scrollback) ? Math.min(1000, Math.max(0, request.Scrollback)) : 0 });
  const links = [];
  terminal.parser.registerOscHandler(8, data => { links.push(data); return false; });
  const result = [];
  try {
    for (const chunk of chunks) {
      if (typeof chunk !== 'string' || Buffer.byteLength(chunk) > 8 * 1024 * 1024) throw new Error('invalid oracle chunk');
      await new Promise(resolve => terminal.write(chunk, resolve));
      const buffer = terminal.buffer.active;
      const lines = [], cells = [];
      for (let y = 0; y < rows; y++) {
        const line = buffer.getLine(buffer.viewportY + y);
        lines.push(line.translateToString(false));
        const row = [];
        for (let x = 0; x < cols; x++) {
          const cell = line.getCell(x);
          row.push({ Text: cell.getChars(), Width: cell.getWidth(), FG: cell.getFgColor(), BG: cell.getBgColor(),
            FGRGB: !!cell.isFgRGB(), BGRGB: !!cell.isBgRGB(), Bold: !!cell.isBold(), Italic: !!cell.isItalic(),
            Underline: !!cell.isUnderline(), Inverse: !!cell.isInverse() });
        }
        cells.push(row);
      }
      const history = [];
      for (let y = 0; y < buffer.baseY; y++) {
        const line = buffer.getLine(y);
        const row = [];
        for (let x = 0; x < cols; x++) { const c = line.getCell(x); row.push({Text:c.getChars(),Width:c.getWidth(),FG:c.getFgColor(),BG:c.getBgColor()}); }
        history.push({Text:line.translateToString(true),Wrapped:line.isWrapped,Cells:row});
      }
      const wraps = [];
      for (let y = 0; y < rows; y++) wraps.push(buffer.getLine(buffer.viewportY+y).isWrapped);
      result.push({ History: history, Wraps: wraps, Lines: lines, Cells: cells, X: buffer.cursorX, Y: buffer.cursorY, Links: [...links],
        Modes: terminal.modes, CursorStyle: terminal.options.cursorStyle, CursorBlink: terminal.options.cursorBlink });
    }
  } finally { terminal.dispose(); }
  process.stdout.write(JSON.stringify(result));
}
main().catch(error => { process.stderr.write(String(error.stack || error) + '\n'); process.exitCode = 1; });
