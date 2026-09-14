package runtime

import (
	"encoding/json"
	"os"
	"sort"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

// saveDeterministicBrain serializes the existing Brain substrate in a stable
// order. The neural state is unchanged; only the representation used for a
// checkpoint is canonicalized so map iteration cannot alter replay equality.
func saveDeterministicBrain(brain *knowledge.Brain, path string) error {
	if brain == nil {
		return os.ErrInvalid
	}

	nodes := append([]*knowledge.ConceptNode(nil), brain.Registry.Nodes()...)
	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i] == nil {
			return false
		}
		if nodes[j] == nil {
			return true
		}
		return nodes[i].ID < nodes[j].ID
	})

	patterns := append([]*knowledge.PatternSynapse(nil), brain.Patterns.All()...)
	sort.Slice(patterns, func(i, j int) bool {
		if patterns[i] == nil {
			return false
		}
		if patterns[j] == nil {
			return true
		}
		return patterns[i].ID < patterns[j].ID
	})

	payload := struct {
		Nodes    []*knowledge.ConceptNode   `json:"nodes"`
		Patterns []*knowledge.PatternSynapse `json:"patterns,omitempty"`
	}{Nodes: nodes, Patterns: patterns}

	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
