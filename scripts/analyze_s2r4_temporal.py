#!/usr/bin/env python3
"""Offline information/representation audit only; never a Horizon predictor."""
import argparse, hashlib, json, collections
from pathlib import Path
import numpy as np

def write(p,v):
    p.parent.mkdir(parents=True,exist_ok=True)
    p.write_text(json.dumps(v,indent=2,sort_keys=True,allow_nan=False)+'\n')
def arrays(rows):
    cur=[]; hist=[]; target=[]; groups=[]
    for r in rows:
        t=r.get('Trial',r);p=t['Pre'];s=p['Before']
        means=[np.mean([s.get(str(j),0) for j in range(i,i+4)])/2 for i in range(1,37,4)]
        previous=p['Previous'] or [0.]*9
        cur.append(means+previous+p['Forecast']+[float(p['Action']),float(p['PreviousAction']),float(p['HistoryAction']),float(len(p['History'] or [])==3)])
        hist.append(p['History'] or [0.]*3);target.append(t['Error']);groups.append(r.get('Stream',t['ID']))
    return np.array(cur),np.array(hist),np.array(target),np.array(groups)
def split(rows):
    groups=list(dict.fromkeys(r['Stream'] for r in rows));n=len(groups)//3
    return [[r for r in rows if r['Stream'] in set(groups[a:b]) and r['Step']>=2] for a,b in [(0,n),(n,2*n),(2*n,len(groups))]]
def rmse(y,p):return float(np.sqrt(np.mean((y-p)**2)))
def paired(y,a,b,g,seed=440037):
    # Positive effect means b (with information / intervention) improves a.
    groups=list(dict.fromkeys(g.tolist())); vals=[];rng=np.random.default_rng(seed)
    sa=np.array([np.sum((y[g==x]-a[g==x])**2) for x in groups]);sb=np.array([np.sum((y[g==x]-b[g==x])**2) for x in groups]);nn=np.array([np.sum(g==x) for x in groups])
    for _ in range(2000):
        idx=rng.integers(0,len(groups),len(groups));va=sa[idx].sum()/nn[idx].sum();vb=sb[idx].sum()/nn[idx].sum()
        vals.append(1-np.sqrt(vb/va) if va>0 else 0.)
    effect=1-rmse(y,b)/rmse(y,a) if rmse(y,a)>0 else 0.
    return {'relative_RMSE_improvement':effect,'CI95':np.quantile(vals,[.025,.975]).tolist(),'clusters':len(groups),'replicates':2000,'resampling_unit':'independent stream'}
def features(x,degree):
    if degree==1:return np.c_[np.ones(len(x)),x]
    ij=np.triu_indices(x.shape[1]);return np.c_[np.ones(len(x)),x,(x[:,:,None]*x[:,None,:])[:,ij[0],ij[1]]]
def probe(xs,ys):
    f,c,t=xs;yf,yc,yt=ys
    mean=f.mean(0);sd=f.std(0);keep=sd>1e-10;sd=np.where(sd>1e-10,sd,1.)
    f,c,t=[((x-mean)/sd)[:,keep] for x in [f,c,t]]
    candidates=[]
    for degree in [1,2]:
        ff,fc,ft=[features(x,degree) for x in [f,c,t]]
        for ridge in [1e-5,.01,1.,100.]:
            a=ff.T@ff+ridge*np.eye(ff.shape[1]);a[0,0]-=ridge
            w=np.linalg.solve(a,ff.T@yf)
            candidates.append((rmse(yc,fc@w),{'family':'polynomial-ridge','degree':degree,'ridge':ridge},ft@w))
    # Flexible independent check-selected comparator; data are standardized
    # from Fit only. No coefficient is exported to neural parameters.
    if f.shape[1]:
        dist=lambda a,b:np.maximum(0,(a*a).sum(1)[:,None]+(b*b).sum(1)[None,:]-2*a@b.T)
        for bandwidth in [1.,3.,10.]:
            ff=np.exp(-dist(f,f)/(bandwidth*max(1,f.shape[1])))
            fc=np.exp(-dist(c,f)/(bandwidth*max(1,f.shape[1])))
            ft=np.exp(-dist(t,f)/(bandwidth*max(1,f.shape[1])))
            for ridge in [.001,.1,10.]:
                w=np.linalg.solve(ff+ridge*np.eye(len(f)),yf-yf.mean())
                candidates.append((rmse(yc,fc@w+yf.mean()),{'family':'rbf-ridge','bandwidth':bandwidth,'ridge':ridge},ft@w+yf.mean()))
    selected=min(candidates,key=lambda a:a[0])
    return selected[2],{'selection':'minimum independent development Check RMSE','check_rmse':selected[0],'configuration':selected[1],'candidates':[{**c,'CheckRMSE':v} for v,c,_ in candidates]}
def information(parts,representation=None):
    a=[arrays(r) for r in parts];ys=[x[2] for x in a]
    base=[x[0] for x in a] if representation is None else representation
    with_history=[np.c_[x[0],x[1]] for x in a] if representation is None else [np.c_[v,x[1]] for v,x in zip(base,a)]
    pb,cb=probe(base,ys);ph,ch=probe(with_history,ys)
    y=ys[2];g=a[2][3]
    return {'n':[len(r) for r in parts],'current_only':{'rmse':rmse(y,pb),'selection':cb},'current_plus_history':{'rmse':rmse(y,ph),'selection':ch},'paired':paired(y,pb,ph,g),'predictions':{'current':pb.tolist(),'history':ph.tolist()},'targets':y.tolist(),'groups':g.tolist()}
def development(work,out):
    result={}
    for condition in ['independent','persistent']:
        d=work/'development'/condition;rows=json.loads((d/'rows.json').read_text());parts=split(rows)
        info=information(parts)
        full=json.loads((d/'representation.json').read_text());zero=json.loads((d/'representation-no-history.json').read_text())
        rep={};index={r['Trial']['ID']:i for i,r in enumerate(rows)}
        for basis in ['Activation','Metric','Code','Contributions']:
            f,z=np.array(full[basis]),np.array(zero[basis]);delta=np.abs(f-z)
            rep[basis]={'features_changed':np.flatnonzero(delta.max(0)>1e-12).tolist(),'maximum_history_intervention_effect':float(delta.max()),'variance':np.var(f,axis=0).tolist(),'saturated_fraction':np.mean(np.abs(f)>=.999999,axis=0).tolist() if basis=='Activation' else None}
            if basis in ['Activation','Metric']:
                v=[np.array([f[index[r['Trial']['ID']]] for r in p]) for p in parts]
                rep[basis]['accessibility']=information(parts,v)
        checks=[]
        for i,r in enumerate(rows):
            t=r['Trial'];err=np.mean(np.abs(np.array(t['Actual'])-t['Pre']['Forecast']))
            if abs(err-t['Error'])>1e-14 or r['OperationSequence']!=r['BeforeSequence']+1:checks.append(t['ID'])
            if r['Step']>0:
                prev=rows[i-1]
                if prev['Stream']!=r['Stream'] or r['PreviousID']!=prev['Trial']['ID'] or not np.allclose(t['Pre']['History'],np.abs(prev['Trial']['Residual']),rtol=0,atol=0):checks.append(t['ID'])
        result[condition]={'information':info,'representation':rep,'target_and_history_provenance_errors':checks,'observed_rows':len(rows)}
    write(out/'development-information.json',result)
    print(json.dumps({c:{'current_RMSE':v['information']['current_only']['rmse'],'history_RMSE':v['information']['current_plus_history']['rmse'],'paired':v['information']['paired'],'provenance_errors':len(v['target_and_history_provenance_errors'])} for c,v in result.items()},indent=2))

ARMS=['A32','B32','C32','B0-32','D32','D0-32','A64','C64']
SEEDS=[37,83,151]
def interventions(work):
    dev=json.loads((work/'development/persistent/rows.json').read_text());fit=split(dev)[0];x,h,y,g=arrays(fit);mean=x[:,:21].mean(0);sd=x[:,:21].std(0);sd=np.where(sd>1e-10,sd,1.)
    settings={'mean':mean.tolist(),'sd':sd.tolist(),'scaling_source':'development persistent Fit streams 0..7, first two steps excluded','neighbors':8,'replicates':10,'seeds':list(range(3100031,3100041)),'low_cut':float(np.quantile(y,1/3)),'high_cut':float(np.quantile(y,2/3)),'calibration_bins':[.01,.03,.06,.10,.20,.30]}
    write(work/'METRIC_LOCK.json',settings)
    if not (work/'FINAL_STARTED').exists():return
    for condition in ['persistent','independent','shifted']:
        d=work/'holdout'/condition;rows=json.loads((d/'rows.json').read_text());x,h,y,g=arrays(rows);z=(x[:,:21]-mean)/sd
        distance=np.maximum(0,(z*z).sum(1)[:,None]+(z*z).sum(1)[None,:]-2*z@z.T)
        contexts=np.array([[r['Trial']['Pre']['Action'],r['Trial']['Pre']['PreviousAction'],r['Trial']['Pre']['HistoryAction'],len(r['Trial']['Pre']['History'] or [])] for r in rows])
        choices=[];distances=[]
        for i in range(len(rows)):
            possible=np.flatnonzero(np.all(contexts==contexts[i],axis=1)&(g!=g[i]))
            if contexts[i,3]!=3 or not len(possible):choices.append(np.array([i]));continue
            nearest=possible[np.argsort(distance[i,possible],kind='stable')[:8]];choices.append(nearest);distances.extend(np.sqrt(distance[i,nearest]).tolist())
        donors={}
        for j,seed in enumerate(settings['seeds']):
            rng=np.random.default_rng(seed);donors[f'permute-{j:02d}']=[int(rng.choice(v)) for v in choices]
        write(d/'input-interventions.json',{'Donors':donors,'target_blind':True,'exact_context':['operation','previous_operation','history_operation','presence'],'approximate_match':'21 standardized current-state/previous-state/forecast coordinates, donor from another independent stream','mean_neighbor_distance':float(np.mean(distances)),'distance_q50_q95':np.quantile(distances,[.5,.95]).tolist(),'no_label_used':True})
    print('saved target-blind conditional input interventions')
def metrics(y,p,lock):
    d=p-y;bins=np.searchsorted(lock['calibration_bins'],p);cal=0.;b=[]
    for i in range(len(lock['calibration_bins'])+1):
        m=bins==i
        if m.any():
            pm,ym=float(p[m].mean()),float(y[m].mean());cal+=m.sum()*abs(pm-ym)/len(y);b.append({'bin':i,'n':int(m.sum()),'predicted':pm,'actual':ym})
    lo=y<=lock['low_cut'];hi=y>=lock['high_cut'];auc=None
    if lo.any() and hi.any():
        pair=p[hi,None]-p[None,lo];auc=float(np.mean((pair>0)+.5*(pair==0)))
    return {'N':len(y),'RMSE':rmse(y,p),'MAE':float(np.mean(abs(d))),'Correlation':float(np.corrcoef(y,p)[0,1]) if p.std()>0 and y.std()>0 else 0.,'CalibrationError':cal,'ErrorDiscriminationAUROC':auc,'CalibrationBins':b}
def hierarchy(a,b,groups,seed=4100037):
    # a,b are paired squared losses: training-seed x final-case.
    gs=list(dict.fromkeys(groups.tolist()));suma=np.array([[v[groups==g].sum() for g in gs] for v in a]);sumb=np.array([[v[groups==g].sum() for g in gs] for v in b]);n=np.array([sum(groups==g) for g in gs]);rng=np.random.default_rng(seed);effects=[]
    for _ in range(2000):
        ss=rng.integers(0,len(a),len(a));gg=rng.integers(0,len(gs),len(gs));va=suma[ss][:,gg].sum()/(len(ss)*n[gg].sum());vb=sumb[ss][:,gg].sum()/(len(ss)*n[gg].sum());effects.append(1-np.sqrt(vb/va) if va>0 else 0.)
    return {'relative_RMSE_improvement':float(1-np.sqrt(np.mean(b)/np.mean(a))),'CI95':np.quantile(effects,[.025,.975]).tolist(),'seed_effects':[float(1-np.sqrt(np.mean(bb)/np.mean(aa))) for aa,bb in zip(a,b)],'bootstrap_replicates':2000,'training_seeds':len(a),'streams':len(gs),'method':'paired hierarchical training-seed and shared independent-stream bootstrap'}
def final(work,out):
    lock=json.loads((work/'METRIC_LOCK.json').read_text());summary={};allpred={};allrows={}
    for condition in ['persistent','independent','shifted']:
        rows=json.loads((work/'holdout'/condition/'rows.json').read_text());allrows[condition]=rows;_,_,y,g=arrays(rows);pred={};arms={}
        for arm in ARMS:
            values=[]
            for seed in SEEDS:
                d=work/f'training/seed-{seed}'/arm;p=json.loads((d/f'predictions-{condition}.json').read_text());values.append({k:np.array(v) for k,v in p.items()})
            pred[arm]=values
            perseed=[metrics(y,p['full'],lock) for p in values];full=np.array([p['full'] for p in values]);base=(full-y)**2
            no=(np.array([p['no-history'] for p in values])-y)**2
            perm=np.array([np.mean([(p[f'permute-{j:02d}']-y)**2 for j in range(10)],axis=0) for p in values])
            streams={s:[metrics(y[g==s],p['full'][g==s],lock) for p in values] for s in dict.fromkeys(g)}
            available=np.array([r['Step']>=1 for r in rows])
            arms[arm]={'pooled':metrics(np.tile(y,3),full.ravel(),lock),'seeds':dict(zip(map(str,SEEDS),perseed)),'history_ablation':hierarchy(no,base,g),'conditional_permutation':hierarchy(perm,base,g),'history_available_cases':int(available.sum()),'history_available_ablation':hierarchy(no[:,available],base[:,available],g[available]),'history_available_permutation':hierarchy(perm[:,available],base[:,available],g[available]),'per_stream':streams,'all_ten_permutation_effects':[hierarchy(np.array([(p[f'permute-{j:02d}']-y)**2 for p in values]),base,g)['relative_RMSE_improvement'] for j in range(10)]}
        contrasts={}
        for reference,candidate in [('C32','A32'),('B0-32','B32'),('A32','B32'),('D0-32','D32'),('C64','A64'),('A32','A64')]:
            a=(np.array([p['full'] for p in pred[reference]])-y)**2;b=(np.array([p['full'] for p in pred[candidate]])-y)**2
            contrasts[candidate+'-vs-'+reference]=hierarchy(a,b,g)
        summary[condition]={'arms':arms,'contrasts':contrasts};allpred[condition]=pred
    write(out/'confirmatory-metrics.json',summary)
    admission={};retentions={};resources={};synapses={}
    for seed in SEEDS:
        for arm in ARMS:
            name=f's{seed}-{arm}';d=work/f'training/seed-{seed}'/arm;receipts=json.loads((d/'receipts.json').read_text());kinds=collections.Counter();eligible=collections.Counter();rejections=collections.Counter();fits=collections.Counter();history_trials=0;selected=[]
            for block,r in enumerate(receipts):
                diag=r['diagnosis'];selected.append({'block':block,'committed':diag['committed'],'cause':diag['cause'],'grown':diag.get('grown',0),'before_error':diag['before_error'],'after_error':diag['after_error']})
                for t in diag.get('trials',[]):
                    kinds[t['kind']]+=1;eligible[t['kind']]+=int(t['eligible']);rejections.update(t.get('rejected_by',[]));c=t.get('candidate') or {};f=c.get('FitDiagnostic') or {};fits[f.get('Failure') or 'valid']+=1;axis=c.get('Axis') or []
                    if len(axis)>61 and any(abs(axis[j])>1e-14 for j in [59,60,61]):history_trials+=1
            admission[name]={'lessons':len(receipts),'commits':sum(r['diagnosis']['committed'] for r in receipts),'kinds':dict(kinds),'eligible':dict(eligible),'rejected_by':dict(rejections),'fit_failures':dict(fits),'trials_with_direct_history_axis':history_trials,'selection':selected}
            retentions[name]=json.loads((d/'final-retention.json').read_text());resources[name]=json.loads((d/'resources.json').read_text())
            # Canonical complete synapse maps live in checkpoints; save an
            # explicit signed parameter change map for review/reproduction.
            before=json.loads((work/'00-source.json').read_text())['brain'];after=json.loads((d/'final.json').read_text())['brain']
            key=lambda s:(s['source_id'],s['target_id'],s['inhibitory'])
            bm=collections.defaultdict(list);am=collections.defaultdict(list)
            for s in before['synapses']:bm[key(s)].append(s)
            for s in after['synapses']:am[key(s)].append(s)
            changed=[{'source':k[0],'target':k[1],'inhibitory':k[2],'before':bm.get(k),'after':am.get(k)} for k in sorted(set(bm)|set(am)) if bm.get(k)!=am.get(k)]
            write(d/'synaptic-changes.json',changed);synapses[name]={'changed_or_new_channel_groups':len(changed),'neurons_before':len(before['neural_units']),'neurons_after':len(after['neural_units']),'features_after':len(after['n_resolution']['features'])}
    write(out/'admission.json',admission);write(out/'retention.json',retentions);write(out/'resources.json',resources);write(out/'structure.json',synapses)
    for c in summary:
        print(c,json.dumps({a:round(v['pooled']['RMSE'],8) for a,v in summary[c]['arms'].items()}));print('primary',summary[c]['contrasts']['B32-vs-B0-32'])

def main():
    p=argparse.ArgumentParser();p.add_argument('--work',type=Path,required=True);p.add_argument('--output',type=Path,required=True);p.add_argument('--phase',choices=['development','interventions','final'],default='development');a=p.parse_args()
    if a.phase=='development':development(a.work,a.output)
    elif a.phase=='interventions':interventions(a.work)
    else:final(a.work,a.output)
if __name__=='__main__':main()
