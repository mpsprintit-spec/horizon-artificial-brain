#!/usr/bin/env python3
"""Authenticate full S2-R5 Brain/WAL evidence, including failures and sources."""
import argparse,hashlib,json,os,subprocess,tarfile,shutil
from pathlib import Path
ROOT=Path(__file__).resolve().parents[1]
def sha(p):
 h=hashlib.sha256()
 with p.open('rb') as f:
  while b:=f.read(1<<20):h.update(b)
 return h.hexdigest()
def verify(p):
 actual={};manifest=None
 with tarfile.open(p,'r|xz') as t:
  for m in t:
   if not m.isfile():continue
   assert not Path(m.name).is_absolute() and '..' not in Path(m.name).parts
   f=t.extractfile(m)
   if m.name=='MANIFEST.json':manifest=json.load(f);continue
   if m.name in actual:raise RuntimeError('duplicate '+m.name)
   h=hashlib.sha256();n=0
   while b:=f.read(1<<20):h.update(b);n+=len(b)
   actual[m.name]={'bytes':n,'sha256':h.hexdigest()}
 if not manifest or manifest!=actual:raise RuntimeError('manifest mismatch')
 return {'archive_bytes':p.stat().st_size,'archive_sha256':sha(p),'authenticated_payloads':len(actual),'manifest_all_pass':True}
def main():
 p=argparse.ArgumentParser();p.add_argument('--workspace',type=Path);p.add_argument('--output',type=Path);p.add_argument('--stage',type=Path);p.add_argument('--verify',type=Path);a=p.parse_args()
 if a.verify:print(json.dumps(verify(a.verify),indent=2));return
 ws=a.workspace.resolve();stage=a.stage.resolve();stage.mkdir(exist_ok=False)
 def install(src,dst):dst.parent.mkdir(parents=True,exist_ok=True);os.link(src,dst)
 for folder in ['r5','execution']:
  for f in sorted((ws/folder).rglob('*')):
   if f.is_file() and not f.name.endswith(('.writer.lock','.pyc')) and '__pycache__' not in f.parts:install(f,stage/folder/f.relative_to(ws/folder))
 for f in sorted(ws.glob('*.log')):install(f,stage/'execution-early'/f.name)
 for f in sorted((ws/'source_download').glob('*VERIFICATION.json')):install(f,stage/'source'/f.name)
 install(ws/'source_download/Horizon_S2_R4_Temporal_Error_History_Learning_20261117.tar.xz',stage/'source/Horizon_S2_R4_Temporal_Error_History_Learning_20261117.tar.xz')
 install(ws/'source-r3/Horizon_S2_R2_Native_Learning_Repair_20261117.tar.xz',stage/'source/Horizon_S2_R2_Native_Learning_Repair_20261117.tar.xz')
 for f in sorted((ROOT/'docs/research/s2_r5_retention_20261117').rglob('*')):
  if f.is_file():install(f,stage/'docs'/f.relative_to(ROOT/'docs/research/s2_r5_retention_20261117'))
 for f in sorted((ROOT/'scripts').glob('*s2r5*.py')):install(f,stage/'reproduce'/f.name)
 lock=json.loads((ws/'r5/CONFIRMATORY_LOCK.json').read_text());commit=lock['SourceCommit']
 subprocess.run(['git','archive','--format=tar.gz','-o',str(stage/'scientific-source.tar.gz'),commit],cwd=ROOT,check=True)
 (stage/'implementation.patch').write_bytes(subprocess.check_output(['git','diff','--binary','0069e6eb27dbe9334746dd80cc31e5ed52214867',commit],cwd=ROOT))
 (stage/'BUNDLE.json').write_text(json.dumps({'scientific_source_commit':commit,'scientific_tree':lock['SourceTree'],'base':'0069e6eb27dbe9334746dd80cc31e5ed52214867','packaging_head':subprocess.check_output(['git','rev-parse','HEAD'],cwd=ROOT,text=True).strip(),'Brain_source_sequence':80735,'Brain_native_checksum':lock['source_native_checksum'],'full_historical_WAL':True,'final_checkpoints':108,'main_changed':False,'promoted':False},indent=2)+'\n')
 manifest={str(f.relative_to(stage)):{'bytes':f.stat().st_size,'sha256':sha(f)} for f in sorted(stage.rglob('*')) if f.is_file()}
 (stage/'MANIFEST.json').write_text(json.dumps(manifest,indent=2,sort_keys=True)+'\n')
 with a.output.open('wb') as target:
  proc=subprocess.Popen(['xz','-T3','--memlimit-compress=3GiB','--lzma2=preset=3,dict=128MiB','-c'],stdin=subprocess.PIPE,stdout=target)
  try:
   with tarfile.open(fileobj=proc.stdin,mode='w|') as t:
    for f in sorted(stage.rglob('*')):
     if f.is_file():t.add(f,arcname=str(f.relative_to(stage)),recursive=False)
   proc.stdin.close()
   if proc.wait():raise RuntimeError('compression failed')
  except BaseException:proc.kill();proc.wait();raise
 print(json.dumps(verify(a.output),indent=2),flush=True)
if __name__=='__main__':main()
