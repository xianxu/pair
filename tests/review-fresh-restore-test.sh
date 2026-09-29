#!/usr/bin/env bash
# Fresh Alt+C crosses the real draft, Go opener, review process and RPC boundary.
# Only Zellij's process/visibility API is faked; the fake persists its children.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PAIR_ROOT="$ROOT" python3 - <<'PY'
import json, os, pathlib, signal, subprocess, sys, tempfile, time
root=pathlib.Path(os.environ['PAIR_ROOT'])
draft_init=pathlib.Path(os.environ.get('PAIR_TEST_DRAFT_INIT',str(root/'nvim/init.lua')))
with tempfile.TemporaryDirectory(prefix='pair-fresh-restore-') as td:
    temp=pathlib.Path(td).resolve(); repo=temp/'repo'; repo.mkdir(); bindir=temp/'bin'; bindir.mkdir()
    def git(*args):
        return subprocess.check_output(['git','-C',str(repo),*args],text=True,timeout=5).strip()
    git('init','-q','-b','main');git('config','user.name','T');git('config','user.email','t@e.com')
    (repo/'a.md').write_text('A\n');(repo/'b.md').write_text('B\n')
    git('add','.');git('commit','-qm','initial')
    for name in ('a','b'):
        git('checkout','-qb','review/'+name,'main')
        (repo/(name+'.md')).write_text(name.upper()+' reviewed\n')
        records=[dict(old=name.upper(),new=name.upper()+' reviewed',occurrence=1,new_occurrence=1,explain=name+' branch explanation')]
        message='review('+name+'): agent r1\n\n```review-records\n'+json.dumps(records)+'\n```\n'
        (temp/'msg').write_text(message);git('add','.');git('commit','-qF',str(temp/'msg'))
    def resolve():
        return json.loads(subprocess.check_output([str(root/'bin/pair'),'review','readiness','--resolve',str(repo)],text=True,timeout=4))
    stale_b=resolve();assert stale_b['file']=='b.md',stale_b
    git('checkout','-qb','review/missing','main')
    git('checkout','-qb','review/ambiguous','main')
    (repo/'a.md').write_text('ambiguous A\n');(repo/'b.md').write_text('ambiguous B\n')
    git('commit','-qam','review(ambiguous): human r1')

    # The real Go launcher executes this fake, which starts the exact requested
    # review command in a distinct headless process and immediately returns.
    fake=bindir/'zellij'
    fake.write_text('#!'+sys.executable+'\n'+r'''
import json,os,pathlib,subprocess,sys
state=pathlib.Path(os.environ['FAKE_ZELLIJ_STATE']);args=sys.argv[1:]
with (state/'calls.jsonl').open('a') as log: log.write(json.dumps(args)+'\n')
if args and args[0]=='run':
    cwd=args[args.index('--cwd')+1];command=args[args.index('--')+1:]
    assert pathlib.Path(command[0]).name=='nvim',command
    env=dict(os.environ,ZELLIJ_PANE_ID='42')
    if env.get('FAKE_WRONG_SESSION')=='1':env['PAIR_SESSION_ID']='wrong-conversation'
    with (state/'review.log').open('w') as log:
        child=subprocess.Popen([command[0],'--headless',*command[1:]],cwd=cwd,env=env,
            stdin=subprocess.DEVNULL,stdout=log,stderr=log,start_new_session=True)
    with (state/'children').open('a') as pids:pids.write(str(child.pid)+'\n')
    sys.exit(0)
if args[:2]==['action','show-floating-panes']:(state/'visible').write_text('true')
elif args[:2]==['action','hide-floating-panes']:(state/'visible').write_text('false')
elif args[:2]==['action','are-floating-panes-visible']:
    visible=(state/'visible').read_text();print(visible);sys.exit(0 if visible=='true' else 1)
elif args[:2]==['action','list-panes']:
    print(json.dumps({'panes':[{'id':3,'terminal_command':'nvim -u '+os.environ['PAIR_HOME']+'/nvim/init.lua'},
                              {'id':42,'terminal_command':'nvim -u '+os.environ['PAIR_HOME']+'/nvim/review.lua'},
                              {'id':4,'is_floating':False,'is_plugin':False,'terminal_command':'pair wrap'}]}))
''');fake.chmod(0o700)
    driver=temp/'draft-driver.lua'
    driver.write_text(r'''
local messages={}
local initial
_pair_review.client.opts.notify=function(message) messages[#messages+1]=message end
local ok,err=xpcall(function()
  assert(_pair_review.read_target()==nil,'fresh session adopted another conversation target')
  PairReviewToggle()
  assert(vim.wait(10000,function() return not _pair_review.client.busy end,10),'Alt+C transaction did not finish')
  if #messages==0 and _pair_review.read_target() then
    local meta=vim.json.decode(vim.fn.readfile(vim.env.PAIR_REVIEW_OPEN_PATH)[3])
    local snapshot="luaeval('vim.json.encode({marks=#vim.api.nvim_buf_get_extmarks(0,vim.api.nvim_create_namespace(\"review\"),0,-1,{}),diagnostics=vim.diagnostic.get(0)})')"
    local captured=vim.system({'nvim','--server',meta.endpoint,'--remote-expr',snapshot},{text=true}):wait(5000)
    assert(captured.code==0,vim.inspect(captured));initial=vim.json.decode(captured.stdout)
    local start="luaeval('(function() PairReviewPane.finish_human_turn(vim.api.nvim_get_current_buf(),vim.api.nvim_buf_get_name(0)); return PairReviewPane.controller.opts.pending() end)()')"
    local started=vim.system({'nvim','--server',meta.endpoint,'--remote-expr',start},{text=true}):wait(5000)
    assert(started.code==0 and started.stdout:find('agent request pending',1,true),vim.inspect(started))
    PairReviewToggle()
    assert(vim.wait(10000,function()return not _pair_review.client.busy end,10))
    assert(vim.fn.readfile(vim.env.FAKE_ZELLIJ_STATE..'/visible')[1]=='false','pending same-review toggle must hide')
    PairReviewToggle()
    assert(vim.wait(10000,function()return not _pair_review.client.busy end,10))
    assert(vim.fn.readfile(vim.env.FAKE_ZELLIJ_STATE..'/visible')[1]=='true','exit-1 false must reopen while agent works')
    assert(#messages==0,vim.inspect(messages))
  end
  vim.fn.writefile({vim.json.encode({messages=messages,initial=initial})},vim.env.DRIVER_RESULT)
end,debug.traceback)
if not ok then io.stderr:write(err..'\n');vim.cmd('cquit 1') end
vim.cmd('qa!')
''')
    def stop(pid):
        try:os.killpg(pid,signal.SIGTERM)
        except ProcessLookupError:return
        # The fake spawned an orphan; poll the process and bound final cleanup.
        end=time.monotonic()+2
        while time.monotonic()<end:
            try:os.kill(pid,0)
            except ProcessLookupError:return
            time.sleep(.025)
        try:os.killpg(pid,signal.SIGKILL)
        except ProcessLookupError:pass
    def run_case(name,branch,expected,wrong_session=False):
        if branch=='detached':git('checkout','--detach','-q','main')
        else:git('checkout','-q',branch)
        data=temp/name;data.mkdir();(data/'visible').write_text('false')
        target=data/'review-target-fresh.json';opened=data/'review-fresh.open'
        stale=dict(file=str(repo/'b.md'),status='ready',session='prior-conversation',identity=stale_b)
        if expected=='missing':
            # Even a valid receipt matching current HEAD cannot cross sessions.
            current=resolve();stale['identity']=dict(current,file='b.md',status='resolved')
        raw=json.dumps(stale);target.write_text(raw)
        draft=data/'draft.md';draft.write_text('fresh conversation draft\n')
        env=dict(os.environ,PATH=str(bindir)+os.pathsep+str(root/'bin')+os.pathsep+os.environ['PATH'],
            PAIR_HOME=str(root),PAIR_DATA_DIR=str(data),PAIR_TAG='fresh',PAIR_AGENT='claude',PAIR_SESSION_ID='fresh-conversation',
            PAIR_DRAFT_PATH=str(draft),PAIR_REVIEW_TARGET_PATH=str(target),PAIR_REVIEW_OPEN_PATH=str(opened),
            PAIR_REVIEW_HANDOFF_PATH=str(data/'handoff.json'),PAIR_REVIEW_LANDED_PATH=str(data/'landed.json'),
            PAIR_REVIEW_MODE_PATH=str(data/'mode'),PAIR_AGENT_CONFIG_PATH=str(data/'agent.json'),PAIR_AGENT_PID_PATH=str(data/'agent.pid'),
            PAIR_LAYOUT_MODE_PATH=str(data/'layout'),PAIR_ZELLIJ_ACTIONS_PATH=str(data/'actions.jsonl'),
            FAKE_ZELLIJ_STATE=str(data),FAKE_WRONG_SESSION='1' if wrong_session else '0',DRIVER_RESULT=str(data/'result.json'),
            ZELLIJ_PANE_ID='3',XDG_STATE_HOME=str(data/'xdg-state'),XDG_DATA_HOME=str(data/'xdg-data'),XDG_CACHE_HOME=str(data/'xdg-cache'))
        before=git('rev-parse','HEAD')
        try:
            result=subprocess.run(['nvim','--headless','-u',str(draft_init),str(draft),'-c','luafile '+str(driver)],
                cwd=repo,env=env,stdin=subprocess.DEVNULL,stdout=subprocess.PIPE,stderr=subprocess.STDOUT,text=True,timeout=15)
            assert result.returncode==0,(name,result.stdout)
            outcome=json.loads((data/'result.json').read_text())
            calls=[json.loads(line) for line in (data/'calls.jsonl').read_text().splitlines()] if (data/'calls.jsonl').exists() else []
            spawns=[call for call in calls if call[0]=='run']
            if expected!='resolved':
                assert resolve()['status']==expected,(name,resolve())
                assert not spawns and not opened.exists(),(name,calls,outcome)
                assert target.read_text()==raw and (data/'visible').read_text()=='false',(name,'refusal changed cache or visibility')
                if expected=='invalid':assert outcome['messages'],outcome
            elif wrong_session:
                assert len(spawns)==1,(name,calls,outcome)
                assert target.read_text()==raw and (data/'visible').read_text()=='false',(name,'unacknowledged pane published')
                assert any('another conversation' in message for message in outcome['messages']),outcome
            else:
                assert len(spawns)==1,(name,calls,outcome)
                assert spawns[0][-1]==str(repo/'a.md'),spawns
                published=json.loads(target.read_text());assert published['session']=='fresh-conversation',published
                assert published['file']==str(repo/'a.md') and published['identity']['branch']=='review/a',published
                assert (data/'visible').read_text()=='true',(calls,outcome)
                lines=opened.read_text().splitlines();meta=json.loads(lines[2])
                assert meta['session']=='fresh-conversation' and meta['context']['file']=='a.md',meta
                assert meta['context']['repo']==str(repo) and meta['context']['branch']=='review/a' and meta['context']['activation'],meta
                expr="luaeval('vim.json.encode({file=vim.api.nvim_buf_get_name(0),lines=vim.api.nvim_buf_get_lines(0,0,-1,false),marks=#vim.api.nvim_buf_get_extmarks(0,vim.api.nvim_create_namespace(\"review\"),0,-1,{}),diagnostics=vim.diagnostic.get(0)})')"
                snap=json.loads(subprocess.check_output(['nvim','--server',meta['endpoint'],'--remote-expr',expr],env=env,text=True,timeout=5))
                assert snap['file']==str(repo/'a.md') and snap['lines']==['A reviewed'] ,snap
                assert outcome['initial']['marks']>0,outcome
                assert any('a branch explanation' in d['message'] for d in outcome['initial']['diagnostics']),outcome
                assert not outcome['messages'],outcome
            assert git('rev-parse','HEAD')==before and not git('status','--porcelain'),(name,'Alt+C changed checkout')
            print('  ok  '+name)
        finally:
            if (data/'children').exists():
                for text in (data/'children').read_text().splitlines():stop(int(text))
    run_case('fresh-restores-history','review/a','resolved')
    run_case('wrong-conversation-ack-refused','review/a','resolved',True)
    run_case('missing-history-does-not-adopt-old-receipt','review/missing','missing')
    run_case('ambiguous-history-does-not-open','review/ambiguous','ambiguous')
    run_case('detached-identity-does-not-open','detached','invalid')
    print('review-fresh-restore-test ok')
PY
