package runtime

import (
	"fmt"
	k "github.com/project-horizon/horizon-core/services/ai/knowledge"
	"math"
	"testing"
	"time"
)

func BenchmarkNativeEncoderVersions(b *testing.B) {
	for _, encoder := range []string{k.NativeRelationalEncoderV1, k.NativeRelationalEncoderV2} {
		for _, n := range []int{1000, 4000, 40000} {
			for _, density := range []float64{.001, 1} {
				b.Run(fmt.Sprintf("%s/n=%d/density=%g", encoder, n, density), func(b *testing.B) {
					r := telemetryBenchmarkRuntime(b, n, 4)
					cfg := nativeControlConfig([]k.NodeID{1})
					cfg.Version = k.NativeControlCausalVersion
					cfg.Encoder = encoder
					cfg.MaxActiveNodes = 100000
					cfg.MaxSynapseReads = 1000000
					if err := r.brain.InitializeNativeControl(cfg, time.Date(2026, 11, 14, 0, 0, 0, 0, time.UTC)); err != nil {
						b.Fatal(err)
					}
					state := map[k.NodeID]float64{}
					for i := 1; i <= int(math.Max(1, float64(n)*density)); i++ {
						state[k.NodeID(i)] = .8
					}
					b.ReportAllocs()
					b.ResetTimer()
					var out k.NativeReadout
					for i := 0; i < b.N; i++ {
						var err error
						out, err = r.brain.NativePolicy(state, true, 0)
						if err != nil {
							b.Fatal(err)
						}
					}
					b.StopTimer()
					b.ReportMetric(float64(out.StructuralReads), "structural_reads")
					b.ReportMetric(float64(out.ParameterReads), "parameter_reads")
				})
			}
		}
	}
}
