package hfcc

import "encoding/json"

// MarshalCandidate serializes a candidate preserving nested structure (not a flat string).
func MarshalCandidate(c Candidate) ([]byte, error) {
	return json.Marshal(c)
}

// UnmarshalCandidate restores a candidate from JSON.
func UnmarshalCandidate(data []byte) (Candidate, error) {
	var c Candidate
	err := json.Unmarshal(data, &c)
	return c, err
}

// MarshalCandidateSet serializes a full multi-candidate set.
func MarshalCandidateSet(cs CandidateSet) ([]byte, error) {
	return json.Marshal(cs)
}

// UnmarshalCandidateSet restores a candidate set.
func UnmarshalCandidateSet(data []byte) (CandidateSet, error) {
	var cs CandidateSet
	err := json.Unmarshal(data, &cs)
	return cs, err
}

// MarshalExperiment serializes an experiment audit record.
func MarshalExperiment(e ExperimentRecord) ([]byte, error) {
	return json.Marshal(e)
}
