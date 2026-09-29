#!/usr/bin/env bash
# Slow Git observation must not block editing or accumulate activation callbacks.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PAIR_ROOT="$ROOT" python3 - <<'PY'
import json,os,pathlib,subprocess,sys,tempfile,time
root=pathlib.Path(os.environ['PAIR_ROOT'])
with tempfile.TemporaryDirectory(prefix='pair-observation-') as td:
    tmp=pathlib.Path(td).resolve();repo=tmp/'repo';repo.mkdir();data=tmp/'data';data.mkdir();home=tmp/'home';(home/'bin').mkdir(parents=True)
    def git(*args):return subprocess.check_output(['git','-C',str(repo),*args],text=True,timeout=5).strip()
    git('init','-qb','main');git('config','user.name','T');git('config','user.email','t@e.com')
    for name in ('a','b'):(repo/(name+'.md')).write_text(name+'\n')
    git('add','.');git('commit','-qm','initial')
    for name in ('a','b'):
        git('checkout','-qb','review/'+name,'main');(repo/(name+'.md')).write_text(name+' reviewed\n')
        git('commit','-qam','review('+name+'): human r1')
    git('checkout','-q','review/a')
    def resolve():return json.loads(subprocess.check_output([str(root/'bin/pair'),'review','readiness','--resolve',str(repo)],text=True,timeout=4))
    slow=tmp/'slow';calls=tmp/'calls'
    wrapper=home/'bin/pair'
    wrapper.write_text('#!'+sys.executable+'\n'+'''import os,sys,time
from pathlib import Path
if sys.argv[1:4]==['review','readiness','--resolve'] and Path(os.environ['SLOW_FILE']).exists():
 with Path(os.environ['CALLS_FILE']).open('a') as f:f.write('scan\\n')
 time.sleep(1)
os.execv(os.environ['REAL_PAIR'],[os.environ['REAL_PAIR'],*sys.argv[1:]])
''');wrapper.chmod(0o700)
    opened=data/'review.open'
    env=dict(os.environ,PAIR_HOME=str(home),REAL_PAIR=str(root/'bin/pair'),SLOW_FILE=str(slow),CALLS_FILE=str(calls),
        PAIR_DATA_DIR=str(data),PAIR_TAG='observation',PAIR_SESSION_ID='sid',PAIR_REVIEW_OPEN_PATH=str(opened),
        PAIR_REVIEW_HANDOFF_PATH=str(data/'handoff.json'),PAIR_REVIEW_LANDED_PATH=str(data/'landed.json'),
        PAIR_REVIEW_IDENTITY=json.dumps(resolve()),XDG_STATE_HOME=str(tmp/'state'),XDG_CACHE_HOME=str(tmp/'cache'),XDG_DATA_HOME=str(tmp/'xdg'))
    with (tmp/'pane.log').open('w') as log:
        pane=subprocess.Popen(['nvim','--headless','-u',str(root/'nvim/review.lua'),str(repo/'a.md')],cwd=repo,env=env,stdout=log,stderr=log)
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
        # Clean same-branch disk edits still reload, after an async observation.
        # The timed region is inside the editor; RPC startup is excluded.
        slow.touch()
        duration=float(lua('(function() local now=vim.uv.hrtime(); assert(PairReviewPane.apply_definition_result(vim.api.nvim_get_current_buf())==false); return (vim.uv.hrtime()-now)/1e6 end)()'))
        if duration>=250:failures.append('empty definition poll blocked for '+str(duration)+'ms')
        (repo/'a.md').write_text('external same branch\n')
        duration=float(lua('(function() local now=vim.uv.hrtime(); vim.api.nvim_exec_autocmds("FocusGained",{}); return (vim.uv.hrtime()-now)/1e6 end)()'))
        if duration>=250:failures.append('clean FocusGained blocked for '+str(duration)+'ms')
        end=time.monotonic()+4
        while time.monotonic()<end:
            if lua('vim.api.nvim_buf_get_lines(0,0,1,false)[1]')=='external same branch':break
            time.sleep(.025)
        else:failures.append('matching branch external edit did not reload')
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
