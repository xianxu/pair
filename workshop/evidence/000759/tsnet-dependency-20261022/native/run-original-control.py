from pathlib import Path
import subprocess,json,time,hashlib,re,threading,stat
r=Path(__file__).parent;d=r/'original-module-control';w=r.parent/'native-upgrades'
e=json.loads((r/'additional-native-command.json').read_text())['env'];e['HOME']=str(d/'home')
a=['go','test','-modfile='+str(d/'baseline.mod'),'./cmd/internal/terminal','-run','^TestHistoryWireNativeOracle$','-count=1','-v','-artifacts','-outputdir='+str(d/'artifacts')]
(d/'command.json').write_text(json.dumps({'cwd':str(w),'argv':a,'env':e},indent=2))
start=time.time();roots=set();pids=set();observations=[]
with (d/'test.log').open('w') as out:
 p=subprocess.Popen(a,cwd=w,env=e,stdout=out,stderr=subprocess.STDOUT)
 while p.poll() is None:
  rows=subprocess.check_output(['/bin/ps','-axo','pid=,ppid=,command='],text=True).splitlines();parsed=[]
  for line in rows:
   v=line.strip().split(None,2)
   if len(v)==3:parsed.append((int(v[0]),int(v[1]),v[2]))
  owned={p.pid}
  for i in range(8):owned.update(pid for pid,ppid,c in parsed if ppid in owned)
  for pid,ppid,c in parsed:
   if pid in owned:
    pids.add(pid)
    for root in re.findall(r'/tmp/pw[a-zA-Z0-9_-]+',c):roots.add(root)
  time.sleep(.1)
 p.wait()
result={'exit':p.returncode,'seconds':time.time()-start,'source':'bfa3f4a22','upgraded_candidate_retried':False}
(d/'results.json').write_text(json.dumps(result,indent=2))
rows=subprocess.check_output(['/bin/ps','-axo','pid=,ppid=,command='],text=True).splitlines();left=[]
for line in rows:
 v=line.strip().split(None,2)
 if v and (int(v[0]) in pids or any(root in line for root in roots) or str(d/'home') in line):left.append(line)
cleanup={'observed_roots':sorted(roots),'observed_pids':sorted(pids),'surviving_process_matches':left,'remaining_roots':[root for root in roots if Path(root).exists()],'private_home_sockets':[str(x) for x in (d/'home').rglob('*') if x.exists() and stat.S_ISSOCK(x.stat().st_mode)]}
(d/'cleanup.json').write_text(json.dumps(cleanup,indent=2))
(d/'module-hashes.json').write_text(json.dumps({n:hashlib.sha256((d/n).read_bytes()).hexdigest() for n in ['baseline.mod','baseline.sum']},indent=2))
print(json.dumps(result),flush=True);print(json.dumps(cleanup),flush=True)
