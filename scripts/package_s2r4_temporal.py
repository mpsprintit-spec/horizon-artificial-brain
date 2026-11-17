#!/usr/bin/env python3
"""Package complete S2-R4 research evidence without truncating any journal."""
import argparse,hashlib,json,shutil,subprocess,tarfile
from pathlib import Path
ROOT=Path(__file__).resolve().parents[1]
def digest(p):
 h=hashlib.sha256()
 with p.open('rb') as f:
  while b:=f.read(1<<20):h.update(b)
 return h.hexdigest()
def verify(archive):
 actual={};m=None
 with tarfile.open(archive,'r|xz') as t:
  for member in t:
   if not member.isfile():continue
   f=t.extractfile(member)
   if member.name=='MANIFEST.json':m=json.load(f);continue
   if member.name in actual:raise RuntimeError('duplicate archive payload')
   h=hashlib.sha256();size=0
   while b:=f.read(1<<20):h.update(b);size+=len(b)
   actual[member.name]={'bytes':size,'sha256':h.hexdigest()}
 if m is None:raise RuntimeError('manifest missing')
 bad=[n for n in set(m)|set(actual) if m.get(n)!=actual.get(n)]
 if bad:raise RuntimeError('invalid payloads '+str(bad))
 return {'archive_sha256':digest(archive),'archive_bytes':archive.stat().st_size,'authenticated_payloads':len(m),'manifest_all_pass':True}
def main():
 p=argparse.ArgumentParser();p.add_argument('--workspace',type=Path);p.add_argument('--output',type=Path);p.add_argument('--stage',type=Path);p.add_argument('--verify',type=Path);a=p.parse_args()
 if a.verify:print(json.dumps(verify(a.verify),indent=2));return
 ws=a.workspace.resolve();stage=a.stage.resolve();stage.mkdir(exist_ok=False)
 folders=[('s2r4','development-v1'),('s2r4-v2','development-v2'),('s2r4-v3','diagnosis-v3'),('s2r4-v4','s2r4'),('execution','execution'),('s2r4-v3-execution','execution-v3'),('s2r4-v4-execution','execution-native')]
 for src,dst in folders:
  shutil.copytree(ws/src,stage/dst,ignore=shutil.ignore_patterns('*.writer.lock','__pycache__','*.pyc','horizon-chat-android-arm64','ai-android-arm64'))
 (stage/'source').mkdir();shutil.copy2(ws/'source/Horizon_S2_R3_Neural_Capacity_Diagnosis_20261117.tar.xz',stage/'source/Horizon_S2_R3_Neural_Capacity_Diagnosis_20261117.tar.xz')
 shutil.copy2(ws/'SOURCE_VERIFICATION.json',stage/'SOURCE_VERIFICATION.json')
 shutil.copytree(ROOT/'docs/research/s2_r4_temporal_20261117',stage/'docs')
 (stage/'reproduce').mkdir()
 for f in sorted((ROOT/'scripts').glob('*s2r4*.py')):shutil.copy2(f,stage/'reproduce'/f.name)
 shutil.copy2(ws/'s2r4-locked.test',stage/'reproduce/s2r4-linux-amd64.test')
 patch=subprocess.check_output(['git','diff','--binary','6d8ad454bb2c5289a747ab7d7e0cdca58dd7db2c','HEAD'],cwd=ROOT);(stage/'implementation.patch').write_bytes(patch)
 lock=json.loads((ws/'s2r4-v4/CONFIRMATORY_LOCK.json').read_text())
 (stage/'BUNDLE.json').write_text(json.dumps({'scientific_source_commit':lock['SourceCommit'],'packaging_repository_head':subprocess.check_output(['git','rev-parse','HEAD'],cwd=ROOT,text=True).strip(),'base':'6d8ad454bb2c5289a747ab7d7e0cdca58dd7db2c','source_sequence':80735,'source_native_checksum':'9016b845eed4e8437e87ada99b9641718a086c570e246dd0a62e57dffd1ce9d3','original_R3_sha256':digest(stage/'source/Horizon_S2_R3_Neural_Capacity_Diagnosis_20261117.tar.xz'),'journals':'complete historical prefix plus complete suffix; no trimming; source checkpoint immutable','production_promoted':False},indent=2)+'\n')
 manifest={str(f.relative_to(stage)):{'bytes':f.stat().st_size,'sha256':digest(f)} for f in sorted(stage.rglob('*')) if f.is_file()}
 (stage/'MANIFEST.json').write_text(json.dumps(manifest,indent=2,sort_keys=True)+'\n')
 # A large dictionary preserves compression of unchanged complete historical
 # prefixes; no journal is deduplicated, truncated or reconstructed on restore.
 with a.output.open('wb') as target:
  compressor=subprocess.Popen(['xz','-T4','--memlimit-compress=4GiB','--lzma2=preset=3,dict=128MiB','-c'],stdin=subprocess.PIPE,stdout=target)
  try:
   with tarfile.open(fileobj=compressor.stdin,mode='w|') as t:
    for f in sorted(stage.rglob('*')):
     if f.is_file():t.add(f,arcname=str(f.relative_to(stage)),recursive=False)
   compressor.stdin.close()
   if compressor.wait()!=0:raise RuntimeError('archive compression failed')
  except BaseException:
   compressor.kill();compressor.wait();raise
 print(json.dumps(verify(a.output),indent=2),flush=True)
if __name__=='__main__':main()
