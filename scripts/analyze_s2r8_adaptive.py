#!/usr/bin/env python3
"""Predeclared paired analysis plus explicitly descriptive diagnostics."""
import argparse,csv,hashlib,json,pathlib,subprocess
import numpy as np
import matplotlib
matplotlib.use('Agg')
import matplotlib.pyplot as plt

SEEDS=[37,83,151,211,307,419];ARMS=list('ABCD');CONDS=['normal','shifted','persistent']
def load(p):
    p=pathlib.Path(p)
    return json.loads(p.read_bytes() if p.exists() else subprocess.check_output(['zstd','-q','-d','-c',str(p)+'.zst']))
def weights(brain):
    return {(x['source_id'],x['target_id'],x.get('inhibitory',False),x.get('kind','')):x['dynamic']['weight'] for x in brain['synapses']}
def main():
    p=argparse.ArgumentParser();p.add_argument('--work',type=pathlib.Path,required=True);p.add_argument('--source',type=pathlib.Path,required=True);p.add_argument('--output',type=pathlib.Path,required=True);a=p.parse_args();a.output.mkdir(exist_ok=True,parents=True)
    values={s:load(a.work/f'seed-{s}/FINAL_RESULTS.json') for s in SEEDS}
    errors={arm:np.array([[values[s][arm][c][i]['squared_errors'] for c in CONDS for i in range(4)] for s in SEEDS]).reshape(6,3,4,24).mean(axis=-1) for arm in ARMS}
    budget={arm:np.array([[values[s][arm]['compute_budget'][c][i]['squared_errors'] for c in CONDS for i in range(4)] for s in SEEDS]).reshape(6,3,4,24).mean(axis=-1) for arm in ARMS}
    rng=np.random.default_rng(8182026);si=rng.integers(0,6,(10000,6));sti=rng.integers(0,4,(10000,6,3,4));ci=np.arange(3)[None,None,:,None]
    def draw(x):return x[si[:,:,None,None],ci,sti].mean(axis=(1,2,3))
    def effect(control,candidate,level=.95):
        base=np.sqrt(control.mean());point=(base-np.sqrt(candidate.mean()))/base
        draws=(np.sqrt(draw(control))-np.sqrt(draw(candidate)))/np.sqrt(draw(control))
        lo,hi=np.quantile(draws,[(1-level)/2,1-(1-level)/2])
        return {'relative_RMSE_improvement':float(point),'CI_level':level,'CI_low':float(lo),'CI_high':float(hi)}
    comparisons={f'B_vs_{arm}':effect(errors[arm],errors['B'],.95 if arm=='A' else .975) for arm in 'ACD'}
    stats={'seeds':SEEDS,'conditions':CONDS,'primary':comparisons,'compute_budget':{f'B_vs_{arm}':effect(budget[arm],budget['B']) for arm in 'ACD'},'models':{},'retention':{},'persistence':{},'growth':[],'inference_ablations':{}}
    rows=[];growth=[];resource=[];all_persistence=[];all_retention=[]
    source_features=len(load(a.source/'s2r2/s2r2-continuation.json')['brain']['n_resolution']['features'])
    for arm in ARMS:
        pooled=float(np.sqrt(errors[arm].mean()))
        stats['models'][arm]={'RMSE':pooled,'condition_RMSE':{c:float(np.sqrt(errors[arm][:,i].mean())) for i,c in enumerate(CONDS)},'budget_RMSE':float(np.sqrt(budget[arm].mean()))}
    ablated={name:[] for name in ['R8-readout-off','S1-direct-readout-off','S1-ancestry-readout-off']}
    for seed in SEEDS:
        before=load(a.work/f'seed-{seed}/training-observer/final.json')['brain'];bw=weights(before)
        for arm in ARMS:
            directory=a.work/f'seed-{seed}/{arm}';curves=load(directory/'curves.json');receipts=load(directory/'receipts.json');ret=load(directory/'retention.json');persist=load(directory/'persistence.json');cp=load(directory/'final.json');head=cp['brain']['n_resolution']['last']
            final=values[seed][arm];sse=np.concatenate([np.concatenate([np.array(x['squared_errors']) for x in final[c]]) for c in CONDS])
            pred=np.concatenate([np.concatenate([np.array(x['predictions']) for x in final[c]]) for c in CONDS]);targets=np.concatenate([np.concatenate([np.array(x['targets']) for x in final[c]]) for c in CONDS])
            rmse=float(np.sqrt(sse.mean()));cor=float(np.corrcoef(pred,targets)[0,1]) if np.std(pred)>1e-12 else 0.
            commits=sum(bool(x['diagnosis']['committed']) for x in receipts) if 'diagnosis' in receipts[0] else sum(bool(x['Diagnosis']['committed']) for x in receipts)
            current=weights(cp['brain']);changed=sum(v!=bw.get(k) for k,v in current.items())
            candidate_ret=[r['historical_S1_S2']['pass'] and r['AE1']['pass'] for r in ret]
            all_retention.extend(candidate_ret);needed=['restart_exact','full_suffix_replay_exact','source_prefix_exact','learning_after_restart_exact','continued_full_replay_exact','source_rollback_exact','single_N_owner'];exact=all(persist.get(k) is True for k in needed);all_persistence.append(exact)
            cpu=sum(x.get('learning_cpu_nanoseconds',0) for x in curves);wall=sum(x.get('learning_nanoseconds',0) for x in curves)
            rows.append({'seed':seed,'arm':arm,'RMSE':rmse,'correlation':cor,'mean_signed_error':float(np.mean(pred-targets)),'negative_predictions':int(np.sum(pred<0)),'features':len(cp['brain']['n_resolution']['features']),'neurons':len(cp['brain']['neural_units']),'synapses':len(cp['brain']['synapses']),'changed_weight_parameters':changed,'commits':commits,'retention_pass_blocks':sum(candidate_ret),'persistence_exact':exact,'learner_CPU_seconds':cpu/1e9,'learner_wall_seconds':wall/1e9,'budget_blocks':final['compute_budget']['block'],'budget_used_CPU_seconds':final['compute_budget']['used_cpu_nanoseconds']/1e9,'normal_RMSE':float(np.sqrt(errors[arm][SEEDS.index(seed),0].mean())),'shifted_RMSE':float(np.sqrt(errors[arm][SEEDS.index(seed),1].mean())),'persistent_RMSE':float(np.sqrt(errors[arm][SEEDS.index(seed),2].mean()))})
            for block,c in enumerate(curves):
                inv=c['inventory'];entry={'seed':seed,'arm':arm,'block':block,'exposures':c['exposures'],'features':inv['features'],'neurons':inv['neurons'],'synapses':inv['synapses'],'sequence':inv['sequence']}
                if block:
                    snap=load(directory/f'block-{block}.json');nw=weights(snap['brain']);entry['changed_weight_parameters']=sum(v!=bw.get(k) for k,v in nw.items())
                else:entry['changed_weight_parameters']=0
                growth.append(entry)
            resource.append({'seed':seed,'arm':arm,'learner_CPU_seconds':cpu/1e9,'learner_wall_seconds':wall/1e9,'inference_stream_means_ns':final['inference_stream_mean_nanoseconds'],'replay_seconds':persist['replay_nanoseconds']/1e9,'final_checkpoint_compressed_bytes':(directory/'final.json.zst').stat().st_size,'continued_learning_committed':load(directory/'continued.json')['brain']['n_resolution']['last']['committed'],'allocations_bytes_by_lesson':[x['memory_after']['total_alloc']-x['memory_before']['total_alloc'] for x in curves[1:]]})
            stats['retention'][f'{seed}-{arm}']=ret;stats['persistence'][f'{seed}-{arm}']=persist
            if arm=='B':
                trace=load(directory/'final-neural-trace.json');model=trace['model'];h=model['Heads'][0];v=np.array(trace['metric'] if h['Metric'] else trace['activation']);w=np.array(h['W']);b=h['B'];native=np.array(trace['predictions']['full'])
                py=np.clip(b+np.array([sum(float(u)*float(z) for u,z in zip(w,row)) for row in v]),-1,1)
                if np.max(np.abs(py-native))>1e-13:raise RuntimeError('readout reconstruction mismatch')
                for name in ablated:
                    altered=w.copy()
                    if name=='R8-readout-off':altered[source_features:]=0
                    elif name=='S1-direct-readout-off':altered[:26]=0
                    else:
                        inherited=set(x['Meta']['node'] for x in model['Circuits'][:26])
                        for j,c in enumerate(model['Circuits']):
                            if c['Meta']['node'] in inherited or any(x in inherited for x in c['Meta'].get('parents',[])):
                                inherited.add(c['Meta']['node']);altered[j]=0
                    yp=np.clip(b+v@altered,-1,1);ablated[name].append(((yp-targets)**2).reshape(3,4,24).mean(axis=-1))
    for name,arr in ablated.items():stats['inference_ablations'][name]=effect(np.array(arr),errors['B'])
    stats['growth']=growth;stats['per_seed']=rows;stats['resources']=resource
    pairs=load(a.work/'distinctness/CASES.json');distinct={}
    for name in ['source']+[f'B-{s}' for s in SEEDS]:
        trace=load(a.work/f'distinctness/{name}-trace.json');selected=trace['metric'] if trace['model']['Heads'][0]['Metric'] else trace['activation'];details={}
        for i,pair in enumerate(pairs):
            kind=pair['Kind'];d=details.setdefault(kind,{'pairs':0,'raw_exact_collisions':0,'metric_exact_collisions':0,'activation_exact_collisions':0,'selected_exact_collisions':0,'selected_near_collisions_1e12':0,'prediction_near_collisions_1e12':0,'target_different':0,'absolute_prediction_errors':[]})
            d['pairs']+=1;d['target_different']+=pair['TargetDifferent']
            for key,field in [('raw_exact_collisions','raw'),('metric_exact_collisions','metric'),('activation_exact_collisions','activation')]:d[key]+=int(np.array_equal(trace[field][2*i],trace[field][2*i+1]))
            va,vb=np.array(selected[2*i]),np.array(selected[2*i+1]);d['selected_exact_collisions']+=int(np.array_equal(va,vb));d['selected_near_collisions_1e12']+=int(np.max(np.abs(va-vb))<=1e-12)
            pa,pb=trace['predictions']['full'][2*i:2*i+2];d['prediction_near_collisions_1e12']+=int(abs(pa-pb)<=1e-12);d['absolute_prediction_errors'].extend([abs(pa-pair['A']['Error']),abs(pb-pair['B']['Error'])])
        for d in details.values():d['MAE']=float(np.mean(d.pop('absolute_prediction_errors')))
        distinct[name]=details
    stats['distinctness']=distinct
    stats['retention_pass_blocks']=sum(all_retention);stats['retention_total_blocks']=len(all_retention);stats['exact_persistence_models']=sum(all_persistence);stats['total_models']=len(all_persistence)
    stats['Gate_D']={'point_improvement_at_least_5pct':comparisons['B_vs_A']['relative_RMSE_improvement']>=.05,'CI95_lower_above_zero':comparisons['B_vs_A']['CI_low']>0,'all_conditions_noninferior_to_A_5pct':all(stats['models']['B']['condition_RMSE'][c]<=stats['models']['A']['condition_RMSE'][c]*1.05 for c in CONDS)}
    for name,data in [('PER_SEED.csv',rows),('GROWTH.csv',growth)]:
        with (a.output/name).open('w') as f:writer=csv.DictWriter(f,fieldnames=list(data[0]));writer.writeheader();writer.writerows(data)
    (a.output/'SCIENTIFIC_RESULTS.json').write_text(json.dumps(stats,indent=2,allow_nan=False))
    plt.rcParams.update({'font.size':10,'axes.spines.top':False,'axes.spines.right':False,'figure.facecolor':'#f7f9fc','axes.facecolor':'#ffffff'})
    colors={'A':'#67758c','B':'#007d78','C':'#b77900','D':'#7b55a3'};labels={'A':'A · Fixed 32','B':'B · Reuse + Growth','C':'C · Selective exclusion','D':'D · Permuted growth'}
    fig,axes=plt.subplots(2,2,figsize=(13,8),layout='constrained');x=np.arange(5)*64
    for arm in ARMS:
        ys=np.array([[c['monitor']['RMSE'] for c in load(a.work/f'seed-{s}/{arm}/curves.json')] for s in SEEDS]);fs=np.array([[c['inventory']['features'] for c in load(a.work/f'seed-{s}/{arm}/curves.json')] for s in SEEDS])
        axes[0,0].plot(x,ys.mean(axis=0),label=labels[arm],color=colors[arm],marker='o');axes[0,0].fill_between(x,ys.min(axis=0),ys.max(axis=0),color=colors[arm],alpha=.09)
        axes[0,1].plot(x,fs.mean(axis=0),label=labels[arm],color=colors[arm],marker='o');axes[0,1].fill_between(x,fs.min(axis=0),fs.max(axis=0),color=colors[arm],alpha=.09)
    axes[0,0].set(title='Learning curve · monitor terpisah',xlabel='Pengalaman Fit + Check baru',ylabel='RMSE · lebih rendah lebih baik');axes[0,0].legend(fontsize=8)
    axes[0,1].set(title='Pertumbuhan representasi aktual',xlabel='Pengalaman Fit + Check baru',ylabel='Jumlah fitur numerik')
    positions=np.arange(3);width=.18
    for i,arm in enumerate(ARMS):axes[1,0].bar(positions+(i-1.5)*width,[stats['models'][arm]['condition_RMSE'][c] for c in CONDS],width,label=labels[arm],color=colors[arm])
    axes[1,0].set_xticks(positions,CONDS);axes[1,0].set(title='Holdout final · 1.728 kasus unik',ylabel='Pooled RMSE')
    for i,arm in enumerate(ARMS):
        axes[1,1].scatter([r['learner_CPU_seconds'] for r in rows if r['arm']==arm],[r['RMSE'] for r in rows if r['arm']==arm],color=colors[arm],label=labels[arm],s=45,alpha=.85)
    axes[1,1].set(title='Biaya belajar dan generalisasi · setiap seed',xlabel='CPU learner untuk 4 kelompok (detik)',ylabel='RMSE final')
    fig.suptitle('HORIZON · S2-R8 | Representasi adaptif · PARTIAL',fontsize=17,fontweight='bold');fig.savefig(a.output/'S2R8_Learning_Curves.png',dpi=170);plt.close(fig)
    print(json.dumps({'models':stats['models'],'primary':stats['primary'],'Gate_D':stats['Gate_D'],'retention':f'{sum(all_retention)}/{len(all_retention)}','persistence':f'{sum(all_persistence)}/{len(all_persistence)}'},indent=2))
if __name__=='__main__':main()
