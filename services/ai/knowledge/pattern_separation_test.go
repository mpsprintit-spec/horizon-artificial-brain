package knowledge

import "testing"

func TestSeparateTraceKeepsSimilarBranchesDistinct(t *testing.T) {
	p := NewPatternIndex()
	a := []PatternStep{{NodeID: 1, Position: 0, Activation: 1}, {NodeID: 2, Position: 1, Activation: 1}, {NodeID: 3, Position: 2, Activation: 1}}
	b := []PatternStep{{NodeID: 1, Position: 0, Activation: 1}, {NodeID: 2, Position: 1, Activation: 1}, {NodeID: 4, Position: 2, Activation: 1}}
	p.LearnTrace(a, nil, 3, 1, 1)
	p.LearnTrace(b, nil, 4, 1, 1)

	matches := p.SeparateTrace(a[:2], nil)
	if len(matches) != 2 { t.Fatalf("expected shared prefix to identify two learned branches, got %d", len(matches)) }

	matches = p.SeparateTrace([]PatternStep{a[0], a[1], a[2]}, nil)
	if len(matches) != 1 || matches[0].Result != 3 { t.Fatalf("expected exact branch A, got %#v", matches) }

	matches = p.SeparateTrace([]PatternStep{b[0], b[1], b[2]}, nil)
	if len(matches) != 1 || matches[0].Result != 4 { t.Fatalf("expected exact branch B, got %#v", matches) }
}

func TestSeparateTraceRejectsInsertedUnlearnedNode(t *testing.T) {
	p := NewPatternIndex()
	p.LearnTrace([]PatternStep{{NodeID: 1, Position: 0, Activation: 1}, {NodeID: 2, Position: 1, Activation: 1}, {NodeID: 3, Position: 2, Activation: 1}}, nil, 3, 1, 1)

	matches := p.SeparateTrace([]PatternStep{{NodeID: 1, Position: 0, Activation: 1}, {NodeID: 2, Position: 1, Activation: 1}, {NodeID: 9, Position: 2, Activation: 1}}, nil)
	if len(matches) != 0 { t.Fatalf("unlearned continuation was not separated: %d matches", len(matches)) }
}
