package knowledge

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"testing"
	"time"
)

// The oracle deliberately bypasses NativeDigest and its specialized encoders.
func TestNativeDigestMatchesFrozenJSON(t *testing.T) {
	r := rand.New(rand.NewSource(151))
	states := []map[NodeID]float64{nil, {}, {1: 0, 2: math.Copysign(0, -1), 10: 1e-7, -1: 1e21, 100: math.SmallestNonzeroFloat64, NodeID(math.MinInt64): math.MaxFloat64, NodeID(math.MaxInt64): -1e-6}}
	for _, count := range []int{9, 1000, 40000} {
		state := make(map[NodeID]float64, count)
		for i := 0; i < count; i++ {
			value := math.Float64frombits(r.Uint64())
			if !nativeFinite(value) {
				value = .8
			}
			state[NodeID(r.Int63())] = value
		}
		states = append(states, state)
	}
	check := func(v any, encoded []byte, err error) {
		t.Helper()
		old, oldErr := json.Marshal(v)
		if (err == nil) != (oldErr == nil) || !bytes.Equal(old, encoded) {
			t.Fatalf("JSON changed for %T: errors %v / %v", v, err, oldErr)
		}
		want := ""
		if oldErr == nil {
			sum := sha256.Sum256(old)
			want = hex.EncodeToString(sum[:])
		}
		if got := NativeDigest(v); got != want {
			t.Fatalf("digest changed for %T", v)
		}
	}
	for i, state := range states {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			data, err := nativeDigestState(state).MarshalJSON()
			check(state, data, err)
			for _, decisions := range []int{0, 1, 3} {
				ep := &NativeEpisode{MatchKey: "<&>\u2028\u2029\n\"\xff", Intervention: true, ID: "episode", Training: true, Started: time.Date(2026, 11, 15, 0, 0, 0, 123, time.FixedZone("test", 7*3600)), Stopped: true, Truncated: true}
				for j := 0; j < decisions; j++ {
					ep.Decisions = append(ep.Decisions, NativeDecision{State: state, Values: []float64{math.Copysign(0, -1), .8}, Probabilities: []float64{.2, .8}, Action: 1, Cost: .03, Before: "before", After: "after", At: ep.Started})
				}
				data, err = marshalNativeEpisodeDigest(ep)
				check(ep, data, err)
			}
		})
	}
	for _, value := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		state := map[NodeID]float64{1: value}
		data, err := nativeDigestState(state).MarshalJSON()
		check(state, data, err)
		ep := &NativeEpisode{Decisions: []NativeDecision{{State: state}}}
		data, err = marshalNativeEpisodeDigest(ep)
		check(ep, data, err)
	}
	for _, ep := range []*NativeEpisode{nil, {}, {Decisions: []NativeDecision{}}, {Started: time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)}, {Decisions: []NativeDecision{{Cost: math.NaN()}}}} {
		data, err := marshalNativeEpisodeDigest(ep)
		check(ep, data, err)
	}
}
