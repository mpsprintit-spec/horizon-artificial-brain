#!/usr/bin/env python3
"""Recovery companion: preserve frozen source and run fixture oracles from runtime.
The original runner's cwd=repo root cannot resolve historical oracle fixtures.
No acceptance rule, learned checkpoint, archived command or fitting is changed.
"""
import argparse,hashlib,json,os,shutil,subprocess
from pathlib import Path
ARMS=list('ABCDEFG')+['history-masked','iid','iid-masked']
def digest(path):
 h=hashlib.sha256();compressed=not path.exists();actual=Path(str(path)+'.zst') if compressed else path
 proc=subprocess.Popen(['zstd','-q','-dc',str(actual)],stdout=subprocess.PIPE) if compressed else None
 f=proc.stdout if proc else actual.open('rb')
 with f:
  for b in iter(lambda:f.read(1<<20),b''):h.update(b)
 if proc and proc.wait():raise RuntimeError('decompression failed '+str(actual))
 return h.hexdigest()
def install(src,dst):
 dst.parent.mkdir(parents=True,exist_ok=True)
 if src.exists():shutil.copy2(src,dst)
 else:
  with dst.open('xb') as out:subprocess.run(['zstd','-q','-dc',str(src)+'.zst'],stdout=out,check=True)
def main():
 p=argparse.ArgumentParser();p.add_argument('--source',type=Path,required=True);p.add_argument('--work',type=Path,required=True);p.add_argument('--repo',type=Path,default=Path(__file__).resolve().parents[1]);p.add_argument('--binary',type=Path,required=True);p.add_argument('--audit-binary',type=Path);p.add_argument('--audit-provenance',type=Path);p.add_argument('--revision-audit-binary',type=Path);p.add_argument('--revision-provenance',type=Path);p.add_argument('--verify-only',action='store_true');p.add_argument('--seed',type=int,choices=[37,83,151,211,307,419],required=True);g=p.add_mutually_exclusive_group(required=True);g.add_argument('--destination',type=Path);g.add_argument('--in-place',action='store_true');a=p.parse_args();w,root,binary,source=a.work.resolve(),a.repo.resolve(),a.binary.resolve(),a.source.resolve()
 if source==w or source in w.parents or w in source.parents:p.error('isolated source/work required')
 lock=json.loads((w/'CONFIRMATORY_LOCK.json').read_text())
 for old,expected in lock['Files'].items():
  if old.endswith('/bin/r7.test'):f=binary
  elif '/horizon/' in old:f=root/old.split('/horizon/',1)[1]
  elif '/r7/' in old:f=w/old.split('/r7/',1)[1]
  else:raise RuntimeError('unmapped frozen path '+old)
  if digest(f)!=expected:raise RuntimeError('frozen bytes changed: '+str(f))
 if a.verify_only:
  print('all',len(lock['Files']),'frozen payload hashes match');return
 if a.revision_audit_binary and not a.audit_binary:p.error('--revision-audit-binary requires --audit-binary')
 execution_binary=binary;test_name='TestS2R7Persistence'
 if a.audit_binary:
  provenance=a.audit_provenance or w.parent/'evidence/AUDIT_COMPANION_PROVENANCE.json';meta=json.loads(provenance.read_text());execution_binary=a.audit_binary.resolve()
  if digest(execution_binary)!=meta['audit_binary_sha256']:raise RuntimeError('audit binary differs from provenance')
  test_name='TestS2R7PersistenceAudit'
 target=w
 if a.destination:
  target=a.destination.resolve()
  if target.exists() or w in target.parents or source in target.parents:p.error('destination must be new, outside source/archive work')
  for arm in ARMS:
   for name in ['final.json','events.jsonl']:install(w/f'confirmatory/seed-{a.seed}/{arm}/{name}',target/f'confirmatory/seed-{a.seed}/{arm}/{name}')
  if a.revision_audit_binary:
   for family in ['history-observer','iid-observer']:install(w/f'confirmatory/seed-{a.seed}/{family}/observer-final.json',target/f'confirmatory/seed-{a.seed}/{family}/observer-final.json')
 env=os.environ.copy();env.update(GOMAXPROCS='2',HORIZON_S2R7_SOURCE=str(source),HORIZON_S2R7_WORK=str(target),HORIZON_S2R7_PHASE='confirmatory',HORIZON_S2R7_SEED=str(a.seed),HORIZON_S2R7_BUDGET='512')
 log=target/f'RECOVERY{"-AUDIT" if a.audit_binary else ""}-{a.seed}.log'
 with log.open('x') as out:r=subprocess.run([str(execution_binary),'-test.run','^'+test_name+'$','-test.v','-test.timeout','60m'],cwd=root/'services/ai/runtime',env=env,stdout=out,stderr=subprocess.STDOUT)
 text=log.read_text()
 if r.returncode or '--- PASS:' not in text or '--- SKIP:' in text:raise RuntimeError(str(log))
 print(str(log),flush=True)
 if a.revision_audit_binary:
  meta=json.loads((a.revision_provenance or w.parent/'evidence/REVISION_TIME_AUDIT_PROVENANCE.json').read_text());revision_binary=a.revision_audit_binary.resolve()
  if digest(revision_binary)!=meta['binary_sha256']:raise RuntimeError('revision audit binary differs from provenance')
  revision_log=target/f'RECOVERY-REVISION-TIME-{a.seed}.log'
  with revision_log.open('x') as out:r=subprocess.run([str(revision_binary),'-test.run','^TestS2R7RevisionTimeAudit$','-test.v','-test.timeout','60m'],cwd=root/'services/ai/runtime',env=env,stdout=out,stderr=subprocess.STDOUT)
  if r.returncode or '--- PASS:' not in revision_log.read_text():raise RuntimeError(str(revision_log))
  print(str(revision_log),flush=True)
if __name__=='__main__':main()
