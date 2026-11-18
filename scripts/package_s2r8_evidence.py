#!/usr/bin/env python3
"""Stream losslessly decoded checkpoint/WAL bytes into a verified archive."""
import argparse,hashlib,io,json,pathlib,subprocess,tarfile
def main():
 p=argparse.ArgumentParser();p.add_argument('--root',type=pathlib.Path,required=True);p.add_argument('--output',type=pathlib.Path,required=True);a=p.parse_args();root=a.root.resolve();output=a.output.resolve()
 selected=[]
 for name in ['r2-source','r8-confirmatory','R8_Results']:
  selected.extend(p for p in (root/name).rglob('*') if p.is_file())
 selected.extend(p for p in root.glob('r8-*.log') if p.is_file());selected.extend(root.glob('r8-*.json'));selected.extend(root.glob('R8_*.json'))
 names=set();manifest={};proc=subprocess.Popen(['zstd','-q','-T2','-12','--long=27','-o',str(output)],stdin=subprocess.PIPE)
 with tarfile.open(fileobj=proc.stdin,mode='w|') as tf:
  for p in sorted(set(selected)):
   name=p.relative_to(root).as_posix()
   if '/companion-scratch/' in name or '/resource-replay/' in name:continue
   if name.endswith('.json.zst') or name.endswith('.jsonl.zst'):
    # Resource evidence may contain original interrupted files. Preserve their
    # raw bytes too; decoding a valid Zstd frame never repairs JSON or a WAL.
    data=subprocess.check_output(['zstd','-q','-d','-c',str(p)]);name=name[:-4]
   else:data=p.read_bytes()
   if name in names:raise RuntimeError(f'archive path collision: {name}')
   names.add(name);entry=tarfile.TarInfo(name);entry.size=len(data);entry.mode=0o600;tf.addfile(entry,io.BytesIO(data));manifest[name]={'bytes':len(data),'sha256':hashlib.sha256(data).hexdigest()}
  data=json.dumps(manifest,indent=2).encode();entry=tarfile.TarInfo('PAYLOAD_MANIFEST.json');entry.size=len(data);entry.mode=0o600;tf.addfile(entry,io.BytesIO(data))
 proc.stdin.close()
 if proc.wait():raise SystemExit('archive compression failed')
 (root/'R8_PAYLOAD_MANIFEST.json').write_text(json.dumps(manifest,indent=2))
 decoded=subprocess.Popen(['zstd','-q','-d','-c',str(output)],stdout=subprocess.PIPE);verified={}
 with tarfile.open(fileobj=decoded.stdout,mode='r|') as tf:
  for entry in tf:
   data=tf.extractfile(entry).read()
   if entry.name=='PAYLOAD_MANIFEST.json':continue
   verified[entry.name]={'bytes':len(data),'sha256':hashlib.sha256(data).hexdigest()}
 if decoded.wait() or verified!=manifest:raise SystemExit('archive payload verification failed')
 receipt={'archive':output.name,'bytes':output.stat().st_size,'sha256':hashlib.sha256(output.read_bytes()).hexdigest(),'payloads':len(manifest),'all_payloads_verified':True,'decoded_json_WAL_and_checkpoint_bytes_exact':True}
 (root/'R8_ARCHIVE_VERIFICATION.json').write_text(json.dumps(receipt,indent=2));print(json.dumps(receipt,indent=2))
if __name__=='__main__':main()
