from pathlib import Path
import subprocess,json,time,os
r=Path(__file__).parent
w=r.parent/'native-upgrades'
e=json.loads((r/'environment.json').read_text())
steps=[('runtimebundle',['make','runtimebundle-generate'],{}),('build',['go','build','-trimpath','-o',str(w/'bin/pair'),'./cmd/pair-go'],{}),('oracle',['sh','tests/terminal-oracle/run.sh'],{}),('native',['go','test','./cmd/internal/couchtty','-run','^(TestLiveConsoleNvimPreservesContentAndChrome|TestLiveReservedRowSurvivesRealScrolling|TestConsoleStatusWrapNativePTY|TestNativeConsoleWrapperZellij)$','-count=1','-v','-artifacts','-outputdir='+str(r/'artifacts')],{'PAIR_TERMINAL_ORACLE':'1','PAIR_LIVE_COUCH':'1','PAIR_LIVE_COUCH_NATIVE':'1','PAIR_NATIVE_BINARY':str(w/'bin/pair')}),('terminal',['go','test','./cmd/internal/terminal','-count=1','-v','-artifacts','-outputdir='+str(r/'artifacts')],{'PAIR_TERMINAL_ORACLE':'1'})]
results=[]
for name,argv,extra in steps:
 env=dict(e,**extra)
 (r/(name+'-command.json')).write_text(json.dumps({'cwd':str(w),'argv':argv,'env':env},indent=2))
 start=time.time()
 with (r/(name+'.log')).open('w') as out:
  p=subprocess.run(argv,cwd=w,env=env,stdout=out,stderr=subprocess.STDOUT)
 result={'name':name,'exit':p.returncode,'seconds':time.time()-start}
 results.append(result); (r/'results.json').write_text(json.dumps(results,indent=2)); print(json.dumps(result),flush=True)
 if p.returncode and name in ['runtimebundle','build','oracle']: break
