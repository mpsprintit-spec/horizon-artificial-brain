#!/usr/bin/env python3
"""Read-only preregistered paired analysis; never fits or selects a Brain."""
import argparse
import hashlib
import json
from pathlib import Path
import numpy as np

SEEDS=[37,83,151]
def read(p):return json.loads(p.read_text())
def write(p,v):p.write_text(json.dumps(v,indent=2,sort_keys=True,allow_nan=False)+'\n')
def corr(a,b):
    a=np.asarray(a);b=np.asarray(b)
    if np.std(a)<1e-16 or np.std(b)<1e-16:return 0.
    return float(np.corrcoef(a,b)[0,1])
def rmse(y,p):return float(np.sqrt(np.mean((np.asarray(y)-np.asarray(p))**2)))

class Paired:
    def __init__(self,n=1024):
        self.n=n;self.cache={}
    def effect(self,ref,treatment,kind='improvement',stream='pooled'):
        ref=np.asarray(ref);treatment=np.asarray(treatment)
        if ref.ndim==1:ref=ref[None,:];treatment=treatment[None,:]
        lo,hi=(0,self.n) if stream=='pooled' else ((0,self.n//2) if stream=='normal' else (self.n//2,self.n))
        a=ref[:,lo:hi];b=treatment[:,lo:hi]
        sign=1 if kind=='improvement' else -1
        point=sign*(1-np.sqrt(b.mean()/a.mean()))
        key=(len(a),lo,hi)
        if key not in self.cache:
            rng=np.random.default_rng(980017);outer=rng.integers(0,len(a),size=(2000,len(a)))
            # Four-case temporal clusters, separate streams for pooled CIs.
            length=hi-lo;half=self.n//2
            chunks=[length] if stream!='pooled' else [half,half]
            pieces=[];offset=0
            for size in chunks:
                blocks=rng.integers(0,size//4,size=(2000,size//4))*4
                pieces.append((blocks[:,:,None]+np.arange(4)).reshape(2000,size)+offset);offset+=size
            self.cache[key]=(outer,np.concatenate(pieces,axis=1))
        outer,indices=self.cache[key]
        av=a[outer[:,:,None],indices[:,None,:]].mean(axis=(1,2));bv=b[outer[:,:,None],indices[:,None,:]].mean(axis=(1,2))
        boots=sign*(1-np.sqrt(bv/av));ci=np.quantile(boots,[.025,.975])
        return {'relative_RMSE_effect':float(point),'CI95':[float(x) for x in ci],'seed_effects':[float(sign*(1-np.sqrt(bb.mean()/aa.mean()))) for aa,bb in zip(a,b)],'n_cases':hi-lo,'training_seeds':len(a),'bootstrap_replicates':2000,'temporal_block_size':4}

def main():
    p=argparse.ArgumentParser(description=__doc__);p.add_argument('--work',type=Path,required=True);p.add_argument('--output',type=Path,required=True);a=p.parse_args();w=a.work;o=a.output;o.mkdir(exist_ok=True,parents=True)
    rows=read(w/'holdout.json');y=np.array([r['Error'] for r in rows]);pred=read(w/'predictions.json');metrics=read(w/'metrics.json');pair=Paired(len(y))
    def loss(cap,condition='full'):
        if condition.startswith('permute-'):
            return np.array([np.mean([(y-np.array(pred[f's{s}-c{cap}/{condition}-{j:02d}']))**2 for j in range(10)],axis=0) for s in SEEDS])
        return np.array([(y-np.array(pred[f's{s}-c{cap}/{condition}']))**2 for s in SEEDS])
    effects={stream:pair.effect(loss(32),loss(64),stream=stream) for stream in ['pooled','normal','shifted']}
    write(o/'capacity-effects.json',effects)
    interventions={}
    for cap in [32,64]:
        interventions[str(cap)]={}
        for condition in ['no-history','no-forecast','no-previous-operation','no-presence','no-history-operation','no-extras','permute-history','permute-forecast','new-feature-zero','new-readout-zero','all-readout-zero']:
            interventions[str(cap)][condition]={stream:pair.effect(loss(cap),loss(cap,condition),'removal-cost',stream) for stream in ['pooled','normal','shifted']}
    write(o/'input-feature-interventions.json',interventions)
    feature_analysis={};growth={};admission={}
    for seed in SEEDS:
        progress=read(w/f'seed-{seed}/progress.json');growth[str(seed)]=progress
        for cap in [32,64]:
            name=f's{seed}-c{cap}';audit=read(w/(name+'-feature-audit.json'));model=audit['Model'];circuits=model['Circuits'];head=model['Heads'][0]
            values=np.array(audit['Metric'] if head['Metric'] else audit['Activation']);contrib=np.array(audit['Contributions']);codes=np.array(audit['Code']);activation=np.array(audit['Activation']);err=np.array(audit['MetricError'])
            target=y-np.array(pred[name+'/full']);old=values[:,:32];design=np.column_stack([old,np.ones(len(old))]);rank=int(np.linalg.matrix_rank(design));fstats=[]
            for i,c in enumerate(circuits):
                is_new=i>=32;node=c['Meta']['node'];ablation=pred[f'{name}/feature-zero-{node}'];full=pred[name+'/full']
                maxcorr=max([abs(corr(values[:,i],values[:,j])) for j in range(i)]+[0.])
                novelty=None
                if is_new:
                    # Unlabeled descriptive projection only, no fitted predictor.
                    residual=values[:,i]-design@np.linalg.lstsq(design,values[:,i],rcond=None)[0]
                    novelty=float(np.var(residual)/max(np.var(values[:,i]),1e-30))
                fstats.append({'node':node,'kind':c['Meta']['kind'],'parents':c['Meta'].get('parents',[]),'new':is_new,'signed_readout':head['W'][i],'center':c['Center'],'delta':c['Delta'],'axis':c['Axis'],'std':float(np.std(values[:,i])),'saturated_activation_fraction':float(np.mean(np.abs(activation[:,i])>=.99)),'unique_codes':len(np.unique(codes[:,i])),'mean_metric_error_bound':float(np.mean(err[:,i])),'contribution_RMS':float(np.sqrt(np.mean(contrib[:,i]**2))),'max_correlation_prior_features':maxcorr,'unexplained_variance_by_old_features':novelty,'corr_full_residual':corr(values[:,i],target),'corr_32_residual':corr(values[:,i],y-np.array(pred[f's{seed}-c32/full'])),'ablation_RMSE':{s:rmse(y[sl],np.array(ablation)[sl]) for s,sl in [('pooled',slice(None)),('normal',slice(0,512)),('shifted',slice(512,None))]},'ablation_paired':pair.effect((y-np.array(full))**2,(y-np.array(ablation))**2,'removal-cost')})
            feature_analysis[name]={'feature_count':len(circuits),'old_feature_matrix_rank_with_bias':rank,'head_metric':head['Metric'],'features':fstats,'pairwise_correlations':[[corr(values[:,i],values[:,j]) for j in range(len(circuits))] for i in range(len(circuits))]}
            receipts=[read(w/f'seed-{seed}/arm-{cap}/receipt-{b}.json') for b in range(8)]
            blocks=[]
            for b,r in enumerate(receipts):
                d=r['diagnosis'];counts={};kinds={}
                for t in d['trials']:
                    kinds[t['kind']]=kinds.get(t['kind'],0)+1
                    for reason in t.get('rejected_by',[]):counts[reason]=counts.get(reason,0)+1
                blocks.append({'block':b,'committed':d['committed'],'grown':d.get('grown',0),'cause':d['cause'],'trial_kinds':kinds,'rejections':counts,'evidence':d.get('evidence'),'winner':next((t for t in reversed(d['trials']) if t['eligible']),None),'trials':len(d['trials'])})
            admission[name]=blocks
    write(o/'feature-representation.json',feature_analysis);write(o/'growth.json',growth);write(o/'admission.json',admission)
    # Explicit feature/neuron/synapse growth map relative to canonical source.
    source=read(w/'00-source.json')['brain'];source_units={u['id'] for u in source['neural_units']};source_edges={(e['source_id'],e['target_id'],e['inhibitory']) for e in source['synapses']}
    maps={};signed_paths={}
    for seed in SEEDS:
        for cap in [32,64]:
            name=f's{seed}-c{cap}';brain=read(w/f'seed-{seed}/arm-{cap}/final.json')['brain'];maps[name]={'new_neurons':[u for u in brain['neural_units'] if u['id'] not in source_units],'new_synapses':[e for e in brain['synapses'] if (e['source_id'],e['target_id'],e['inhibitory']) not in source_edges],'feature_metadata':brain['n_resolution']['features'],'heads':brain['n_resolution']['heads']}
            edges={(e['source_id'],e['target_id'],e['inhibitory']):e['dynamic']['weight'] for e in brain['synapses']}
            def parameter(froms,tos):return sum(8*(edges.get((x,z,False),0)-edges.get((x,z,True),0)) for x in froms for z in tos)/(len(froms)*len(tos))
            graph=read(w/(name+'-feature-audit.json'))['Model'];head=graph['Heads'][0];paths=[]
            for i,f in enumerate(graph['Circuits']):
                node=f['Meta']['node'];actual=parameter([node],head['Query'])
                if abs(actual-head['W'][i])>1e-12:raise ValueError('signed readout mismatch')
                path={'node':node,'kind':f['Meta']['kind'],'new':i>=32,'signed_readout':actual,'query':head['Query']}
                if f['Meta']['kind']==1:
                    path['axis_paths']=[]
                    for j,axis in enumerate(f['Axis']):
                        sid=brain['n_resolution']['sources'][j]
                        if sid:
                            actual_axis=parameter([sid],[node])
                            if abs(actual_axis-axis)>1e-12:raise ValueError('signed axis mismatch')
                            if axis:path['axis_paths'].append({'coordinate':j,'source_node':sid,'signed_weight':actual_axis})
                else:
                    path['parents']=[{'node':parent,'signed_weight':parameter([parent],[node])} for parent in f['Meta']['parents']]
                    if any(v['signed_weight']!=1 for v in path['parents']):raise ValueError('signed parent mismatch')
                paths.append(path)
            signed_paths[name]=paths
    write(o/'new-neural-structure.json',maps)
    write(o/'signed-synaptic-paths.json',signed_paths)
    controls={};positive_pass=True;negative_ok=True
    for seed in SEEDS:
        for positive in [True,False]:
            for cap in [32,64]:
                name=f's{seed}-positive{str(positive).lower()}-c{cap}';d=w/'information-controls'/name;cr=read(d/'test.json');cy=np.array([r['Error'] for r in cr]);cpred=read(d/'predictions.json');pb=Paired(len(cy))
                scores={key:rmse(cy,val) for key,val in cpred.items()}
                ab={key:pb.effect((cy-np.array(cpred['full']))**2,(cy-np.array(cpred[key]))**2,'removal-cost') for key in ['no-history','permute-history']}
                gain=1-scores['full']/scores['constant'];passed=gain>=.5 and all(v['CI95'][0]>0 for v in ab.values())
                if positive:positive_pass=positive_pass and passed
                else:negative_ok=negative_ok and gain<.02 and ab['permute-history']['CI95'][0]<=0
                audit=read(d/'representation-audit.json');head=audit['Model']['Heads'][0];av=np.array(audit['Metric'] if head['Metric'] else audit['Activation'])
                control_rep={'readout_mode':'metric' if head['Metric'] else 'activation','features':av.shape[1],'varying_feature_count':int(np.sum(np.std(av,axis=0)>1e-12)),'feature_std':[float(v) for v in np.std(av,axis=0)],'old_varying_feature_count':int(np.sum(np.std(av[:,:32],axis=0)>1e-12))}
                controls[name]={'RMSE':scores,'constant_improvement':gain,'interventions':ab,'positive_gate':passed if positive else None,'synthetic_only':True,'representation':control_rep}
    write(o/'information-controls.json',controls)
    persistence=read(w/'persistence.json');ret=read(w/'holdout-S1-retention.json')
    retention=all(v['pass'] for v in ret.values()) and all(v['AE1']['pass'] and v['policy_exact'] and v['witnesses_exact'] for v in persistence.values())
    persist=all(v['restart_exact'] and v['full_suffix_replay_exact'] and v['rollback_exact'] and v['historical_journal_prefix_exact'] and v['feature_metadata_exact'] for v in persistence.values())
    added=all(feature_analysis[f's{s}-c64']['feature_count']>32 for s in SEEDS)
    causal=all(interventions['64'][kind]['pooled']['CI95'][0]>0 for kind in ['new-feature-zero','new-readout-zero'])
    capacity_confirmed=added and effects['pooled']['CI95'][0]>0 and all(effects[s]['relative_RMSE_effect']>=.02 for s in ['normal','shifted']) and all(v>0 for v in effects['pooled']['seed_effects']) and causal and retention and persist
    history_equivalent=all(interventions[str(c)][cond][stream]['CI95'][1]<.02 for c in [32,64] for cond in ['no-history','permute-history'] for stream in ['normal','shifted'])
    history_supported=history_equivalent and positive_pass and negative_ok
    redundant_new=sum(f['new'] and f['unexplained_variance_by_old_features'] is not None and f['unexplained_variance_by_old_features']<.01 for v in feature_analysis.values() for f in v['features'])
    # Representation is a diagnostic finding, never inferred from a failed score alone.
    # Descriptive redundancy alone cannot establish the cause. Leave H2 open
    # unless an additional preregistered causal contrast identifies its failure.
    representation_supported=False
    status='DIAGNOSED' if capacity_confirmed or representation_supported or history_supported else ('PARTIAL' if effects['pooled']['relative_RMSE_effect']>0 and added else 'INCONCLUSIVE')
    summary={'status':status,'capacity_bottleneck_confirmed':capacity_confirmed,'representation_bottleneck_supported':representation_supported,'predictive_information_limitation_supported':history_supported,'capacity_effects':effects,'extra_features_recruited_all_seeds':added,'new_feature_causal_benefit_gate':causal,'positive_control_all_seeds_both_capacities':positive_pass,'negative_control_expected':negative_ok,'history_equivalence_under_2pct':history_equivalent,'strict_retention':retention,'persistence':persist,'redundant_new_features_below_1pct_novel_variance':redundant_new,'historical_S2_R2_ability':'PARTIAL','new_actual_cases':{'development':256,'training':3072,'confirmatory':1024},'training_seeds':SEEDS,'no_production_promotion':True}
    write(o/'summary.json',summary)
    # Actual S2 gates remain distinct from a capacity diagnosis.
    protocol=read(w/'METRIC_PROTOCOL.json');ability={}
    for seed in SEEDS:
        for cap in [32,64]:
            name=f's{seed}-c{cap}';full=metrics[name+'/full'];gates={}
            for stream in ['normal','shifted']:
                s=full[stream];baseline=metrics['constant'][stream]
                gates[stream]={'relative_RMSE':s['RMSE']<=.8*baseline['RMSE'],'correlation':s['Correlation']>=.5,'calibration':s['CalibrationError']<=protocol['CalibrationLimit']}
                sl=slice(0,512) if stream=='normal' else slice(512,None);rr=rows[sl];pv=np.array(pred[name+'/full'])[sl];gaps={}
                for op in [0,1]:
                    low=[i for i,r in enumerate(rr) if r['Pre']['Action']==op and r['Error']<=protocol['LowCut']];high=[i for i,r in enumerate(rr) if r['Pre']['Action']==op and r['Error']>=protocol['HighCut']]
                    gaps[str(op)]=float(np.mean(pv[high])-np.mean(pv[low])) if low and high else None
                gates[stream]['context_gaps']=gaps;gates[stream]['positive_context_gaps']=all(v is not None and v>0 for v in gaps.values())
            gates['all_readout_ablation']=metrics[name+'/all-readout-zero']['pooled']['RMSE']>=1.2*full['pooled']['RMSE']
            gates['independent_new_representation']=causal if cap==64 else False;gates['retention']=retention
            passed=all(all(v for k,v in gates[s].items() if k!='context_gaps') for s in ['normal','shifted']) and all(gates[x] for x in ['all_readout_ablation','independent_new_representation','retention'])
            ability[name]={'status':'PASS' if passed else 'PARTIAL','gates':gates,'metrics':full}
    write(o/'actual-S2-ability.json',ability)
    # Exact resource comparisons, with inclusive transaction cost labeled.
    resources={}
    for seed in SEEDS:
        for cap in [32,64]:
            arm=w/f'seed-{seed}/arm-{cap}';resources[f's{seed}-c{cap}']={'inference':read(arm/'cost.json'),'learning':read(arm/'learning-cost.json'),'persistence':persistence[f's{seed}-c{cap}']}
    write(o/'resources.json',resources)
    # Verify paired data, causal observation ordering and lack of set overlap.
    observed={};hashes={};ledger_count=0
    sets=[]
    for stream in range(2):
        for role in ['fit','check']:sets.append((f'development-{stream}-{role}',w/f'development-{stream}-{role}.json'))
    for seed in SEEDS:
        for block in range(8):
            for role in ['fit','check']:sets.append((f'seed-{seed}-block-{block}-{role}',w/f'seed-{seed}/block-{block}-{role}.json'))
    sets.append(('final',w/'holdout.json'))
    for name,path in sets:
        for row in read(path):
            ident=row['ID'];h=row['InputSHA256']
            if ident in observed or h in hashes:raise ValueError('experience or encoded-input overlap '+name)
            observed[ident]=(name,row);hashes[h]=ident
    ledger_paths=[(w/'preoperation.jsonl',w/'development-observer/events.jsonl')]
    ledger_paths += [(w/f'seed-{s}/preoperation.jsonl',w/f'seed-{s}/observer/events.jsonl') for s in SEEDS]
    ledger_paths += [(w/'final-observer/preoperation.jsonl',w/'final-observer/events.jsonl')]
    for ledger,journal in ledger_paths:
        targets={}
        for line in journal.open():
            ev=json.loads(line);cmd=ev.get('native_control') or {};ident=cmd.get('id','')
            if ident.endswith('-target'):targets[ident[:-7]]=ev['sequence']
        for line in ledger.open():
            item=json.loads(line);ident=item['id'];row=observed[ident][1]
            if row['InputSHA256']!=item['input_sha256'] or row['Pre']['Forecast']!=item['forecast'] or targets[ident]<=item['before_sequence']:raise ValueError('pre-operation provenance mismatch '+ident)
            ledger_count+=1
    if ledger_count!=len(observed):raise ValueError('incomplete pre-operation ledger')
    for seed in SEEDS:
        for block in range(8):
            a=read(w/f'seed-{seed}/arm-32/receipt-{block}.json')['diagnosis']['evidence'];b=read(w/f'seed-{seed}/arm-64/receipt-{block}.json')['diagnosis']['evidence']
            if a!=b:raise ValueError('arms received different fit/check/witness evidence')
    write(o/'data-provenance.json',{'actual_observations':len(observed),'pre_operation_ledger_entries':ledger_count,'episode_overlap':0,'encoded_input_overlap':0,'paired_evidence_hashes_exact':True,'pre_input_before_target_native_event':True,'prior_R2_holdout_used_for_learning_or_selection':False})
    write(o/'analysis-provenance.json',{'script_sha256':hashlib.sha256(Path(__file__).read_bytes()).hexdigest(),'numpy':np.__version__,'no_predictive_model_fit':True,'descriptive_unlabeled_projection_only':True,'holdout_evaluations':1,'bootstrap_seed':980017})
    print(json.dumps(summary,indent=2))

if __name__=='__main__':main()
