#!/usr/bin/env python3
"""Seal source/budgets before training, then all final checkpoints before holdout."""
import argparse,hashlib,json,subprocess
from pathlib import Path
import numpy as np
import analyze_s2r5_retention as history
ROOT=Path(__file__).resolve().parents[1]
SEEDS=[37,83,151,211,307,419]
ARMS=['A','A0','B','B0','BI','BI0','C','F']
def sha(p):
 h=hashlib.sha256()
 with p.open('rb') as f:
  while b:=f.read(1<<20):h.update(b)
 return h.hexdigest()
def write(p,v):
 with p.open('x') as f:json.dump(v,f,indent=2,sort_keys=True);f.write('\n')
def main():
 p=argparse.ArgumentParser();p.add_argument('--work',type=Path,required=True);p.add_argument('--binary',type=Path,required=True);p.add_argument('--stage',choices=['training','confirmatory'],required=True);a=p.parse_args();w=a.work.resolve()
 if (w/'FINAL_STARTED').exists():raise RuntimeError('final already started')
 if subprocess.check_output(['git','status','--porcelain'],cwd=ROOT,text=True).strip():raise RuntimeError('commit scientific source first')
 files={}
 for name in subprocess.check_output(['git','ls-files'],cwd=ROOT,text=True).splitlines():
  if name.endswith('.go') or name in ['go.mod','go.sum','AGENTS.md'] or name.startswith('scripts/') and name.endswith('.py') or name.startswith('docs/research/s2_r6_native_utility_20261118/'):
   files[str(ROOT/name)]=sha(ROOT/name)
 for name in ['00-source.json','source-events.jsonl','SOURCE_VERIFICATION.json','ORIGINAL_WITNESSES.json','DEVELOPMENT_DECISION.json','RETENTION_LOCK.json','anchors/rows.json','coverage/normal/rows.json','coverage/shifted/rows.json']:files[str(w/name)]=sha(w/name)
 if a.stage=='training':
  dev=history.read(w/'development/seed-37/observer/rows.json')[:128];x,_,_,_=history.arrays(dev);sd=x.std(0);sd=np.where(sd>1e-10,sd,1.)
  metric={'mean':x.mean(0).tolist(),'sd':sd.tolist(),'scaling_source':'development seed37 Fit streams0-7 only','neighbors':12,'permutation_seeds':list(range(181930001,181930011)),'history_gate':.10,'IID_equivalence_margin':.03,'calibration_ratio':[.8,1.2],'H1_gain':.10,'H1_class_threshold':1e-5,'H1_macroF1':.40,'H1_bootstrap':10000,'history_bootstrap':5000}
  write(w/'METRIC_LOCK.json',metric);files[str(w/'METRIC_LOCK.json')]=sha(w/'METRIC_LOCK.json')
 else:
  for seed in SEEDS:
   for arm in ARMS:
    d=w/f'training/seed-{seed}/{arm}'
    for name in ['final.json','events.jsonl','receipts.json','costs.json']:files[str(d/name)]=sha(d/name)
  constants={}
  for seed in SEEDS:
   d=w/f'training/seed-{seed}';fit=[]
   for b in range(4):fit+=history.read(d/f'A/block-{b}-fit.json')
   constants[str(seed)]={'history_constant':float(np.mean([r['Trial']['Error'] for r in fit])),'Fit_ids':[r['Trial']['ID'] for r in fit]}
  write(w/'OFFLINE_BASELINE_LOCK.json',constants)
  for name in ['METRIC_LOCK.json','TRAINING_LOCK.json','OFFLINE_BASELINE_LOCK.json']:files[str(w/name)]=sha(w/name)
 files[str(a.binary.resolve())]=sha(a.binary.resolve())
 lock={'SourceCommit':subprocess.check_output(['git','rev-parse','HEAD'],cwd=ROOT,text=True).strip(),'SourceTree':subprocess.check_output(['git','rev-parse','HEAD^{tree}'],cwd=ROOT,text=True).strip(),'Files':files,'seeds':SEEDS,'arms':ARMS,'H2_H3_H4':'not run: H1 development prerequisite failed','source_sequence':80735,'source_checksum':'9016b845eed4e8437e87ada99b9641718a086c570e246dd0a62e57dffd1ce9d3','fixed_features':32,'no_main_change':True,'no_final_selection':True,'phase':a.stage,'final_history_seeds':[181600017,181610017,181620017],'final_old_seeds':[181700031,181710031]}
 write(w/('TRAINING_LOCK.json' if a.stage=='training' else 'CONFIRMATORY_LOCK.json'),lock)
 print(a.stage,'sealed',len(files),'files',lock['SourceCommit'])
if __name__=='__main__':main()
