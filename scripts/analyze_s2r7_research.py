#!/usr/bin/env python3
"""Offline R7 statistical research only. No fitted probe is used by Horizon."""
import argparse,collections,hashlib,json,math,re
from pathlib import Path
import numpy as np
from scipy.optimize import lsq_linear
SEEDS=[37,83,151,211,307,419];ARMS=list('ABCDEFG');PREFIX=48170318

def read(p):return json.loads(Path(p).read_text())
def write(p,v):
 p=Path(p);p.parent.mkdir(parents=True,exist_ok=True);p.write_text(json.dumps(v,indent=2,sort_keys=True,allow_nan=False)+'\n')
def metric(y,p):
 y,p=np.asarray(y),np.asarray(p);truth=np.sign(np.where(abs(y)<=1e-5,0,y));pred=np.sign(np.where(abs(p)<=1e-5,0,p));f=[];counts=[]
 for c in [-1,0,1]:
  tp=np.sum((truth==c)&(pred==c));fp=np.sum((truth!=c)&(pred==c));fn=np.sum((truth==c)&(pred!=c));f.append(float(2*tp/(2*tp+fp+fn)) if 2*tp+fp+fn else 0.);counts.append(int(np.sum(truth==c)))
 return {'N':len(y),'RMSE':float(np.sqrt(np.mean((y-p)**2))),'MAE':float(np.mean(abs(y-p))),'macro_F1':float(np.mean(f)),'class_counts_negative_neutral_positive':counts,'mean_actual':float(y.mean()),'mean_prediction':float(p.mean()),'calibration_ratio':float(p.mean()/y.mean()) if y.mean()!=0 else None,'R_squared':float(1-np.mean((y-p)**2)/np.var(y)) if np.var(y)>0 else None,'prediction_quantiles':np.quantile(p,[0,.05,.5,.95,1]).tolist()}

def split(rows):
 streams=list(dict.fromkeys(r['Stream'] for r in rows));fit=np.array([streams.index(r['Stream'])%4<2 for r in rows]);a={r['OriginID'] for r,m in zip(rows,fit) if m};b={r['OriginID'] for r,m in zip(rows,fit) if not m};assert not a&b
 return fit,~fit

def matrix(rows,mode):
 if mode=='raw':return np.array([r['Representation']['Input'] for r in rows])
 if mode in ['activation','metric']:return np.array([r['Representation'][mode.capitalize()] for r in rows])
 if mode=='activation_only':return np.array([[r['Context']['Original']['ActivationMagnitude']] for r in rows])
 if mode=='weight_only':return np.array([[r['Context']['Original']['WeightMagnitude']] for r in rows])
 raise ValueError(mode)

def fit_probe(x,y,kind='ridge',ridge=.01):
 x=np.asarray(x);y=np.asarray(y);mu=x.mean(0);sd=x.std(0);vary=sd>1e-12;scale=np.where(vary,sd,1.);z=((x-mu)/scale)[:,vary];my=y.mean()
 if kind=='bounded' and z.shape[1]:
  bound=7.99*sd[vary];result=lsq_linear(z,(y-my),bounds=(-bound,bound),max_iter=128,tol=1e-10);coef=result.x;status={'success':bool(result.success),'iterations':int(result.nit),'optimality':float(result.optimality)}
 elif kind=='ols':coef=np.linalg.lstsq(z,y-my,rcond=1e-9)[0];status={'success':True}
 elif z.shape[1]:coef=np.linalg.solve(z.T@z/len(z)+np.eye(z.shape[1])*ridge,z.T@(y-my)/len(z));status={'success':True}
 else:coef=np.zeros(0);status={'success':True}
 def predict(xx):return np.clip(((np.asarray(xx)-mu)/scale)[:,vary]@coef+my,-1,1)
 return predict,{'kind':kind,'ridge':ridge,'varying_columns':int(vary.sum()),'rank':int(np.linalg.matrix_rank(z)) if z.size else 0,'coefficient_peak_standardized':float(max(abs(coef),default=0.)),**status}

def fit_polynomial_probe(x,y,degree=2,ridge=1e-4):
 x=np.asarray(x);y=np.asarray(y);mu=x.mean(0);sd=x.std(0);vary=sd>1e-12;scale=np.where(vary,sd,1.);z=((x-mu)/scale)[:,vary];dim=max(1,z.shape[1]);my=y.mean();kernel=(1+z@z.T/dim)**degree
 coef=np.linalg.solve(kernel+np.eye(len(z))*len(z)*ridge,y-my)
 def predict(xx):
  zz=((np.asarray(xx)-mu)/scale)[:,vary];return np.clip((1+zz@z.T/dim)**degree@coef+my,-1,1)
 return predict,{'kind':'polynomial-kernel-ridge','degree':degree,'ridge':ridge,'fitting':'Fit only','information':'pre-operation descriptor','diagnostic_only':True,'post_outcome_input':False}

COMPARATOR_CACHE={}
def comparators(train,test):
 mask,_=split(train);fit=[r for r,m in zip(train,mask) if m];y=np.array([r['Utility'] for r in fit]);n=len(test);cache=hashlib.sha256()
 for data in [matrix(fit,'raw'),matrix(test,'raw'),matrix(fit,'activation'),matrix(test,'activation'),matrix(fit,'metric'),matrix(test,'metric'),y]:cache.update(data.tobytes())
 key=cache.hexdigest()
 if key in COMPARATOR_CACHE:return COMPARATOR_CACHE[key]
 pred={'zero':np.zeros(n),'constant_mean':np.full(n,y.mean())};details={}
 for mode,kind,name in [('activation_only','ols','activation_only'),('weight_only','ols','weight_only'),('raw','bounded','bounded_raw_linear'),('raw','ridge','regularized_raw_linear'),('activation','ridge','regularized_native_activation'),('metric','ridge','regularized_native_metric')]:
  fn,detail=fit_probe(matrix(fit,mode),y,kind);pred[name]=fn(matrix(test,mode));details[name]={**detail,'information':mode,'fitting':'Fit origins only','post_outcome_input':False}
 for mode in ['raw','metric']:
  for degree in [2,3]:
   name=f'polynomial_{degree}_{mode}';fn,detail=fit_polynomial_probe(matrix(fit,mode),y,degree);pred[name]=fn(matrix(test,mode));details[name]=detail
 COMPARATOR_CACHE[key]=(pred,details)
 return pred,details

def collisions(x,y):
 x,y=np.array(x),np.array(y);result={}
 for places in [None,7,5,3]:
  a=x if places is None else np.round(x,places);_,inv,count=np.unique(a,axis=0,return_inverse=True,return_counts=True);sy=np.bincount(inv,weights=y);syy=np.bincount(inv,weights=y*y);var=np.maximum(0,syy-sy*sy/count);groups=np.flatnonzero(count>1)
  examples=[]
  for g in groups[:12]:
   ids=np.flatnonzero(inv==g);examples.append({'row_indices':ids[:12].tolist(),'N':len(ids),'utility_min':float(y[ids].min()),'utility_max':float(y[ids].max())})
  result['exact_float64' if places is None else f'rounded_1e-{places}']={'distinct':len(count),'collision_groups':int(len(groups)),'collision_pairs':int(np.sum(count*(count-1)//2)),'within_cell_utility_variance_MSE':float(var.sum()/len(y)),'examples':examples}
 return result

def representation_audit(train,test):
 y=np.array([r['Utility'] for r in test]);mask,_=split(train);fit=[r for r,m in zip(train,mask) if m];fy=np.array([r['Utility'] for r in fit]);out={}
 for mode in ['raw','activation','metric']:
  xx=matrix(fit,mode);fn,detail=fit_probe(xx,fy);sd=xx.std(0);vary=sd>1e-12;z=(xx[:,vary]-xx[:,vary].mean(0))/sd[vary] if vary.any() else np.zeros((len(xx),0));sv=np.linalg.svd(z,compute_uv=False);pos=sv[sv>1e-10]
  poly,poly_detail=fit_polynomial_probe(xx,fy,3)
  out[mode]={'polynomial_3_holdout':metric(y,poly(matrix(test,mode))),'polynomial_diagnostic':poly_detail,'ridge_holdout':metric(y,fn(matrix(test,mode))),'fit_diagnostic':detail,'condition_nonzero':float(pos[0]/pos[-1]) if len(pos) else 0,'collisions':collisions(matrix(test,mode),y),'refinable_leaves':int(test[0]['Representation']['RefinableLeaves'])}
 return out

def hierarchical(records,replicates=5000):
 # Acquisition streams are independent across R7 seeds. Resample streams
 # separately inside each sampled seed; keep model and comparator paired.
 sa=[];sb=[]
 for y,a,b,g in records:
  groups=list(dict.fromkeys(g));sa.append(np.array([np.sum((a[np.array(g)==s]-y[np.array(g)==s])**2) for s in groups]));sb.append(np.array([np.sum((b[np.array(g)==s]-y[np.array(g)==s])**2) for s in groups]))
 effect=1-math.sqrt(sum(x.sum() for x in sb)/max(sum(x.sum() for x in sa),1e-30));rng=np.random.default_rng(294000037);effects=[]
 for _ in range(replicates):
  aa=bb=0.
  for i in rng.integers(0,len(sa),len(sa)):
   jj=rng.integers(0,len(sa[i]),len(sa[i]));aa+=sa[i][jj].sum();bb+=sb[i][jj].sum()
  effects.append(1-math.sqrt(bb/max(aa,1e-30)))
 return {'RMSE_improvement':float(effect),'CI95':np.quantile(effects,[.025,.975]).tolist(),'CI90':np.quantile(effects,[.05,.95]).tolist(),'replicates':replicates,'seed_count':len(records),'resampling':'seed, then independent streams within each sampled seed; paired losses'}

def development(w):
 result={'seeds':{},'selection_used_final':False,'target':'prospective actual utility','post_outcome_oracle_is_not_native':True}
 for seed in [37,211]:
  base=w/f'development/seed-{seed}';info={};curves={};native={}
  for d in ['D0','D1','D2','D3','D4']:
   train=read(base/f'utility-acquisition/{d}-rows.json');test=read(base/f'development-acquisition/{d}-rows.json');info[d]=representation_audit(train,test)
  for arm in ARMS:
   d='D0' if arm=='A' else 'D1' if arm=='B' else 'D3';train=read(base/f'utility-acquisition/{d}-rows.json');test=read(base/f'development-acquisition/{d}-rows.json');y=np.array([r['Utility'] for r in test]);c=read(base/arm/'curves.json');pred,details=comparators(train,test);baselines={k:metric(y,v) for k,v in pred.items()};best=min(baselines,key=lambda k:baselines[k]['RMSE']);cc={}
   for n,v in c.items():
    m=metric(y,np.array(v['prediction']));m['best_comparator_2048_fit']=best;m['best_comparator_RMSE_2048_fit']=baselines[best]['RMSE'];m['note']='Comparator uses full development Fit; comparisons at earlier native learning-curve points are descriptive only.';cc[n]=m
   transfer={}
   for condition in ['unseen','shifted']:
    rr=read(base/f'transfer-{condition}/acquisition/{arm}-rows.json');yy=np.array([r['Utility'] for r in rr]);pp=np.array([r['Prediction'] for r in rr]);transfer[condition]=metric(yy,pp)
   curves[arm]=cc;native[arm]={'transfer':transfer,**{'baselines':baselines,'information_access':details,'full_budget_native':cc.get('2048'),'receipt_commits':sum(x['diagnosis'].get('committed',False) for x in read(base/arm/'receipts.json'))}}
  result['seeds'][str(seed)]={'information':info,'learning_curves':curves,'native':native}
 # Final freezes the minimum requested comparison budget. All 2048 dev
 # examples and failures remain; no final enlargement or seed selection.
 seen_pass={a:all((result['seeds'][str(s)]['native'][a]['full_budget_native'] or {}).get('RMSE',float('inf')) <= .9*min(v['RMSE'] for v in result['seeds'][str(s)]['native'][a]['baselines'].values()) and (result['seeds'][str(s)]['native'][a]['full_budget_native'] or {}).get('macro_F1',0)>=.4 for s in [37,211]) for a in ['D','E']}
 h1=False
 if any(seen_pass.values()):raise RuntimeError('H1 seen criteria reached: run independent transfer/CI gate before freezing next stage')
 result['development_seen_pass']=seen_pass
 result['decision']={'H1_pass':h1,'H2':'NOT RUN: independent development H1 not met','H3':'NOT RUN: H2 unavailable','descriptor':'D3','ridge':.01,'final_budget':512,'confirmatory_arms':ARMS,'final_seeds':SEEDS,'reason':'Keep original D0/D1/D3 comparisons and the generic conditioning repair plus curriculum/control. Development utility does not satisfy H1; data increase through 2048 does not establish a pass. Final is a frozen diagnostic validation, never an opportunity to reopen H2.'}
 write(w/'DEVELOPMENT_ANALYSIS.json',result);write(w/'DEVELOPMENT_DECISION.json',result['decision'])
 print(json.dumps(result['decision'],indent=2))
 for s,v in result['seeds'].items():
  print('seed',s,{a:(q['full_budget_native'] or {}).get('RMSE') for a,q in v['native'].items()})

def final(w):
 summary={'utility':{},'history':{},'retention':{},'performance':{},'per_seed':{},'controls':{},'gates':{}}
 for condition in ['seen','unseen','shifted']:
  pooled={a:{'y':[],'p':[],'base':collections.defaultdict(list),'records':collections.defaultdict(list),'ablations':collections.defaultdict(list),'by_seed':{}} for a in ARMS}
  for seed in SEEDS:
   base=w/f'confirmatory/seed-{seed}';fd=w/f'final/seed-{seed}/{condition}'
   for arm in ARMS:
    descriptor='D0' if arm=='A' else 'D1' if arm=='B' else 'D3';train=read(base/f'utility-acquisition/{descriptor}-rows.json');rows=read(fd/f'utility/{arm}-rows.json');y=np.array([r['Utility'] for r in rows]);p=np.array([r['Prediction'] for r in rows]);assert all(r['BeforeSequence']<r['OperationSequence'] for r in rows)
    comparisons,details=comparators(train,rows);pm=pooled[arm];pm['y'].extend(y);pm['p'].extend(p);stats=metric(y,p);stats['baseline_metrics']={k:metric(y,q) for k,q in comparisons.items()};stats['best_baseline']=min(comparisons,key=lambda k:stats['baseline_metrics'][k]['RMSE']);stats['best_baseline_RMSE']=stats['baseline_metrics'][stats['best_baseline']]['RMSE'];stats['catastrophic']=stats['RMSE']>.02 or stats['RMSE']>3*stats['best_baseline_RMSE'];stats['information_access']=details;pm['by_seed'][str(seed)]=stats
    for name,q in comparisons.items():pm['base'][name].extend(q);pm['records'][name].append((y,q,p,[r['Stream'] for r in rows]))
    ab=read(fd/f'ablation-{arm}.json')
    for name in ['delta_mask','association_swap','permutation','retrospective']:pm['ablations'][name].extend(ab[name])
    assert ab['permutation_exact_inputs']==len(rows)
    if len(rows):
     perf=summary['performance'].setdefault(arm,{'inference_ns':[],'descriptor_ns':[],'active_features':[]});perf['inference_ns'] += [r['PredictionNanoseconds'] for r in rows];perf['descriptor_ns'] += [r['DescriptorNanoseconds'] for r in rows];perf['active_features'] += [r['Representation']['ActiveFeatures'] for r in rows]
  cond={}
  for arm,pm in pooled.items():
   y,p=np.array(pm['y']),np.array(pm['p']);baselines={k:metric(y,v) for k,v in pm['base'].items()};best=min(baselines,key=lambda k:baselines[k]['RMSE']);contrast=hierarchical(pm['records'][best]);m=metric(y,p)
   ablation={k:metric(y,np.array(v)) for k,v in pm['ablations'].items()};gate=m['macro_F1']>=.4 and contrast['RMSE_improvement']>=.1 and contrast['CI95'][0]>0 and min(m['class_counts_negative_neutral_positive'])>=10 and not any(s['catastrophic'] for s in pm['by_seed'].values())
   cond[arm]={'native':m,'baselines':baselines,'best_baseline':best,'contrast':contrast,'per_seed':pm['by_seed'],'ablation':ablation,'gate_D_condition_pass':gate,'permutation_exact_all':True}
  summary['utility'][condition]=cond
 # History, all existing capability gates remain separate from utility.
 for condition in ['seen','unseen','shifted','independent']:
  full=[];zero=[];mag=[];actual=[];records=[];mask_records=[];perm_records=[];ratios=[]
  for seed in SEEDS:
   fd=w/f'final/seed-{seed}/{condition}';rows=read(fd/'native-rows.json');v=read(fd/'history.json');y=np.array([r['Trial']['Error'] for r in rows]);p=np.array(v['full']);z=np.array(v['matched_no_history']);g=[r['Stream'] for r in rows];actual.extend(y);full.extend(p);zero.extend(z);mag.extend(v['magnitude_ablation']);records.append((y,z,p,g));mask_records.append((y,np.array(v['magnitude_ablation']),p,g));perm=np.mean([(np.array(v[f'permute-{j}'])-y)**2 for j in range(10)],axis=0);perm_records.append((np.zeros(len(y)),np.sqrt(perm),p-y,g));ratios.append(metric(y,p))
  summary['history'][condition]={'native':metric(actual,full),'matched_no_history':metric(actual,zero),'benefit':hierarchical(records),'magnitude_ablation':hierarchical(mask_records),'conditional_permutation':hierarchical(perm_records),'per_seed':dict(zip(map(str,SEEDS),ratios)),'IID_interpretation':'independently IID-trained matched models on independent IID holdout' if condition=='independent' else None}
 # New-source distribution retention: report both absolute old and paired R7.
 old_limits={k:v['limit'] for k,v in read(w/'R6_RETENTION_LOCK.json').items()}
 for seed in SEEDS:
  v=read(w/f'final/seed-{seed}/retention.json');per={}
  for arm,item in v.items():
   stat={'historical':item['historical'],'new':{}}
   for name,bank in item['new'].items():
    rows=read(w/f'final/seed-{seed}/retention-{name}/rows.json');y=np.array([r['Error'] for r in rows]);a=np.array(bank['source_predictions']);b=np.array(bank['candidate_predictions']);contrast=hierarchical([(y,a,b,[r['ID'] for r in rows])],2000);ratio=[1-contrast['CI95'][1],1-contrast['CI95'][0]];stat['new'][name]={'source_RMSE':metric(y,a)['RMSE'],'candidate_RMSE':metric(y,b)['RMSE'],'R6_exact_absolute_limit':old_limits[name],'R6_source_absolute_pass':metric(y,a)['RMSE']<=old_limits[name],'R6_candidate_absolute_pass':metric(y,b)['RMSE']<=old_limits[name],'R7_paired_RMSE_ratio_CI95':ratio,'R7_paired_pass':ratio[1]<=1.05,'same_input_source_candidate':True,'S1_delta':bank['S1_max_delta']}
   per[arm]=stat
  summary['retention'][str(seed)]=per
 for arm,v in summary['performance'].items():
  for field in ['inference_ns','descriptor_ns','active_features']:
   x=v[field];v[field]={'N':len(x),'p50':float(np.quantile(x,.5)),'p95':float(np.quantile(x,.95)),'max':max(x)}
  costs=[];sizes=[];commits=opportunities=0;peak=0.
  for seed in SEEDS:
   d=w/f'confirmatory/seed-{seed}/{arm}';c=read(d/'costs.json');costs += [r['learning_nanoseconds'] for r in c];sizes.append({'seed':seed,'checkpoint_bytes':(d/'final.json').stat().st_size,'WAL_bytes':(d/'events.jsonl').stat().st_size,'WAL_suffix_bytes':(d/'events.jsonl').stat().st_size-PREFIX});receipts=read(d/'receipts.json');opportunities+=len(receipts);commits+=sum(r['diagnosis'].get('committed',False) for r in receipts)
   for r in receipts:
    for t in r['diagnosis'].get('trials') or []:
     if t.get('candidate'):peak=max(peak,max(abs(v) for v in t['candidate'].get('Readout',[]) or [0]))
  v['learning_latency_ns']={'N':len(costs),'p50':float(np.median(costs)) if costs else None,'p95':float(np.quantile(costs,.95)) if costs else None};v['sizes']=sizes;v['commits']=commits;v['opportunities']=opportunities;v['peak_candidate_coefficient']=peak
 h1={a:all(summary['utility'][c][a]['gate_D_condition_pass'] for c in ['seen','unseen','shifted']) for a in ARMS}
 hist=summary['history']['seen'];iid=summary['history']['independent']['benefit'];cal=hist['native']['calibration_ratio'];history_pass=hist['benefit']['RMSE_improvement']>=.1 and hist['benefit']['CI95'][0]>0 and hist['magnitude_ablation']['CI95'][0]>0 and hist['conditional_permutation']['CI95'][0]>0 and iid['CI90'][0]>=-.03 and iid['CI90'][1]<=.03 and .8<=cal<=1.2
 summary['gates']={'H1_by_arm':h1,'H2':'NOT RUN','H3':'NOT RUN','consolidation':'NOT PROVEN','history_and_calibration_pass':history_pass,'IID_training_equivalence': iid['CI90'][0]>=-.03 and iid['CI90'][1]<=.03,'verdict':'PARTIAL — INFORMATION LOSS DIAGNOSED; NATIVE UTILITY PREREQUISITE NOT MET'}
 write(w/'SCIENTIFIC_RESULTS.json',summary);print(json.dumps(summary['gates'],indent=2))
 for c in ['seen','unseen','shifted']:print(c,{a:round(v['native']['RMSE'],7) for a,v in summary['utility'][c].items()})

def main():
 p=argparse.ArgumentParser();p.add_argument('--work',type=Path,required=True);p.add_argument('--phase',choices=['development','final'],required=True);a=p.parse_args();(development if a.phase=='development' else final)(a.work)
if __name__=='__main__':main()
