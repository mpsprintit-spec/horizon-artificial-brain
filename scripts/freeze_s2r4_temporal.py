#!/usr/bin/env python3
"""Seal exact scientific source/checkpoints before one-shot final acquisition."""
import argparse,hashlib,json,subprocess
from pathlib import Path
ROOT=Path(__file__).resolve().parents[1]
def sha(p):return hashlib.sha256(p.read_bytes()).hexdigest()
def main():
 p=argparse.ArgumentParser();p.add_argument('--work',type=Path,required=True);a=p.parse_args();w=a.work.resolve()
 if (w/'CONFIRMATORY_LOCK.json').exists() or (w/'FINAL_STARTED').exists():raise RuntimeError('final lock/acquisition already exists; never overwrite')
 if subprocess.check_output(['git','status','--porcelain'],cwd=ROOT,text=True).strip():raise RuntimeError('commit scientific source before freezing')
 tracked=subprocess.check_output(['git','ls-files'],cwd=ROOT,text=True).splitlines()
 files={str(ROOT/f):sha(ROOT/f) for f in tracked if f.endswith('.go') or f in ['go.mod','go.sum'] or ('s2r4' in f and f.endswith('.py')) or f in ['docs/research/s2_r4_temporal_20261117/PROTOCOL.md','docs/research/s2_r4_temporal_20261117/CONFIRMATORY_PROTOCOL.md','docs/research/s2_r4_temporal_20261117/FULL_STATE_PROTOCOL.md']}
 for name in ['TRAINING_LOCK.json','METRIC_LOCK.json','00-source.json','source-events.jsonl']:
  files[str(w/name)]=sha(w/name)
 checkpoints=[]
 for seed in [37,83,151]:
  for arm in ['A32','B32','C32','B0-32','D32','D0-32','A64','C64']:
   d=w/f'training/seed-{seed}'/arm
   for name in ['final.json','receipts.json','events.jsonl']:files[str(d/name)]=sha(d/name)
   checkpoints.append(str(d/'final.json'))
 result={'SourceCommit':subprocess.check_output(['git','rev-parse','HEAD'],cwd=ROOT,text=True).strip(),'Files':files,'fixed_final_checkpoints':checkpoints,'seeds':[37,83,151],'final_seeds':[3500037,3600037,3700037],'streams_per_condition':32,'steps':24,'all_arms_frozen_before_final':True,'protocol':['docs/research/s2_r4_temporal_20261117/PROTOCOL.md','docs/research/s2_r4_temporal_20261117/CONFIRMATORY_PROTOCOL.md','docs/research/s2_r4_temporal_20261117/FULL_STATE_PROTOCOL.md'],'model_selection_using_final':False,'retention_failures_are_ineligible_for_promotion':True}
 (w/'CONFIRMATORY_LOCK.json').write_text(json.dumps(result,indent=2,sort_keys=True)+'\n');print(result['SourceCommit'],len(files),'files sealed; final data not acquired')
if __name__=='__main__':main()
