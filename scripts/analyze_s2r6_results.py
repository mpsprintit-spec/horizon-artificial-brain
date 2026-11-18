#!/usr/bin/env python3
"""Frozen independent analyses; never called by Horizon decision paths."""
import argparse,collections,json
from pathlib import Path
import numpy as np
import analyze_s2r5_retention as old
import analyze_s2r6_utility as utility
SEEDS=[37,83,151,211,307,419]
ARMS=['A','A0','B','B0','BI','BI0','C','F']
def write(p,v):old.write(p,v)
def analyze(w):
 result={'utility':utility.calculate(w,'training',SEEDS),'history':{},'retention':{},'resources':{},'continual':{}}
 constants=old.read(w/'OFFLINE_BASELINE_LOCK.json');retlock=old.read(w/'RETENTION_LOCK.json')
 for condition in ['persistent','independent','shifted']:
  rows=old.read(w/f'holdout/{condition}/rows.json');_,ctx,y,g=old.arrays(rows);pred={};metrics={}
  for arm in ARMS:
   pp=[{k:np.array(v) for k,v in old.read(w/f'training/seed-{s}/{arm}/predictions-{condition}.json').items()} for s in SEEDS];pred[arm]=pp
   p=np.array([v['full'] for v in pp]);m=np.array([v['no-history'] for v in pp]);permuted=np.array([np.mean([(v[f'permute-{i:02d}']-y)**2 for i in range(10)],axis=0) for v in pp])
   pooled=old.metrics(np.tile(y,len(SEEDS)),p.ravel(),np.tile(ctx[:,0],len(SEEDS)))
   cs=np.array([np.repeat(constants[str(s)]['history_constant'],len(y)) for s in SEEDS])
   pooled['constant_Fit_RMSE']=float(np.sqrt(np.mean((cs-y)**2)))
   metrics[arm]={'pooled':pooled,'per_seed':{str(s):old.metrics(y,v['full'],ctx[:,0]) for s,v in zip(SEEDS,pp)},'magnitude_ablation':old.hierarchy((m-y)**2,(p-y)**2,g),'conditional_permutation':old.hierarchy(permuted,(p-y)**2,g)}
  contrasts={}
  for a,b in [('A','A0'),('B','B0'),('BI','BI0'),('C','B0'),('B','A'),('C','B'),('C','F')]:
   pa=np.array([v['full'] for v in pred[a]]);pb=np.array([v['full'] for v in pred[b]])
   contrasts[a+'-vs-'+b]=old.hierarchy((pb-y)**2,(pa-y)**2,g)
  result['history'][condition]={'arms':metrics,'contrasts':contrasts}
 for arm in ARMS:
  per=[];commits=0;trials=0;latency=[];checkpoint=[];walgrowth=[];reuse_updates=[];reject=collections.Counter();bank_reject=collections.Counter();syn_changes=[]
  for seed in SEEDS:
   d=w/f'training/seed-{seed}/{arm}';h=old.read(d/'historical-retention.json');oldresults={name:old.read(d/f'old-retention-{name}.json') for name in ['normal','shifted']};ps=old.read(w/f'persistence-{seed}.json')[arm]
   rec=old.read(d/'receipts.json');cost=old.read(d/'costs.json');commits+=sum(r['diagnosis']['committed'] for r in rec);trials+=sum(len(r['diagnosis']['trials']) for r in rec)
   for r in rec:
    for trial in r['diagnosis']['trials']:
     reject.update(trial.get('rejected_by') or [])
     for bank,n in enumerate(trial.get('retention_bank_regressions') or []):bank_reject[str(bank)]+=int(n>0)
   latency.extend(c['learning_nanoseconds'] for c in cost)
   for name in ['final.json','events.jsonl']:checkpoint.append((d/name).stat().st_size) if name=='final.json' else walgrowth.append((d/name).stat().st_size-(w/'source-events.jsonl').stat().st_size)
   passed={'historical':h['fixed_S2_pass'],'strict_S1':h['S1_max_delta']==0,'AE1':ps['AE1']['pass']}
   for name,v in oldresults.items():passed[name]=v['S2_candidate']['RMSE']<=retlock[name]['limit'] and v['S1_max_delta']==0
   per.append({'seed':seed,'historical_RMSE':h['S2_candidate']['RMSE'],'historical_S1_shift':h['S1_max_delta'],'new':{name:{'candidate_RMSE':v['S2_candidate']['RMSE'],'source_RMSE':v['S2_source']['RMSE'],'frozen_limit':retlock[name]['limit']} for name,v in oldresults.items()},'gates':passed,'all_retention_pass':all(passed.values()),'persistence':ps})
   # Quantify actual canonical signed parameter/eligibility changes per block.
   before=old.read(d/'state.json')['brain']
   trajectory=[]
   for block in range(4):
    after=old.read(d/f'block-{block}.json')['brain']
    def encode(brain):
     parameters=collections.defaultdict(float)
     for syn in brain['synapses']:
      parameters[(syn['source_id'],syn['target_id'])]+=(-1 if syn['inhibitory'] else 1)*8*syn['weight']
     return dict(parameters)
    a,b=encode(before),encode(after);keys=set(a)|set(b);delta=[b.get(k,0.)-a.get(k,0.) for k in keys if b.get(k,0.)!=a.get(k,0.)]
    trajectory.append({'block':block,'changed_signed_parameter_pairs':len(delta),'max_signed_delta':max(map(abs,delta),default=0.),'l1_signed_delta':float(sum(map(abs,delta))),'units':len(after['neural_units']),'synapses':len(after['synapses']),'features':len(after['n_resolution']['features']),'native_policy_updates':after['native_control']['updates']})
    before=after
   syn_changes.append({'seed':seed,'blocks':trajectory})
  result['retention'][arm]={'per_seed':per,'pass_counts':{k:sum(r['gates'][k] for r in per) for k in per[0]['gates']},'all_pass':all(r['all_retention_pass'] for r in per)}
  resource_rows=[old.read(w/f'training/seed-{s}/{arm}/resources.json') for s in SEEDS]
  inference=[v for r in resource_rows for v in r['persistent']['inference_nanoseconds']]
  resources={'commits':commits,'opportunities':24,'candidate_trials':trials,'learning_nanoseconds_median':float(np.median(latency)),'inference_p50_nanoseconds':float(np.percentile(inference,50)),'inference_p95_nanoseconds':float(np.percentile(inference,95)),'checkpoint_bytes_median':float(np.median(checkpoint)),'WAL_suffix_bytes_median':float(np.median(walgrowth)),'rejection_counts':dict(reject),'additional_bank_rejected_trials':dict(bank_reject),'per_seed_inventory':[r['persistent']['inventory'] for r in resource_rows]}
  if arm in ['C','F']:
   costs=[]
   for seed in SEEDS:
    rows=old.read(w/f'training/seed-{seed}/{arm}/eval-seen/rows.json');costs.extend(r['PredictionNanoseconds'] for r in rows)
   resources.update(utility_p50_nanoseconds=float(np.percentile(costs,50)),utility_p95_nanoseconds=float(np.percentile(costs,95)))
  result['resources'][arm]=resources;result['continual'][arm]=syn_changes
 c=result['history']['persistent']['arms']['C'];hist=result['history']['persistent']['contrasts']['C-vs-B0'];iid=result['history']['independent']['contrasts']['BI-vs-BI0'];shift=result['history']['shifted']['contrasts']['C-vs-B0']
 historypass=hist['relative_RMSE_improvement']>=.10 and hist['CI95'][0]>0 and c['magnitude_ablation']['CI95'][0]>0 and c['conditional_permutation']['CI95'][0]>0 and iid['CI90'][0]>=-.03 and iid['CI90'][1]<=.03
 calibration=c['pooled'];calpass=.8<=calibration['mean_ratio']<=1.2 and calibration['RMSE']<calibration['constant_Fit_RMSE'] and result['retention']['C']['all_pass']
 result['gates']={'A':{'status':'PASS','basis':'authenticated R2/R5 source, source hashes, six frozen seeds'},'B':{'status':'PASS' if result['utility']['H1_pass'] else 'FAIL','basis':'independent utility forecasts vs Fit-only baselines, unseen pathways and shifted contexts'},'C':{'status':'NOT RUN','basis':'H1 development prerequisite failed'},'D':{'status':'NOT RUN','basis':'H1/H2 development prerequisites unavailable; no adaptive selector installed'},'E':{'status':'PASS' if result['retention']['C']['all_pass'] else 'FAIL','basis':result['retention']['C']['pass_counts']},'F':{'status':'PASS' if historypass else 'FAIL','basis':{'paired':hist,'shifted':shift,'IID':iid}},'G':{'status':'PASS' if calpass else 'FAIL','basis':calibration},'H':{'status':'NOT PROVEN','basis':'four serial utility blocks executed; distinct-ability consolidation/correction protocol not reached'},'I':{'status':'PASS','basis':'one N, 32 shared features, all learned coefficients canonical signed synapses; no utility controller'},'J':{'status':'PASS','basis':'48 final checkpoints: exact restart/full suffix replay/signed reconstruction/rollback, historical prefix intact'}}
 result['verdict']='PARTIAL — UTILITY PREREQUISITE NOT MET' if not result['utility']['H1_pass'] else 'PARTIAL — ADAPTIVE PLASTICITY NOT TESTED'
 result['H1']=result['gates']['B']['status'];result['H2']='NOT RUN';result['H3']='NOT RUN';result['H4']='NOT RUN'
 return result
def main():
 p=argparse.ArgumentParser();p.add_argument('--work',type=Path,required=True);p.add_argument('--phase',choices=['interventions','final'],required=True);a=p.parse_args()
 if a.phase=='interventions':old.interventions(a.work);return
 out=analyze(a.work);write(a.work/'SCIENTIFIC_RESULTS.json',out)
 print(json.dumps({'verdict':out['verdict'],'gates':{k:v['status'] for k,v in out['gates'].items()},'utility':out['utility']['results']},indent=2))
if __name__=='__main__':main()
