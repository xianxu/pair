#!/usr/bin/env bash
# Real Git + real review Neovim RPC; no editor or Git behavior mocked.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PAIR_ROOT="$ROOT" python3 - <<'PY'
import json, os, pathlib, subprocess, tempfile, time
root=pathlib.Path(os.environ['PAIR_ROOT'])
with tempfile.TemporaryDirectory(prefix='pair-branch-review-') as td:
    temp=pathlib.Path(td).resolve(); repo=temp/'repo'; repo.mkdir(); data=temp/'data'; data.mkdir()
    def git(*args): return subprocess.check_output(['git','-C',str(repo),*args],text=True).strip()
    git('init','-q','-b','main'); git('config','user.email','t@e.com'); git('config','user.name','T')
    (repo/'a.md').write_text('A\n'); (repo/'b.md').write_text('B\n')
    git('add','.'); git('commit','-qm','initial')
    for name in ['a','b']:
        git('checkout','-qb','review/'+name,'main')
        (repo/(name+'.md')).write_text(name.upper()+' reviewed\n')
        records=[dict(old=name.upper(),new=name.upper()+' reviewed',occurrence=1,new_occurrence=1,explain=name+' explanation')]
        msg='review('+name+'): agent r1 — first\n\n```review-records\n'+json.dumps(records)+'\n```\n'
        (temp/'msg').write_text(msg); git('add','.'); git('commit','-qF',str(temp/'msg'))
    git('checkout','-q','review/a')
    def resolve(): return json.loads(subprocess.check_output([str(root/'bin/pair'),'review','readiness','--resolve',str(repo)],text=True))
    opened=data/'review.open'; handoff=data/'handoff.json'
    bindir=temp/'bin'; bindir.mkdir()
    (bindir/'zellij').write_text('#!/usr/bin/env python3\nimport sys,os,json\nfrom pathlib import Path\np=Path(os.environ["HOST_LOG"])\nwith p.open("a") as f: f.write(" ".join(sys.argv[1:])+"\\n")\nif sys.argv[1:]==["action","are-floating-panes-visible"]: print("true")\nelif sys.argv[1:3]==["action","list-panes"]: print(json.dumps({"panes":[{"id":3,"terminal_command":"nvim -u /pair/nvim/init.lua"}]}))\n')
    (bindir/'zellij').chmod(0o755)
    env=dict(os.environ,PATH=str(bindir)+':'+str(root/'bin')+':'+os.environ['PATH'],HOST_LOG=str(temp/'host.log'),PAIR_HOME=str(root),PAIR_DATA_DIR=str(data),PAIR_TAG='restore',PAIR_SESSION_ID='session',
        PAIR_REVIEW_OPEN_PATH=str(opened),PAIR_REVIEW_HANDOFF_PATH=str(handoff),PAIR_REVIEW_LANDED_PATH=str(data/'landed.json'),
        PAIR_REVIEW_MODE_PATH=str(data/'mode'),XDG_STATE_HOME=str(temp/'state'),XDG_DATA_HOME=str(temp/'xdg'),XDG_CACHE_HOME=str(temp/'cache'))
    children=[]; logs=[]
    def launch():
        e=dict(env,PAIR_REVIEW_IDENTITY=json.dumps(resolve()))
        log=open(temp/('nvim-'+str(len(children))+'.log'),'w'); logs.append(log)
        p=subprocess.Popen(['nvim','--headless','-u',str(root/'nvim/review.lua'),str(repo/'a.md')],cwd=repo,env=e,stdout=log,stderr=log); children.append(p)
        deadline=time.monotonic()+8
        while time.monotonic()<deadline:
            if opened.exists() and len(opened.read_text().splitlines())>=3: return json.loads(opened.read_text().splitlines()[2])
            if p.poll() is not None: break
            time.sleep(.025)
        raise AssertionError('pane did not publish readiness: '+(temp/('nvim-'+str(len(children)-1)+'.log')).read_text())
    def evaluate(meta,expr):
        return subprocess.check_output(['nvim','--server',meta['endpoint'],'--remote-expr',expr],env=env,text=True,timeout=8).strip()
    def lua(meta,code):
        return evaluate(meta,"luaeval('"+code.replace("'","''")+"')")
    def restore(meta,observed):
        req=json.dumps(dict(token=meta['token'],session='session',identity=observed))
        return json.loads(evaluate(meta,"luaeval('vim.json.encode(PairReviewPane.restore(_A))', '"+req.replace("'","''")+"')"))
    def snapshot(meta):
        return json.loads(lua(meta,'vim.json.encode({file=vim.api.nvim_buf_get_name(0),text=vim.api.nvim_buf_get_lines(0,0,-1,false),marks=#vim.api.nvim_buf_get_extmarks(0,vim.api.nvim_create_namespace("review"),0,-1,{})})'))
    try:
        meta=launch()
        draft=temp/'draft.md'; draft.write_text('draft\n'); draft_socket=str(temp/'draft.sock')
        log=open(temp/'draft.log','w'); logs.append(log)
        d=subprocess.Popen(['nvim','--headless','--listen',draft_socket,'-u',str(root/'nvim/init.lua'),str(draft)],cwd=repo,
            env=dict(env,PAIR_DRAFT_PATH=str(draft)),stdout=log,stderr=log); children.append(d)
        deadline=time.monotonic()+8
        while not pathlib.Path(draft_socket).exists() and time.monotonic()<deadline: time.sleep(.025)
        draft_meta={'endpoint':draft_socket}
        def toggle():
            lua(draft_meta,'PairReviewToggle()')
            deadline=time.monotonic()+8
            while time.monotonic()<deadline:
                if lua(draft_meta,'_pair_review.client.busy')=='false': return
                time.sleep(.025)
            raise AssertionError('Alt+C did not complete')
        first=json.loads(opened.read_text().splitlines()[2])['context']
        assert snapshot(meta)['text']==['A reviewed'] and snapshot(meta)['marks']>0
        for name in ['b','a']:
            git('checkout','-q','review/'+name)
            toggle()
            snap=snapshot(meta); assert snap['file']==str(repo/(name+'.md')) and snap['text']==[name.upper()+' reviewed'] and snap['marks']>0,snap
            if name=='b':
                attempt='select(1,pcall(vim.api.nvim_buf_call,vim.fn.bufnr('+json.dumps(str(repo/'a.md'))+'),function() vim.cmd("write") end))'
                assert lua(meta,attempt)=='false','inactive retained buffer wrote into another checkout'
                assert (repo/'a.md').read_text()=='A\n'
        # A late response from A's previous activation must survive unconsumed.
        handoff.write_text(json.dumps(dict(context=first,records=[dict(old='A reviewed',new='WRONG',occurrence=1)])))
        time.sleep(.2); assert handoff.exists() and snapshot(meta)['text']==['A reviewed']
        handoff.unlink()
        # A completed agent round is no longer pending even if branch switching
        # happens before the pane has observed its commit.
        active=json.loads(opened.read_text().splitlines()[2])['context']
        handoff.write_text(json.dumps(dict(context=active,records=[dict(old='A reviewed',new='A final',occurrence=1,explain='final')])) )
        deadline=time.monotonic()+5
        while not (data/'landed.json').exists() and time.monotonic()<deadline: time.sleep(.025)
        landed=json.loads((data/'landed.json').read_text())
        assert landed['context']==active
        (temp/'msg').write_text('review(a): agent r2 — final\n\n'+landed['body']+'\n')
        git('add','a.md'); git('commit','-qF',str(temp/'msg'))
        git('checkout','-q','review/b'); toggle(); assert snapshot(meta)['text']==['B reviewed']
        git('checkout','-q','review/a'); toggle(); assert snapshot(meta)['text']==['A final']
        # Unsaved edits block retargeting and must survive an orderly mismatched exit.
        lua(meta,'vim.api.nvim_buf_set_lines(0,0,-1,false,{"human unsaved"})')
        git('checkout','-q','review/b')
        response=restore(meta,resolve()); assert not response['ok'] and 'unsaved' in response['error'],response
        assert (repo/'a.md').read_text()=='A\n'
        recovery_dir=data/'review-recovery'; recovery_backup=data/'recovery-backup'
        if recovery_dir.exists(): recovery_dir.rename(recovery_backup)
        recovery_dir.symlink_to(repo,target_is_directory=True)
        lua(meta,'vim.schedule(function() vim.cmd("qa") end)')
        time.sleep(.2)
        assert children[0].poll() is None,'failed recovery did not block orderly quit'
        assert (repo/'a.md').read_text()=='A\n'
        recovery_dir.unlink()
        if recovery_backup.exists(): recovery_backup.rename(recovery_dir)
        lua(meta,'vim.schedule(function() vim.cmd("qa!") end)')
        children[0].wait(timeout=5)
        assert (repo/'a.md').read_text()=='A\n','mismatched exit overwrote checkout'
        snapshots=list((data/'review-recovery').glob('*.json')); assert len(snapshots)==1
        assert json.loads(snapshots[0].read_text())['lines']==['human unsaved']
        git('checkout','-q','review/a'); meta=launch()
        lua(meta,'vim.cmd("PairReviewRecover")'); assert snapshot(meta)['text']==['human unsaved']
        lua(meta,'vim.cmd("write")'); assert not snapshots[0].exists()
        assert (repo/'a.md').read_text()=='human unsaved\n'
        print('review-branch-restore-test ok')
    finally:
        for p in children:
            if p.poll() is None:
                p.terminate()
                try: p.wait(timeout=5)
                except subprocess.TimeoutExpired: p.kill(); p.wait()
        for log in logs: log.close()
PY
