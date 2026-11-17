#!/usr/bin/env python3
"""Resolve only the independently diagnosed Go nil/empty diagnostic-slice comparison."""
import argparse
import hashlib
import json
from pathlib import Path
import re

def main():
    p=argparse.ArgumentParser(description=__doc__);p.add_argument('--work',type=Path,required=True);a=p.parse_args();w=a.work
    original=w/'persistence-initial-struct-comparison.json'
    if not original.exists():original.write_bytes((w/'persistence.json').read_bytes())
    initial=json.loads(original.read_text());audit=json.loads((w/'metadata-audit.json').read_text())
    expected={f's{s}-c{c}' for s in [37,83,151] for c in [32,64]}
    if set(initial)!=expected or set(audit)!=expected:raise ValueError('incomplete original or independent audit')
    pattern=re.compile(r'^NResolution\.Last\.Trials\[\d+\]\.RejectedBy: nil/empty slice \(lengths 0/0\)$')
    result={}
    for key in sorted(expected):
        v=initial[key];m=audit[key]
        if not m['pass'] or not m['complete_state_exact'] or not m['serialized_NResolution_exact']:raise ValueError('actual state/metadata differs')
        if m['serialized_NResolution_sha256']!=m['replayed_NResolution_sha256']:raise ValueError('metadata hash differs')
        if not all(m[k] for k in ['feature_topology_deep_equal','head_topology_deep_equal','source_metadata_deep_equal','controls_deep_equal','capacity_exact']):raise ValueError('structural metadata differs')
        if not m['struct_differences'] or not all(pattern.fullmatch(x) for x in m['struct_differences']):raise ValueError('unexpected raw structure difference')
        if not all(v[k] for k in ['full_suffix_replay_exact','restart_exact','rollback_exact','policy_exact','witnesses_exact','historical_journal_prefix_exact']):raise ValueError('an independent persistence gate failed')
        v['initial_raw_struct_metadata_comparison']=v['feature_metadata_exact'];v['feature_metadata_exact']=True
        v['metadata_contract_audit']='metadata-audit.json/'+key;v['diagnostic_empty_slice_difference_only']=True
        result[key]=v
    (w/'persistence.json').write_text(json.dumps(result,indent=2,sort_keys=True)+'\n')
    amendment={'kind':'measurement correction; no neural or scientific-rule changes','original_persistence_sha256':hashlib.sha256(original.read_bytes()).hexdigest(),'independent_audit_sha256':hashlib.sha256((w/'metadata-audit.json').read_bytes()).hexdigest(),'difference':'Last.Trials[].RejectedBy nil versus allocated empty slices, both omitted by JSON; feature topology/heads/sources/controls/capacity and full serialized states exact','new_experiences':0,'new_learning':0,'final_holdout_reevaluated':False,'original_negative_comparison_preserved':True,'models_checked':6}
    (w/'PERSISTENCE_AUDIT_CORRECTION.json').write_text(json.dumps(amendment,indent=2,sort_keys=True)+'\n');print(json.dumps(amendment))

if __name__=='__main__':main()
