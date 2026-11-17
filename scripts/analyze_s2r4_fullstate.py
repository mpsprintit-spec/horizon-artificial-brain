#!/usr/bin/env python3
"""S2-R4 v4: complete-state information diagnosis, never runtime inference."""
import json
from pathlib import Path
import numpy as np
import analyze_s2r4_temporal as core

original_arrays=core.arrays
def arrays(rows):
    reduced,h,y,g=original_arrays(rows)
    full=np.array([[r['Trial']['Pre']['Before'].get(str(j),0)/2 for j in range(1,37)] for r in rows])
    grouped=full.reshape(-1,9,4);means=grouped.mean(2)
    # An invertible complete-state representation (means plus within-group
    # contrasts), not conditioning only on the means. Fit-only scaling makes
    # small spatial differences accessible to the diagnostic comparators.
    current=np.c_[means,(grouped-means[:,:,None]).reshape(-1,36),reduced[:,9:]]
    return current,h,y,g

def probe(xs,ys):
    f,c,t=xs;yf,yc,yt=ys;mean=f.mean(0);sd=f.std(0);keep=sd>1e-10;sd=np.where(sd>1e-10,sd,1.)
    f,c,t=[((x-mean)/sd)[:,keep] for x in [f,c,t]];candidates=[]
    for degree in [1,2]:
        ff,fc,ft=[core.features(x,degree)[:,1:] for x in [f,c,t]]
        center=ff.mean(0);ff,fc,ft=[x-center for x in [ff,fc,ft]]
        # Dual ridge is the same bounded-size offline comparator in sample
        # space; it avoids a quadratic feature-space solve for full state.
        gram=ff@ff.T;check=fc@ff.T;test=ft@ff.T
        for ridge in [1e-5,.01,1.,100.]:
            alpha=np.linalg.solve(gram+ridge*np.eye(len(ff)),yf-yf.mean())
            candidates.append((core.rmse(yc,check@alpha+yf.mean()),{'family':'full-state-polynomial-ridge','degree':degree,'ridge':ridge},test@alpha+yf.mean()))
    if f.shape[1]:
        dist=lambda a,b:np.maximum(0,(a*a).sum(1)[:,None]+(b*b).sum(1)[None,:]-2*a@b.T)
        for bandwidth in [1.,3.,10.]:
            ff=np.exp(-dist(f,f)/(bandwidth*f.shape[1]));fc=np.exp(-dist(c,f)/(bandwidth*f.shape[1]));ft=np.exp(-dist(t,f)/(bandwidth*f.shape[1]))
            for ridge in [.001,.1,10.]:
                alpha=np.linalg.solve(ff+ridge*np.eye(len(f)),yf-yf.mean())
                candidates.append((core.rmse(yc,fc@alpha+yf.mean()),{'family':'full-state-rbf-ridge','bandwidth':bandwidth,'ridge':ridge},ft@alpha+yf.mean()))
    selected=min(candidates,key=lambda x:x[0])
    return selected[2],{'selection':'independent development Check only','full_neural_state_conditioned':True,'check_rmse':selected[0],'configuration':selected[1],'candidates':[{**c,'CheckRMSE':v} for v,c,_ in candidates]}

def interventions(work):
    dev=json.loads((work/'development/persistent/rows.json').read_text());fit=core.split(dev)[0];x,h,y,g=arrays(fit);continuous=x[:,:-4];mean=continuous.mean(0);sd=continuous.std(0);sd=np.where(sd>1e-10,sd,1.)
    settings={'mean':mean.tolist(),'sd':sd.tolist(),'scaling_source':'complete 36-neuron state via means+contrasts, previous state, forecast; development Fit only','conditioned_continuous_dimensions':len(mean),'neighbors':8,'replicates':10,'seeds':list(range(3100031,3100041)),'low_cut':float(np.quantile(y,1/3)),'high_cut':float(np.quantile(y,2/3)),'calibration_bins':[.01,.03,.06,.10,.20,.30]}
    core.write(work/'METRIC_LOCK.json',settings)
    if not (work/'FINAL_STARTED').exists():return
    for condition in ['persistent','independent','shifted']:
        d=work/'holdout'/condition;rows=json.loads((d/'rows.json').read_text());x,h,y,g=arrays(rows);z=(x[:,:-4]-mean)/sd;distance=np.maximum(0,(z*z).sum(1)[:,None]+(z*z).sum(1)[None,:]-2*z@z.T)
        contexts=np.array([[r['Trial']['Pre']['Action'],r['Trial']['Pre']['PreviousAction'],r['Trial']['Pre']['HistoryAction'],len(r['Trial']['Pre']['History'] or [])] for r in rows]);choices=[];distances=[]
        for i in range(len(rows)):
            possible=np.flatnonzero(np.all(contexts==contexts[i],axis=1)&(g!=g[i]))
            if contexts[i,3]!=3 or not len(possible):choices.append(np.array([i]));continue
            nearest=possible[np.argsort(distance[i,possible],kind='stable')[:8]];choices.append(nearest);distances.extend(np.sqrt(distance[i,nearest]).tolist())
        donors={}
        for j,seed in enumerate(settings['seeds']):
            rng=np.random.default_rng(seed);donors[f'permute-{j:02d}']=[int(rng.choice(v)) for v in choices]
        core.write(d/'input-interventions.json',{'Donors':donors,'target_blind':True,'complete_state_conditioned':True,'continuous_dimensions':len(mean),'mean_neighbor_distance':float(np.mean(distances)),'distance_q50_q95':np.quantile(distances,[.5,.95]).tolist(),'approximate_nearest_neighbor_permutation':True,'no_label_used':True})

core.arrays=arrays;core.probe=probe;core.interventions=interventions
if __name__=='__main__':core.main()
