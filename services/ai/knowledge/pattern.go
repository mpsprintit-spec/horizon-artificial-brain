package knowledge

import (
	"sort"
	"strconv"
	"strings"
	"sync"
)

type PatternID int64

// PatternSynapse: kombinasi beberapa node (Members, urutan diabaikan) yang
// AKTIF BERSAMAAN memicu satu hasil (Result). Ini bukan node baru -- ini
// jenis sinaps kedua, disimpan di KnowledgeBase yang sama. Apa arti "Result"
// (konsep pengganti, jenis relasi, sinyal konteks, sinyal intent) BUKAN
// urusan PatternSynapse -- itu keputusan siapa pun yang membacanya nanti.
type PatternSynapse struct {
	ID         PatternID `json:"id"`
	Members    []NodeID  `json:"members"`
	Result     NodeID    `json:"result"`
	Weight     float64   `json:"weight"`
	Confidence float64   `json:"confidence"`
	Frequency  int64     `json:"frequency"`
}

type PatternIndex struct {
	mu       sync.RWMutex
	nextID   PatternID
	patterns map[PatternID]*PatternSynapse
}

func NewPatternIndex() *PatternIndex {
	return &PatternIndex{nextID: 1, patterns: map[PatternID]*PatternSynapse{}}
}

func memberKey(members []NodeID) string {
	sorted := append([]NodeID{}, members...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	parts := make([]string, len(sorted))
	for i, id := range sorted {
		parts[i] = strconv.FormatInt(int64(id), 10)
	}
	return strings.Join(parts, ",")
}

// Learn mencatat/memperkuat pola -- kalau kombinasi+hasil yang sama pernah
// diajarkan, keyakinannya TUMBUH (formula sama seperti sinaps biasa), bukan
// bikin entri baru terus-menerus.
func (p *PatternIndex) Learn(members []NodeID, result NodeID, weight, confidence float64) *PatternSynapse {
	p.mu.Lock()
	defer p.mu.Unlock()
	key := memberKey(members)
	for _, existing := range p.patterns {
		if memberKey(existing.Members) == key && existing.Result == result {
			existing.Weight = clamp01((existing.Weight*float64(existing.Frequency) + weight) / float64(existing.Frequency+1))
			existing.Confidence = clamp01(1 - (1-existing.Confidence)*(1-confidence))
			existing.Frequency++
			return existing
		}
	}
	ps := &PatternSynapse{ID: p.nextID, Members: append([]NodeID{}, members...), Result: result, Weight: weight, Confidence: confidence, Frequency: 1}
	p.patterns[p.nextID] = ps
	p.nextID++
	return ps
}

// Match mencari pola yang SELURUH anggotanya ada di candidateIDs (mis.
// seluruh node dalam satu klausa yang sedang diproses). Pola paling
// spesifik (anggota terbanyak) ditaruh duluan.
func (p *PatternIndex) Match(candidateIDs []NodeID) []*PatternSynapse {
	p.mu.RLock()
	defer p.mu.RUnlock()
	set := map[NodeID]bool{}
	for _, id := range candidateIDs {
		set[id] = true
	}
	var matches []*PatternSynapse
	for _, pattern := range p.patterns {
		if len(pattern.Members) == 0 {
			continue
		}
		allPresent := true
		for _, m := range pattern.Members {
			if !set[m] {
				allPresent = false
				break
			}
		}
		if allPresent {
			matches = append(matches, pattern)
		}
	}
	sort.Slice(matches, func(i, j int) bool { return len(matches[i].Members) > len(matches[j].Members) })
	return matches
}

func (p *PatternIndex) All() []*PatternSynapse {
	p.mu.RLock()
	defer p.mu.RUnlock()
	out := make([]*PatternSynapse, 0, len(p.patterns))
	for _, ps := range p.patterns {
		out = append(out, ps)
	}
	return out
}
// ResultsFor mengembalikan semua pola yang HASILNYA node tertentu -- dipakai
// waktu node itu SENDIRI yang aktif (bukan seluruh anggota polanya), supaya
// "tidak" tetap bisa membawa pola {saya,bercanda}->tidak sebagai bukti,
// walau cuma "tidak" sendiri yang diketik.
func (p *PatternIndex) ResultsFor(id NodeID) []*PatternSynapse {
	p.mu.RLock()
	defer p.mu.RUnlock()
	var out []*PatternSynapse
	for _, ps := range p.patterns {
		if ps.Result == id {
			out = append(out, ps)
		}
	}
	return out
}
