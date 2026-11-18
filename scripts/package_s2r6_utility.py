#!/usr/bin/env python3
"""Create/stream-authenticate the complete S2-R6 owner recovery bundle."""
import argparse,hashlib,json,os,shutil,subprocess,tarfile
from pathlib import Path
ROOT=Path(__file__).resolve().parents[1]
def sha(p):
 h=hashlib.sha256()
 with p.open('rb') as f:
  while b:=f.read(1<<20):h.update(b)
 return h.hexdigest()
def verify(p):
 actual={};manifest=None
 with tarfile.open(p,'r|xz') as tar:
  for member in tar:
   assert not Path(member.name).is_absolute() and '..' not in Path(member.name).parts
   if not member.isfile():continue
   f=tar.extractfile(member)
   if member.name=='MANIFEST.json':manifest=json.load(f);continue
   if member.name in actual:raise RuntimeError('duplicate '+member.name)
   h=hashlib.sha256();n=0
   while b:=f.read(1<<20):h.update(b);n+=len(b)
   actual[member.name]={'bytes':n,'sha256':h.hexdigest()}
 if not manifest or manifest!=actual:raise RuntimeError('manifest mismatch')
 return {'archive_bytes':p.stat().st_size,'archive_sha256':sha(p),'authenticated_payloads':len(actual),'all_payloads_pass':True,'uncompressed_payload_bytes':sum(v['bytes'] for v in actual.values())}
def main():
 p=argparse.ArgumentParser();p.add_argument('--workspace',type=Path);p.add_argument('--output',type=Path);p.add_argument('--stage',type=Path);p.add_argument('--verify',type=Path);a=p.parse_args()
 if a.verify:print(json.dumps(verify(a.verify),indent=2));return
 ws=a.workspace.resolve();stage=a.stage.resolve();stage.mkdir(exist_ok=False)
 def install(src,dest):
  dest.parent.mkdir(parents=True,exist_ok=True)
  try:os.link(src,dest)
  except OSError:shutil.copyfile(src,dest)
 def tree(src,dest):
  for f in sorted(src.rglob('*')):
   if f.is_file() and not f.name.endswith(('.writer.lock','.pyc')) and '__pycache__' not in f.parts:install(f,dest/f.relative_to(src))
 tree(ws/'r6-v2',stage/'r6');tree(ws/'r6',stage/'negative-development-v1')
 for folder in ['r6-execution','r6-v2-execution','execution']:tree(ws/folder,stage/'execution'/folder)
 tree(ws/'source-r2-essential',stage/'source/R2-active')
 install(ws/'sources-r5/Horizon_S2_R5_Retention_Constrained_Native_Learning_20261117.tar.xz',stage/'source/Horizon_S2_R5_Retention_Constrained_Native_Learning_20261117.tar.xz')
 install(ws/'source-r2-download/Horizon_S2_R2_Native_Learning_Repair_20261117.tar.xz',stage/'source/Horizon_S2_R2_Native_Learning_Repair_20261117.tar.xz')
 for file in [ws/'sources-r5/VERIFICATION.json',ws/'SOURCE_CACHE_VIEW.json']:install(file,stage/'source'/file.name)
 for name in ['s2r6.test','s2r6-v2.test','s2r6-science.test','horizon-ai-android-arm64','horizon-chat-android-arm64']:install(ws/name,stage/'binaries'/name)
 tree(ROOT/'docs/research/s2_r6_native_utility_20261118',stage/'docs')
 for f in sorted((ROOT/'scripts').glob('*s2r6*')):
  if f.is_file():install(f,stage/'reproduce'/f.name)
 lock=json.loads((ws/'r6-v2/CONFIRMATORY_LOCK.json').read_text());commit=lock['SourceCommit']
 subprocess.run(['git','archive','--format=tar.gz','-o',str(stage/'scientific-source.tar.gz'),commit],cwd=ROOT,check=True)
 (stage/'implementation.patch').write_bytes(subprocess.check_output(['git','diff','--binary','099ae285737cdc62fc121791ba4e6f4e463a1cb3',commit],cwd=ROOT))
 bundle={'scientific_source_commit':commit,'scientific_tree':lock['SourceTree'],'R5_base':'099ae285737cdc62fc121791ba4e6f4e463a1cb3','training_source_commit':json.loads((ws/'r6-v2/TRAINING_LOCK.json').read_text())['SourceCommit'],'source_sequence':80735,'source_native_checksum':'9016b845eed4e8437e87ada99b9641718a086c570e246dd0a62e57dffd1ce9d3','seeds':[37,83,151,211,307,419],'final_models':48,'source_R5_complete_archive_included':True,'fixed_features':32,'full_historical_WAL':True,'main_changed':False,'promoted':False,'verdict':json.loads((ws/'r6-v2/SCIENTIFIC_RESULTS.json').read_text())['verdict']}
 (stage/'BUNDLE.json').write_text(json.dumps(bundle,indent=2)+'\n')
 (stage/'RESTORE.md').write_text('''# S2-R6 owner recovery

Combine all numbered parts in numeric order and verify the outer SHA256 in the owner delivery note. Run the verifier from the compatible source:

`python3 scripts/package_s2r6_utility.py --verify /absolute/archive.tar.xz`

Extract the archive into a NEW directory. The complete R5 archive and original complete R2 archive are included under source/. R2-active is a byte-identical minimal runtime view; MANIFEST.json retains all443 original authenticated R2 payload hashes. Extract the original R2 archive to reconstruct its complete historical tree when needed.

scientific-source.tar.gz contains the compatible repository at the sealed commit in BUNDLE.json. Extract into a new source checkout. Build the research binary using Go1.22.12:

`go test -tags 'horizon_s2r2 horizon_s2r3 horizon_s2r4 horizon_s2r5 horizon_s2r6' -c ./services/ai/runtime -o /absolute/s2r6-owner.test`

From that source directory, recover one entire seed (all eight research arms) into a NEW destination without changing originals:

`python3 scripts/recover_s2r6_utility.py --work /absolute/restored/r6 --source /absolute/restored/source/R2-active --binary /absolute/s2r6-owner.test --destination /absolute/recovered-seed-37 --seed 37`

Repeat for83/151/211/307/419 with distinct destinations. No arm is selected as canonical. The recovery runner verifies exact checkpoint restart, complete suffix replay from80735, signed N reconstruction, original WAL prefix and A–E1 oracles. It retains restarted.json and full events.jsonl for every arm; original files are hash-checked again afterward. An amd64 test binary is included for compatible Linux hosts; Android arm64 service/chat binaries are cross-build evidence only. Compile native source for a different host; cross-architecture exact replay is not asserted.

Direct API recovery uses OpenEventLog on a COPY of the complete journal, then RebuildRuntimeFromCheckpointAndJournal(log, checkpointPath). final.json+events.jsonl restores a final research model; source/R2-active/s2r2/s2r2-continuation.json+the same complete journal replays the whole suffix. Original rollback uses the R2 checkpoint+original48MB journal. Do not truncate historical WAL or transplant coefficients into production.

TRAINING_LOCK and CONFIRMATORY_LOCK retain original absolute acquisition paths as provenance. Recovery above does not rewrite those locks. A new reproduction must use fresh work paths and create new path-specific locks before acquisition. Never overwrite or reacquire the archived holdout in place. negative-development-v1 contains the failed unbounded-fitter prototype; r6/development contains bounded-solver development. Both protocol versions are supported by the included recovery source.
''')
 manifest={str(f.relative_to(stage)):{'bytes':f.stat().st_size,'sha256':sha(f)} for f in sorted(stage.rglob('*')) if f.is_file()}
 (stage/'MANIFEST.json').write_text(json.dumps(manifest,indent=2,sort_keys=True)+'\n')
 with a.output.open('wb') as out:
  proc=subprocess.Popen(['xz','-T3','--memlimit-compress=3GiB','--lzma2=preset=3,dict=128MiB','-c'],stdin=subprocess.PIPE,stdout=out)
  try:
   with tarfile.open(fileobj=proc.stdin,mode='w|') as tar:
    for f in sorted(stage.rglob('*')):
     if f.is_file():tar.add(f,arcname=str(f.relative_to(stage)),recursive=False)
   proc.stdin.close()
   if proc.wait():raise RuntimeError('compression failed')
  except BaseException:proc.kill();proc.wait();raise
 print(json.dumps(verify(a.output),indent=2),flush=True)
if __name__=='__main__':main()
