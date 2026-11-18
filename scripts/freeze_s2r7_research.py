#!/usr/bin/env python3
"""Freeze scientific source/parameters before final training and acquisition."""
import argparse,hashlib,json,subprocess
from pathlib import Path
import numpy as np
ROOT=Path(__file__).resolve().parents[1];SEEDS=[37,83,151,211,307,419];ARMS=list('ABCDEFG')+['history-masked','iid','iid-masked']
def sha(p):
 h=hashlib.sha256()
 with Path(p).open('rb') as f:
  for b in iter(lambda:f.read(1<<20),b''):h.update(b)
 return h.hexdigest()
def write(p,x):
 with Path(p).open('x') as f:json.dump(x,f,indent=2,sort_keys=True);f.write('\n')
def main():
 p=argparse.ArgumentParser();p.add_argument('--work',type=Path,required=True);p.add_argument('--binary',type=Path,required=True);p.add_argument('--stage',choices=['training','confirmatory'],required=True);a=p.parse_args();w=a.work.resolve()
 if subprocess.check_output(['git','status','--porcelain'],cwd=ROOT,text=True).strip():raise RuntimeError('commit scientific source first')
 decision=json.loads((w/'DEVELOPMENT_DECISION.json').read_text());assert decision['final_budget']==512 and not decision['H1_pass']
 if a.stage=='training':
  rows=json.loads((w/'development/seed-37/history-observer/rows.json').read_text());streams=list(dict.fromkeys(r['Stream'] for r in rows));fit=[r for r in rows if streams.index(r['Stream'])%4<2];x=[]
  for r in fit:
   p=r['Trial']['Pre'];x.append([p['Before'].get(str(i),0.) for i in range(1,37)]+(p['Previous'] or [0.]*9)+p['Forecast'])
  x=np.array(x);sd=x.std(0);sd=np.where(sd>1e-10,sd,1.)
  write(w/'PERMUTATION_LOCK.json',{'Mean':x.mean(0).tolist(),'SD':sd.tolist(),'source':'new development seed37 Fit streams only','neighbors':12,'target_blind':True,'permutation_seeds':list(range(292000037,292000047))})
 files={}
 for name in subprocess.check_output(['git','ls-files'],cwd=ROOT,text=True).splitlines():
  if name.endswith('.go') or name.endswith('.py') and name.startswith('scripts/') or name in ['AGENTS.md','go.mod','go.sum'] or name.startswith('docs/research/s2_r7_information_20261118/'):
   files[str(ROOT/name)]=sha(ROOT/name)
 for name in ['SOURCE_VERIFICATION.json','00-source.json','source-events.jsonl','DEVELOPMENT_DECISION.json','DEVELOPMENT_ANALYSIS.json','R7_RETENTION_CONTRACT.json','R6_RETENTION_LOCK.json','PERMUTATION_LOCK.json']:
  files[str(w/name)]=sha(w/name)
 for f in (w/'history-contract').rglob('rows.json'):files[str(f)]=sha(f)
 if a.stage=='confirmatory':
  for seed in SEEDS:
   for arm in ARMS:
    d=w/f'confirmatory/seed-{seed}/{arm}'
    for name in ['final.json','events.jsonl']:files[str(d/name)]=sha(d/name)
  files[str(w/'TRAINING_LOCK.json')]=sha(w/'TRAINING_LOCK.json')
 files[str(a.binary.resolve())]=sha(a.binary.resolve())
 lock={'SourceCommit':subprocess.check_output(['git','rev-parse','HEAD'],cwd=ROOT,text=True).strip(),'SourceTree':subprocess.check_output(['git','rev-parse','HEAD^{tree}'],cwd=ROOT,text=True).strip(),'Files':files,'stage':a.stage,'seeds':SEEDS,'arms':ARMS,'utility_budget':512,'auxiliary_budget_per_D_E_F':256,'final_utility_operations_per_condition':256,'final_history_streams':16,'final_utility_conditions':['seen','unseen','shifted'],'R6_original_gates_unchanged':True,'new_paired_retention_ratio_upper_CI95':1.05,'H1_improvement':.1,'H1_macro_F1':.4,'class_threshold':1e-5,'catastrophic_absolute_RMSE':.02,'catastrophic_baseline_multiple':3,'bootstrap_replicates':5000,'hierarchy':'independent streams within independent training/acquisition seeds','IID_models':'independently IID-trained matched history/no-history pair','H2_H3':'NOT RUN; independent H1 development failed','source_sequence':80735,'fixed_features':32,'main_changed':False,'production_changed':False}
 write(w/('TRAINING_LOCK.json' if a.stage=='training' else 'CONFIRMATORY_LOCK.json'),lock);print(a.stage,'frozen',len(files),lock['SourceCommit'])
if __name__=='__main__':main()
