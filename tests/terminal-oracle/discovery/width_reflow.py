from pathlib import Path
import json,subprocess
from importlib.machinery import SourceFileLoader
z=SourceFileLoader('z',str(Path(__file__).with_name('zellij_oracle.py'))).load_module()
texts=['AAA界Z','AA  BZ','XYZ🙂Q','e\u0301FGH']
wire=''.join(s+'\r\n' for s in texts)+''.join('T'+str(i)+'\r\n' for i in range(8))
results=[]
for name,args in [('native-resize',dict(wire=wire,cols=4,rows=4,resize=[6,4])),('typed-rebuild',dict(wire=wire,cols=6,rows=4))]:
 x=json.loads(subprocess.check_output(['node',str(Path(__file__).with_name('xterm_oracle.cjs'))],input=json.dumps(args).encode()))
 logical=[]
 for l in x['lines']:
  if l['wrapped'] and logical:logical[-1]+=l['trimmed']
  else:logical.append(l['trimmed'])
 r=dict(name=name,xterm_lines=x['lines'],xterm_logical=logical,zellij=z.run(**args));results.append(r)
assert results[0]['xterm_lines']==results[1]['xterm_lines'],results
assert results[0]['zellij']['stdout']==results[1]['zellij']['stdout'],results
print(json.dumps(results,ensure_ascii=False))
