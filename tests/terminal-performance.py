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
line=bytearray();hover=False
while True:
 data=os.read(0,4096)
 if not data:break
 if hover:continue
 for b in data:
  if b not in (10,13):line.append(b);continue
  text=line.decode();line.clear()
  if text=='exit':sys.exit(0)
  if text=='hover':
   hover=True;os.write(1,('\x1b[2J\x1b[HHOVER:%d'%n).encode());continue
  if text.startswith('bulk:'):
   count=int(text.split(':')[1]);block=b'0123456789abcdef'*64+b'\r\n'
   for _ in range(count):os.write(1,block)
   os.write(1,('\x1b[2J\x1b[HDONE:%d'%n).encode())
  else:os.write(1,('\x1b[2J\x1b[HACK:%d:%s'%(n,text)).encode())
'''

class Oracle:
 def __init__(self, cols, rows):
  self.processing_seconds=0
  self.p=subprocess.Popen(['node',str(pathlib.Path(__file__).parent/'terminal-oracle/stream.cjs')],stdin=subprocess.PIPE,stdout=subprocess.PIPE,stderr=subprocess.PIPE)
  self.ask(dict(init=True,cols=cols,rows=rows))
 def ask(self,q):
  started=time.monotonic()
  self.p.stdin.write(json.dumps(q).encode()+b'\n');self.p.stdin.flush()
  if not select.select([self.p.stdout],[],[],5)[0]:raise RuntimeError('oracle timeout')
  line=self.p.stdout.readline()
  if not line:raise RuntimeError('oracle exited: '+self.p.stderr.read(4096).decode())
  result='\n'.join(json.loads(line)['lines'])
  self.processing_seconds+=time.monotonic()-started
  return result
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

def trial(binary,cols,rows,bulk,idle,hover_seconds,hover_hz):
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
  def measured(data,marker,details=None):
   before_bytes=parent_bytes;before_oracle=oracle.processing_seconds
   sent=time.monotonic();os.write(master,data);elapsed=(until(marker)-sent)*1000
   if details is not None:details.append(dict(elapsed_ms=elapsed,parent_bytes=parent_bytes-before_bytes,oracle_ipc_processing_ms=(oracle.processing_seconds-before_oracle)*1000))
   return elapsed
  def stage_start():
   # CPU reads are outside the timed interval, not per-input hot path.
   cpu=cpu_seconds(process.pid);oracle_cpu=cpu_seconds(oracle.p.pid)
   return dict(cpu=cpu,oracle_cpu=oracle_cpu,wall=time.monotonic(),bytes=parent_bytes,oracle=oracle.processing_seconds)
  def stage_end(started):
   elapsed=time.monotonic()-started['wall']
   return dict(elapsed_ms=elapsed*1000,parent_bytes=parent_bytes-started['bytes'],oracle_ipc_processing_ms=(oracle.processing_seconds-started['oracle'])*1000,wrapper_cpu_seconds=cpu_seconds(process.pid)-started['cpu'],oracle_cpu_seconds=cpu_seconds(oracle.p.pid)-started['oracle_cpu'])
  try:
   startup=(until('READY:1')-start)*1000
   inputs=[measured(('ping%d\r'%i).encode(),'ACK:1:ping%d'%i) for i in range(10)]
   newtab=measured(b'\x1b[116;3u','READY:2')
   measured(b'second\r','ACK:2:second')
   switches=[]
   for _ in range(10):
    switches.append(measured(b'\x1b[1;3D','ACK:1:ping9'))
    switches.append(measured(b'\x1b[1;3C','ACK:2:second'))
   # At least 256KiB passes both the baseline 128KiB capture ring and
   # candidate 65536-cell history ceiling plus the largest visible screen.
   # No private runtime introspection is used; this is saturation-by-workload.
   emitted_blocks=max(bulk,256)
   output_stage=stage_start()
   elapsed=measured(('bulk:%d\r'%emitted_blocks).encode(),'DONE:2')
   output_detail=stage_end(output_stage)
   output_wall=output_detail['elapsed_ms']/1000;output_cpu=output_detail['wrapper_cpu_seconds']
   input_details=[];input_stage=stage_start()
   after_inputs=[measured(('after%d\r'%i).encode(),'ACK:2:after%d'%i,input_details) for i in range(10)]
   input_detail=stage_end(input_stage)
   after_switches=[];switch_details=[];switch_stage=stage_start()
   for _ in range(10):
    after_switches.append(measured(b'\x1b[1;3D','ACK:1:ping9',switch_details))
    after_switches.append(measured(b'\x1b[1;3C','ACK:2:after9',switch_details))
   switch_detail=stage_end(switch_stage)
   cpu_before=cpu_seconds(process.pid);idle_start=time.monotonic();time.sleep(idle)
   cpu=(cpu_seconds(process.pid)-cpu_before)/(time.monotonic()-idle_start)*100
   # Last workload: the helper drains bytes without storing them. Baseline
   # passthrough may deliver motion even when the child requested no tracking.
   measured(b'hover\r','HOVER:2')
   hover_cpu_before=cpu_seconds(process.pid);hover_start=time.monotonic();sent=0
   while time.monotonic()-hover_start<hover_seconds:
    os.write(master,('\x1b[<35;%d;%dM'%(1+sent%cols,1+(sent//cols)%(rows-1))).encode());sent+=1
    # Drain every available parent write so the injected workload cannot
    # measure a full PTY pipe instead of routing cost. Keep only current cells.
    while select.select([master],[],[],0)[0]:
     data=os.read(master,65536)
     if not data:raise RuntimeError('PTY EOF during hover')
     parent_bytes+=len(data);screen=oracle.ask(dict(data=base64.b64encode(data).decode()))
    time.sleep(max(0,hover_start+sent/hover_hz-time.monotonic()))
   hover_wall=time.monotonic()-hover_start;hover_cpu=cpu_seconds(process.pid)-hover_cpu_before
   rss=int(subprocess.check_output(['ps','-o','rss=','-p',str(process.pid)],text=True))
   return dict(stage_costs=dict(active_output=output_detail,after_history_input=input_detail,after_history_switch=switch_detail),after_history_input_samples=input_details,after_history_switch_samples=switch_details,startup_ms=startup,new_tab_ms=newtab,input_ms=inputs,switch_ms=switches,after_history_input_ms=after_inputs,after_history_switch_ms=after_switches,output_bytes=emitted_blocks*1026,output_complete_ms=elapsed,ingested_mib_per_second=emitted_blocks*1026/(elapsed/1000)/(1<<20),wrapper_output_cpu_seconds=output_cpu,wrapper_output_cpu_percent=output_cpu/output_wall*100,wrapper_idle_cpu_percent=cpu,hover_events=sent,hover_wall_seconds=hover_wall,wrapper_hover_cpu_seconds=hover_cpu,wrapper_hover_cpu_percent=hover_cpu/hover_wall*100,wrapper_rss_kib=rss,parent_bytes=parent_bytes)

  finally:
   process.terminate()
   try:process.wait(timeout=5)
   except subprocess.TimeoutExpired:process.kill();process.wait()
   os.close(master);oracle.close()

def main():
 parser=argparse.ArgumentParser(description=__doc__)
 parser.add_argument('--baseline',required=True);parser.add_argument('--candidate',required=True)
 parser.add_argument('--hover-seconds',type=float,default=2);parser.add_argument('--hover-hz',type=float,default=120)
 parser.add_argument('--trials',type=int,default=5);parser.add_argument('--bulk-blocks',type=int,default=1024);parser.add_argument('--idle-seconds',type=float,default=2)
 args=parser.parse_args()
 if args.trials<1 or args.bulk_blocks<1 or args.idle_seconds<=0 or args.hover_seconds<=0 or args.hover_hz<=0:parser.error('positive workload required')
 result=dict(platform=platform.platform(), node=subprocess.check_output(['node','--version'],text=True).strip(), oracle='@xterm/headless 5.5.0', binary_sha256={role:hashlib.sha256(pathlib.Path(binary).read_bytes()).hexdigest() for role,binary in [('baseline',args.baseline),('candidate',args.candidate)]}, kind='isolated production pair term; independent xterm screen acknowledgments; no native Zellij',timing='includes oracle IPC+parsing for both binaries; startup includes cold per-trial runtime extraction',cpu='wrapper process only, ps cumulative CPU resolution; child/helper and independent oracle excluded; short samples may quantize to zero',history='>=256KiB emitted before post-history samples; saturation inferred from65536 history-cell ceiling and128KiB baseline ring, not introspected',hover=dict(kind='unsolicited SGR button-none motion; no child tracking requested; last workload with child drain-only',target_hz=args.hover_hz,seconds=args.hover_seconds),samples=[])
 for cols,rows in [(80,24),(240,80)]:
  for index in range(args.trials):
   for role,binary in [('baseline',args.baseline),('candidate',args.candidate)]:
    sample=dict(role=role,binary=str(pathlib.Path(binary).resolve()),cols=cols,rows=rows,trial=index)
    sample.update(trial(binary,cols,rows,args.bulk_blocks,args.idle_seconds,args.hover_seconds,args.hover_hz));result['samples'].append(sample)
    print(json.dumps(sample),file=sys.stderr,flush=True)
 result['summary']=[]
 for role in ('baseline','candidate'):
  for cols,rows in ((80,24),(240,80)):
   selected=[s for s in result['samples'] if s['role']==role and s['cols']==cols]
   summary=dict(role=role,cols=cols,rows=rows)
   for name in ('startup_ms','new_tab_ms','input_ms','switch_ms','after_history_input_ms','after_history_switch_ms','ingested_mib_per_second','wrapper_output_cpu_seconds','wrapper_output_cpu_percent','wrapper_idle_cpu_percent','wrapper_hover_cpu_seconds','wrapper_hover_cpu_percent','wrapper_rss_kib'):
    values=[]
    for sample in selected:
     value=sample[name];values.extend(value if isinstance(value,list) else [value])
    values.sort();summary[name]=dict(min=values[0],median=values[len(values)//2],p95=values[max(0,(len(values)*95+99)//100-1)],max=values[-1])
   result['summary'].append(summary)
 print(json.dumps(result,indent=2))
if __name__=='__main__':main()
