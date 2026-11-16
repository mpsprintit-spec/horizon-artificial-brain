#!/usr/bin/env python3
"""Read-only S2 evidence analysis. Never acquires data or changes a checkpoint."""
import argparse
from collections import Counter
import csv
import hashlib
import json
from pathlib import Path

import numpy as np


def metric(rows, predictions):
    y = np.array([r['Error'] for r in rows])
    x = np.array(predictions)
    return {'n': len(rows), 'rmse': float(np.sqrt(np.mean((x-y)**2))),
            'mae': float(np.mean(abs(x-y))), 'correlation': float(np.corrcoef(x, y)[0, 1]) if np.std(x)>1e-12 else 0.0,
            'negative_estimates': int(np.sum(x<0)), 'actual_min': float(y.min()), 'actual_max': float(y.max())}


def analyze(work):
    read = lambda name: json.loads((work/name).read_text())
    rows, pred, p, result = read('holdout.json'), read('predictions.json'), read('PROTOCOL.json'), read('results.json')
    final = read('lineage/s2-final.json')
    learning = read('learning.json')
    # Check target recognition independently from the experiment's accumulator.
    for name in ('development.json', 'training.json', 'training-new.json', 'holdout.json', 'continuation-trials.json'):
        for r in read(name):
            residual = np.array(r['Actual'])-np.array(r['Pre']['Forecast'])
            if not np.array_equal(residual, np.array(r['Residual'])) or abs(float(abs(residual).mean())-r['Error'])>1e-15:
                raise ValueError('post-operation error target differs: '+r['ID'])
    ledger = [json.loads(line) for line in (work/'preoperation.jsonl').read_text().splitlines()]
    ids = [x['id'] for x in ledger]
    if len(set(ids)) != len(ids):
        raise ValueError('duplicate pre-operation ID')
    by_id = {x['id']: x for x in ledger}
    all_rows = sum([read(n) for n in ('development.json', 'training.json', 'training-new.json', 'holdout.json', 'continuation-trials.json')], [])
    for r in all_rows:
        before = by_id[r['ID']]
        if before['input_sha256'] != r['InputSHA256'] or before['forecast'] != r['Pre']['Forecast'] or before['estimate'] != r['Estimate']:
            raise ValueError('pre-operation receipt changed: '+r['ID'])
    if len(ledger) != len(all_rows):
        raise ValueError('pre-operation receipt count differs')
    # Diagnostic uncertainty only: no new acceptance rule and no model selection.
    # Paired blocks keep local prior-error dependence within each eight-trial block.
    rng = np.random.default_rng(1703)
    y = np.array([r['Error'] for r in rows])
    candidate, constant = np.array(pred['D_S2']), np.array(pred['B_constant'])
    differences = []
    for _ in range(4000):
        indices = []
        for start in (0, 96):
            for block in rng.integers(0, 12, size=12):
                indices.extend(range(start+int(block)*8, start+(int(block)+1)*8))
        ix = np.array(indices)
        differences.append(np.sqrt(np.mean((constant[ix]-y[ix])**2))-np.sqrt(np.mean((candidate[ix]-y[ix])**2)))
    operation_metrics = {}
    operation_constant = []
    training = read('training.json')+read('training-new.json')
    means = {a: float(np.mean([r['Error'] for r in training if r['Pre']['Action']==a])) for a in (0, 1)}
    for a, name in enumerate(('propagate', 'stop')):
        ix = [i for i,r in enumerate(rows) if r['Pre']['Action']==a]
        part = [rows[i] for i in ix]
        operation_metrics[name] = {arm: metric(part, [values[i] for i in ix]) for arm, values in pred.items()}
    operation_constant = [means[r['Pre']['Action']] for r in rows]
    new = read('training-new.json')
    # Its recorded estimates were made before these new outcomes were supplied
    # to lessons 9–16. All those lessons were rejected; the readout is unchanged.
    new_commits = sum(d['committed'] for d in learning[8:])
    diagnostic = {
        'analysis_scope': 'post-holdout descriptive audit only; frozen gates unchanged',
        'preoperation_receipts_verified': len(ledger),
        'all_error_targets_independently_verified': len(all_rows),
        'holdout_RMSE_improvement_fraction': 1-result['metrics']['D_S2']['RMSE']/result['metrics']['B_constant']['RMSE'],
        'paired_block_bootstrap': {'repetitions': 4000, 'seed': 1703, 'block_trials': 8,
                                  'RMSE_gain_CI95': [float(v) for v in np.quantile(differences, [.025, .975])]},
        'by_operation': operation_metrics,
        'operation_mean_training_control': metric(rows, operation_constant),
        'operation_training_means': means,
        'new_error_block_before_learning': metric(new, [r['Estimate'] for r in new]),
        'new_error_block_native_commits': new_commits,
        'previous_error_context_removed_RMSE_delta': result['history_incremental_rmse'],
        'active_extra_forecast_or_history_source_coordinates': [i for i,x in enumerate(final['brain']['n_resolution']['sources']) if i>=56 and x],
        'numeric_features': len(final['brain']['n_resolution']['features']),
        'numeric_feature_ceiling_unchanged': 32,
        'neurons': len(final['brain']['neural_units']), 'synapses': len(final['brain']['synapses']),
        'PatternSynapses': len(final['brain']['temporal_patterns']),
        'lesson_rejections': [dict(Counter(reason for trial in d['trials'] for reason in trial.get('rejected_by', []))) for d in learning],
        'status': result['status'],
    }
    (work/'analysis.json').write_text(json.dumps(diagnostic, indent=2, sort_keys=True)+'\n')
    with (work/'holdout-errors.csv').open('w', newline='') as f:
        writer = csv.writer(f)
        writer.writerow(['id','action','actual_error','S1_without_S2','constant_first_block','constant_full_training','untrained','S2','ablated'])
        for i,r in enumerate(rows):
            writer.writerow([r['ID'], ('propagate','stop')[r['Pre']['Action']], r['Error']]+[pred[key][i] for key in ('A_S1','B_constant','B_full_training_constant','C_untrained','D_S2','E_ablated')])
    import matplotlib
    matplotlib.use('Agg')
    import matplotlib.pyplot as plt
    plt.rcParams.update({'font.size': 10, 'axes.spines.top': False, 'axes.spines.right': False})
    fig, ax = plt.subplots(1, 3, figsize=(13, 3.8), constrained_layout=True)
    labels = ['S1 / tanpa S2', 'Konstan', 'S2', 'Ablasi']
    values = [result['metrics'][key]['RMSE'] for key in ('A_S1','B_constant','D_S2','E_ablated')]
    ax[0].bar(range(4), values, color=['#9AA9BA','#71869C','#178D8A','#C3815B'])
    ax[0].set_xticks(range(4), labels, rotation=22, ha='right')
    ax[0].set_ylabel('RMSE estimasi error')
    ax[0].set_title('192 percobaan holdout akhir')
    for a,name,color in ((0,'propagate','#178D8A'),(1,'stop','#C3815B')):
        ix = np.array([r['Pre']['Action']==a for r in rows])
        ax[1].scatter(candidate[ix], y[ix], s=16, alpha=.7, label=name, color=color)
    limit = float(max(y.max(), candidate.max()))*1.04
    for a in ax[1:]:
        a.plot([0,limit],[0,limit], '--', color='#9AA9BA', linewidth=1)
        a.set_xlabel('Estimasi rata-rata error absolut')
        a.set_ylabel('Rata-rata error absolut aktual')
        a.set_xlim(min(0, float(candidate.min())*1.1),limit);a.set_ylim(0,limit)
    ax[1].set_title('Korelasi = %.3f'%result['metrics']['D_S2']['Correlation'])
    ax[1].legend(frameon=False)
    bins = result['metrics']['D_S2']['Bins']
    ax[2].plot([b['estimated'] for b in bins], [b['actual'] for b in bins], 'o-', color='#178D8A')
    for b in bins:
        ax[2].annotate('n='+str(b['n']), (b['estimated'],b['actual']), xytext=(3,6), textcoords='offset points', fontsize=8)
    ax[2].set_title('Error kalibrasi kelompok tetap = %.5f'%result['metrics']['D_S2']['CalibrationError'])
    fig.suptitle('Horizon S2 — PARTIAL; kriteria protokol dipertahankan', fontsize=13)
    fig.savefig(work/'s2-evaluation.png', dpi=180)
    fig.savefig(work/'s2-evaluation.svg')
    plt.close(fig)
    print(json.dumps({k:diagnostic[k] for k in ('status','preoperation_receipts_verified','holdout_RMSE_improvement_fraction','paired_block_bootstrap','new_error_block_native_commits')}))


if __name__ == '__main__':
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--work', type=Path, required=True)
    a = p.parse_args()
    analyze(a.work.resolve())
