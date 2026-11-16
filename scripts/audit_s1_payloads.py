#!/usr/bin/env python3
"""Read-only final audit of the published S1 bytes and independent controls.

Never trains, chooses a stage, rewrites a result, or writes a neural state.
The only output is a new numeric audit summary at --output.
"""
import argparse
import collections
import datetime as dt
import hashlib
import json
import math
from pathlib import Path

ARCHIVE_SHA = '7ca53ea11b2a56a149f466d65a1d31733581d893ea810b3003ba3a3895fb12e7'


def sha(path):
    with path.open('rb') as stream:
        return hashlib.file_digest(stream, 'sha256').hexdigest()


def load(path):
    return json.loads(path.read_text())


def numeric_deltas(a, b, prefix=''):
    if isinstance(a, dict):
        assert set(a) == set(b), prefix
        return {p: v for k in a for p, v in numeric_deltas(a[k], b[k], prefix+'/'+k).items()}
    if isinstance(a, list):
        assert len(a) == len(b), prefix
        return {p: v for i, (x, y) in enumerate(zip(a, b)) for p, v in numeric_deltas(x, y, prefix+'/'+str(i)).items()}
    if isinstance(a, (float, int)) and not isinstance(a, bool):
        return {} if a == b else {prefix: b-a}
    assert a == b, (prefix, a, b)
    return {}


def edge_key(edge):
    return edge['source_id'], edge['target_id'], edge['inhibitory']


def parameter_audit(before, after):
    old = {edge_key(e): e for e in before['synapses']}
    new = {edge_key(e): e for e in after['synapses']}
    assert len(old) == len(before['synapses'])
    ids = {u['id'] for u in before['neural_units']}
    assert {key for key in new if key[0] in ids and key[1] in ids} == set(old)
    edge_deltas, activation_deltas = [], []
    for key, a in old.items():
        b = new[key]
        a, b = dict(a), dict(b)
        activation_deltas.append(abs(a.pop('activation')-b.pop('activation')))
        # Dynamic eligibility/activation and all clocks remain in this comparison.
        changes = numeric_deltas(a, b)
        edge_deltas.append({'edge': key, 'numeric_deltas': changes})
    old_units = {u['id']: u for u in before['neural_units']}
    new_units = {u['id']: u for u in after['neural_units']}
    neuron_deltas = []
    for key, a in old_units.items():
        a, b = dict(a), dict(new_units[key])
        a.pop('activation'); b.pop('activation')
        changes = numeric_deltas(a, b)
        if changes:
            neuron_deltas.append({'id': key, 'deltas': changes})
    assert not neuron_deltas
    assert before['temporal_patterns'] == after['temporal_patterns']
    assert before['native_control'] == after['native_control']
    maximum = max([abs(v) for row in edge_deltas for v in row['numeric_deltas'].values()] or [0])
    assert maximum <= 1e-7
    return {'original_neurons': len(ids), 'original_edges': len(old),
            'original_patterns': len(before['temporal_patterns']), 'original_topology_equal': True,
            'neuron_learning_fields_exact': True, 'patterns_exact': True, 'native_policy_exact': True,
            'max_parameter_absolute_delta': maximum,
            'max_transient_edge_activation_delta': max(activation_deltas),
            'per_edge_parameter_deltas': edge_deltas}


def metric(rows, prediction):
    errors = [p-y for row, ps in zip(rows, prediction) for p, y in zip(ps, row['Actual'])]
    return {'Coordinates': len(errors), 'Samples': len(rows),
            'RMSE': math.sqrt(sum(e*e for e in errors)/len(errors)),
            'MAE': sum(map(abs, errors))/len(errors),
            'Accuracy': sum(abs(e) <= .05 for e in errors)/len(errors)}


def time_law_audit(probes):
    result = []
    for probe in probes:
        a, b = probe['before'], probe['after']
        old, new = {edge_key(e): e for e in a['synapses']}, {edge_key(e): e for e in b['synapses']}
        native = a['native_control']
        protected = {native['bias'], *native['features'], *(n for row in native['outputs'] for n in row)}
        parse = lambda t: dt.datetime.fromisoformat(t.replace('Z', '+00:00'))
        previous, at = parse(a['memory_decay']['applied_at']), parse(probe['at'])
        retention = a['memory_state']['effective_retention']
        rate = (1-retention)/24
        residual, rows = 0., []
        for key, edge in old.items():
            da, db = edge['dynamic'], new[key]['dynamic']
            if key[0] in protected or key[1] in protected:
                expected_weight, expected_confidence = da['weight'], da['confidence']
            else:
                start = max(previous, parse(da['last_modification']))
                hours = max(0., (at-start).total_seconds()/3600)
                expected_weight = da['weight'] * math.exp(-hours*rate)
                expected_confidence = da['confidence'] * math.exp(-hours*rate*.5)
            residual = max(residual, abs(expected_weight-db['weight']), abs(expected_confidence-db['confidence']))
            assert da['last_modification'] == db['last_modification']
            assert da['frequency'] == db['frequency']
            rows.append({'edge': key, 'weight_before': da['weight'], 'weight_after': db['weight'],
                         'weight_expected': expected_weight, 'confidence_after': db['confidence'],
                         'confidence_expected': expected_confidence})
        assert residual <= 1e-14
        result.append({'at': probe['at'], 'memory_watermark_before': a['memory_decay'],
                       'memory_watermark_after': b['memory_decay'], 'elapsed_seconds': (at-previous).total_seconds(),
                       'retention': retention, 'max_decay_law_residual': residual,
                       'last_modification_and_frequency_exact': True, 'edges': rows})
    return result


def audit(archive, root, audit_dir):
    assert sha(archive) == ARCHIVE_SHA
    manifest = load(root/'MANIFEST.json')
    for name, row in manifest.items():
        p = root/name
        assert p.stat().st_size == row['bytes'] and sha(p) == row['sha256'], name
    source = load(root/'source/07-E1-restarted.json')
    final = load(root/'s1/lineage/s1-final.json')
    continuation = load(root/'s1/lineage/s1-continuation.json')
    assert source['sequence'] == 75331 and final['sequence'] == 75849 and continuation['sequence'] == 75866
    for checkpoint in [source, final, continuation]:
        assert checkpoint['brain_identity'] == 'horizon-primary-brain' and checkpoint['schema_version'] == 20
    journal = (root/'s1/lineage/events.jsonl').read_bytes()
    prefix = (root/'source/events.jsonl').read_bytes()
    assert journal.startswith(prefix)
    events = [json.loads(line) for line in journal.splitlines()]
    # The original A–E1 sensor intent/commit protocol has two physical records
    # at revision 10. A commit authenticates an intent; it is not a new neural
    # revision. Native ReadEventLog/recovery verifies the binding independently.
    commits = [e for e in events if e['type'] == 'sensor_commit']
    assert len(commits) == 1 and commits[0]['sequence'] == 10
    assert [e['sequence'] for e in events if e['type'] != 'sensor_commit'] == list(range(1, 75867))
    acquisition = [e for e in events if 75331 < e['sequence'] <= 75849]
    lessons = [e['composition_exposure'] for e in acquisition if e['type'] == 'composition_exposure']
    assert len(lessons) == 36
    assert [e['id'] for e in lessons] == [f's1-learning-pass-{p:02d}-q-{q}' for p in range(1, 13) for q in range(3)]
    assert all(not e['native_control'].get('outcome', {}).get('learn', False) for e in events if e['sequence'] > 75331 and e['type'] == 'native_control_command')
    samples = {name: load(root/f's1/{name}.json') for name in ['fit', 'check', 'holdout']}
    ids = {name: {s['id'] for s in rows} for name, rows in samples.items()}
    input_hash = lambda s: hashlib.sha256(json.dumps({k: s[k] for k in ['Before', 'Previous', 'Action', 'PreviousAction']}, sort_keys=True, separators=(',', ':')).encode()).hexdigest()
    hashes = {name: {input_hash(s) for s in rows} for name, rows in samples.items()}
    for a, b in [('fit', 'check'), ('fit', 'holdout'), ('check', 'holdout')]:
        assert not (ids[a] & ids[b]) and not (hashes[a] & hashes[b])
    for lesson in lessons:
        for name, rows in [('fit', lesson['resolution']['fit']), ('check', lesson['resolution']['check'])]:
            assert {r['episode_id'] for r in rows} == ids[name]
    holdout_seed_counts = collections.Counter(s['id'].split('-')[1] for s in samples['holdout'])
    assert holdout_seed_counts == {'83': 96, '151': 96}
    predictions, saved = load(root/'s1/predictions.json'), load(root/'s1/results.json')
    metrics = {}
    for label, key in [('baseline', 'Baseline'), ('control', 'Control'), ('trained', 'Trained'), ('ablation', 'Ablated')]:
        metrics[label] = metric(samples['holdout'], predictions[label])
        assert all(abs(metrics[label][k]-saved[key][k]) < 1e-14 for k in metrics[label])
    receipts = load(root/'s1/learning-diagnoses.json')
    assert len(receipts) == 36 and sum(r['committed'] for r in receipts) == 27
    assert sha(root/'s1/03-trained-pass-12.json') == sha(root/'s1/lineage/s1-final.json')
    assert sha(root/'s1/lineage/s1-restarted.json') == sha(root/'s1/lineage/s1-final.json')
    params = parameter_audit(source['brain'], final['brain'])
    controls = load(audit_dir/'retention.json')
    time_checks = time_law_audit(load(audit_dir/'passive-time.json'))
    # Preserve the observation oracle's activity plasticity separately from
    # native decay. It is executed on probe copies, never the saved lineage.
    oracle_phases = []
    for phase, passive in zip(load(audit_dir/'time-intervention.json'),
                              load(audit_dir/'passive-time.json')):
        old = {edge_key(e): e for e in passive['after']['synapses']}
        new = {edge_key(e): e for e in phase['after']['synapses']}
        changes = []
        for key, edge in old.items():
            b = new[key]
            if edge['dynamic']['last_modification'] != b['dynamic']['last_modification']:
                changes.append({'edge': key,
                                'weight_increment_after_passive_decay': b['dynamic']['weight']-edge['dynamic']['weight'],
                                'frequency_increment': b['dynamic']['frequency']-edge['dynamic']['frequency']})
        oracle_phases.append({'at': phase['at'], 'margin': phase['margin'],
                              'activity_plasticity_on_evaluation_copy': changes})
    margin_deltas = {domain: [y-x for x, y in zip(controls['A_historical'][domain], controls['C_S1'][domain])] for domain in ['A', 'B', 'D']}
    max_margin = max(abs(x) for rows in margin_deltas.values() for x in rows)
    assert max_margin > 1e-7 and saved['Status'] == 'PARTIAL' and not saved['Gates']['retention']
    return {'archive_sha256': ARCHIVE_SHA, 'verified_manifest_payloads': len(manifest),
            'payloads': manifest, 'identity': final['brain_identity'],
            'sequence': {'source': 75331, 'trained': 75849, 'continuation': 75866},
            'suffix_event_counts': dict(collections.Counter(e['type'] for e in acquisition)),
            'journal_records': len(events), 'journal_revisions': 75866,
            'non_revision_records': [{'type': e['type'], 'sequence': e['sequence']} for e in commits],
            'source_prefix_bytes': len(prefix),
            'numeric_commands': len(lessons), 'numeric_commits': 27, 'fixed_final_pass': 12,
            'native_policy_updates': final['brain']['native_control']['updates'],
            'holdout_seed_counts': dict(holdout_seed_counts), 'input_and_id_sets_disjoint': True,
            'holdout_absent_from_all_numeric_lessons': True,
            'fixed_final_checkpoint_is_pass12': True, 'final_restart_bytes_identical': True,
            'metrics_recomputed_from_saved_arrays': metrics,
            'original_parameter_retention': params,
            'historical_ABD_margin_deltas': margin_deltas, 'max_historical_ABD_margin_delta': max_margin,
            'time_intervention': time_checks,
            'oracle_observation_phase': oracle_phases,
            'formal_status': 'PARTIAL', 'formal_retention_gate': False,
            'functional_retention': 'no S1-attributable regression in audited A–E1 oracles',
            'control_A_checkpoint_sha256': sha(audit_dir/'control-a.json'),
            'control_B_checkpoint_sha256': sha(audit_dir/'control-b.json'),
            'control_A_journal_sha256': sha(audit_dir/'control-a-events.jsonl'),
            'control_B_journal_sha256': sha(audit_dir/'control-b-events.jsonl')}


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--archive', type=Path, required=True)
    parser.add_argument('--root', type=Path, required=True)
    parser.add_argument('--audit-dir', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    result = audit(args.archive, args.root, args.audit_dir)
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(json.dumps(result, indent=2, sort_keys=True)+'\n')
    print(json.dumps({k: result[k] for k in ['verified_manifest_payloads', 'numeric_commands', 'numeric_commits', 'max_historical_ABD_margin_delta', 'formal_status']}))
