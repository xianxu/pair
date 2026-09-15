from pathlib import Path
import json,subprocess
from importlib.machinery import SourceFileLoader
z=SourceFileLoader('z',str(Path(__file__).with_name('zellij_oracle.py'))).load_module()
groups=[ [('A',1)]*3+[('界',2),('Z',1)], [('A',1)]*2+[(' ',1)]*2+[('B',1),('Z',1)], [('X',1),('Y',1),('界',2),('Q',1)], [('e\u0301',1),('F',1),('G',1),('H',1)]]

def serialize(cols):
 wire='\x1b[1;2r\x1b[?7h\x1b[1;1H\x1b[2K'
 for group in groups:
  physical=[];row=[];used=0
  for cluster,width in group:
   if used+width>cols:physical.append((row,used));row=[];used=0
   row.append((cluster,width));used+=width
  if row:physical.append((row,used))
  for i,(row,used) in enumerate(physical):
   # Canonical scratch row2, untouched incoming link on row1.
   wire+='\x1b[2;1H\x1b[M\x1b[L\x1b[2K\x1b[1;1H'+''.join(c for c,w in row)
   if i+1<len(physical):wire+=('界' if used<cols else 'x')
   wire+='\x1b[2;1H\n'
 wire+='\x1b[r\x1b[?7l'
 for y,text in enumerate(['ONE','TWO','THR','CHR'],1):wire+=f'\x1b[{y};1H\x1b[{cols}X'+text
 return wire+'\x1b[?7h'

for cols in [4,6]:
 direct=''.join(''.join(c for c,w in g)+'\r\n' for g in groups)+'ONE\r\nTWO\r\nTHR\r\nCHR'
 results=[]
 for name,wire in [('direct',direct),('typed-rebuild','OLDWRAPPED'*12+'\x1b[3J\x1b[r\x1b[1;1H\x1b[4L'+serialize(cols))]:
  a=dict(wire=wire,cols=cols,rows=4)
  x=json.loads(subprocess.check_output(['node',str(Path(__file__).with_name('xterm_oracle.cjs'))],input=json.dumps(a).encode()))
  results.append(dict(name=name,xterm=x,zellij=z.run(**a)))
 assert results[0]['xterm']==results[1]['xterm'],results
 assert results[0]['zellij']['stdout']==results[1]['zellij']['stdout'],results
 print(json.dumps(dict(cols=cols,equal_xterm=results[0]['xterm']==results[1]['xterm'],equal_zellij=results[0]['zellij']['stdout']==results[1]['zellij']['stdout'],results=results),ensure_ascii=False))
