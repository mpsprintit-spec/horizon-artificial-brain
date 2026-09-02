package learning

import (
	"strings"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

var stopwords = map[string]bool{
	"dan": true, "yang": true, "di": true, "ke": true, "dari": true,
	"itu": true, "ini": true, "juga": true, "atau": true, "adalah": true,
	"untuk": true, "dengan": true, "pada": true, "akan": true, "ada": true,
	"tidak": true, "saja": true, "pun": true, "lah": true, "kah": true,
	"karena": true, "menyebabkan": true, "sebab": true, "fungsi": true,
	"berguna": true, "bagian": true, "memiliki": true, "punya": true,
	"lokasi": true, "sebelum": true, "sesudah": true, "ketika": true, "saat": true,
	"merupakan": true, "yaitu": true, "ialah": true,
	"bisa": true, "dapat": true, "mampu": true,
}

// clauseConnectors: kata yang memisahkan kalimat jadi beberapa klausa, beserta
// jenis hubungan antar klausa yang dipicunya. Beda dari kata pemicu relasi
// biasa (menyebabkan, karena, dst) yang menghubungkan DUA KATA dalam satu
// klausa yang sama -- ini menghubungkan DUA KLAUSA yang berbeda.
var clauseConnectors = map[string]knowledge.RelationKind{
	"tapi":      knowledge.RelationContrast,
	"namun":     knowledge.RelationContrast,
	"sedangkan": knowledge.RelationContrast,
	"meskipun":  knowledge.RelationContrast,
	"walaupun":  knowledge.RelationContrast,
	"kalau":     knowledge.RelationCondition,
	"jika":      knowledge.RelationCondition,
	"apabila":   knowledge.RelationCondition,
	"lalu":      knowledge.RelationSequence,
	"kemudian":  knowledge.RelationSequence,
	"sehingga":  knowledge.RelationCause,
	"maka":      knowledge.RelationCause,
	"supaya":    knowledge.RelationFunction,
	"agar":      knowledge.RelationFunction,
}

// clause adalah satu unit makna dalam kalimat -- token di dalamnya boleh
// saling terhubung, tapi TIDAK otomatis terhubung ke token di klausa lain.
type clause struct {
	tokens    []string
	connector knowledge.RelationKind // relasi ke klausa SEBELUMNYA; kosong untuk klausa pertama
}

// splitClauses memecah kalimat di titik kata sambung struktural -- supaya
// makna satu bagian kalimat tidak tercampur rata dengan bagian lain yang
// sebenarnya konteksnya berbeda.
func splitClauses(allTokens []string) []clause {
	var clauses []clause
	current := clause{}
	for _, tok := range allTokens {
		if kind, isConnector := clauseConnectors[tok]; isConnector && len(current.tokens) > 0 {
			clauses = append(clauses, current)
			current = clause{connector: kind}
			continue
		}
		current.tokens = append(current.tokens, tok)
	}
	if len(current.tokens) > 0 {
		clauses = append(clauses, current)
	}
	return clauses
}

// splitSentences memecah teks jadi kalimat-kalimat sungguhan di titik/seru/
// tanya -- SEBELUM diproses -- supaya kalimat yang nempel tanpa spasi
// ("...berfikir.horizon adalah...") tidak ikut tergabung jadi satu kata rusak
// atau satu pembelajaran besar yang salah.
func splitSentences(text string) []string {
	replacer := strings.NewReplacer("!", ".", "?", ".")
	normalized := replacer.Replace(text)
	parts := strings.Split(normalized, ".")
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func (l *LearningUnit) Assimilate(text string, weight, baseConfidence float64) {
	l.LastTouched = nil
	for _, sentence := range splitSentences(text) {
		l.assimilateSentence(sentence, weight, baseConfidence)
	}
}

func (l *LearningUnit) assimilateSentence(text string, weight, baseConfidence float64) {
	allTokens := normalize(text)
	if len(allTokens) == 0 {
		return
	}
	clauses := splitClauses(allTokens)

	var prevNodes []*knowledge.ConceptNode
	for _, cl := range clauses {
		nodes := l.assimilateClause(cl.tokens, weight, baseConfidence)
		if cl.connector != "" && len(prevNodes) > 0 && len(nodes) > 0 {
			for _, a := range prevNodes {
				for _, b := range nodes {
					l.Kb.ConnectKind(a, b, cl.connector, weight, 0.75, false)
				}
			}
		}
		if len(nodes) > 0 {
			prevNodes = nodes
		}
	}
}

// assimilateClause mempelajari SATU klausa: stopword dibuang, sisanya saling
// dihubungkan, jenis hubungan dideteksi HANYA dari kata di klausa yang sama
// (bukan dari seluruh kalimat seperti sebelumnya).
func (l *LearningUnit) assimilateClause(clauseTokens []string, weight, baseConfidence float64) []*knowledge.ConceptNode {
	// Setiap kata SELALU jadi node -- tanpa kecuali. Parser boleh MEMAKAI
	// kata (mis. sebagai pemicu jenis relasi), tapi tidak boleh MEMILIKI
	// kata itu (mencegahnya jadi node). Filter stopword di bawah cuma
	// membatasi mana yang otomatis dihubungkan padat ke tetangganya --
	// bukan mencegah identitasnya sebagai node.
	for _, token := range clauseTokens {
		l.Kb.Store(token)
	}

	tokens, positions := filterStopwords(clauseTokens)

	// Kata PERTAMA klausa selalu diikutkan sebagai kandidat relasi, walau
	// dia stopword -- posisi pertama biasanya subjek yang sedang dibahas/
	// diajarkan (mis. "adalah ialah kata..." sedang mengajarkan TENTANG
	// kata "adalah" itu sendiri). Stopword di tengah klausa tetap
	// dikecualikan, supaya kebisingan tidak kembali.
	if len(clauseTokens) > 0 && stopwords[clauseTokens[0]] {
		alreadyIncluded := len(positions) > 0 && positions[0] == 0
		if !alreadyIncluded {
			tokens = append([]string{clauseTokens[0]}, tokens...)
			positions = append([]int{0}, positions...)
		}
	}

	if len(tokens) == 0 {
		return nil
	}
	var nodes []*knowledge.ConceptNode
	for _, token := range tokens {
		nodes = append(nodes, l.Kb.Store(token))
	}
	if l.handleEquivalence(clauseTokens, weight, baseConfidence) {
		return nodes
	}
	for i := 0; i < len(nodes); i++ {
		for j := 0; j < len(nodes); j++ {
			if i == j {
				continue
			}
			if positions[i] > positions[j] {
				continue // cuma proses satu arah: dari kata yang lebih dulu ke yang belakangan
			}
			kind := l.relationKind(clauseTokens, positions[i], positions[j])
			adjacent := abs(positions[i]-positions[j]) == 1
			if kind == knowledge.RelationAssociation && !adjacent {
				continue
			}
			w := weight
			if kind == knowledge.RelationAssociation {
				w = weight * 0.5
			}
			l.Kb.ConnectKind(nodes[i], nodes[j], kind, w, baseConfidence, false)
			if kind == knowledge.RelationEquivalentTo {
				// Kesetaraan itu SIMETRIS -- "A sama dengan B" berarti
				// A≡B dan B≡A, tidak peduli mana yang disebut lebih dulu.
				l.Kb.ConnectKind(nodes[j], nodes[i], kind, w, baseConfidence, false)
			}
			l.LastTouched = append(l.LastTouched, TouchedRelation{SourceID: nodes[i].ID, TargetID: nodes[j].ID, Kind: kind, Inhibitory: false})
		}
	}
	for _, token := range tokens {
		prefixes, suffixes, root := SplitAffixChain(token)
		if len(prefixes) == 0 && len(suffixes) == 0 {
			continue
		}
		wordNode := l.Kb.Store(token)
		rootNode := l.Kb.Store(root)
		l.Kb.ConnectKind(wordNode, rootNode, knowledge.RelationAffix, weight, 0.9, false)
		for _, p := range prefixes {
			pNode := l.Kb.Store(p + "-")
			l.Kb.ConnectKind(wordNode, pNode, knowledge.RelationAffix, weight, 0.9, false)
		}
		for _, s := range suffixes {
			sNode := l.Kb.Store("-" + s)
			l.Kb.ConnectKind(wordNode, sNode, knowledge.RelationAffix, weight, 0.9, false)
		}
	}
	return nodes
}

func (l *LearningUnit) Inhibit(a, b string, weight float64) {
	l.Kb.ConnectKind(l.Kb.Store(a), l.Kb.Store(b), knowledge.RelationAssociation, weight, 0.8, true)
}

func (l *LearningUnit) Optimize(now time.Time) {
	l.Memory.Optimize(now, 30*24*time.Hour, 0.08)
}

func stripPunctuation(s string) string {
	return strings.TrimFunc(s, func(r rune) bool {
		return !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'))
	})
}

func normalize(text string) []string {
	raw := strings.Fields(strings.ToLower(strings.TrimSpace(text)))
	var out []string
	for _, w := range raw {
		w = stripPunctuation(w)
		if w != "" {
			out = append(out, w)
		}
	}
	return out
}

func filterStopwords(tokens []string) (out []string, positions []int) {
	for i, w := range tokens {
		if !stopwords[w] {
			out = append(out, w)
			positions = append(positions, i)
		}
	}
	return out, positions
}

func triggerKind(token string) (knowledge.RelationKind, bool) {
	switch token {
	case "adalah", "merupakan", "yaitu", "ialah", "itu":
		return knowledge.RelationIsA, true
	case "memiliki", "punya":
		return knowledge.RelationHas, true
	case "karena", "menyebabkan", "sebab":
		return knowledge.RelationCause, true
	case "untuk", "fungsi", "berguna":
		return knowledge.RelationFunction, true
	case "bagian":
		return knowledge.RelationPartWhole, true
	case "di", "dari", "lokasi":
		return knowledge.RelationLocation, true
	case "sebelum", "sesudah", "ketika", "saat":
		return knowledge.RelationTime, true
	case "bisa", "dapat", "mampu":
		return knowledge.RelationCanDo, true
	}
	return "", false
}

// relationKind cuma memberi relasi SPESIFIK (IsA, Cause, dst) kalau posB
// adalah kata konten PERTAMA persis setelah kata pemicu -- mencegah
// "manusia adalah makhluk hidup" salah menghubungkan "manusia" ke "hidup"
// juga (padahal "hidup" itu sifat "makhluk", bukan objek langsung "adalah").
// triggerKindOrEquivalent mengecek pemicu bawaan (bootstrap) DULU. Kalau
// tidak ketemu, cek apakah kata ini punya relasi EquivalentTo ke kata lain
// yang SUDAH dikenal sebagai pemicu -- ini yang membuat bootstrap bisa
// "dikalahkan"/diperluas oleh pengetahuan yang diajarkan, tanpa programmer
// menambah case baru di kode.
func (l *LearningUnit) triggerKindOrEquivalent(token string) (knowledge.RelationKind, bool) {
	if kind, ok := triggerKind(token); ok {
		return kind, true
	}
	node := l.Kb.Fetch(token)
	if node == nil {
		return "", false
	}
	for _, s := range node.OutboundAll() {
			id := s.TargetID
		if s.Kind != knowledge.RelationEquivalentTo {
			continue
		}
		if target := l.Kb.Registry.GetByID(id); target != nil {
			if kind, ok := triggerKind(target.Token); ok {
				return kind, true
			}
		}
	}
	return "", false
}

// relationKind cuma memberi relasi SPESIFIK (IsA, Cause, dst) kalau posB
// adalah kata konten PERTAMA persis setelah kata pemicu.
func (l *LearningUnit) relationKind(allTokens []string, posA, posB int) knowledge.RelationKind {
	lo, hi := posA, posB
	if lo > hi {
		lo, hi = hi, lo
	}
	between := allTokens[lo+1 : hi]
	for k, token := range between {
		triggerPos := lo + 1 + k
		kind, ok := l.triggerKindOrEquivalent(token)
		if !ok {
			continue
		}
		// "sama dengan" dihitung sebagai satu frasa pemicu -- geser posisi
		// efektifnya ke kata "dengan" supaya pengecekan tetangga-langsung
		// tetap benar.
		if token == "sama" && triggerPos+1 < len(allTokens) && allTokens[triggerPos+1] == "dengan" {
			triggerPos++
		}
		if hi == triggerPos+1 {
			return kind
		}
		return knowledge.RelationAssociation
	}
	return knowledge.RelationAssociation
}
func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
// IsStopword diekspor supaya package lain (core) bisa mengecek kata konten
// vs kata sambung tanpa menduplikasi daftarnya sendiri.
func IsStopword(token string) bool {
	return stopwords[token]
}
// handleEquivalence mendeteksi "...sama dengan..." dalam satu klausa. Kalau
// kedua sisinya cuma satu kata, tetap dihubungkan seperti sinaps biasa
// (EquivalentTo dua arah). Kalau salah satu sisi berupa FRASA (>1 kata),
// itu berarti kombinasi kata-kata itu -- bukan salah satu katanya sendiri
// -- yang bermakna setara, jadi disimpan sebagai PatternSynapse: seluruh
// anggota frasa itu harus aktif BERSAMAAN baru cocok, sesuai prinsip
// "node yang aktif bersamaan", bukan kata yang kebetulan berdekatan.
func (l *LearningUnit) handleEquivalence(clauseTokens []string, weight, baseConfidence float64) bool {
	idx := -1
	for i := 0; i+1 < len(clauseTokens); i++ {
		if clauseTokens[i] == "sama" && clauseTokens[i+1] == "dengan" {
			idx = i
			break
		}
	}
	if idx == -1 {
		return false
	}
	leftTokens := clauseTokens[:idx]
	rightTokens := clauseTokens[idx+2:]
	if len(leftTokens) == 0 || len(rightTokens) == 0 {
		return false
	}

	leftIDs := make([]knowledge.NodeID, 0, len(leftTokens))
	for _, t := range leftTokens {
		leftIDs = append(leftIDs, l.Kb.Store(t).ID)
	}
	rightIDs := make([]knowledge.NodeID, 0, len(rightTokens))
	for _, t := range rightTokens {
		rightIDs = append(rightIDs, l.Kb.Store(t).ID)
	}

	if len(leftIDs) == 1 && len(rightIDs) == 1 {
		a := l.Kb.Registry.GetByID(leftIDs[0])
		b := l.Kb.Registry.GetByID(rightIDs[0])
		l.Kb.ConnectKind(a, b, knowledge.RelationEquivalentTo, weight, baseConfidence, false)
		l.Kb.ConnectKind(b, a, knowledge.RelationEquivalentTo, weight, baseConfidence, false)
		return true
	}

	if len(leftIDs) > 1 {
		l.Kb.Patterns.Learn(leftIDs, rightIDs[0], weight, baseConfidence)
	}
	if len(rightIDs) > 1 {
		l.Kb.Patterns.Learn(rightIDs, leftIDs[0], weight, baseConfidence)
	}
	return true
}
