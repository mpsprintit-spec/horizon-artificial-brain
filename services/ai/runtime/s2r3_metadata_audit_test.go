//go:build horizon_s2r3 && horizon_s2r2

package runtime

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"testing"
)

func s2r3StructDiff(a, b reflect.Value, path string, out *[]string) {
	if a.Type() != b.Type() {
		*out = append(*out, path+": type mismatch")
		return
	}
	switch a.Kind() {
	case reflect.Pointer:
		if a.IsNil() != b.IsNil() {
			*out = append(*out, path+": nil pointer mismatch")
			return
		}
		if !a.IsNil() {
			s2r3StructDiff(a.Elem(), b.Elem(), path, out)
		}
	case reflect.Struct:
		for i := 0; i < a.NumField(); i++ {
			s2r3StructDiff(a.Field(i), b.Field(i), path+"."+a.Type().Field(i).Name, out)
		}
	case reflect.Slice:
		if a.IsNil() != b.IsNil() {
			*out = append(*out, fmt.Sprintf("%s: nil/empty slice (lengths %d/%d)", path, a.Len(), b.Len()))
		}
		if a.Len() != b.Len() {
			*out = append(*out, path+": length mismatch")
			return
		}
		for i := 0; i < a.Len(); i++ {
			s2r3StructDiff(a.Index(i), b.Index(i), fmt.Sprintf("%s[%d]", path, i), out)
		}
	default:
		if !reflect.DeepEqual(a.Interface(), b.Interface()) {
			*out = append(*out, path+": value mismatch")
		}
	}
}

func TestS2R3FeatureMetadataAudit(t *testing.T) {
	_, work, _ := s2r3Paths(t)
	results := map[string]any{}
	for _, seed := range s2r3Seeds {
		for _, cap := range []int{32, 64} {
			name := fmt.Sprintf("s%d-c%d", seed, cap)
			dir := filepath.Join(work, fmt.Sprintf("seed-%d/arm-%d", seed, cap))
			r, w := s1Open(t, filepath.Join(dir, "final.json"), filepath.Join(dir, "events.jsonl"))
			// Full 80735 suffix bytes were already proven equal. Diagnose the
			// in-memory metadata distinction by replaying the recorded last block.
			replayed, e := RebuildRuntimeFromCheckpointAndJournal(w, filepath.Join(dir, "block-6.json"))
			s1Check(t, e)
			a, b := r.brain.BrainState.NResolution, replayed.brain.BrainState.NResolution
			x, e := json.Marshal(a)
			s1Check(t, e)
			y, e := json.Marshal(b)
			s1Check(t, e)
			diff := []string{}
			s2r3StructDiff(reflect.ValueOf(a), reflect.ValueOf(b), "NResolution", &diff)
			features := reflect.DeepEqual(a.Features, b.Features)
			heads := reflect.DeepEqual(a.Heads, b.Heads)
			sources := reflect.DeepEqual(a.Sources, b.Sources)
			controls := reflect.DeepEqual(a.Controls, b.Controls)
			full := bytes.Equal(continuationBytes(t, r), continuationBytes(t, replayed))
			serialized := bytes.Equal(x, y)
			pass := features && heads && sources && controls && full && serialized && a.ExperimentalFeatureLimit == b.ExperimentalFeatureLimit && a.Contract == b.Contract
			results[name] = map[string]any{"feature_topology_deep_equal": features, "head_topology_deep_equal": heads, "source_metadata_deep_equal": sources, "controls_deep_equal": controls, "capacity_exact": a.ExperimentalFeatureLimit == b.ExperimentalFeatureLimit, "serialized_NResolution_exact": serialized, "complete_state_exact": full, "serialized_NResolution_sha256": s1Digest(x), "replayed_NResolution_sha256": s1Digest(y), "raw_struct_deep_equal": reflect.DeepEqual(a, b), "struct_differences": diff, "pass": pass}
			s1Check(t, w.Close())
			s1Write(t, filepath.Join(work, "metadata-audit.json"), results)
			t.Log(name, "metadata exact", pass, "raw struct differences", diff)
			if !pass {
				t.Fatal("actual feature metadata or serialized state differs")
			}
		}
	}
}
