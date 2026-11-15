package knowledge

import (
	"cmp"
	"encoding/json"
	"errors"
	"math"
	"slices"
	"strconv"
	"time"
)

// Digest encoding is a transient view, never a second state or a checkpoint
// format. Keep encoding/json's field order, decimal key order, nil distinctions,
// escaping, float spelling and error behavior. The standard encoder still owns
// episode metadata; only the large integer-key activity maps avoid reflection.
type nativeDigestState map[NodeID]float64

func (state nativeDigestState) MarshalJSON() ([]byte, error) {
	if state == nil {
		return []byte("null"), nil
	}
	type entry struct {
		key   string
		value float64
	}
	entries := make([]entry, 0, len(state))
	for id, value := range state {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return nil, errors.New("nonfinite native digest activity")
		}
		entries = append(entries, entry{strconv.FormatInt(int64(id), 10), value})
	}
	// JSON sorts the decimal spelling, not the numeric address ("10" < "2").
	slices.SortFunc(entries, func(a, b entry) int { return cmp.Compare(a.key, b.key) })
	out := make([]byte, 0, 2+len(entries)*24)
	out = append(out, '{')
	for i, item := range entries {
		if i != 0 {
			out = append(out, ',')
		}
		out = append(out, '"')
		out = append(out, item.key...)
		out = append(out, '"', ':')
		format := byte('f')
		if magnitude := math.Abs(item.value); magnitude != 0 && (magnitude < 1e-6 || magnitude >= 1e21) {
			format = 'e'
		}
		out = strconv.AppendFloat(out, item.value, format, -1, 64)
		// encoding/json emits e-9 rather than strconv's e-09.
		if n := len(out); format == 'e' && n >= 4 && out[n-4] == 'e' && out[n-3] == '-' && out[n-2] == '0' {
			out[n-2] = out[n-1]
			out = out[:n-1]
		}
	}
	return append(out, '}'), nil
}

func marshalNativeEpisodeDigest(ep *NativeEpisode) ([]byte, error) {
	if ep == nil {
		return []byte("null"), nil
	}
	type decision struct {
		State         nativeDigestState `json:"state"`
		Values        []float64         `json:"values"`
		Probabilities []float64         `json:"probabilities"`
		Action        int               `json:"action"`
		Cost          float64           `json:"cost"`
		Before        string            `json:"before"`
		After         string            `json:"after"`
		At            time.Time         `json:"at"`
	}
	decisions := make([]decision, len(ep.Decisions))
	for i, d := range ep.Decisions {
		decisions[i] = decision{nativeDigestState(d.State), d.Values, d.Probabilities, d.Action, d.Cost, d.Before, d.After, d.At}
	}
	return json.Marshal(struct {
		MatchKey     string     `json:"match_key,omitempty"`
		Intervention bool       `json:"intervention,omitempty"`
		ID           string     `json:"id"`
		Training     bool       `json:"training"`
		Started      time.Time  `json:"started"`
		Decisions    []decision `json:"decisions,omitempty"`
		Stopped      bool       `json:"stopped"`
		Truncated    bool       `json:"truncated"`
	}{ep.MatchKey, ep.Intervention, ep.ID, ep.Training, ep.Started, decisions, ep.Stopped, ep.Truncated})
}
