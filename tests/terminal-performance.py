#!/usr/bin/env python3
"""Compare actual pair term binaries through isolated PTYs and independent xterm.
Requires npm ci in tests/terminal-oracle. No installed binaries/session changes.
Timing includes the same bounded independent-screen acknowledgment for both.
"""
import argparse, base64, fcntl, hashlib, json, os, pathlib, platform, pty, select
import struct, subprocess, sys, tempfile, termios, time

HELPER = r'''
import fcntl, os, pathlib, sys, tty
root=pathlib.Path(os.environ['PAIR_PERF_ROOT'])
with open(root/'counter','a+') as f:
 fcntl.flock(f,fcntl.LOCK_EX);f.seek(0);n=int(f.read() or '0')+1;f.seek(0);f.truncate();f.write(str(n));f.flush()
tty.setraw(0)
os.write(1, ('\x1b[2J\x1b[HREADY:%d'%n).encode())
line=bytearray()
while True:
 data=os.read(0,4096)
 if not data:break
 for b in data:
  if b not in (10,13):line.append(b);continue
  text=line.decode();line.clear()
  if text=='exit':sys.exit(0)
  if text.startswith('bulk:'):
   count=int(text.split(':')[1]);block=b'0123456789abcdef'*64+b'\r\n'
   for _ in range(count):os.write(1,block)
   os.write(1,('\x1b[2J\x1b[HDONE:%d'%n).encode())
  else:os.write(1,('\x1b[2J\x1b[HACK:%d:%s'%(n,text)).encode())
'''

class Oracle:
 def __init__(self, cols, rows):
  self.p=subprocess.Popen(['node',str(pathlib.Path(__file__).parent/'terminal-oracle/stream.cjs')],stdin=subprocess.PIPE,stdout=subprocess.PIPE,stderr=subprocess.PIPE)
  self.ask(dict(init=True,cols=cols,rows=rows))
 def ask(self,q):
  self.p.stdin.write(json.dumps(q).encode()+b'\n');self.p.stdin.flush()
  if not select.select([self.p.stdout],[],[],5)[0]:raise RuntimeError('oracle timeout')
  line=self.p.stdout.readline()
  if not line:raise RuntimeError('oracle exited: '+self.p.stderr.read(4096).decode())
  return '\n'.join(json.loads(line)['lines'])
 def close(self):
  self.p.stdin.close()
  try:self.p.wait(timeout=3)
  except subprocess.TimeoutExpired:self.p.kill();self.p.wait()

def cpu_seconds(pid):
 raw=subprocess.check_output(['ps','-o','time=','-p',str(pid)],text=True).strip()
 days=0
 if '-' in raw: day,raw=raw.split('-');days=int(day)
 parts=[float(p) for p in raw.split(':')]
 return days*86400+sum(v*60**i for i,v in enumerate(reversed(parts)))

def trial(binary,cols,rows,bulk,idle):
 with tempfile.TemporaryDirectory(prefix='pair-terminal-perf-') as directory:
  root=pathlib.Path(directory);shell=root/'shell'
  shell.write_text('#!'+sys.executable+'\n'+HELPER);shell.chmod(0o700)
  env={k:v for k,v in os.environ.items() if not k.startswith(('PAIR_','ZELLIJ','COUCH_'))}
  env.update(SHELL=str(shell),TERM='xterm-256color',PAIR_PERF_ROOT=directory,PAIR_DATA_DIR=str(root/'data'),XDG_CACHE_HOME=str(root/'cache'))
  oracle=Oracle(cols,rows);master,slave=pty.openpty()
  fcntl.ioctl(slave,termios.TIOCSWINSZ,struct.pack('HHHH',rows,cols,0,0))
  start=time.monotonic();process=subprocess.Popen([binary,'term'],stdin=slave,stdout=slave,stderr=slave,env=env,start_new_session=True);os.close(slave)
  parent_bytes=0;screen=''
  def until(marker):
   nonlocal parent_bytes,screen
   deadline=time.monotonic()+15
   while time.monotonic()<deadline:
    if select.select([master],[],[],0.1)[0]:
     try:data=os.read(master,65536)
     except OSError:raise RuntimeError('PTY ended; bounded screen='+repr(screen))
     if not data:raise RuntimeError('PTY EOF')
     parent_bytes+=len(data);screen=oracle.ask(dict(data=base64.b64encode(data).decode()))
     if marker in screen:return time.monotonic()
   raise RuntimeError('missing '+marker+'; bounded screen='+repr(screen))
  def measured(data,marker):
   sent=time.monotonic();os.write(master,data);return (until(marker)-sent)*1000
  try:
   startup=(until('READY:1')-start)*1000
   inputs=[measured(('ping%d\r'%i).encode(),'ACK:1:ping%d'%i) for i in range(10)]
   newtab=measured(b'\x1b[116;3u','READY:2')
   measured(b'second\r','ACK:2:second')
   switches=[]
   for _ in range(10):
    switches.append(measured(b'\x1b[1;3D','ACK:1:ping9'))
    switches.append(measured(b'\x1b[1;3C','ACK:2:second'))
   elapsed=measured(('bulk:%d\r'%bulk).encode(),'DONE:2')
   cpu_before=cpu_seconds(process.pid);idle_start=time.monotonic();time.sleep(idle)
   cpu=(cpu_seconds(process.pid)-cpu_before)/(time.monotonic()-idle_start)*100
   rss=int(subprocess.check_output(['ps','-o','rss=','-p',str(process.pid)],text=True))
   return dict(startup_ms=startup,new_tab_ms=newtab,input_ms=inputs,switch_ms=switches,output_bytes=bulk*1026,output_complete_ms=elapsed,ingested_mib_per_second=bulk*1026/(elapsed/1000)/(1<<20),wrapper_idle_cpu_percent=cpu,wrapper_rss_kib=rss,parent_bytes=parent_bytes)
  finally:
   process.terminate()
   try:process.wait(timeout=5)
   except subprocess.TimeoutExpired:process.kill();process.wait()
   os.close(master);oracle.close()

def main():
 parser=argparse.ArgumentParser(description=__doc__)
 parser.add_argument('--baseline',required=True);parser.add_argument('--candidate',required=True)
 parser.add_argument('--trials',type=int,default=5);parser.add_argument('--bulk-blocks',type=int,default=1024);parser.add_argument('--idle-seconds',type=float,default=2)
 args=parser.parse_args()
 if args.trials<1 or args.bulk_blocks<1 or args.idle_seconds<=0:parser.error('positive workload required')
 result=dict(platform=platform.platform(), node=subprocess.check_output(['node','--version'],text=True).strip(), oracle='@xterm/headless 5.5.0', binary_sha256={role:hashlib.sha256(pathlib.Path(binary).read_bytes()).hexdigest() for role,binary in [('baseline',args.baseline),('candidate',args.candidate)]}, kind='isolated production pair term; independent xterm screen acknowledgments; no native Zellij',timing='includes oracle IPC+parsing for both binaries; startup includes cold per-trial runtime extraction',cpu='wrapper process only, ps cumulative CPU resolution; child/helper excluded',samples=[])
 for cols,rows in [(80,24),(240,80)]:
  for index in range(args.trials):
   for role,binary in [('baseline',args.baseline),('candidate',args.candidate)]:
    sample=dict(role=role,binary=str(pathlib.Path(binary).resolve()),cols=cols,rows=rows,trial=index)
    sample.update(trial(binary,cols,rows,args.bulk_blocks,args.idle_seconds));result['samples'].append(sample)
    print(json.dumps(sample),file=sys.stderr,flush=True)
 result['summary']=[]
 for role in ('baseline','candidate'):
  for cols,rows in ((80,24),(240,80)):
   selected=[s for s in result['samples'] if s['role']==role and s['cols']==cols]
   summary=dict(role=role,cols=cols,rows=rows)
   for name in ('startup_ms','new_tab_ms','input_ms','switch_ms','ingested_mib_per_second','wrapper_idle_cpu_percent','wrapper_rss_kib'):
    values=[]
    for sample in selected:
     value=sample[name];values.extend(value if isinstance(value,list) else [value])
    values.sort();summary[name]=dict(min=values[0],median=values[len(values)//2],p95=values[max(0,(len(values)*95+99)//100-1)],max=values[-1])
   result['summary'].append(summary)
 print(json.dumps(result,indent=2))
if __name__=='__main__':main()
