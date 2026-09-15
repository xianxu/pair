from pathlib import Path
import json,subprocess
from importlib.machinery import SourceFileLoader
z=SourceFileLoader('z',str(Path(__file__).with_name('zellij_oracle.py'))).load_module()
# Existing history soft-wrap into viewport row1.
wire='\x1b[1;5rHHHHx\x1b[5;1H\n'
# First force all visible row links soft, to test clearing previously set flags.
for y in range(2,6):wire+=f'\x1b[{y-1};4Hxx'
# Restore selected incoming flags. Row1 intentionally keeps history link.
for y,wrapped in [(2,True),(3,False),(4,True),(5,False)]:
 wire+=(f'\x1b[{y-1};4Hxx' if wrapped else f'\x1b[{y};1H\x1b[M\x1b[L')
wire+='\x1b[?7l'+''.join(f'\x1b[{y};1H'+text for y,text in enumerate(['AAAA','BBBB','CCCC','DDDD','CHRO'],1))+'\x1b[?7h'
wire+='\x1b[5;1H'+ '\n'*4
a=dict(wire=wire,cols=4,rows=5)
x=json.loads(subprocess.check_output(['node',str(Path(__file__).with_name('xterm_oracle.cjs'))],input=json.dumps(a).encode()))
print(json.dumps(dict(wire=wire,xterm=x,zellij=z.run(**a))))
