from pathlib import Path
import subprocess,json,time,hashlib,re
r=Path(__file__).parent;w=r.parent/'native-upgrades'
e=json.loads((r/'environment.json').read_text())
e.update(HOME=str(r/'additional-native-home'),GOCACHE='/Users/xianxu/Library/Caches/go-build',GOPATH='/Users/xianxu/go',PAIR_TERMINAL_ORACLE='1',PAIR_TERMINAL_NATIVE='1')
names=json.loads((r/'additional-native-audit.json').read_text())['tests']
argv=['go','test','./cmd/internal/terminal','-run','^('+'|'.join(names)+')$','-count=1','-v','-artifacts','-outputdir='+str(r/'artifacts')]
(r/'additional-native-command.json').write_text(json.dumps({'cwd':str(w),'argv':argv,'env':e},indent=2))
start=time.time()
with (r/'additional-native.log').open('w') as out:p=subprocess.run(argv,cwd=w,env=e,stdout=out,stderr=subprocess.STDOUT)
s=(r/'additional-native.log').read_text()
result={'exit':p.returncode,'seconds':time.time()-start,'top_level_pass':len(re.findall(r'^--- PASS:',s,re.M)),'top_level_skip':len(re.findall(r'^--- SKIP:',s,re.M)),'fail':len(re.findall(r'--- FAIL:',s))}
(r/'additional-native-results.json').write_text(json.dumps(result,indent=2));print(json.dumps(result),flush=True)
