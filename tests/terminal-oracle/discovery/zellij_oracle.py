from pathlib import Path
import os,pty,fcntl,termios,struct,subprocess,tempfile,time,json,threading,pathlib,base64,sys

def run(wire, cols=8,rows=4,resize=None):
 root=pathlib.Path(tempfile.mkdtemp(prefix='pw',dir='/tmp'));session='pw'+str(os.getpid())+str(time.monotonic_ns())[-5:]
 child=root/'child.py'
 child.write_text('import os,time,base64\nos.write(1,base64.b64decode('+repr(base64.b64encode(wire.encode()).decode())+'))\nopen('+repr(str(root/'ready'))+',"w").write("ready")\ntime.sleep(90)\n')
 (root/'layout.kdl').write_text('layout { pane borderless=true command="/usr/bin/python3" { args "'+str(child)+'"; }; }')
 (root/'config.kdl').write_text('pane_frames false\ndefault_shell "/bin/sh"\non_force_close "quit"\n')
 env=os.environ.copy()
 for k in ['ZELLIJ','ZELLIJ_SESSION_NAME','ZELLIJ_PANE_ID']:env.pop(k,None)
 env.update(TERM='xterm-256color',XDG_CACHE_HOME=str(root/'cache'),XDG_CONFIG_HOME=str(root/'config'),ZELLIJ_SOCKET_DIR=str(root/'socket'))
 master,slave=pty.openpty();fcntl.ioctl(slave,termios.TIOCSWINSZ,struct.pack('HHHH',rows,cols,0,0))
 p=subprocess.Popen(['zellij','--session',session,'--config',str(root/'config.kdl'),'--new-session-with-layout',str(root/'layout.kdl'),'--data-dir',str(root/'data')],stdin=slave,stdout=slave,stderr=slave,env=env,start_new_session=True);os.close(slave)
 captured=bytearray()
 def drain():
  try:
   while True:
    chunk=os.read(master,65536)
    if not chunk:break
    captured.extend(chunk)
  except OSError:pass
 reader=threading.Thread(target=drain,daemon=True);reader.start()
 try:
  for _ in range(100):
   if (root/'ready').exists() or p.poll()!=None:break
   time.sleep(.05)
  time.sleep(.15)
  if resize:
   fcntl.ioctl(master,termios.TIOCSWINSZ,struct.pack('HHHH',resize[1],resize[0],0,0));time.sleep(.4)
  r=subprocess.run(['zellij','--session',session,'action','dump-screen','--full'],env=env,capture_output=True,text=True,timeout=10)
  assert r.returncode==0, dict(root=str(root),returncode=r.returncode,stderr=r.stderr,pty=captured.decode(errors='replace')[-2000:])
  return dict(root=str(root),cols=cols,rows=rows,resize=resize,returncode=r.returncode,stdout=r.stdout,stderr=r.stderr,ptyerror=captured.decode(errors='replace')[-2000:] if r.returncode else '')
 finally:
  try:
   subprocess.run(['zellij','kill-session',session],env=env,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL,timeout=10)
  finally:
   try:
    try:p.wait(timeout=5)
    except subprocess.TimeoutExpired:
     p.terminate()
     try:p.wait(timeout=2)
     except subprocess.TimeoutExpired:p.kill();p.wait(timeout=2)
   finally:
    os.close(master)
    reader.join(timeout=1)
if __name__=='__main__':
 print(json.dumps(run(**json.load(sys.stdin)),ensure_ascii=False))
