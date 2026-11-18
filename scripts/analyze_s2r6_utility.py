#!/usr/bin/env python3
"""Locked outcome analysis; no model, threshold or checkpoint selection."""
import argparse,json,math
from pathlib import Path
import numpy as np

def read(p):return json.loads(p.read_text())
def rmse(a,b):return float(np.sqrt(np.mean((np.asarray(a)-np.asarray(b))**2)))
def classes(a):return np.where(np.asarray(a)>1e-5,2,np.where(np.asarray(a)<-1e-5,0,1))
def macro(a,b):
 a,b=classes(a),classes(b);f=[]
 for c in range(3):
  tp=int(np.sum((a==c)&(b==c)));fp=int(np.sum((a!=c)&(b==c)));fn=int(np.sum((a==c)&(b!=c)))
  f.append(2*tp/max(1,2*tp+fp+fn))
 return float(np.mean(f)),[int(np.sum(a==c)) for c in range(3)]
def baseline(train,rows):
 y=np.array([v['Utility'] for v in train]);out={'constant':[float(y.mean())]*len(rows),'zero':[0.]*len(rows)}
 for key,name in [('ActivationMagnitude','activation_only'),('WeightMagnitude','weight_only')]:
  x=np.array([v['Context'][key] for v in train]);z=np.array([v['Context'][key] for v in rows])
  xx=np.column_stack([np.ones(len(x)),x]);coef=np.linalg.lstsq(xx,y,rcond=None)[0]
  out[name]=(np.column_stack([np.ones(len(z)),z])@coef).tolist()
 return out
def calculate(root,stage,seeds):
 results={};cluster={};allrows={}
 for label in (['seen','unseen'] if stage=='development' else ['seen','unseen','shifted']):
  results[label]={};cluster[label]={};allrows[label]={}
  for arm in ['C','F']:
   y=[];p=[];bs={};per=[];groups=[]
   for seed in seeds:
    d=root/stage/f'seed-{seed}'
    alltrain=read(d/'utility-acquisition/rows.json')
    train=alltrain[:128] if stage=='development' else [r for i,r in enumerate(alltrain) if i%128<64]
    rows=read(d/arm/f'eval-{label}/rows.json')
    b=baseline(train,rows);yy=[r['Utility'] for r in rows];pp=[r['Prediction'] for r in rows]
    y+=yy;p+=pp
    for k,v in b.items():bs.setdefault(k,[]).extend(v)
    per.append({'seed':seed,'rmse':rmse(yy,pp),'baseline_rmse':{k:rmse(yy,v) for k,v in b.items()}})
    for stream in sorted({r['Stream'] for r in rows}):
     ii=[i for i,r in enumerate(rows) if r['Stream']==stream]
     groups.append({'seed':seed,'stream':stream,'y':np.array(yy)[ii],'p':np.array(pp)[ii],'b':{k:np.array(v)[ii] for k,v in b.items()}})
   br={k:rmse(y,v) for k,v in bs.items()};best=min(br,key=br.get);rr=rmse(y,p);gain=1-rr/br[best] if br[best] else -math.inf
   rng=np.random.default_rng(181899991);effects=[]
   for _ in range(10000):
    chosen=[]
    shared_streams=None
    if stage=='training':
     streams=sorted({g['stream'] for g in groups});shared_streams=rng.choice(streams,len(streams),replace=True)
    for seed in rng.choice(seeds,len(seeds),replace=True):
     pool=[g for g in groups if g['seed']==seed]
     if shared_streams is None:chosen.extend(pool[i] for i in rng.integers(0,len(pool),len(pool)))
     else:
      bystream={g['stream']:g for g in pool};chosen.extend(bystream[s] for s in shared_streams)
    a=np.concatenate([g['y'] for g in chosen]);c=np.concatenate([g['p'] for g in chosen]);b=np.concatenate([g['b'][best] for g in chosen])
    effects.append(1-rmse(a,c)/max(1e-30,rmse(a,b)))
   ci=np.percentile(effects,[2.5,97.5]).tolist();f1,counts=macro(y,p)
   results[label][arm]={'N':len(y),'rmse':rr,'baseline_rmse':br,'best_baseline':best,'benefit':gain,'CI95':ci,'macro_F1':f1,'class_counts_negative_neutral_positive':counts,'per_seed':per}
  c,f=results[label]['C'],results[label]['F']
  results[label]['pass']=c['benefit']>=.10 and c['CI95'][0]>0 and c['macro_F1']>=.40 and min(c['class_counts_negative_neutral_positive'])>=10 and f['benefit']<.5*c['benefit']
 return {'stage':stage,'seeds':seeds,'H1_pass':all(results[v]['pass'] for v in results),'results':results,'H2_authorized_by_development':all(results[v]['pass'] for v in results) if stage=='development' else None}
def main():
 p=argparse.ArgumentParser();p.add_argument('--work',type=Path,required=True);p.add_argument('--stage',choices=['development','training'],default='development');p.add_argument('--output',type=Path,required=True);a=p.parse_args()
 out=calculate(a.work,a.stage,[37,83] if a.stage=='development' else [37,83,151,211,307,419]);a.output.write_text(json.dumps(out,indent=2)+'\n');print(json.dumps(out,indent=2))
if __name__=='__main__':main()
