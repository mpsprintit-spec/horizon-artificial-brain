#!/usr/bin/env python3
"""Complete multipart tar+zstd with per-payload and combined authentication."""
import argparse,hashlib,json,subprocess,tarfile
from pathlib import Path
def digest(p):
 h=hashlib.sha256()
 with p.open('rb') as f:
  for b in iter(lambda:f.read(1<<20),b''):h.update(b)
 return h.hexdigest()
def verify(parts):
 parts=sorted(parts);combined=hashlib.sha256();total=0
 for p in parts:
  with p.open('rb') as f:
   for b in iter(lambda:f.read(1<<20),b''):combined.update(b);total+=len(b)
 cat=subprocess.Popen(['cat',*map(str,parts)],stdout=subprocess.PIPE)
 proc=subprocess.Popen(['zstd','-q','-dc'],stdin=cat.stdout,stdout=subprocess.PIPE);cat.stdout.close()
 actual={};manifest=None
 with tarfile.open(fileobj=proc.stdout,mode='r|') as tar:
  for m in tar:
   if Path(m.name).is_absolute() or '..' in Path(m.name).parts:raise RuntimeError('unsafe archive name')
   if not m.isfile():raise RuntimeError('unexpected nonfile '+m.name)
   f=tar.extractfile(m)
   if m.name=='MANIFEST.json':
    if manifest is not None:raise RuntimeError('duplicate manifest')
    manifest=json.load(f);continue
   if m.name in actual:raise RuntimeError('duplicate '+m.name)
   h=hashlib.sha256();n=0
   for b in iter(lambda:f.read(1<<20),b''):h.update(b);n+=len(b)
   actual[m.name]={'bytes':n,'sha256':h.hexdigest()}
 if proc.wait() or cat.wait() or not manifest or manifest!=actual:raise RuntimeError('payload manifest mismatch')
 return {'archive_bytes':total,'archive_sha256':combined.hexdigest(),'authenticated_payloads':len(actual),'uncompressed_payload_bytes':sum(v['bytes'] for v in actual.values()),'all_payloads_pass':True,'parts':[{'name':p.name,'bytes':p.stat().st_size,'sha256':digest(p)} for p in parts]}
def main():
 p=argparse.ArgumentParser();p.add_argument('--root',type=Path);p.add_argument('--output',type=Path);p.add_argument('--verify-parts',type=Path,nargs='+');p.add_argument('--result',type=Path,required=True);a=p.parse_args()
 if a.verify_parts:result=verify(a.verify_parts)
 else:
  root=a.root.resolve();out=a.output.resolve();out.mkdir(exist_ok=False)
  mappings=[(root/'r7','r7'),(root/'restored/source/R2-active','source/R2-active'),(root/'evidence','evidence'),(root/'execution','execution'),(root/'r7-execution','r7-execution'),(root/'bin','bin'),(root/'deliverables/reproduction','scientific')]
  files={}
  for folder,prefix in mappings:
   for f in sorted(folder.rglob('*')):
    if f.is_file() and '__pycache__' not in f.parts:files[prefix+'/'+str(f.relative_to(folder))]=f
  for f in sorted((root/'deliverables').iterdir()):
   if f.is_file():files['reports/'+f.name]=f
  files['RESTORE.md']=root/'deliverables/RESTORE.md'
  files['prior/Horizon_S2_R6_Native_Utility_Retention_Learning_20261118.tar.zst']=root/'input/Horizon_S2_R6_Native_Utility_Retention_Learning_20261118.tar.zst'
  manifest={name:{'bytes':f.stat().st_size,'sha256':digest(f)} for name,f in sorted(files.items())}
  mp=out/'MANIFEST.json';mp.write_text(json.dumps(manifest,indent=2,sort_keys=True)+'\n')
  files['MANIFEST.json']=mp
  prefix=out/'Horizon_S2_R7_Native_Utility_Information_Learning_Repair_20261118.tar.zst.part'
  compressor=subprocess.Popen(['zstd','-q','-T2','--long=27','-3','-c'],stdin=subprocess.PIPE,stdout=subprocess.PIPE)
  splitter=subprocess.Popen(['split','-b','300M','-d','-a','2','--numeric-suffixes=1','-',str(prefix)],stdin=compressor.stdout);compressor.stdout.close()
  try:
   with tarfile.open(fileobj=compressor.stdin,mode='w|',dereference=True) as tar:
    for name,f in sorted(files.items()):tar.add(f,arcname=name,recursive=False)
   compressor.stdin.close()
   if compressor.wait() or splitter.wait():raise RuntimeError('archive compressor/split failed')
  except BaseException:compressor.kill();splitter.kill();raise
  result=verify(list(out.glob('*.part[0-9][0-9]')))
 a.result.write_text(json.dumps(result,indent=2)+'\n');print(json.dumps(result),flush=True)
if __name__=='__main__':main()
