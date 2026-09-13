package knowledge

// SeparateTrace selects learned temporal structures using the ordered cue as a
// discriminative signal. Unlike loose overlap completion, every cue element
// must occur contiguously and at the corresponding learned position. This
// keeps similar experiences separable without introducing semantic rules.
func (p *PatternIndex) SeparateTrace(cue []PatternStep, context []ContextFrame) []*PatternSynapse {
	cue = normalizeSequence(cue)
	context = normalizeContext(context)
	if len(cue) == 0 || p == nil {
		return nil
	}

	patterns := p.All()
	out := make([]*PatternSynapse, 0, len(patterns))
	for _, pattern := range patterns {
		if len(pattern.Sequence) < len(cue) {
			continue
		}
		matched := false
		for start := 0; start+len(cue) <= len(pattern.Sequence); start++ {
			matched = true
			for i := range cue {
				if pattern.Sequence[start+i].NodeID != cue[i].NodeID {
					matched = false
					break
				}
			}
			if matched {
				break
			}
		}
		if !matched {
			continue
		}
		if len(context) > 0 && contextSimilarity(pattern.Context, context) == 0 {
			continue
		}
		out = append(out, pattern)
	}
	return out
}
