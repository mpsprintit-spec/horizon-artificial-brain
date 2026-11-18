#!/usr/bin/env python3
"""Validate every frozen byte, then rebind only absolute paths on a new host."""
import argparse,hashlib,json,pathlib,shutil
def main():
 p=argparse.ArgumentParser();p.add_argument('--repo',type=pathlib.Path,required=True);p.add_argument('--source',type=pathlib.Path,required=True);p.add_argument('--work',type=pathlib.Path,required=True);a=p.parse_args()
 path=a.work/'CONFIRMATORY_LOCK.json';v=json.loads(path.read_text());rebound={}
 for old,want in v['Files'].items():
  target=None
  for marker,root in [('/horizon-r8/',a.repo),('/r2-source/',a.source),('/r8-confirmatory/',a.work)]:
   if marker in old:target=root.resolve()/old.split(marker,1)[1];break
  if target is None:raise SystemExit(f'unrecognized original path: {old}')
  if hashlib.sha256(target.read_bytes()).hexdigest()!=want:raise SystemExit(f'frozen bytes differ: {target}')
  rebound[str(target)]=want
 backup=path.with_name('CONFIRMATORY_LOCK.original.json')
 if backup.exists():raise SystemExit('original lock backup already exists; no overwrite')
 shutil.copy2(path,backup);v['Files']=rebound;v['portable_path_rebind_only']=True;path.write_text(json.dumps(v,indent=2));print('verified and rebound',len(rebound),'files; original lock retained')
if __name__=='__main__':main()
