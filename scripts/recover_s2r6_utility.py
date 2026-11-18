#!/usr/bin/env python3
"""Recover one complete eight-arm seed into a NEW owner directory.

Never opens an original journal for writing and never chooses/promotes an arm.
"""
import argparse,hashlib,json,os,shutil,subprocess
from pathlib import Path
ROOT=Path(__file__).resolve().parents[1]
ARMS=['A','A0','B','B0','BI','BI0','C','F']
def sha(p):
 h=hashlib.sha256()
 with p.open('rb') as f:
  while b:=f.read(1<<20):h.update(b)
 return h.hexdigest()
def main():
 p=argparse.ArgumentParser();p.add_argument('--work',type=Path,required=True);p.add_argument('--source',type=Path,required=True);p.add_argument('--binary',type=Path,required=True);p.add_argument('--destination',type=Path,required=True);p.add_argument('--seed',type=int,choices=[37,83,151,211,307,419],required=True);a=p.parse_args()
 work,source,dest=a.work.resolve(),a.source.resolve(),a.destination.resolve()
 if dest.exists() or dest==work or work in dest.parents or source in dest.parents:p.error('destination must be new and disjoint')
 dest.mkdir(parents=True);files=[Path('holdout/persistent/rows.json')]
 for arm in ARMS:
  for name in ['final.json','events.jsonl','predictions-persistent.json']:files.append(Path(f'training/seed-{a.seed}/{arm}/{name}'))
 for condition in ['persistent','independent']:files.append(Path(f'training/seed-{a.seed}/observer-{condition}/observer-final.json'))
 before={}
 for name in files:
  src=work/name;out=dest/name;out.parent.mkdir(parents=True,exist_ok=True);shutil.copyfile(src,out);before[str(name)]=sha(src)
 subprocess.run(['python3',str(ROOT/'scripts/run_s2r6_utility.py'),'--work',str(dest),'--source',str(source),'--binary',str(a.binary.resolve()),'--phase','persistence','--seed',str(a.seed)],cwd=ROOT,check=True)
 for name,expected in before.items():
  if sha(work/name)!=expected:raise RuntimeError('original file mutated: '+name)
 result={'seed':a.seed,'all_eight_arms_recovered':True,'original_files_unchanged':True,'original_hashes':before,'proof':json.loads((dest/f'persistence-{a.seed}.json').read_text()),'promoted':False}
 (dest/'OWNER_RECOVERY.json').write_text(json.dumps(result,indent=2)+'\n');print(dest/'OWNER_RECOVERY.json')
if __name__=='__main__':main()
