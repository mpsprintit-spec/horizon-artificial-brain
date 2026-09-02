package language

import (
	"fmt"
	"strings"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/thinking"
)

type Engine struct {
	Kb *knowledge.KnowledgeBase
}

func NewEngine(kb *knowledge.KnowledgeBase) *Engine {
	return &Engine{Kb: kb}
}

// sentencePattern mengubah subjek + jenis relasi + objek jadi kalimat Indonesia.
func sentencePattern(subject string, kind knowledge.RelationKind, object string) string {
	subject = capitalize(subject)
	switch kind {
	case knowledge.RelationIsA:
		return fmt.Sprintf("%s adalah %s", subject, object)
	case knowledge.RelationHas:
		return fmt.Sprintf("%s punya %s", subject, object)
	case knowledge.RelationCause:
		return fmt.Sprintf("%s menyebabkan %s", subject, object)
	case knowledge.RelationFunction:
		return fmt.Sprintf("%s berfungsi untuk %s", subject, object)
	case knowledge.RelationPartWhole:
		return fmt.Sprintf("%s adalah bagian dari %s", subject, object)
	case knowledge.RelationLocation:
		return fmt.Sprintf("%s berada di %s", subject, object)
	case knowledge.RelationTime:
		return fmt.Sprintf("%s terjadi saat %s", subject, object)
	case knowledge.RelationContrast:
		return fmt.Sprintf("%s, tapi %s", subject, object)
	case knowledge.RelationCondition:
		return fmt.Sprintf("Kalau %s, maka %s", strings.ToLower(subject), object)
	case knowledge.RelationSequence:
		return fmt.Sprintf("%s, lalu %s", subject, object)
	case knowledge.RelationCanDo:
		return fmt.Sprintf("%s bisa %s", subject, object)
	default:
		return fmt.Sprintf("%s berkaitan dengan %s", subject, object)
	}
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// hedge membungkus kalimat inti dengan tingkat keyakinan dalam bentuk kata,
// bukan angka -- dan memberi jawaban jujur kalau memang belum tahu apa-apa.
func hedge(core string, confidence float64, needsWebSearch bool) string {
	core = strings.TrimSpace(core)
	if core == "" {
		return "Saya belum cukup tahu untuk menjawab itu."
	}
	if !strings.HasSuffix(core, ".") {
		core += "."
	}
	switch {
	case confidence >= 0.8:
		return core
	case confidence >= 0.5:
		return "Kemungkinan besar " + strings.ToLower(core[:1]) + core[1:]
	case confidence >= 0.2:
		msg := "Sepertinya " + strings.ToLower(core[:1]) + core[1:]
		if needsWebSearch {
			msg += " Saya belum sepenuhnya yakin, masih perlu mencari tahu lebih lanjut."
		}
		return msg
	default:
		return "Saya belum cukup tahu untuk menjawab itu."
	}
}

// Generate menyusun kalimat dari SATU keputusan (hasil pilihan Decision),
// menelusuri relasi asli di Knowledge -- bukan lagi template tetap.
func (e *Engine) Generate(best thinking.Hypothesis, confidence float64, needsWebSearch bool, inferences []thinking.InferredFact, preferIsA bool) string {
	if len(best.Nodes) == 0 {
		return hedge("", confidence, needsWebSearch)
	}
	subject := e.Kb.Registry.GetByID(best.Nodes[0])
	if subject == nil {
		return hedge("", confidence, needsWebSearch)
	}

	if preferIsA {
		for _, s := range subject.OutboundAll() {
			id := s.TargetID
			if s.Kind == knowledge.RelationIsA && !s.Inhibitory {
				if object := e.Kb.Registry.GetByID(id); object != nil {
					return hedge(sentencePattern(subject.Token, knowledge.RelationIsA, object.Token), s.Confidence, needsWebSearch)
				}
			}
		}
	}

	var bestObjectID knowledge.NodeID
	var bestKind knowledge.RelationKind
	var bestRelConfidence float64
	bestScore := -1.0
	for _, s := range subject.OutboundAll() {
			id := s.TargetID
		if s.Inhibitory || s.Kind == knowledge.RelationAffix {
			continue
		}
		score := s.Weight * s.Confidence
		if s.Kind != knowledge.RelationAssociation {
			score += 0.5
		}
		if score > bestScore {
			bestScore, bestKind, bestObjectID, bestRelConfidence = score, s.Kind, id, s.Confidence
		}
	}

	if bestScore >= 0 {
		if object := e.Kb.Registry.GetByID(bestObjectID); object != nil {
			return hedge(sentencePattern(subject.Token, bestKind, object.Token), bestRelConfidence, needsWebSearch)
		}
	}

	// Tidak ada relasi langsung -- coba pakai hasil simpulan tidak langsung
	// (rantai IS_A), tapi ditandai TEGAS sebagai simpulan, bukan fakta yang
	// langsung diajarkan -- supaya Horizon tidak mengaku tahu lebih dari
	// yang sebenarnya dia alami.
	var bestInference *thinking.InferredFact
	for i := range inferences {
		inf := &inferences[i]
		if inf.Confidence < 0.15 {
			continue
		}
		if bestInference == nil || inf.Confidence > bestInference.Confidence {
			bestInference = inf
		}
	}
	if bestInference != nil {
		if object := e.Kb.Registry.GetByID(bestInference.TargetID); object != nil {
			core := sentencePattern(subject.Token, bestInference.Kind, object.Token)
			return hedge(core, bestInference.Confidence, false) + " (disimpulkan, bukan diajarkan langsung)"
		}
	}

	return hedge(subject.Token, confidence, needsWebSearch)
}
// Confirm menjawab pertanyaan ya/tidak -- beda dari Generate yang menyusun
// kalimat pernyataan biasa.
func (e *Engine) Confirm(subjectToken, claim string, found bool, kind knowledge.RelationKind, viaInference bool) string {
	if !found {
		return fmt.Sprintf("Saya belum menemukan hubungan antara %s dan %s.", subjectToken, claim)
	}
	body := sentencePattern(subjectToken, kind, claim)
	if viaInference {
		return "Kemungkinan besar ya — " + body + " (disimpulkan, bukan diajarkan langsung)."
	}
	return "Ya — " + body + "."
}

// Realize performs surface realization of an internal understanding state (I*).
// It does not perform cognitive selection: it only renders the relations that
// already belong to the interpretation chosen by Reasoning/Decision.

// Realize is pure surface realization of I*. Cognitive decisions are finished.
// Presentation prioritizes typed structure about the emergent focus; association
// noise on peripheral nodes is deprioritized for length limits only.
func (e *Engine) Realize(interp *thinking.Interpretation, confidence float64, needsWebSearch bool) string {
	if interp == nil || len(interp.Nodes) == 0 {
		return hedge("", confidence, needsWebSearch)
	}
	focus := e.Kb.Registry.GetByID(interp.FocusID)
	if focus == nil {
		return hedge("", confidence, needsWebSearch)
	}

	type item struct {
		subj, obj string
		kind      knowledge.RelationKind
		priority  int
	}
	isTyped := func(k knowledge.RelationKind) bool {
		return k != knowledge.RelationAssociation && k != knowledge.RelationAffix
	}
	focusNeighbor := map[knowledge.NodeID]bool{interp.FocusID: true}
	for _, r := range interp.Relations {
		if r.Inhibitory || !isTyped(r.Kind) {
			continue
		}
		if r.SourceID == interp.FocusID {
			focusNeighbor[r.TargetID] = true
		}
		if r.TargetID == interp.FocusID {
			focusNeighbor[r.SourceID] = true
		}
	}

	var items []item
	used := map[string]bool{}
	for _, r := range interp.Relations {
		if r.Inhibitory {
			continue
		}
		subjN := e.Kb.Registry.GetByID(r.SourceID)
		objN := e.Kb.Registry.GetByID(r.TargetID)
		if subjN == nil || objN == nil {
			continue
		}
		key := subjN.Token + "|" + string(r.Kind) + "|" + objN.Token
		if used[key] {
			continue
		}
		used[key] = true
		typed := isTyped(r.Kind)
		fromFocus := r.SourceID == interp.FocusID
		near := focusNeighbor[r.SourceID] || focusNeighbor[r.TargetID]
		pri := 0
		switch {
		case fromFocus && typed:
			pri = 5
		case near && typed:
			pri = 4
		case fromFocus:
			pri = 2
		case near:
			pri = 1
		default:
			pri = 0
		}
		items = append(items, item{subjN.Token, objN.Token, r.Kind, pri})
	}
	if len(items) == 0 {
		return hedge(focus.Token, confidence, needsWebSearch)
	}
	for i := 0; i < len(items); i++ {
		for j := i + 1; j < len(items); j++ {
			if items[j].priority > items[i].priority {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
	var filtered []item
	for _, it := range items {
		if it.priority >= 4 {
			filtered = append(filtered, it)
		}
	}
	if len(filtered) == 0 {
		for _, it := range items {
			if it.priority > 0 {
				filtered = append(filtered, it)
			}
		}
	}
	if len(filtered) > 0 {
		items = filtered
	}
	const maxParts = 6
	if len(items) > maxParts {
		items = items[:maxParts]
	}
	var parts []string
	for _, it := range items {
		parts = append(parts, sentencePattern(it.subj, it.kind, it.obj))
	}
	return hedge(strings.Join(parts, " "), confidence, needsWebSearch)
}


// RealizeEvaluation realizes Decision outcome from I* (FSU Phase 2).
// No knowledge search; no cognitive selection.
func (e *Engine) RealizeEvaluation(interp *thinking.Interpretation, confidence float64, needsWebSearch bool) string {
	if interp == nil {
		return hedge("", confidence, needsWebSearch)
	}
	status := interp.EvalStatus
	// Identity: Language realizes the same requested proposition Decision evaluated.
	prop := interp.RequestedProposition
	if prop.TargetID == 0 && len(interp.Propositions) > 0 {
		for _, pr := range interp.Propositions {
			if pr.Requested {
				prop = pr
				break
			}
		}
		if prop.TargetID == 0 {
			prop = interp.Propositions[0]
		}
	}
	focus := e.Kb.Registry.GetByID(interp.FocusID)
	focusTok := ""
	if focus != nil {
		focusTok = focus.Token
	}
	switch status {
	case thinking.EvalSupported:
		if prop.TargetTok != "" && prop.ObjectTok != "" {
			return hedge(sentencePattern(prop.TargetTok, prop.Relation, prop.ObjectTok), confidence, needsWebSearch)
		}
		return e.Realize(interp, confidence, needsWebSearch)
	case thinking.EvalConflicted:
		msg := "Hubungan yang ditanyakan bentrok dengan batasan/constraint yang diketahui"
		if prop.TargetTok != "" {
			msg = "Tidak dapat menyimpulkan bahwa " + prop.TargetTok + " " + string(prop.Relation) + " " + prop.ObjectTok + " di bawah constraint yang diberikan"
		}
		if len(interp.Constraints) > 0 {
			msg += " (constraint: " + interp.Constraints[0].Token + ")"
		}
		return hedge(msg, confidence, needsWebSearch)
	case thinking.EvalUnsupported, thinking.EvalUnknown:
		if prop.TargetTok != "" {
			return hedge("Saya belum menemukan bukti untuk "+prop.TargetTok+" "+string(prop.Relation)+" "+prop.ObjectTok, confidence, needsWebSearch)
		}
		return hedge("Saya belum menemukan bukti untuk klaim yang diminta", confidence, needsWebSearch)
	case thinking.EvalContradicted:
		if prop.TargetTok != "" {
			return hedge("Bukti menentang bahwa "+prop.TargetTok+" "+string(prop.Relation)+" "+prop.ObjectTok, confidence, needsWebSearch)
		}
		return hedge("Bukti menentang klaim yang diminta", confidence, needsWebSearch)
	default:
		if focusTok != "" {
			return e.Realize(interp, confidence, needsWebSearch)
		}
		return hedge("", confidence, needsWebSearch)
	}
}
