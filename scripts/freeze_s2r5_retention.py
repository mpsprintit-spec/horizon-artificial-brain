#!/usr/bin/env python3
"""Seal code, all 108 final models, analysis/configuration before fresh holdout."""
import argparse,json,hashlib,subprocess
from pathlib import Path
ROOT=Path(__file__).resolve().parents[1]
SEEDS=[37,83,151,211,307,419]
ARMS=['A','A0','B','B0','C','C0','Cw','Cw0','D','D0','E','E0','AI','AI0','DI','DI0','EI','EI0']
def sha(p):return hashlib.sha256(p.read_bytes()).hexdigest()
def main():
 p=argparse.ArgumentParser();p.add_argument('--work',type=Path,required=True);p.add_argument('--binary',type=Path,required=True);a=p.parse_args();w=a.work.resolve()
 if (w/'FINAL_STARTED').exists() or (w/'CONFIRMATORY_LOCK.json').exists():raise RuntimeError('do not replace final lock/data')
 if subprocess.check_output(['git','status','--porcelain'],cwd=ROOT,text=True).strip():raise RuntimeError('commit scientific source before freezing')
 files={}
 for name in subprocess.check_output(['git','ls-files'],cwd=ROOT,text=True).splitlines():
  if name.endswith('.go') or name in ['go.mod','go.sum'] or ('s2r5' in name and name.endswith('.py')) or name.startswith('docs/research/s2_r5_retention_20261117/'):
   files[str(ROOT/name)]=sha(ROOT/name)
 for name in ['00-source.json','source-events.jsonl','SOURCE_VERIFICATION.json','TRAINING_LOCK.json','METRIC_LOCK.json','OFFLINE_CALIBRATION_LOCK.json','anchors/rows.json','RECOVERY_PREFLIGHT.json']:
  files[str(w/name)]=sha(w/name)
 cps=[]
 for seed in SEEDS:
  for arm in ARMS:
   d=w/f'training/seed-{seed}/{arm}'
   for name in ['final.json','receipts.json','events.jsonl','costs.json']:files[str(d/name)]=sha(d/name)
   cps.append(str(d/'final.json'))
 files[str(a.binary.resolve())]=sha(a.binary.resolve())
 lock={'SourceCommit':subprocess.check_output(['git','rev-parse','HEAD'],cwd=ROOT,text=True).strip(),'SourceTree':subprocess.check_output(['git','rev-parse','HEAD^{tree}'],cwd=ROOT,text=True).strip(),'Files':files,'fixed_final_checkpoints':cps,'seeds':SEEDS,'arms':ARMS,'final_seeds':[8100017,8200031,8300043],'new_retention_seeds':[8900003,9000007],'streams':32,'steps':24,'source_sequence':80735,'source_native_checksum':'9016b845eed4e8437e87ada99b9641718a086c570e246dd0a62e57dffd1ce9d3','no_final_model_selection':True,'no_main_change':True,'decision_rules':'PROTOCOL.md + exact original native evaluator + fixed historical external limit 0.01766457'}
 (w/'CONFIRMATORY_LOCK.json').write_text(json.dumps(lock,indent=2,sort_keys=True)+'\n');print('Sealed',len(cps),'models;',len(files),'files; final not yet acquired')
if __name__=='__main__':main()
