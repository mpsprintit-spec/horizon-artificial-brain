package runtime

// Section markers, coordinate-group lengths and confidence retain exact
// agreement. Tolerance applies only to the declared numeric measurement axes,
// preventing a different field layout from being reused through noisy markers.
func (s SensorObservation) projectionTolerance() []float64 {
	if s.NoiseTolerance == 0 {
		return nil
	}
	out := []float64{0, 0}
	for range s.ChannelStates {
		out = append(out, s.NoiseTolerance)
	}
	out = append(out, 0, 0)
	for range s.SpatialCoordinates {
		out = append(out, s.NoiseTolerance)
	}
	out = append(out, 0, 0)
	for range s.TemporalDelta {
		out = append(out, s.NoiseTolerance)
	}
	return append(out, 0)
}
