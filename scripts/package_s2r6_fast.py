#!/usr/bin/env python3
"""Fast, lossless owner encoding of the same complete S2-R6 evidence tree."""
import argparse,hashlib,json,subprocess,tarfile
from pathlib import Path
def sha(p):
 h=hashlib.sha256()
 with p.open('rb') as f:
  while b:=f.read(1<<20):h.update(b)
 return h.hexdigest()
def verify(p):
 actual={};manifest=None
 proc=subprocess.Popen(['zstd','-q','-dc',str(p)],stdout=subprocess.PIPE)
 with tarfile.open(fileobj=proc.stdout,mode='r|') as tar:
  for member in tar:
   assert not Path(member.name).is_absolute() and '..' not in Path(member.name).parts
   if not member.isfile():continue
   f=tar.extractfile(member)
   if member.name=='MANIFEST.json':manifest=json.load(f);continue
   if member.name in actual:raise RuntimeError('duplicate '+member.name)
   h=hashlib.sha256();n=0
   while b:=f.read(1<<20):h.update(b);n+=len(b)
   actual[member.name]={'bytes':n,'sha256':h.hexdigest()}
 if proc.wait()!=0 or not manifest or manifest!=actual:raise RuntimeError('manifest mismatch')
 return {'archive_bytes':p.stat().st_size,'archive_sha256':sha(p),'authenticated_payloads':len(actual),'all_payloads_pass':True,'uncompressed_payload_bytes':sum(v['bytes'] for v in actual.values()),'encoding':'tar+zstd; lossless, no evidence reduction'}
def main():
 p=argparse.ArgumentParser();p.add_argument('--stage',type=Path);p.add_argument('--output',type=Path);p.add_argument('--verify',type=Path);a=p.parse_args()
 if a.verify:print(json.dumps(verify(a.verify),indent=2));return
 stage=a.stage.resolve()
 manifest={str(f.relative_to(stage)):{'bytes':f.stat().st_size,'sha256':sha(f)} for f in sorted(stage.rglob('*')) if f.is_file() and f!=stage/'MANIFEST.json'}
 (stage/'MANIFEST.json').write_text(json.dumps(manifest,indent=2,sort_keys=True)+'\n')
 with a.output.open('wb') as out:
  proc=subprocess.Popen(['zstd','-q','-T2','--long=27','-3','-c'],stdin=subprocess.PIPE,stdout=out)
  try:
   with tarfile.open(fileobj=proc.stdin,mode='w|',dereference=True) as tar:
    for f in sorted(stage.rglob('*')):
     if f.is_file():tar.add(f,arcname=str(f.relative_to(stage)),recursive=False)
   proc.stdin.close()
   if proc.wait():raise RuntimeError('zstd compression failed')
  except BaseException:proc.kill();proc.wait();raise
 print(json.dumps(verify(a.output),indent=2),flush=True)
if __name__=='__main__':main()
