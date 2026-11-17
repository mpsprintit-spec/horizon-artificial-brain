#!/usr/bin/env python3
"""Frozen S2-R5 statistical diagnosis only. Never a Horizon runtime model."""
import argparse,json,hashlib,collections
from pathlib import Path
import numpy as np
SEEDS=[37,83,151,211,307,419]
ARMS=['A','A0','B','B0','C','C0','Cw','Cw0','D','D0','E','E0','AI','AI0','DI','DI0','EI','EI0']
def read(p):return json.loads(p.read_text())
def write(p,v):p.parent.mkdir(parents=True,exist_ok=True);p.write_text(json.dumps(v,indent=2,sort_keys=True,allow_nan=False)+'\n')
def arrays(rows):
 y=[];groups=[];x=[];contexts=[]
 for r in rows:
  p=r['Trial']['Pre'];x.append([p['Before'].get(str(j),0.) for j in range(1,37)]+(p['Previous'] or [0.]*9)+p['Forecast']);contexts.append([p['Action'],p['PreviousAction'],p['HistoryAction'],len(p['History'] or [])]);y.append(r['Trial']['Error']);groups.append(r['Stream'])
 return np.array(x),np.array(contexts),np.array(y),np.array(groups)
def metrics(y,p,ops=None):
 e=p-y;bins=np.searchsorted([0,.01,.02,.04,.06,.08,.12,.2,.4],p);cal=sum(abs(e[bins==j].mean())*(bins==j).mean() for j in set(bins));v={'N':len(y),'RMSE':float(np.sqrt(np.mean(e*e))),'MAE':float(np.mean(abs(e))),'mean_actual':float(y.mean()),'mean_prediction':float(p.mean()),'mean_bias':float(e.mean()),'mean_ratio':float(p.mean()/y.mean()) if y.mean()!=0 else None,'calibration_error':float(cal),'residual_quantiles':np.quantile(e,[0,.05,.25,.5,.75,.95,1]).tolist()}
 if ops is not None:v['operations']={str(a):metrics(y[ops==a],p[ops==a]) for a in sorted(set(ops))}
 return v
def hierarchy(a,b,g):
 gs=list(dict.fromkeys(g));sa=np.array([[r[g==x].sum() for x in gs] for r in a]);sb=np.array([[r[g==x].sum() for x in gs] for r in b]);rng=np.random.default_rng(9500033);effects=[]
 for _ in range(5000):
  ii=rng.integers(0,len(a),len(a));jj=rng.integers(0,len(gs),len(gs));aa=sa[ii][:,jj].sum();bb=sb[ii][:,jj].sum();effects.append(1-np.sqrt(bb/aa) if aa>0 else 0.)
 return {'relative_RMSE_improvement':float(1-np.sqrt(np.mean(b)/np.mean(a))),'CI95':np.quantile(effects,[.025,.975]).tolist(),'CI90':np.quantile(effects,[.05,.95]).tolist(),'seed_effects':[float(1-np.sqrt(bb.mean()/aa.mean())) for aa,bb in zip(a,b)],'replicates':5000,'training_seeds':len(a),'streams':len(gs),'resampling':'paired training-seeds and independent whole streams'}
def freeze_metrics(w):
 if (w/'FINAL_STARTED').exists():raise RuntimeError('cannot change controls after final')
 rows=read(w/'development/persistent/rows.json');groups=list(dict.fromkeys(r['Stream'] for r in rows));fit=[r for r in rows if r['Stream'] in groups[:8]];x,_,_,_=arrays(fit);sd=x.std(0);sd=np.where(sd>1e-10,sd,1.)
 lock={'mean':x.mean(0).tolist(),'sd':sd.tolist(),'scaling':'development Fit only, all 36 observed neurons + previous state + forecast','neighbors':12,'permutation_seeds':list(range(9100031,9100041)),'bootstrap_replicates':5000,'IID_equivalence_margin':.03,'history_gate':.1,'historical_retention_limit':.01766457,'source_checksum':'9016b845eed4e8437e87ada99b9641718a086c570e246dd0a62e57dffd1ce9d3','calibration_bins':[0,.01,.02,.04,.06,.08,.12,.2,.4]}
 write(w/'METRIC_LOCK.json',lock)
 coeff={}
 for seed in SEEDS:
  d=read(w/f'calibration-controls/seed-{seed}.json');rows=d['rows'];y=np.array([r['Trial']['Error'] for r in rows]);streams=list(dict.fromkeys(r['Stream'] for r in rows));fit=np.array([streams.index(r['Stream'])%4<2 for r in rows]);check=~fit;coeff[str(seed)]={}
  for model in ['source','A']:
   p=np.array(d[model]);beta=float(np.mean(y[fit]-p[fit]));aff=np.linalg.lstsq(np.c_[p[fit],np.ones(fit.sum())],y[fit],rcond=None)[0];constant=float(y[fit].mean());oldy=np.array([r['Error'] for r in d['historical_rows']]);oldp=np.array(d['historical_'+model]);v={'bias':beta,'affine':aff.tolist(),'constant':constant,'Fit_ids':[r['Trial']['ID'] for i,r in enumerate(rows) if fit[i]],'Check_ids':[r['Trial']['ID'] for i,r in enumerate(rows) if check[i]],'check':{},'historical':{}}
   for name,qq in {'original':p,'bias':np.clip(p+beta,-1,1),'affine':np.clip(aff[0]*p+aff[1],-1,1),'constant':np.full_like(p,constant)}.items():v['check'][name]=metrics(y[check],qq[check])
   for name,qq in {'original':oldp,'bias':np.clip(oldp+beta,-1,1),'affine':np.clip(aff[0]*oldp+aff[1],-1,1),'constant':np.full_like(oldp,constant)}.items():v['historical'][name]=metrics(oldy,qq)
   coeff[str(seed)][model]=v
 write(w/'OFFLINE_CALIBRATION_LOCK.json',coeff)
 print('Fit-only metrics and calibration controls frozen',flush=True)
def interventions(w):
 lock=read(w/'METRIC_LOCK.json');mean=np.array(lock['mean']);sd=np.array(lock['sd'])
 for condition in ['persistent','independent','shifted']:
  d=w/'holdout'/condition;rows=read(d/'rows.json');x,c,y,g=arrays(rows);z=(x-mean)/sd;distance=np.maximum(0,(z*z).sum(1)[:,None]+(z*z).sum(1)[None,:]-2*z@z.T);choices=[];dd=[]
  for i in range(len(rows)):
   possible=np.flatnonzero(np.all(c==c[i],axis=1)&(g!=g[i]))
   if c[i,3]!=3 or not len(possible):choices.append(np.array([i]));continue
   near=possible[np.argsort(distance[i,possible],kind='stable')[:12]];choices.append(near);dd.extend(np.sqrt(distance[i,near]).tolist())
  donors={}
  for j,seed in enumerate(lock['permutation_seeds']):
   rng=np.random.default_rng(seed);donors[f'permute-{j:02d}']=[int(rng.choice(v)) for v in choices]
  write(d/'input-interventions.json',{'Donors':donors,'target_blind':True,'same_discrete_context':True,'donors_other_streams':True,'neighbors':12,'approximate_conditional_permutation':True,'distance_quantiles':np.quantile(dd,[.5,.95]).tolist(),'continuous_dimensions':len(mean)})
 print('Target-blind interventions saved',flush=True)
def final(w):
 summary={};calibrations={};coeff=read(w/'OFFLINE_CALIBRATION_LOCK.json')
 for cond in ['persistent','independent','shifted']:
  rows=read(w/f'holdout/{cond}/rows.json');x,ctx,y,g=arrays(rows);pred={};armstats={};ops=ctx[:,0]
  for arm in ARMS:
   pp=[{k:np.array(v) for k,v in read(w/f'training/seed-{s}/{arm}/predictions-{cond}.json').items()} for s in SEEDS];pred[arm]=pp;full=np.array([p['full'] for p in pp]);loss=(full-y)**2;maskloss=(np.array([p['no-history'] for p in pp])-y)**2;permloss=np.array([np.mean([(p[f'permute-{j:02d}']-y)**2 for j in range(10)],axis=0) for p in pp])
   armstats[arm]={'pooled':metrics(np.tile(y,len(SEEDS)),full.ravel(),np.tile(ops,len(SEEDS))),'seeds':{str(s):metrics(y,p['full'],ops) for s,p in zip(SEEDS,pp)},'history_ablation':hierarchy(maskloss,loss,g),'conditional_permutation':hierarchy(permloss,loss,g),'changed_predictions_on_ablation':int(np.count_nonzero(full-np.array([p['no-history'] for p in pp]))),'maximum_ablation_shift':float(np.max(abs(full-np.array([p['no-history'] for p in pp])))),'per_stream':{stream:{str(s):metrics(y[g==stream],pp[i]['full'][g==stream]) for i,s in enumerate(SEEDS)} for stream in dict.fromkeys(g)}}
  contrasts={}
  for with_hist,no_hist in [(a,a+'0') for a in ['A','B','C','Cw','D','E','AI','DI','EI']]+[('C','A'),('Cw','C'),('D','Cw'),('E','D')]:
   aa=(np.array([p['full'] for p in pred[no_hist]])-y)**2;bb=(np.array([p['full'] for p in pred[with_hist]])-y)**2;contrasts[with_hist+'-vs-'+no_hist]=hierarchy(aa,bb,g)
  summary[cond]={'arms':armstats,'contrasts':contrasts}
  calibrations[cond]={}
  for model in ['source','A']:
   pairs={name:[] for name in ['original','bias','affine','constant']}
   for i,seed in enumerate(SEEDS):
    p=np.array([r['Trial']['Estimate'] for r in rows]) if model=='source' else pred['A'][i]['full'];c=coeff[str(seed)][model]
    for name,qq in {'original':p,'bias':np.clip(p+c['bias'],-1,1),'affine':np.clip(c['affine'][0]*p+c['affine'][1],-1,1),'constant':np.full_like(p,c['constant'])}.items():pairs[name].append(qq)
   calibrations[cond][model]={name:metrics(np.tile(y,len(SEEDS)),np.array(pp).ravel()) for name,pp in pairs.items()}
  print(cond,[(a,round(armstats[a]['pooled']['RMSE'],6)) for a in ['A','B','C','Cw','D','E']],flush=True)
 write(w/'analysis/confirmatory-metrics.json',summary);write(w/'analysis/offline-calibration-final.json',calibrations)
 admission={};retention={};cost={};pareto=[]
 for seed in SEEDS:
  for arm in ARMS:
   d=w/f'training/seed-{seed}/{arm}';receipts=read(d/'receipts.json');trials=[tr for rec in receipts for tr in rec['diagnosis']['trials']];why=collections.Counter(r for tr in trials for r in tr.get('rejected_by',[]));commits=sum(rec['diagnosis']['committed'] for rec in receipts)
   admission[f's{seed}-{arm}']={'commits':commits,'opportunities':len(receipts),'trials':len(trials),'retention_feasible_candidates':sum(tr['retention_regressions']==0 for tr in trials),'eligible_candidates':sum(tr['eligible'] for tr in trials),'rejections':dict(why),'committed_retention_violations':sum(any(tr['eligible'] and tr['retention_regressions']>0 for tr in rec['diagnosis']['trials']) for rec in receipts)}
   ret=read(d/'final-retention.json');extra=read(d/'new-retention.json');s1=read(d/'strict-s1.json');retention[f's{seed}-{arm}']={'historical_RMSE':ret['S2_candidate']['RMSE'],'historical_pass':ret['fixed_external_pass'],'strict_S1_pass':ret['strict_S1_pass'] and all(v['max_delta']==0 and v['exact'] for v in s1.values()) and all(v['strict_S1_pass'] for v in extra),'new_retention':extra,'new_pass':all(v['pass'] for v in extra),'original_witnesses_unchanged':ret['witnesses_exact'],'policy_unchanged':ret['native_policy_exact']}
   cc=read(d/'costs.json');res=read(d/'resources.json');constraint=[tr['candidate']['Constraint'] for tr in trials if tr.get('candidate',{}).get('Constraint')]
   cost[f's{seed}-{arm}']={'training_ns':sum(c['wall_nanoseconds'] for c in cc),'candidate_evaluations':len(trials),'solver_sweeps':sum(c['Iterations'] for c in constraint),'line_evaluations':sum(c['LineSearches'] for c in constraint),'signed_bound_activity':sum(c['BoundActive'] for c in constraint),'final_checkpoint_bytes':(d/'final.json').stat().st_size,'journal_bytes':(d/'events.jsonl').stat().st_size,'resources':res,'solver_statuses':dict(collections.Counter(c['Status'] for c in constraint))}
   for i,rec in enumerate(receipts):
    for tr in rec['diagnosis']['trials']:
     if tr.get('candidate',{}).get('Constraint'):pareto.append({'seed':seed,'arm':arm,'lesson':i,'kind':tr['kind'],'CheckMSE':tr['check_error'],'eligible':tr['eligible'],'rejected':tr.get('rejected_by',[]),**tr['candidate']['Constraint']})
 write(w/'analysis/admission.json',admission);write(w/'analysis/retention.json',retention);write(w/'analysis/resources.json',cost);write(w/'analysis/feasibility-pareto.json',pareto)
 decisions={}
 for arm in ['A','B','C','Cw','D','E']:
  p=summary['persistent']['contrasts'][arm+'-vs-'+arm+'0'];ret=[retention[f's{s}-{arm}'] for s in SEEDS];rpass=all(r['historical_pass'] and r['strict_S1_pass'] and r['new_pass'] for r in ret);cal=summary['persistent']['arms'][arm]['pooled']['mean_ratio'];iid=summary['independent']['contrasts'].get(arm+'I-vs-'+arm+'I0');equiv=iid is not None and iid['CI90'][0]>-.03 and iid['CI90'][1]<.03
  decisions[arm]={'mean_calibration_pass':.8<=cal<=1.2,'history_improvement_gate':p['relative_RMSE_improvement']>=.1 and p['CI95'][0]>0,'IID_equivalence_pass':equiv,'retention_S1_S2_pass':rpass,'historical_pass_seeds':sum(r['historical_pass'] for r in ret),'new_pass_seeds':sum(r['new_pass'] for r in ret),'strict_S1_pass_seeds':sum(r['strict_S1_pass'] for r in ret),'NOTE':'AE1 and exact replay gates joined after persistence; no promotion by RMSE alone'}
 write(w/'analysis/preliminary-decisions.json',decisions)
def main():
 p=argparse.ArgumentParser();p.add_argument('--work',type=Path,required=True);p.add_argument('--phase',choices=['freeze-metrics','interventions','final'],required=True);a=p.parse_args();{'freeze-metrics':freeze_metrics,'interventions':interventions,'final':final}[a.phase](a.work.resolve())
if __name__=='__main__':main()
