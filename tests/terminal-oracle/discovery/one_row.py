from pathlib import Path
import json,subprocess
from zellij_oracle import run
args=dict(wire="ABCDx\x1b[?7l\x1b[1;1HEFGH\x1b[?7h\n",cols=4,rows=1)
x=json.loads(subprocess.check_output(["node",str(Path(__file__).with_name("xterm_oracle.cjs"))],input=json.dumps(args).encode()))
z=run(**args)
assert [(line["trimmed"],line["wrapped"]) for line in x["lines"]]==[("ABCD",False),("EFGH",True),("",False)]
assert z["stdout"]=="ABCDEFGH\n\n",z
print(json.dumps(dict(xterm=x,zellij=z)))
