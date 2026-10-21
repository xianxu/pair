#!/usr/bin/env python3
"""One-shot public-CLI qualification. No retries and no direct Tart calls."""
import argparse, hashlib, json, os, subprocess, sys, time
from pathlib import Path

LITERAL = ['space argument', '$(touch NEVER)', '`touch NEVER`', '; exit 99', '', '界']
GUEST = r"""import json, os, re, subprocess, sys
from pathlib import Path
head, hostroot, *literal = sys.argv[1:]
root=Path.cwd()
assert literal == ['space argument', '$(touch NEVER)', '`touch NEVER`', '; exit 99', '', '界'], literal
allowed={'PATH','LANG','TERM','HOME','TMPDIR','XDG_CONFIG_HOME','XDG_DATA_HOME','XDG_CACHE_HOME','XDG_STATE_HOME','ZELLIJ_SOCKET_DIR','PAIR_DATA_DIR','COUCH_ISOLATED_ROOT','WF_SDLC_QUEUE_ROOT','PWD','SHLVL','_','LC_CTYPE','OLDPWD','__CF_USER_TEXT_ENCODING'}
assert set(os.environ) <= allowed, sorted(set(os.environ)-allowed)
assert not any('HOST-POISON' in v for v in os.environ.values())
# Darwin runtime synthesized values: strict format/UID, and guest-local prior cwd.
cf=os.environ.get('__CF_USER_TEXT_ENCODING')
if cf is not None:
    match=re.fullmatch(r'0x([0-9A-Fa-f]+):0x0:0x0',cf)
    assert match and int(match.group(1),16)==os.getuid(), ('CF encoding',cf)
oldpwd=os.environ.get('OLDPWD')
if oldpwd is not None:
    previous=Path(oldpwd)
    assert previous.is_absolute() and previous.is_dir() and str(previous)==str(previous.resolve()), ('OLDPWD',oldpwd)
    assert previous != Path(hostroot) and Path(hostroot) not in previous.parents
    assert str(previous) != '/Volumes/My Shared Files' and '/Volumes/My Shared Files' not in str(previous), ('shared OLDPWD',oldpwd)
assert not Path(hostroot).exists(), hostroot
home=Path(os.environ['HOME']); work=home.parent
for key,leaf in [('HOME','home'),('TMPDIR','tmp'),('XDG_CONFIG_HOME','config'),('XDG_DATA_HOME','data'),('XDG_CACHE_HOME','cache'),('XDG_STATE_HOME','state'),('ZELLIJ_SOCKET_DIR','sockets'),('PAIR_DATA_DIR','pair'),('COUCH_ISOLATED_ROOT','couch'),('WF_SDLC_QUEUE_ROOT','queue')]:
    p=Path(os.environ[key]); assert p == work/leaf and p.is_dir(), (key,str(p)); assert not p.is_symlink()
assert root == work/'checkout' and root != Path(hostroot)/'checkout'
assert (root/'.git').is_dir() and not (root/'.git').is_symlink()
assert not (root/'.git/objects/info/alternates').exists()
def git(*a): return subprocess.check_output(['git',*a],text=True).strip()
assert git('rev-parse','HEAD') == head
assert git('rev-parse','--git-common-dir') == '.git'
git('cat-file','-e','HEAD^{tree}')
assert (root/'tracked.txt').read_bytes() == b'original tracked bytes\n'
assert (root/'tracked-link').is_symlink() and os.readlink(root/'tracked-link') == 'tracked.txt'
assert (root/'tracked-link').read_bytes() == b'original tracked bytes\n'
assert not (root/'UNTRACKED-HOST-SENTINEL').exists()
assert not (root/'NEVER').exists()
(root/'tracked.txt').write_text('guest mutation\n')
assert (root/'tracked-link').read_text() == 'guest mutation\n'
print('QUALIFIED literal argv / private environment / independent Git / isolated mutation',flush=True)
print('QUALIFIED stderr separately',file=sys.stderr,flush=True)
"""


def limited(path, cap=65536):
    with path.open('rb') as f: raw=f.read(cap+1)
    assert len(raw)<=cap, f'oversized evidence: {path}'
    return raw


def main():
    ap=argparse.ArgumentParser(); ap.add_argument('--pair',required=True); args=ap.parse_args()
    pair=Path(args.pair).resolve(); assert pair.is_file() and os.access(pair,os.X_OK)
    base=Path(__file__).resolve().parent
    # Exclusive mkdir makes accidental rerunning a refusal, never another VM run.
    owned=base/'qualification'; owned.mkdir(mode=0o700)
    fixture=owned/'checkout'; fixture.mkdir()
    poison=owned/'HOST-POISON'; poison.mkdir()
    (poison/'credential-marker').write_text('dummy HOST-POISON; no credentials\n')
    (fixture/'tracked.txt').write_bytes(b'original tracked bytes\n')
    (fixture/'tracked-link').symlink_to('tracked.txt')
    env=dict(os.environ)
    for k in list(env):
        if k.startswith('GIT_'): del env[k]
    env.update(GIT_CONFIG_GLOBAL='/dev/null',GIT_CONFIG_NOSYSTEM='1')
    def git(*a): return subprocess.check_output(['git','-C',str(fixture),*a],env=env,text=True).strip()
    git('init','--quiet','--template='); git('add','tracked.txt','tracked-link')
    git('-c','user.name=Native fixture','-c','user.email=fixture@example.invalid','-c','commit.gpgsign=false','commit','--quiet','-m','private native fixture')
    head=git('rev-parse','HEAD')
    (fixture/'UNTRACKED-HOST-SENTINEL').write_text('HOST-POISON untracked\n')
    before=git('status','--porcelain=v1','--untracked-files=all')
    env.update(HOME=str(poison),COUCH_TREE='HOST-POISON-tree',COUCH_SESSION='HOST-POISON-session',COUCH_SLOT='HOST-POISON-slot',PAIR_DATA_DIR=str(poison/'pair'),PAIR_SESSION='HOST-POISON-session',ZELLIJ='HOST-POISON',ZELLIJ_SESSION_NAME='HOST-POISON',ZELLIJ_SOCKET_DIR=str(poison/'sockets'),AWS_SHARED_CREDENTIALS_FILE=str(poison/'credential-marker'),SSH_AUTH_SOCK=str(poison/'dummy-agent.sock'),GH_TOKEN='HOST-POISON-dummy-token',OPENAI_API_KEY='HOST-POISON-dummy-key',PAIR_QUALIFICATION_SENTINEL='HOST-POISON')
    rows=[]; report=owned/'qualification.json'
    def save(**extra):
        data=dict(version=1,pair=str(pair),fixture=str(fixture),head=head,pair_sha256=hashlib.sha256(pair.read_bytes()).hexdigest(),cases=rows,**extra)
        raw=json.dumps(data,indent=2).encode(); assert len(raw)<=65536
        report.write_bytes(raw+b'\n')
    def unchanged():
        assert (fixture/'tracked.txt').read_bytes()==b'original tracked bytes\n'
        assert os.readlink(fixture/'tracked-link')=='tracked.txt'
        assert (fixture/'UNTRACKED-HOST-SENTINEL').read_text()=='HOST-POISON untracked\n'
        assert git('rev-parse','HEAD')==head and git('status','--porcelain=v1','--untracked-files=all')==before
    def run(name,timeout,command,expected,check_streams=False):
        artifacts=owned/(name+'-artifacts')
        argv=[str(pair),'vm','run','--checkout',str(fixture),'--lane','adhoc','--timeout',timeout,'--artifacts',str(artifacts),'--',*command]
        started=time.monotonic(); row=dict(name=name,timeout=timeout,artifacts=str(artifacts),expected_exit=expected)
        rows.append(row); save(state='running')
        with (owned/(name+'-cli.stdout')).open('xb') as out,(owned/(name+'-cli.stderr')).open('xb') as err:
            rc=subprocess.call(argv,env=env,stdout=out,stderr=err)
        row.update(elapsed_seconds=round(time.monotonic()-started,3),exit_code=rc)
        unchanged(); row['source_unchanged']=True
        result=json.loads(limited(artifacts/'result.json')); row['result']=result
        # Missing/unknown cleanup is a hard stop before any following invocation.
        assert result.get('lane_claim_retained') is False, 'STOP: cleanup unknown; inspect retained claim, do not retry'
        assert rc==expected and result.get('exit_code')==expected, (rc,result)
        assert result.get('snapshot',{}).get('Head')==head, result
        if check_streams:
            assert limited(artifacts/'stdout.log')==b'QUALIFIED literal argv / private environment / independent Git / isolated mutation\n'
            assert limited(artifacts/'stderr.log')==b'QUALIFIED stderr separately\n'
            row['exact_streams']=True
        if name=='startup-timeout':
            assert limited(artifacts/'stdout.log')==b'', 'guest command reached: timeout not confined to pre-command phase'
            assert result.get('launch_pid',0)>0 and result.get('launch_start_identity'), 'native startup not established'
            assert result.get('command_requested') is False, 'guest execution already requested'
            row['scope']='native startup interrupted after recorded child identity and before guest execution' 
        row['verified']=True; save(state='between-cases')
    guest=['python3','-c',GUEST,head,str(owned),*LITERAL]
    try:
        run('success','4m',guest,0,True)
        run('exit37','4m',['python3','-c',GUEST+'\nraise SystemExit(37)\n',head,str(owned),*LITERAL],37,True)
        run('startup-timeout','8s',['python3','-c',"print('COMMAND-REACHED',flush=True)"],124)
        run('following-clean-success','4m',guest,0,True)
        save(state='passed'); print(report)
    except BaseException as exc:
        save(state='stopped',error=str(exc)[:4096]); print('STOP; no retry: '+str(report),file=sys.stderr); raise

if __name__=='__main__': main()
