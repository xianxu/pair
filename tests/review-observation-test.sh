#!/usr/bin/env bash
# Slow Git observation must not block editing or accumulate activation callbacks.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PAIR_ROOT="$ROOT" python3 - <<'PY'
import json,os,pathlib,subprocess,sys,tempfile,time
root=pathlib.Path(os.environ['PAIR_ROOT'])
pane_init=pathlib.Path(os.environ.get('PAIR_TEST_REVIEW_INIT',str(root/'nvim/review.lua')))
with tempfile.TemporaryDirectory(prefix='pair-observation-') as td:
    tmp=pathlib.Path(td).resolve();repo=tmp/'repo';repo.mkdir();data=tmp/'data';data.mkdir();home=tmp/'home';(home/'bin').mkdir(parents=True)
    def git(*args):return subprocess.check_output(['git','-C',str(repo),*args],text=True,timeout=5).strip()
    git('init','-qb','main');git('config','user.name','T');git('config','user.email','t@e.com')
    for name in ('a','b'):(repo/(name+'.md')).write_text(name+'\n')
    git('add','.');git('commit','-qm','initial')
    for name in ('a','b'):
        git('checkout','-qb','review/'+name,'main');(repo/(name+'.md')).write_text(name+' reviewed\n')
        record=dict(old=name,new=name+' reviewed',occurrence=1,new_occurrence=1,explain=name+' snapshot explanation')
        (tmp/'msg').write_text('review('+name+'): agent r1\n\n```review-records\n'+json.dumps([record])+'\n```\n')
        git('commit','-qaF',str(tmp/'msg'))
    git('checkout','-q','review/a')
    def resolve():return json.loads(subprocess.check_output([str(root/'bin/pair'),'review','readiness','--resolve',str(repo)],text=True,timeout=4))
    slow=tmp/'slow';calls=tmp/'calls'
    wrapper=home/'bin/pair'
    wrapper.write_text('#!'+sys.executable+'\n'+'''import os,sys,time,subprocess
from pathlib import Path
if sys.argv[1:4]==['review','readiness','--resolve'] and Path(os.environ['SLOW_FILE']).exists():
 with Path(os.environ['CALLS_FILE']).open('a') as f:f.write('scan\\n')
 time.sleep(1)
if sys.argv[1:4]==['review','readiness','--resolve'] and Path(os.environ['HOLD_FILE']).exists():
 result=subprocess.run([os.environ['REAL_PAIR'],*sys.argv[1:]],stdout=subprocess.PIPE,stderr=subprocess.PIPE)
 Path(os.environ['CAPTURED_FILE']).write_text('captured')
 while not Path(os.environ['RELEASE_FILE']).exists():time.sleep(.01)
 sys.stdout.buffer.write(result.stdout);sys.stderr.buffer.write(result.stderr);sys.exit(result.returncode)
os.execv(os.environ['REAL_PAIR'],[os.environ['REAL_PAIR'],*sys.argv[1:]])
''');wrapper.chmod(0o700)
    opened=data/'review.open'
    env=dict(os.environ,PAIR_HOME=str(home),REAL_PAIR=str(root/'bin/pair'),SLOW_FILE=str(slow),CALLS_FILE=str(calls),HOLD_FILE=str(tmp/'hold'),CAPTURED_FILE=str(tmp/'captured'),RELEASE_FILE=str(tmp/'release'),
        PAIR_DATA_DIR=str(data),PAIR_TAG='observation',PAIR_SESSION_ID='sid',PAIR_REVIEW_OPEN_PATH=str(opened),
        PAIR_REVIEW_HANDOFF_PATH=str(data/'handoff.json'),PAIR_REVIEW_LANDED_PATH=str(data/'landed.json'),
        PAIR_REVIEW_IDENTITY=json.dumps(resolve()),XDG_STATE_HOME=str(tmp/'state'),XDG_CACHE_HOME=str(tmp/'cache'),XDG_DATA_HOME=str(tmp/'xdg'))
    with (tmp/'pane.log').open('w') as log:
        pane=subprocess.Popen(['nvim','--headless','-u',str(pane_init),str(repo/'a.md')],cwd=repo,env=env,stdout=log,stderr=log)
    def evaluate(expr):return subprocess.check_output(['nvim','--server',meta['endpoint'],'--remote-expr',expr],env=env,text=True,timeout=8).strip()
    def lua(code):return evaluate("luaeval('"+code.replace("'","''")+"')")
    def request(want):
        raw=json.dumps(dict(token=meta['token'],session='sid',identity=want))
        return json.loads(evaluate("luaeval('vim.json.encode(PairReviewPane.restore(_A))', '"+raw.replace("'","''")+"')"))
    try:
        end=time.monotonic()+6
        while time.monotonic()<end:
            if opened.exists() and len(opened.read_text().splitlines())>=3:break
            assert pane.poll() is None,(tmp/'pane.log').read_text();time.sleep(.025)
        meta=json.loads(opened.read_text().splitlines()[2]);failures=[]
        count=lambda:int(lua('#vim.api.nvim_get_autocmds({event="TextChanged",buffer=vim.api.nvim_get_current_buf()})'))
        before=count()
        for name in ('b','a','b','a'):
            git('checkout','-q','review/'+name);ack=request(resolve());assert ack['ok'],ack
        after=count()
        if before!=after:failures.append('render callbacks accumulated: '+str((before,after)))
        # A completed identity observation is deliberately withheld across a
        # checkout. The callback must apply captured A bytes, never reread B.
        (repo/'a.md').write_text('a reviewed captured before checkout\n')
        (tmp/'hold').touch();lua('vim.api.nvim_exec_autocmds("FocusGained",{})')
        end=time.monotonic()+3
        while not (tmp/'captured').exists() and time.monotonic()<end:time.sleep(.01)
        assert (tmp/'captured').exists(),'resolver did not reach controlled delivery boundary'
        git('checkout','--','a.md');git('checkout','-q','review/b')
        (tmp/'release').touch();time.sleep(.3)
        displayed=lua('vim.api.nvim_buf_get_lines(0,0,1,false)[1]')
        if displayed!='a reviewed captured before checkout':failures.append('late observation loaded unbound bytes: '+repr(displayed))
        undo_marks=int(lua('(function() vim.cmd("undo"); vim.api.nvim_exec_autocmds("TextChanged",{buffer=0}); return #vim.api.nvim_buf_get_extmarks(0,vim.api.nvim_create_namespace("review"),0,-1,{}) end)()'))
        if lua('vim.api.nvim_buf_get_lines(0,0,1,false)[1]')!='a reviewed' or undo_marks<1:failures.append('snapshot undo lost original text or decorations')
        redo_marks=int(lua('(function() vim.cmd("redo"); vim.api.nvim_exec_autocmds("TextChanged",{buffer=0}); return #vim.api.nvim_buf_get_extmarks(0,vim.api.nvim_create_namespace("review"),0,-1,{}) end)()'))
        if lua('vim.api.nvim_buf_get_lines(0,0,1,false)[1]')!='a reviewed captured before checkout' or redo_marks<1:failures.append('snapshot redo lost captured text or decorations')
        if lua('select(1,pcall(vim.cmd,"write"))')!='false':failures.append('late snapshot allowed write into different checkout')
        if (repo/'a.md').read_text()!='a\n':failures.append('late snapshot changed checkout bytes')
        (tmp/'hold').unlink();git('checkout','-q','review/a')
        # Clean same-branch disk edits still reload, after an async observation.
        # The timed region is inside the editor; RPC startup is excluded.
        slow.touch()
        duration=float(lua('(function() local now=vim.uv.hrtime(); assert(PairReviewPane.apply_definition_result(vim.api.nvim_get_current_buf())==false); return (vim.uv.hrtime()-now)/1e6 end)()'))
        if duration>=250:failures.append('empty definition poll blocked for '+str(duration)+'ms')
        (repo/'a.md').write_text('external same branch')
        duration=float(lua('(function() local now=vim.uv.hrtime(); vim.api.nvim_exec_autocmds("FocusGained",{}); return (vim.uv.hrtime()-now)/1e6 end)()'))
        if duration>=250:failures.append('clean FocusGained blocked for '+str(duration)+'ms')
        end=time.monotonic()+4
        while time.monotonic()<end:
            if lua('vim.api.nvim_buf_get_lines(0,0,1,false)[1]')=='external same branch':break
            time.sleep(.025)
        else:failures.append('matching branch external edit did not reload')
        if lua('vim.bo.endofline')!='false':failures.append('snapshot lost no-final-newline state')
        slow.unlink()
        for body,first,eol,fmt,bomb in [
            ('dos first\r\ndos last\r\n','dos first','true','dos','false'),
            ('\ufeffBOM first\r\nBOM last\r\n','BOM first','true','dos','true'),
            ('\ufeffBOM no EOL','BOM no EOL','false','unix','true'),
            ('\ufeff','','false','unix','true'),
            ('','','false','unix','false'),
            ('LF first\nLF last\n','LF first','true','unix','false'),
            ('lone\rcarriage return','lone\rcarriage return','false','unix','false'),
            ('external same branch','external same branch','false','unix','false')]:
            (repo/'a.md').write_bytes(body.encode())
            lua('vim.api.nvim_exec_autocmds("FocusGained",{})')
            end=time.monotonic()+3
            while time.monotonic()<end:
                if json.loads(lua('vim.json.encode(vim.api.nvim_buf_get_lines(0,0,1,false)[1])'))==first and lua('vim.bo.endofline')==eol and lua('vim.bo.fileformat')==fmt and lua('vim.bo.bomb')==bomb:break
                time.sleep(.025)
            else:failures.append('snapshot line ending or empty-file mismatch: '+repr(body))
            lua('vim.cmd("silent write!")')
            if (repo/'a.md').read_bytes()!=body.encode():failures.append('snapshot save changed exact bytes: '+repr(body))
        slow.touch()

        # An external checkout may replace the file at the same path, but its
        # different branch must never reload into this clean activation.
        slow.unlink();git('checkout','--','a.md');git('checkout','-q','review/b');slow.touch()
        lua('vim.api.nvim_exec_autocmds("FocusGained",{})');time.sleep(1.5)
        if lua('vim.api.nvim_buf_get_lines(0,0,1,false)[1]')!='external same branch':failures.append('different branch reloaded active document')
        slow.unlink();git('checkout','-q','review/a');slow.touch()
        if calls.exists():calls.write_text('')
        duration=float(lua('(function() vim.api.nvim_buf_set_lines(0,0,-1,false,{"edited"}); local now=vim.uv.hrtime(); vim.api.nvim_exec_autocmds("TextChanged",{buffer=0}); return (vim.uv.hrtime()-now)/1e6 end)()'))
        if duration>=250:failures.append('TextChanged blocked for '+str(duration)+'ms')
        # Repeated edits during the same slow observation must share the work.
        for i in range(4):lua('vim.api.nvim_exec_autocmds("TextChangedI",{buffer=0})')
        time.sleep(1.5)
        scans=len(calls.read_text().splitlines()) if calls.exists() else 0
        if scans>2:failures.append('uncoalesced observation scans: '+str(scans))
        # Focus events may render chrome but must not synchronously scan history.
        duration=float(lua('(function() local now=vim.uv.hrtime(); vim.api.nvim_exec_autocmds("FocusGained",{}); return (vim.uv.hrtime()-now)/1e6 end)()'))
        if duration>=250:failures.append('FocusGained blocked for '+str(duration)+'ms')
        assert not failures,'; '.join(failures)
        print('review-observation-test ok')
    finally:
        slow.unlink(missing_ok=True)
        pane.terminate()
        try:pane.wait(timeout=5)
        except subprocess.TimeoutExpired:pane.kill();pane.wait()
PY
