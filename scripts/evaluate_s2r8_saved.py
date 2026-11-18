#!/usr/bin/env python3
"""Evaluate only complete frozen training runs; retain the hot-restore error."""
import concurrent.futures,hashlib,json,os,pathlib,subprocess,time
ROOT=pathlib.Path(__file__).resolve().parents[1];BASE=ROOT.parent;WORK=BASE/'r8-confirmatory';SEEDS=[37,83,151,211,307,419]
def evaluate(seed):
 env=os.environ.copy();env.update(R8_SOURCE=str(BASE/'r2-source'),R8_WORK=str(WORK),R8_SEED=str(seed),GOMAXPROCS='2')
 meta={'seed':seed,'started_unix':time.time(),'binary_sha256':hashlib.sha256((BASE/'r8.test').read_bytes()).hexdigest(),'frozen_training_failures_retained':True}
 with (WORK/'execution'/f'FinalEvaluation-{seed}.log').open('w') as f:p=subprocess.run([str(BASE/'r8.test'),'-test.run','^TestS2R8Evaluate$','-test.v','-test.timeout','15m'],cwd=ROOT/'services/ai/runtime',env=env,stdout=f,stderr=subprocess.STDOUT)
 meta.update(returncode=p.returncode,finished_unix=time.time());(WORK/'execution'/f'FinalEvaluation-{seed}.json').write_text(json.dumps(meta,indent=2));print('evaluation',seed,p.returncode,flush=True);return p.returncode
def main():
 for _ in range(240):
  files=list((WORK/'execution').glob('TestS2R8Train*.json'))
  if len(files)==24:
   # Seed-level compression follows the last command; wait until its source
   # files are compressed before any evaluator makes a private working copy.
   if all(not (WORK/f'seed-{s}/training-observer/events.jsonl').exists() for s in SEEDS):break
  time.sleep(5)
 else:raise SystemExit('training completion timeout')
 for file in files:
  metadata=json.loads(file.read_text());log=file.with_suffix('.log').read_text();resource=WORK/f"seed-{metadata['seed']}"/metadata['arm']/'RESOURCE_RECOVERY.json'
  recovered=resource.exists() and json.loads(resource.read_text()).get('four_scientific_checkpoints_match_exactly') is True
  if metadata['returncode']!=1 or ('native checkpoint restore requires a fresh detached runtime' not in log and not recovered):raise SystemExit(f'independent failure needs diagnosis: {file}')
 start=WORK/'FINAL_EVALUATION_STARTED.json';assert not start.exists();start.write_text(json.dumps({'started_unix':time.time(),'selected_block':4,'selection_by_score':False,'failure_scope':'only frozen unsupported hot-restore test; separate companion supplies recovery evidence'},indent=2))
 with concurrent.futures.ThreadPoolExecutor(max_workers=3) as ex:results=list(ex.map(evaluate,SEEDS))
 raise SystemExit(max(results))
if __name__=='__main__':main()
