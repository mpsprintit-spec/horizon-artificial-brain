package activation

// ThinkContinuously advances the existing internal brain state repeatedly
// without injecting a new external stimulus. Each step is committed back to
// the same Engine state, so the next step starts from the previous internal
// state and prediction rather than from fresh input.
//
// This is an orchestration primitive for autonomous cognition, not a
// hard-coded reasoning loop: the transitions themselves still come from the
// learned recurrent neural substrate.
func (e *Engine) ThinkContinuously(steps, cyclesPerStep int) []ThoughtResult {
	if e == nil || e.Memory == nil || steps <= 0 {
		return nil
	}
	if cyclesPerStep < 1 {
		cyclesPerStep = 1
	}

	thoughts := make([]ThoughtResult, 0, steps)
	for i := 0; i < steps; i++ {
		thought := e.ThinkWithPrediction(cyclesPerStep)
		if len(thought.Activations) == 0 {
			break
		}
		thoughts = append(thoughts, thought)
	}
	return thoughts
}
