from pathlib import Path
import json,os,subprocess,time
root=Path(__file__).resolve().parent
revision=(root/'integrated-revision').read_text().strip()
assert subprocess.check_output(['git','-C','/Users/xianxu/workspace/pair','rev-parse','HEAD'],text=True).strip()==revision
command=['go','test','-p','1','-overlay',str(root/'combined-overlay.json'),'./cmd/couch','-run','^TestCombinedStageMainWarmUpgrade$','-count=1','-v','-timeout=20m']
extra={'COMBINED_MAIN_SOURCE':str(root/'main'),'COMBINED_STAGE_SOURCE':'/Users/xianxu/workspace/pair','COMBINED_STAGE_REVISION':revision,'COMBINED_SCRATCH':'/Volumes/TestScratch/tmp/pair-slot4'}
meta={'command':command,'environment':extra,'pins':json.loads((root/'pins.json').read_text()),'stage':revision,'started':time.strftime('%Y-%m-%dT%H:%M:%SZ',time.gmtime())}
(root/'warm-metadata.json').write_text(json.dumps(meta,indent=2)+'\n')
started=time.monotonic()
with (root/'warm-upgrade.log').open('w') as log: result=subprocess.run(command,env={**os.environ,**extra},stdout=log,stderr=subprocess.STDOUT)
meta.update(exit_code=result.returncode,elapsed_seconds=round(time.monotonic()-started,3))
(root/'warm-metadata.json').write_text(json.dumps(meta,indent=2)+'\n')
print(json.dumps(meta,indent=2))
print((root/'warm-upgrade.log').read_text()[-16000:])
raise SystemExit(result.returncode)
