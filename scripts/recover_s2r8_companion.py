#!/usr/bin/env python3
"""Replay saved results; preserve every frozen hot-rollback harness failure."""
import argparse,hashlib,json,os,pathlib,shutil,subprocess,time
ROOT=pathlib.Path(__file__).resolve().parents[1]
def main():
 p=argparse.ArgumentParser();p.add_argument('--source',type=pathlib.Path,required=True);p.add_argument('--work',type=pathlib.Path,required=True);p.add_argument('--binary',type=pathlib.Path,required=True);a=p.parse_args();a.source=a.source.resolve();a.work=a.work.resolve();a.binary=a.binary.resolve();done=set()
 for tick in range(360):
  for metadata in sorted((a.work/'execution').glob('TestS2R8Train*.json')):
   v=json.loads(metadata.read_text());seed,arm=v['seed'],v['arm'];key=f'{seed}-{arm}'
   if key in done:continue
   d=a.work/f'seed-{seed}'/arm
   if (d/'RECOVERY_COMPANION.json').exists():done.add(key);continue
   if not all((d/(name+'.zst')).exists() and not (d/name).exists() for name in ['final.json','continued.json','events.jsonl']):continue
   if shutil.disk_usage(a.work).free<300000000:continue
   env=os.environ.copy();env.update(R8_SOURCE=str(a.source),R8_WORK=str(a.work),R8_SEED=str(seed),R8_RECOVERY_ARM=arm,GOMAXPROCS='2')
   meta={'seed':seed,'arm':arm,'binary_sha256':hashlib.sha256(a.binary.read_bytes()).hexdigest(),'started_unix':time.time(),'frozen_training_returncode':v['returncode']}
   log=a.work/'execution'/f'RecoveryCompanion-{key}.log'
   with log.open('w') as f:r=subprocess.run([str(a.binary),'-test.run','^TestS2R8RecoveryCompanion$','-test.v','-test.timeout','15m'],cwd=ROOT/'services/ai/runtime',env=env,stdout=f,stderr=subprocess.STDOUT)
   meta.update(returncode=r.returncode,finished_unix=time.time());(a.work/'execution'/f'RecoveryCompanion-{key}.json').write_text(json.dumps(meta,indent=2));done.add(key);print(key,r.returncode,round(meta['finished_unix']-meta['started_unix'],2),flush=True)
   if r.returncode:raise SystemExit(r.returncode)
  if len(done)==24:return
  time.sleep(5)
 raise SystemExit('recovery companion timed out waiting for completed models')
if __name__=='__main__':main()
