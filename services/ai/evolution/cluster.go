package evolution

import (
	"math"
	"sort"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

// Cluster adalah sekumpulan node yang pola statistiknya MIRIP -- belum
// punya arti/nama apa pun. Nama/peran baru muncul di tahap berikutnya
// (Role Discovery), setelah cluster ini terbentuk sendiri dari kemiripan.
type Cluster struct {
	ID       int
	Members  []knowledge.NodeID
	Centroid []float64
}

// vector mengubah statistik mentah jadi angka yang bisa dibandingkan --
// log1p dipakai supaya jumlah yang besar (mis. 1 vs 100 koneksi) tidak
// mendominasi cuma karena skalanya jauh beda dari confidence (0-1).
func vector(s NodeStats) []float64 {
	return []float64{
		math.Log1p(float64(s.IsAIncoming)),
		math.Log1p(float64(s.IsAOutgoing)),
		math.Log1p(float64(s.DescriptiveIncoming)),
		math.Log1p(float64(s.DistinctRelationKinds)),
		math.Log1p(float64(s.OutDegree)),
		math.Log1p(float64(s.InDegree)),
		s.AverageConfidence,
	}
}

func distance(a, b []float64) float64 {
	var sum float64
	for i := range a {
		d := a[i] - b[i]
		sum += d * d
	}
	return math.Sqrt(sum)
}

// ClusterThreshold adalah jarak maksimum supaya dua node dianggap cukup
// mirip untuk masuk cluster yang sama -- parameter MEKANISME (mirip ambang
// plastisitas sinaps), bukan aturan makna.
const ClusterThreshold = 0.6

// ClusterNodes mengelompokkan seluruh node berdasarkan kemiripan pola
// statistiknya. Cluster BELUM punya nama/arti -- cuma nomor urut.
func ClusterNodes(kb *knowledge.KnowledgeBase) []Cluster {
	nodes := kb.Registry.Nodes()
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].ID < nodes[j].ID })
	var clusters []Cluster
	for _, node := range nodes {
		vec := vector(ComputeStats(kb, node))
		bestIdx := -1
		bestDist := ClusterThreshold
		for i, c := range clusters {
			d := distance(vec, c.Centroid)
			if d < bestDist {
				bestDist = d
				bestIdx = i
			}
		}
		if bestIdx == -1 {
			clusters = append(clusters, Cluster{
				ID:       len(clusters) + 1,
				Members:  []knowledge.NodeID{node.ID},
				Centroid: vec,
			})
			continue
		}
		c := &clusters[bestIdx]
		n := float64(len(c.Members))
		for i := range c.Centroid {
			c.Centroid[i] = (c.Centroid[i]*n + vec[i]) / (n + 1)
		}
		c.Members = append(c.Members, node.ID)
	}
	return clusters
}
