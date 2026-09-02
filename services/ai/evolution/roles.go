package evolution

import (
	"encoding/json"
	"os"
)

// StabilityThreshold: cluster minimal seukuran ini sebelum dianggap cukup
// mapan untuk dipertimbangkan diberi nama sama sekali.
const StabilityThreshold = 3

// NamedRole adalah pola cluster yang SUDAH pernah diberi nama -- dipakai
// sebagai preseden supaya cluster baru yang mirip mewarisi nama yang sama
// secara otomatis, tanpa tanya lagi.
type NamedRole struct {
	Name     string    `json:"name"`
	Centroid []float64 `json:"centroid"`
}

type RoleCatalog struct {
	Roles []NamedRole `json:"roles"`
}

func LoadRoleCatalog(path string) (*RoleCatalog, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &RoleCatalog{}, nil
		}
		return nil, err
	}
	var catalog RoleCatalog
	if err := json.Unmarshal(data, &catalog); err != nil {
		return nil, err
	}
	return &catalog, nil
}

func (rc *RoleCatalog) Save(path string) error {
	data, err := json.MarshalIndent(rc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// Match mencari nama yang sudah dikenal paling mirip dengan centroid sebuah
// cluster. Balikan "" kalau tidak ada yang cukup mirip (genuinely baru).
func (rc *RoleCatalog) Match(centroid []float64) string {
	best := ""
	bestDist := ClusterThreshold
	for _, role := range rc.Roles {
		d := distance(centroid, role.Centroid)
		if d < bestDist {
			bestDist = d
			best = role.Name
		}
	}
	return best
}

// Learn menambahkan nama baru ke katalog -- HANYA dipanggil saat user
// memberi nama secara eksplisit untuk cluster yang genuinely belum dikenal.
func (rc *RoleCatalog) Learn(name string, centroid []float64) {
	rc.Roles = append(rc.Roles, NamedRole{Name: name, Centroid: centroid})
}
