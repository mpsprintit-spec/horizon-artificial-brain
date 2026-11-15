package runtime

import (
	k "github.com/project-horizon/horizon-core/services/ai/knowledge"
	"testing"
	"time"
)

func BenchmarkGateGAdmission(b *testing.B) {
	rt := telemetryBenchmarkRuntime(b, 1000, 4)
	cfg := nativeControlConfig([]k.NodeID{1})
	if _, e := rt.ApplyNativeControl(NativeControlCommand{ID: "initialize", Initialize: &cfg}, time.Date(2026, 11, 6, 0, 0, 0, 0, time.UTC)); e != nil {
		b.Fatal(e)
	}
	samples := make([]float64, b.N)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()
		rt.mu.Lock()
		e := rt.admitCommandLocked(EventTypeSensorIntent)
		rt.mu.Unlock()
		if e != nil {
			b.Fatal(e)
		}
		samples[i] = float64(time.Since(start)) / 1e6
	}
	b.StopTimer()
	gateGBenchMetrics(b, "admission", samples)
}
