#!/usr/bin/env python3
"""Summarize S2-R1's read-only traces; never opens the final S2 holdout."""
import argparse
import csv
import hashlib
import json
import math
from collections import Counter
from pathlib import Path


def load(path):
    return json.loads(path.read_text())


def write(path, data):
    path.write_text(json.dumps(data, indent=2, sort_keys=True, allow_nan=False)+'\n')


def csv_write(path, rows):
    with path.open('w', newline='') as f:
        w = csv.DictWriter(f, fieldnames=list(rows[0]), lineterminator='\n')
        w.writeheader()
        w.writerows(rows)


def corr(a, b):
    ma, mb = sum(a)/len(a), sum(b)/len(b)
    va = sum((v-ma)**2 for v in a)
    vb = sum((v-mb)**2 for v in b)
    return sum((v-ma)*(w-mb) for v,w in zip(a,b))/math.sqrt(va*vb) if va*vb else None


def main(work, out):
    out.mkdir(parents=True, exist_ok=True)
    admission = load(work/'admission-traces.json')
    inputs = load(work/'input-path.json')
    causal = load(work/'causal-interventions.json')
    graph = inputs['source_graph']
    model = graph['model']
    index = {f['node']: f for f in graph['features']}
    def ancestors(node):
        f = index[node]
        if f['kind'] == 1:
            return [j for j,v in enumerate(f['axis']) if v != 0]
        return sorted(set(j for p in f['parents'] for j in ancestors(p)))
    feature_map = []
    baseline_rmse = causal['baseline']['rmse']
    for f in graph['features']:
        node = f['node']
        values = [next(c for c in r['inspection']['features'] if c['node']==node) for r in inputs['rows']]
        ablation = next(c for c in causal['interventions'] if c['intervention']==f'readout/{node}/factor/0')
        feature_map.append({'node': node, 'kind': f['kind'], 'parents': f['parents'],
                            'source_coordinates': ancestors(node),
                            'signed_readout_weight': f['signed_readout_weight'],
                            'activation_min': min(v['activation'] for v in values),
                            'activation_max': max(v['activation'] for v in values),
                            'saturated_activation_n': sum(abs(v['activation'])==1 for v in values),
                            'metric_unique_n': len(set(v['metric'] for v in values)),
                            'ablation_delta_rmse': ablation['metrics']['rmse']-baseline_rmse,
                            'ablation_max_prediction_delta': ablation['max_prediction_delta'],
                            'ablation_by_operation': ablation['metrics']['by_operation']})
    write(out/'feature-contributions.json', feature_map)
    coordinate_map = []
    for j in range(70):
        values = [r['inspection']['x'][j] for r in inputs['rows']]
        mean = sum(values)/len(values)
        downstream = [f['node'] for f in feature_map if j in f['source_coordinates']]
        intervention = next(c for c in causal['interventions'] if c.get('coordinate')==j)
        source = graph['sources'][j]
        status = 'available-but-not-recruited' if source==0 else ('active-and-contributing' if intervention['changed_predictions'] else 'active-but-no-measured-readout-effect')
        coordinate_map.append({'coordinate':j,'source_neuron':source or None,
            'min':min(values),'max':max(values),'sd':math.sqrt(sum((v-mean)**2 for v in values)/len(values)),
            'downstream_features':downstream,'source_to_feature_signed_synapses':[
                {'target_feature':f['node'],'signed_parameter':f['axis'][j],
                 'exc_dynamic_weight':max(0,f['axis'][j])/8,
                 'inh_dynamic_weight':max(0,-f['axis'][j])/8}
                for f in graph['features'] if f['kind']==1 and f['axis'][j]!=0],
            'zero_coordinate_max_prediction_delta':intervention['max_prediction_delta'],
            'zero_coordinate_mse_delta':intervention['metrics']['mse']-causal['baseline']['mse'],
            'status':status,'status_scope':'numerical forward pass, not persistent recurrent activation'})
    write(out/'input-coordinate-map.json', coordinate_map)
    csv_write(out/'input-coordinate-map.csv', [dict(coordinate=r['coordinate'],source_neuron=r['source_neuron'],min=r['min'],max=r['max'],sd=r['sd'],downstream_features=' '.join(map(str,r['downstream_features'])),max_prediction_delta=r['zero_coordinate_max_prediction_delta'],status=r['status']) for r in coordinate_map])
    lesson_rows, candidate_rows = [], []
    for lesson in admission['lessons']:
        tr = lesson['trace']
        reasons = Counter(r for c in tr['Trials'] for r in c.get('rejected_by') or [])
        direction = [c for c in tr['Trials'] if c['kind'] in ['missing-direction','missing-boundary']]
        axis = tr['ResidualAxis'] or []
        best = min(tr['Trials'], key=lambda c:c['check_mse'])
        lesson_rows.append({'lesson':lesson['lesson'],'sequence_before':lesson['sequence_before'],
            'committed':tr['Diagnosis']['committed'],'features_before':len(tr['Before']['Circuits']),
            'before_mse':tr['Diagnosis']['before_error'],'after_mse':tr['Diagnosis']['after_error'],
            'candidate_n':len(tr['Trials']),'no_check_improvement_n':reasons['no-check-improvement'],
            'retention_regression_n':reasons['retention-regression'],
            'duplicate_n':reasons['functionally-duplicate'],'invalid_fit_n':reasons['fit-invalid-or-parameter-bound'],
            'retention_only_n':sum(c.get('rejected_by')==['retention-regression'] for c in tr['Trials']),
            'check_overlap_fit_n':tr['Evidence']['check_overlap_fit_n'],
            'forecast_axis_l2':math.sqrt(sum(v*v for v in axis[56:59])),
            'history_axis_l2':math.sqrt(sum(v*v for v in axis[59:62])),
            'history_operation_axis_l2':math.sqrt(sum(v*v for v in axis[63:65])),
            'direction_activation_mse':next((c['check_mse'] for c in direction if not c['metric']),None),
            'direction_metric_mse':next((c['check_mse'] for c in direction if c['metric']),None),
            'minimum_candidate_check_mse':best['check_mse'],
            'best_candidate_kind':best['kind'],
            'best_candidate_model_sha256':best['candidate_model_sha256']})
        for c in tr['Trials']:
            candidate_rows.append({'lesson':lesson['lesson'],'index':c['index'],'kind':c['kind'],'feature':c['feature'],
                'mode':'metric' if c['metric'] else 'activation','fit_valid':c['fit_valid'],'eligible':c['eligible'],
                'fit_mse':c['fit_mse'],'check_mse':c['check_mse'],'best_mse_at_decision':c['best_check_mse_at_decision'],
                'regressions':c['regressions'],'rejected_by':' '.join(c.get('rejected_by') or []),
                'candidate_model_sha256':c['candidate_model_sha256']})
    csv_write(out/'admission-summary.csv', lesson_rows)
    csv_write(out/'candidate-trials.csv', candidate_rows)
    positive = load(work/'positive-controls.json')
    positive_summary = []
    for p in positive:
        tr=p['trace'];extra = tr['ResidualAxis'][56:70] if tr['ResidualAxis'] else []
        positive_summary.append({'seed':p['seed'],'coordinate':p['coordinate'],'other_coordinates_vary':p['other_coordinates_vary'],
             'selected_by_unchanged_native_rules':tr['Diagnosis']['committed'],'cause':tr['Diagnosis']['cause'],
             'before_check_mse':tr['Diagnosis']['before_error'],'after_check_mse':tr['Diagnosis']['after_error'],
             'extra_axis_l2':math.sqrt(sum(v*v for v in extra)),
             'eligible_direction_n':sum(c['eligible'] and c['kind'].startswith('missing-direction') for c in tr['Trials'])})
    write(out/'positive-control-summary.json',positive_summary)
    probes=load(work/'basis-probes.json')
    basis_summary=[]
    for group in probes:
        passing=[p for p in group['trials'] if p['fit_valid'] and p['check_gain'] and not p['regressions'] and not p['duplicate']]
        compact=[{k:v for k,v in p.items() if k not in ['new_axis','retention_by_query']} for p in passing]
        basis_summary.append({'lesson':group['lesson'],'trials':len(group['trials']),'gain_without_retention_regression':compact,
                              'fit_valid_n':sum(p['fit_valid'] for p in group['trials']),
                              'best_valid_check_mse':min(p['check_mse'] for p in group['trials'] if p['fit_valid'])})
    write(out/'basis-probe-summary.json',basis_summary)
    duplicates=[]
    for i,f in enumerate(graph['features']):
        for other in graph['features'][:i]:
            a=[next(c for c in r['inspection']['features'] if c['node']==f['node'])['activation'] for r in inputs['rows']]
            b=[next(c for c in r['inspection']['features'] if c['node']==other['node'])['activation'] for r in inputs['rows']]
            if a==b or a==[-v for v in b]:duplicates.append({'a':f['node'],'b':other['node'],'identical_or_opposite_activation':True})
    write(out/'activation-duplicates.json',duplicates)
    target=[r['target'] for r in inputs['rows']]
    signal=[]
    for j in range(56,65):
        x=[r['inspection']['x'][j] for r in inputs['rows']]
        signal.append({'coordinate':j,'pearson_target':corr(x,target),
            'pearson_residual':corr(x,[t-r['inspection']['value'] for t,r in zip(target,inputs['rows'])]),
            'interpretation':'descriptive development correlation, not causal or independent predictive evidence'})
    write(out/'extra-signal-correlation.json',signal)
    summary={'source':load(work/'source-validation.json'),'exact_16_historical_receipts':admission['exact_16_receipt_matches'],
        'input_conditions':[{k:c[k] for k in ['condition','n','changed_predictions','max_prediction_delta']} for c in inputs['conditions']],
        'lessons':lesson_rows,'causal_baseline':causal['baseline'],'positive_controls':positive_summary,
        'basis_probes':[{k:v for k,v in p.items() if k!='gain_without_retention_regression'}|{'gain_without_retention_regression_n':len(p['gain_without_retention_regression'])} for p in basis_summary],
        'final_holdout_used':False,'new_learning_commands':0,'canonical_source_mutated':False}
    if (work/'partial-step-interventions.json').exists():
        step_summary=[]
        for group in load(work/'partial-step-interventions.json'):
            eligible=[r for r in group['probes'] if r['check_gain'] and r['regressions']==0]
            step_summary.append({'lesson':group['lesson'],'tested':len(group['probes']),
                'gains_without_retention_regression':len(eligible),
                'best':min(eligible,key=lambda r:r['check_mse']) if eligible else None,
                'direction_gains':[r for r in eligible if r['kind']=='missing-direction']})
        write(out/'partial-step-summary.json',step_summary)
        summary['partial_steps']=step_summary
        conditions=[]
        for group in load(work/'learning-input-interventions.json'):
            tr=group['trace'];conditions.append({'lesson':group['lesson'],'condition':group['condition'],
                'selected':tr['Diagnosis']['committed'],'after_mse':tr['Diagnosis']['after_error'],
                'minimum_full_candidate_mse':min(c['check_mse'] for c in tr['Trials'])})
        write(out/'learning-input-summary.json',conditions)
        witness=[]
        for group in load(work/'witness-controls.json'):
            tr=group['trace'];witness.append({'lesson':group['lesson'],'control':group['control'],
                'selected':tr['Diagnosis']['committed'],'after_mse':tr['Diagnosis']['after_error'],
                'minimum_full_candidate_mse':min(c['check_mse'] for c in tr['Trials']),
                'check_overlap_fit_n':tr['Evidence']['check_overlap_fit_n'],
                'selected_vs_original_retention':group['selected_vs_original_retention']})
        write(out/'witness-control-summary.json',witness)
    if (work/'ranked-tail-probes.json').exists():
        tail=[]
        for group in load(work/'ranked-tail-probes.json'):
            tested=group['probes'];tail.append({'lesson':group['lesson'],'ranked_n':group['native_ranking_n'],
                'tested_n':len(tested),'eligible_n':sum(r.get('eligible',False) for r in tested),
                'best_valid_check_mse':min(r['check_mse'] for r in tested if r.get('fit_valid'))})
        write(out/'ranked-tail-summary.json',tail);summary['ranked_tail']=tail
    if (work/'direction-step-contributions.json').exists():
        contribution=[{k:v for k,v in row.items() if k!='parameters'} for row in load(work/'direction-step-contributions.json')]
        write(out/'direction-step-contributions.json',contribution)
        summary['direction_counterfactual']=[{'candidate_index':row['candidate_index'],'fraction':row['fraction'],
             'new_feature_weight':row['new_feature_weight'],'full_mse':row['full_metrics']['mse'],
             'old_readout_only_mse':row['old_readout_change_only']['mse'],
             'new_feature_only_mse':row['new_feature_change_only']['mse'],
             'regressions':row['regressions'],
             'input_ablations':[{k:v for k,v in a.items() if k!='metrics'}|{'mse':a['metrics']['mse']} for a in row['input_ablations']]} for row in contribution]
    write(out/'summary.json',summary)
    print(json.dumps({'lessons':len(lesson_rows),'trials':len(candidate_rows),'positive_controls':len(positive_summary),'basis_probes':summary['basis_probes']},indent=2))


if __name__=='__main__':
    p=argparse.ArgumentParser(description=__doc__)
    p.add_argument('--work',type=Path,required=True)
    p.add_argument('--output',type=Path,required=True)
    a=p.parse_args();main(a.work.resolve(),a.output.resolve())
