#!/usr/bin/env bash
. "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib/pair-test-bootstrap.sh"
# Delayed observation results must not block editing or accumulate callbacks.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PAIR_ROOT="$ROOT" python3 -B - <<'PY'
import json,os,pathlib,subprocess,sys,tempfile,time
root=pathlib.Path(os.environ['PAIR_ROOT'])
sys.path.insert(0,str(root/'tests/lib'))
from review_test_env import ReviewTestEnvironment
from review_observation_diagnostics import observe_wanted, identity_metadata
pane_init=pathlib.Path(os.environ.get('PAIR_TEST_REVIEW_INIT',str(root/'nvim/review.lua')))
with ReviewTestEnvironment() as isolated, tempfile.TemporaryDirectory(prefix='pair-observation-') as td:
    tmp=pathlib.Path(td).resolve();repo=tmp/'repo';repo.mkdir();data=tmp/'data';data.mkdir();home=tmp/'home';(home/'bin').mkdir(parents=True)
    base_env=isolated.environment(tmp,root)
    def git(*args):return subprocess.check_output(['git','-C',str(repo),*args],env=base_env,text=True,timeout=5).strip()
    git('init','-qb','main');git('config','user.name','T');git('config','user.email','t@e.com')
    for name in ('a','b'):(repo/(name+'.md')).write_text(name+'\n')
    git('add','.');git('commit','-qm','initial')
    for name in ('a','b'):
        git('checkout','-qb','review/'+name,'main');(repo/(name+'.md')).write_text(name+' reviewed\n')
        record=dict(old=name,new=name+' reviewed',occurrence=1,new_occurrence=1,explain=name+' snapshot explanation')
        (tmp/'msg').write_text('review('+name+'): agent r1\n\n```review-records\n'+json.dumps([record])+'\n```\n')
        git('commit','-qaF',str(tmp/'msg'))
    git('checkout','-q','review/a')
    phase='startup';wanted_history=[];requests=[];phase_evidence=[]
    wanted_binary=os.environ.get('PAIR_TEST_OBSERVATION_WANTED_BIN',str(root/'bin/pair'))
    def resolve():
        return observe_wanted([wanted_binary,'review','readiness','--resolve',str(repo)],
                              base_env,wanted_history,'wanted:'+phase)

    (home/'bin/pair').symlink_to(root/'bin/pair')
    opened=data/'review.open'
    env=dict(base_env,PAIR_HOME=str(home),PAIR_DATA_DIR=str(data),PAIR_TAG='observation',PAIR_SESSION_ID='sid',PAIR_REVIEW_OPEN_PATH=str(opened),
        PAIR_REVIEW_HANDOFF_PATH=str(data/'handoff.json'),PAIR_REVIEW_LANDED_PATH=str(data/'landed.json'),
        PAIR_REVIEW_IDENTITY=json.dumps(resolve()),XDG_STATE_HOME=str(tmp/'state'),XDG_CACHE_HOME=str(tmp/'cache'),XDG_DATA_HOME=str(tmp/'xdg'))
    trace_path=tmp/'pane-observations.json';trace_init=tmp/'observation-init.lua'
    trace_init.write_text('dofile('+json.dumps(str(pane_init))+')\n'+
        'dofile('+json.dumps(str(root/'tests/lib/review_observation_trace.lua'))+').install({path='+json.dumps(str(trace_path))+'})\n'+
        'dofile('+json.dumps(str(root/'tests/lib/review_observation_gate.lua'))+').install()\n')
    pane=None;meta=None

    def evaluate(expr):return subprocess.check_output(['nvim','--server',meta['endpoint'],'--remote-expr',expr],env=env,text=True,timeout=8).strip()
    def lua(code):return evaluate("luaeval('"+code.replace("'","''")+"')")
    def until(predicate, reason, timeout=6):
        end=time.monotonic()+timeout
        while time.monotonic()<end:
            value=predicate()
            if value:return value
            time.sleep(.025)
        raise AssertionError(reason)
    def gate():return json.loads(lua('vim.json.encode(PairReviewObservationGate:snapshot())'))
    def record(event,entry):
        phase_evidence.append(dict(phase=phase,event=event,**entry))
    def arm():
        def attempt():
            raw=lua('(function() local entries=PairReviewObservationGate:snapshot().entries; '
                    'local after=#entries>0 and entries[#entries].id or 0; '
                    'if not PairReviewObservationGate:arm() then return "" end; '
                    'return vim.json.encode({after=after}) end)()')
            return json.loads(raw) if raw else None
        return until(attempt,'previous observation did not settle')['after']
    def result(after):
        started=until(lambda: next((e for e in gate()['entries'] if e['id']>after),None),
                      'controlled resolver did not start')
        record('start',dict(id=started['id'],at_ns=started['started_ns'],timeout=started['timeout']))
        entry=until(lambda: next((e for e in gate()['entries'] if e['id']==started['id'] and e['state']=='result'),None),
                    'controlled resolver did not complete')
        record('result',entry)
        assert entry['code']==0 and entry['signal']==0, 'real controlled resolver failed: '+repr(entry)
        return entry
    def release(entry,keep=False):
        code=('' if keep else 'PairReviewObservationGate:disarm();')
        code+='assert(PairReviewObservationGate:release('+str(entry['id'])+'));'
        code+='vim.schedule(function() PairReviewObservationSettled='+str(entry['id'])+' end);return true'
        assert lua('(function() '+code+' end)()')=='true'
        released=next(e for e in gate()['entries'] if e['id']==entry['id'])
        record('release',dict(id=entry['id'],at_ns=released['released_ns']))
        until(lambda: lua('PairReviewObservationSettled=='+str(entry['id']))=='true','released callback did not settle')
        record('settled',dict(id=entry['id']))
    def applied(entry,predicate,reason):
        until(predicate,reason)
        record('applied',dict(id=entry['id']))
    def request(want):
        raw=json.dumps(dict(token=meta['token'],session='sid',identity=want))
        ack=json.loads(evaluate("luaeval('vim.json.encode(PairReviewPane.restore(_A))', '"+raw.replace("'","''")+"')"))
        requests.append(dict(phase=phase,wanted=identity_metadata(want),ack=dict(
            ok=ack.get('ok'),same=ack.get('same'),error=str(ack.get('error',''))[:2048],
            identity=identity_metadata(ack.get('identity')),context=identity_metadata(ack.get('context')))))
        return ack
    def evidence():
        fresh={'unavailable':'pane endpoint not published'}
        if pane is not None and pane.poll() is None and meta is not None:
            try:fresh=json.loads(lua('vim.json.encode(PairReviewObservationTraceSnapshot())'))
            except Exception as error:fresh={'unavailable':str(error)}
        if 'unavailable' in fresh and trace_path.exists():
            try:fresh=json.loads(trace_path.read_text())
            except Exception as error:fresh={'unavailable':'invalid final trace: '+str(error)}
        return dict(phase=phase,wanted=wanted_history,requests=requests,fresh=fresh,phase_evidence=phase_evidence,
                      returncode=pane.poll() if pane is not None else None,
                      pane_log=(tmp/'pane.log').read_text(errors='replace')[-16384:] if (tmp/'pane.log').exists() else '')
    try:
        with (tmp/'pane.log').open('w') as log:
            pane=subprocess.Popen(['nvim','--headless','-u',str(trace_init),str(repo/'a.md')],cwd=repo,env=env,stdout=log,stderr=log)
        end=time.monotonic()+6
        while time.monotonic()<end:
            if opened.exists() and len(opened.read_text().splitlines())>=3:break
            assert pane.poll() is None,(tmp/'pane.log').read_text();time.sleep(.025)
        meta=json.loads(opened.read_text().splitlines()[2]);failures=[]
        count=lambda:int(lua('#vim.api.nvim_get_autocmds({event="TextChanged",buffer=vim.api.nvim_get_current_buf()})'))
        before=count()
        for name in ('b','a','b','a'):
            phase='restore:'+name
            git('checkout','-q','review/'+name);ack=request(resolve());assert ack['ok'],ack
        after=count()
        if before!=after:failures.append('render callbacks accumulated: '+str((before,after)))
        # A completed identity observation is deliberately withheld across a
        # checkout. The callback must apply captured A bytes, never reread B.
        phase='withheld-observation-across-checkout'
        (repo/'a.md').write_text('a reviewed captured before checkout\n')
        start=arm();lua('vim.api.nvim_exec_autocmds("FocusGained",{})')
        captured=result(start)
        git('checkout','--','a.md');git('checkout','-q','review/b')
        release(captured)
        applied(captured,lambda: lua('vim.api.nvim_buf_get_lines(0,0,1,false)[1]')=='a reviewed captured before checkout',
                'late captured snapshot did not apply')
        displayed=lua('vim.api.nvim_buf_get_lines(0,0,1,false)[1]')
        if displayed!='a reviewed captured before checkout':failures.append('late observation loaded unbound bytes: '+repr(displayed))
        undo_marks=int(lua('(function() vim.cmd("undo"); vim.api.nvim_exec_autocmds("TextChanged",{buffer=0}); return #vim.api.nvim_buf_get_extmarks(0,vim.api.nvim_create_namespace("review"),0,-1,{}) end)()'))
        if lua('vim.api.nvim_buf_get_lines(0,0,1,false)[1]')!='a reviewed' or undo_marks<1:failures.append('snapshot undo lost original text or decorations')
        redo_marks=int(lua('(function() vim.cmd("redo"); vim.api.nvim_exec_autocmds("TextChanged",{buffer=0}); return #vim.api.nvim_buf_get_extmarks(0,vim.api.nvim_create_namespace("review"),0,-1,{}) end)()'))
        if lua('vim.api.nvim_buf_get_lines(0,0,1,false)[1]')!='a reviewed captured before checkout' or redo_marks<1:failures.append('snapshot redo lost captured text or decorations')
        if lua('select(1,pcall(vim.cmd,"write"))')!='false':failures.append('late snapshot allowed write into different checkout')
        if (repo/'a.md').read_text()!='a\n':failures.append('late snapshot changed checkout bytes')
        git('checkout','-q','review/a')
        # Clean same-branch disk edits still reload, after an async observation.
        # The timed region is inside the editor; RPC startup is excluded.
        phase='slow-same-branch-observation'
        duration=float(lua('(function() local now=vim.uv.hrtime(); assert(PairReviewPane.apply_definition_result(vim.api.nvim_get_current_buf())==false); return (vim.uv.hrtime()-now)/1e6 end)()'))
        if duration>=250:failures.append('empty definition poll blocked for '+str(duration)+'ms')
        (repo/'a.md').write_text('external same branch')
        start=arm()
        duration=float(lua('(function() local now=vim.uv.hrtime(); vim.api.nvim_exec_autocmds("FocusGained",{}); return (vim.uv.hrtime()-now)/1e6 end)()'))
        if duration>=250:failures.append('clean FocusGained blocked for '+str(duration)+'ms')
        refreshed=result(start);release(refreshed)
        applied(refreshed,lambda: lua('vim.api.nvim_buf_get_lines(0,0,1,false)[1]')=='external same branch'
                and lua('vim.bo.endofline')=='false',
                'matching branch external edit did not reload exact no-final-newline state')
        phase='snapshot-bytes'
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
            start=arm();lua('vim.api.nvim_exec_autocmds("FocusGained",{})')
            snapshot=result(start);release(snapshot)
            applied(snapshot,lambda: json.loads(lua('vim.json.encode(vim.api.nvim_buf_get_lines(0,0,1,false)[1])'))==first
                    and lua('vim.bo.endofline')==eol and lua('vim.bo.fileformat')==fmt and lua('vim.bo.bomb')==bomb,
                    'snapshot line ending or empty-file mismatch: '+repr(body))
            lua('vim.cmd("silent write!")')
            if (repo/'a.md').read_bytes()!=body.encode():failures.append('snapshot save changed exact bytes: '+repr(body))

        # An external checkout may replace the file at the same path, but its
        # different branch must never reload into this clean activation.
        phase='different-branch-must-not-reload'
        git('checkout','--','a.md');git('checkout','-q','review/b')
        start=arm();lua('vim.api.nvim_exec_autocmds("FocusGained",{})')
        different=result(start);release(different)
        if lua('vim.api.nvim_buf_get_lines(0,0,1,false)[1]')!='external same branch':failures.append('different branch reloaded active document')
        phase='coalesced-edits'
        git('checkout','-q','review/a');start=arm()
        duration=float(lua('(function() vim.api.nvim_buf_set_lines(0,0,-1,false,{"edited"}); local now=vim.uv.hrtime(); vim.api.nvim_exec_autocmds("TextChanged",{buffer=0}); return (vim.uv.hrtime()-now)/1e6 end)()'))
        if duration>=250:failures.append('TextChanged blocked for '+str(duration)+'ms')
        # The real process completed, but its held callback still owns the observer.
        first_result=result(start)
        for i in range(4):lua('vim.api.nvim_exec_autocmds("TextChangedI",{buffer=0})')
        assert len([e for e in gate()['entries'] if e['id']>start])==1,'parallel observation scans'
        release(first_result,keep=True)
        second_result=result(first_result['id']);release(second_result)
        scans=len([e for e in gate()['entries'] if e['id']>start])
        if scans!=2:failures.append('uncoalesced observation scans: '+str(scans))
        # Focus events may render chrome but must not synchronously scan history.
        phase='modified-focus-observation';start=arm()
        duration=float(lua('(function() local now=vim.uv.hrtime(); vim.api.nvim_exec_autocmds("FocusGained",{}); return (vim.uv.hrtime()-now)/1e6 end)()'))
        if duration>=250:failures.append('FocusGained blocked for '+str(duration)+'ms')
        focused=result(start);release(focused)
        assert not failures,'; '.join(failures)
        if os.environ.get('PAIR_TEST_OBSERVATION_REPORT'):
            report=evidence();report['outcome']='passed'
            pathlib.Path(os.environ['PAIR_TEST_OBSERVATION_REPORT']).write_text(json.dumps(report,sort_keys=True))
        print('review-observation-test ok')
    except BaseException:
        report=evidence();report['outcome']='failed'
        print('review observation failure: '+json.dumps(report,sort_keys=True),file=sys.stderr)
        raise
    finally:
        if pane is not None:
            pane.terminate()
            try:pane.wait(timeout=5)
            except subprocess.TimeoutExpired:pane.kill();pane.wait()

PY
