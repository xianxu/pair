from pathlib import Path
import os,subprocess,json,time,shutil
root=Path('/Volumes/TestScratch/tmp/pair-slot6/p759-go1273.fQOeV7')
cwd=root/'shared-upgrades'
out=root/'setup-control';out.mkdir(exist_ok=True)
tools=out/'bin';tools.mkdir(exist_ok=True)
env={'HOME':'/Users/xianxu','PATH':'/Users/xianxu/workspace/ariadne/bin:/opt/homebrew/bin:/usr/bin:/bin:/usr/sbin:/sbin','TMPDIR':str(root),'GOTOOLCHAIN':'local','GOWORK':'off','LANG':'en_US.UTF-8','TERM':'xterm-256color'}
results=[]
def run(name,argv):
 start=time.time()
 with (out/(name+'.log')).open('w') as f:r=subprocess.run(argv,cwd=cwd,env=env,stdout=f,stderr=subprocess.STDOUT)
 results.append({'name':name,'argv':argv,'exit':r.returncode,'seconds':time.time()-start})
 (out/'results.json').write_text(json.dumps(results,indent=2)+'\n')
 return r.returncode
(out/'preflight.json').write_text(json.dumps({'before_path':env['PATH'],'before_helpers':{n:shutil.which(n,path=env['PATH']) for n in ['pair','pair-launch-helper']},'stimulus':'Add only owned same-source/same-module executable prerequisites to PATH; no test assertions, code, deadlines or modules change.'},indent=2)+'\n')
if run('build-helper',['go','build','-o',str(tools/'pair-launch-helper'),'./cmd/pair-launch-helper']):raise SystemExit(1)
if run('build-pair',['go','build','-o',str(tools/'pair'),'./cmd/pair-go']):raise SystemExit(1)
env['PATH']=str(tools)+':'+env['PATH']
(out/'environment.json').write_text(json.dumps(env,indent=2)+'\n')
run('installed-couch',['go','test','./cmd/couch','-run','^TestBareCouchInstalledCommand$','-count=1','-v'])
run('recovery-composition',['go','test','./cmd/internal/couchcmd','-run','^TestRecoveryNativeOnlyProductionComposition$','-count=1','-v'])
print(json.dumps(results,indent=2),flush=True)
