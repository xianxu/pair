from pathlib import Path
import json,subprocess
from importlib.machinery import SourceFileLoader
z=SourceFileLoader('z',str(Path(__file__).with_name('zellij_oracle.py'))).load_module()
def check(name,wire):
 a=dict(wire=wire,cols=4,rows=4)
 x=json.loads(subprocess.check_output(['node',str(Path(__file__).with_name('xterm_oracle.cjs'))],input=json.dumps(a).encode()))
 print(json.dumps(dict(name=name,xterm=x,zellij=z.run(**a)),ensure_ascii=False))
# Compare direct early-wide to typed staging; then overwrite the visible
# continuation and erase the tail with ECH while preserving its wrap metadata.
check('direct','AAA界Z\r\nHARD\r\nTAIL\r\n')
wire='AAA界\x1b[4;1H\n\x1b[?7l\x1b[1;1H界Z\x1b[X\x1b[?7h'
check('stage-early-wide',wire+'\x1b[4;1H\n')
check('stage-trailing-space','AA  x\x1b[4;1H\n\x1b[?7l\x1b[1;1HBZ\x1b[2X\x1b[?7h\x1b[4;1H\n')
# ECH preserves wrap while EL2 clears it in xterm; compare actual flags.
check('ech-wrap','ABCDx\x1b[2;1H\x1b[4X\x1b[?7lBBBB\x1b[?7h')
