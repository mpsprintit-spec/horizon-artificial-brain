import json,pathlib,statistics as s,math,sys
root=pathlib.Path(sys.argv[1])
seeds=[2027,2039,2053,2063,2081]
def ci(x):
 m=s.mean(x);half=2.7764451051977987*s.stdev(x)/math.sqrt(len(x)) if len(x)>1 else 0
 return dict(mean=m,ci95=[m-half,m+half],seeds=len(x))
models={}
for arm in ['relational','identity','softmax']:
 paths=[root/f'confirm-{arm}/seed{seed}.json' for seed in seeds]
 if not all(p.exists() for p in paths):continue
 data=[json.load(open(p)) for p in paths]
 models[arm]={
 'evaluation':{split:{a:{metric:ci([v['evaluation'][split][a][metric] for v in data]) for metric in ['net_return','accuracy','operations','stops']} for a in data[0]['evaluation'][split]} for split in data[0]['evaluation']},
 'adaptation':{split:{metric:ci([v['adaptation'][split][metric] for v in data]) for metric in ['net_return','accuracy','operations']} for split in data[0]['adaptation']},
 'parameters':{'total':[v['parameters'] for v in data],'changed':[v['changed_parameters'] for v in data]},
 'stochastic_training_return':ci([v['stochastic_training']['net_return'] for v in data])}
comparisons={}
if all(k in models for k in ['relational','identity','softmax']):
 data={arm:[json.load(open(root/f'confirm-{arm}/seed{seed}.json')) for seed in seeds] for arm in models}
 for split in data['relational'][0]['evaluation']:
  comparisons[split]={}
  for arm,policy in [('identity','trained'),('relational','static3'),('relational','static5'),('relational','static7')]:
   comparisons[split][f'against-{arm}-{policy}']=ci([a['evaluation'][split]['trained']['net_return']-b['evaluation'][split][policy]['net_return'] for a,b in zip(data['relational'],data[arm])])
credit={}
for arm in ['paired','unpaired']:
 paths=[root/f'confirm-credit-{arm}/seed{seed}.json' for seed in seeds]
 if not all(p.exists() for p in paths):continue
 data=[json.load(open(p)) for p in paths]
 credit[arm]={k:ci([v[k] for v in data]) for k in ['Precision','NuisanceSensitivity','RelevantChanges','IrrelevantChanges','InactiveChanges']}
 credit[arm]['return']=ci([v['After']['net_return'] for v in data]);credit[arm]['before']=ci([v['Before']['net_return'] for v in data]);credit[arm]['episodes']=[v['executed_episodes'] for v in data]
summary={'models':models,'paired_comparisons':comparisons,'credit':credit}
(root/'summary.json').write_text(json.dumps(summary,indent=2)+'\n')
print('complete',list(models),list(credit))
for arm,m in models.items():print(arm,{split:round(v['trained']['net_return']['mean'],6) for split,v in m['evaluation'].items()},'adaptation',{key:round(v['net_return']['mean'],6) for key,v in m['adaptation'].items()})
print('credit',credit)
