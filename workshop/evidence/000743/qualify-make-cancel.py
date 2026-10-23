import json,os,pwd,signal,subprocess,sys,time
from pathlib import Path
root=Path(sys.argv[1]);repo=Path(sys.argv[2]);artifacts=root/'artifacts'
claim=Path(pwd.getpwuid(os.getuid()).pw_dir)/'.tart/pair-runner/positions/train/claim.json'
report={'barrier':'guest command stderr and positive owned launch claim','cleanup_grace_seconds':30}
log=(root/'make.log').open('wb');p=subprocess.Popen(['make','conformance-vm'],cwd=repo,env=dict(os.environ,PAIR_VM_ARTIFACTS_DIR=str(artifacts)),stdout=log,stderr=subprocess.STDOUT,start_new_session=True)
signalled=False;complete=False
try:
 deadline=time.monotonic()+180
 while time.monotonic()<deadline:
  try:
   c=json.loads(claim.read_text())
   if (artifacts/'stderr.log').stat().st_size and c.get('pid',0)>0:
    report.update(launch_pid=c['pid'],launch_start_identity=c['start_identity'],nonce=c['nonce']);break
  except (OSError,ValueError):pass
  time.sleep(.05)
 else:raise RuntimeError('guest command barrier not reached')
 before=time.monotonic();os.killpg(p.pid,signal.SIGTERM);signalled=True
 while time.monotonic()-before<30:
  try:
   result=json.loads((artifacts/'result.json').read_text())
   report.update(cleanup_seconds=time.monotonic()-before,result_exit=result['exit_code'],lane_claim_retained=result['lane_claim_retained'])
   assert result['exit_code']==130 and not result['lane_claim_retained'],result
   complete=True;break
  except (OSError,ValueError):pass
  time.sleep(.02)
 if not complete:raise RuntimeError('cleanup result did not complete within30s')
 report['make_exit']=p.wait(timeout=5);report['passed']=True
finally:
 if not complete:
  # The direct child is deliberately unreaped, so its process-group identity
  # remains pinned until this final signal. Never select another process by ps.
  if not signalled:os.killpg(p.pid,signal.SIGTERM)
  time.sleep(10)
  try:os.killpg(p.pid,signal.SIGKILL)
  except ProcessLookupError:pass
  p.wait(timeout=5)
 log.close();(root/'report.json').write_text(json.dumps(report,indent=2)+'\n');print(json.dumps(report),flush=True)
