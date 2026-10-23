from pathlib import Path
import subprocess,json,time
r=Path('/Volumes/TestScratch/tmp/pair-slot6/p759-go1273.fQOeV7')
o=r/'launcher-control';o.mkdir(exist_ok=True)
env=json.loads((r/'setup-control/environment.json').read_text())
argv=['go','test','-modfile='+str(r/'native-results/original-module-control/baseline.mod'),'./cmd/internal/launcher','-run','^TestLookupSocketUsesNativeSymlinkParentTraversal$','-count=1','-v']
(o/'command.json').write_text(json.dumps({'argv':argv,'environment':env,'source':'bfa3f4a22','control':'Original modules, exact failed test, same scratch environment, no source changes'},indent=2)+'\n')
s=time.time()
with (o/'test.log').open('w') as f:p=subprocess.run(argv,cwd=r/'shared-upgrades',env=env,stdout=f,stderr=subprocess.STDOUT)
z={'exit':p.returncode,'seconds':time.time()-s};(o/'result.json').write_text(json.dumps(z,indent=2)+'\n');print(json.dumps(z))
