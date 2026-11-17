#!/usr/bin/env python3
"""Audit persisted S2-R2 evidence without fitting, selecting or changing a model."""
import argparse
import hashlib
import json
from pathlib import Path

ROOT=Path(__file__).resolve().parents[1]
def read(p):return json.loads(p.read_text())
def write(p,v):p.write_text(json.dumps(v,indent=2,sort_keys=True)+'\n')
def sha(p):return hashlib.sha256(p.read_bytes()).hexdigest()
def main():
    p=argparse.ArgumentParser(description=__doc__);p.add_argument('--work',type=Path,required=True);p.add_argument('--failed',type=Path,required=True);p.add_argument('--output',type=Path,required=True);a=p.parse_args();w=a.work;o=a.output;o.mkdir(exist_ok=True,parents=True)
    result=read(w/'results.json');metrics=read(w/'final-metrics.json');pred=read(w/'predictions.json');progress=read(w/'progress.json');learning=read(w/'learning.json')
    source=read(w/'00-source.json');final=read(w/'s2r2-final.json');continued=read(w/'s2r2-continuation.json');s,b=source['brain'],final['brain'];c=continued['brain']
    assert s['n_resolution']['witnesses']==b['n_resolution']['witnesses']==c['n_resolution']['witnesses'],'historical witnesses changed'
    assert (w/'training-combined/events.jsonl').read_bytes().startswith((w/'source-events.jsonl').read_bytes()),'historical journal changed'
    admission={}
    csv=['arm,block,committed,kind,proposal,fraction,before_mse,after_mse,trial_count']
    for arm,receipts in learning.items():
        out=[]
        for i,receipt in enumerate(receipts):
            d=receipt['diagnosis'];trials=d.get('trials') or [];eligible=[t for t in trials if t['eligible']];winner=eligible[-1] if eligible else {}
            if d['committed']:assert winner['retention_regressions']==0
            out.append({'block':i,'committed':d['committed'],'cause':d['cause'],'winner':winner,'before_mse':d['before_error'],'after_mse':d['after_error'],'evidence':d.get('evidence'),'trials':len(trials)})
            csv.append(','.join(map(str,[arm,i,d['committed'],winner.get('kind',''),winner.get('proposal',''),winner.get('fraction',''),d['before_error'],d['after_error'],len(trials)])))
        admission[arm]=out
    write(o/'admission-summary.json',admission);(o/'admission-summary.csv').write_text('\n'.join(csv)+'\n')
    graph=read(w/'learned-graph.json');head=graph['Heads'][0];edges={(e['source_id'],e['target_id'],e['inhibitory']):e['dynamic']['weight'] for e in b['synapses']}
    def parameter(sources,targets):
        return sum(8*(edges.get((x,y,False),0)-edges.get((x,y,True),0)) for x in sources for y in targets)/(len(sources)*len(targets))
    channels=[]
    for j,name in [(56,'forecast-0'),(57,'forecast-1'),(58,'forecast-2'),(59,'history-error-0'),(60,'history-error-1'),(61,'history-error-2'),(63,'history-presence'),(64,'history-action')]:
        sid=b['n_resolution']['sources'][j];paths=[]
        for i,f in enumerate(graph['Circuits']):
            if f['Meta']['kind']!=1 or f['Axis'][j]==0:continue
            actual=parameter([sid],[f['Meta']['node']]);assert abs(actual-f['Axis'][j])<=1e-12
            readout=parameter([f['Meta']['node']],head['Query']);assert abs(readout-head['W'][i])<=1e-12
            paths.append({'feature':f['Meta']['node'],'axis_weight':actual,'query_readout_weight':readout})
        channels.append({'coordinate':j,'channel':name,'source_node':sid,'signed_synaptic_paths':paths})
    write(o/'signed-channel-attribution.json',channels)
    candidate=metrics['combined']['pooled']['RMSE'];attribution={}
    for name in ['no-forecast','no-history-magnitude','no-history-action','no-history-presence','no-history-magnitude-and-action','permute-history-conditional','permute-action-conditional','permute-forecast','permute-extras','new-feature-clamp','new-readout-clamp','all-readout-clamp']:
        attribution[name]={'RMSE':metrics[name]['pooled']['RMSE'],'RMSE_delta':metrics[name]['pooled']['RMSE']-candidate,'maximum_prediction_delta':max(abs(x-y) for x,y in zip(pred['combined'],pred[name])),'changed_predictions':sum(x!=y for x,y in zip(pred['combined'],pred[name]))}
    write(o/'forecast-history-attribution.json',attribution)
    inventory={}
    for name,snap in [('source',source),('final',final),('continuation',continued)]:
        brain=snap['brain'];inventory[name]={'sequence':snap['sequence'],'native_checksum':snap['checksum'],'neurons':len(brain['neural_units']),'synapses':len(brain['synapses']),'PatternSynapses':len(brain['temporal_patterns']),'features':len(brain['n_resolution']['features']),'native_policy_updates':brain['native_control']['updates'] if 'updates' in brain['native_control'] else None}
    write(o/'inventory.json',inventory)
    summary={'repair_status':result['repair_status'],'ability_status':result['ability_status'],'gates':result['gates'],'source':inventory['source'],'final':inventory['final'],'continuation':inventory['continuation'],'commits':{arm:sum(x['committed'] for x in rows) for arm,rows in admission.items()},'historical_witnesses_preserved_exact':True,'historical_WAL_prefix_exact':True,'candidate_RMSE':candidate,'constant_RMSE':metrics['constant']['pooled']['RMSE'],'constant_improvement_percent':100*(1-candidate/metrics['constant']['pooled']['RMSE']),'original_S2_RMSE':metrics['original-S2']['pooled']['RMSE'],'correlation':metrics['combined']['pooled']['Correlation'],'calibration':metrics['combined']['pooled']['CalibrationError'],'history_independent_benefit':result['history_magnitude_independently_beneficial'],'initial_failed_revision':read(a.failed/'FAILED_REVISION_STATUS.json'),'final_holdout_evaluations':result['final_holdout_evaluations'],'protocol':read(w/'METRIC_PROTOCOL.json'),'AE1_retention':{k:v for k,v in read(w/'retention.json').items() if k in ['delta','E1_delta','C_correct','E1_correct','pass']},'S1_retention':read(w/'s1-retention.json')}
    write(o/'summary.json',summary)
    for name in ['results.json','final-metrics.json','progress.json','mechanism-admission-summary.json','persistence.json','retention.json','s1-retention.json','CONFIRMATORY_LOCK.json','METRIC_PROTOCOL.json','ACQUISITION_PROTOCOL.json','DEVELOPMENT_REUSE.json','continued-retention.json']:
        if (w/name).is_file():(o/name).write_bytes((w/name).read_bytes())
    print(json.dumps(summary,indent=2))

if __name__=='__main__':main()
