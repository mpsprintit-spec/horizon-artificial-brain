#!/usr/bin/env python3
"""Isolated paired native experiments. Never pushes, changes main or opens PRs."""
import argparse, concurrent.futures, hashlib, json, os, pathlib, subprocess, time

ROOT=pathlib.Path(__file__).resolve().parents[1]
SEEDS=[37,83,151,211,307,419]
def digest(p):return hashlib.sha256(pathlib.Path(p).read_bytes()).hexdigest()
def compress_tree(path,keep=()):
    for p in pathlib.Path(path).rglob('*'):
        if p.is_file() and str(p) not in keep and p.suffix in ('.json','.jsonl') and p.stat().st_size>3000000:
            subprocess.run(['zstd','-q','-T1','--rm',str(p)],check=True)
def invoke(args,seed,test,arm=None):
    env=os.environ.copy();env.update(R8_SOURCE=str(args.source),R8_WORK=str(args.work),R8_SEED=str(seed),GOMAXPROCS='2')
    if arm:env['R8_ARM']=arm
    logs=args.work/'execution';logs.mkdir(exist_ok=True)
    name=f'{test}-{seed}'+(f'-{arm}' if arm else '')
    meta={'seed':seed,'arm':arm,'test':test,'binary_sha256':digest(args.binary),'started_unix':time.time(),'GOMAXPROCS':2,'source':str(args.source),'work':str(args.work)}
    with (logs/f'{name}.log').open('w') as f:
        p=subprocess.run([str(args.binary),'-test.run',f'^{test}$','-test.v','-test.timeout','30m'],cwd=ROOT/'services/ai/runtime',env=env,stdout=f,stderr=subprocess.STDOUT)
    meta.update(returncode=p.returncode,finished_unix=time.time());(logs/f'{name}.json').write_text(json.dumps(meta,indent=2))
    print(name,p.returncode,round(meta['finished_unix']-meta['started_unix'],2),flush=True)
    return p.returncode
def job(args,seed):
    if args.phase=='prepare':
        rc=invoke(args,seed,'TestS2R8Prepare')
        if rc==0:compress_tree(args.work/f'seed-{seed}')
        return rc
    if args.phase=='train':
        rc=0;offset=SEEDS.index(seed)%4;arms=list('ABCD');arms=arms[offset:]+arms[:offset]
        for arm in arms:rc=max(rc,invoke(args,seed,'TestS2R8Train',arm))
        compress_tree(args.work/f'seed-{seed}')
        return rc
    rc=invoke(args,seed,'TestS2R8Evaluate');compress_tree(args.work/f'seed-{seed}');return rc
def main():
    p=argparse.ArgumentParser();p.add_argument('phase',choices=['prepare','train','evaluate']);p.add_argument('--source',type=pathlib.Path,required=True);p.add_argument('--work',type=pathlib.Path,required=True);p.add_argument('--binary',type=pathlib.Path,required=True);p.add_argument('--workers',type=int,default=2);p.add_argument('--seeds',type=int,nargs='+',default=SEEDS);a=p.parse_args()
    a.source=a.source.resolve();a.work=a.work.resolve();a.binary=a.binary.resolve();a.work.mkdir(exist_ok=True,parents=True)
    if a.phase=='evaluate':
        for seed in a.seeds:
            for arm in 'ABCD':
                receipt=a.work/'execution'/f'TestS2R8Train-{seed}-{arm}.json'
                if not receipt.exists() or json.loads(receipt.read_text())['returncode']!=0:raise SystemExit(f'preserve failure; final evaluation blocked: {seed}/{arm}')
    with concurrent.futures.ThreadPoolExecutor(max_workers=a.workers) as ex:results=list(ex.map(lambda seed:job(a,seed),a.seeds))
    raise SystemExit(max(results))
if __name__=='__main__':main()
