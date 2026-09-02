package core

import (
	"strings"

)
// Intent adalah tujuan komunikasi di balik sebuah kalimat -- ini yang
// menentukan jalur kerja seluruh otak, bukan sekadar label. Decision tidak
// lagi menebak sendiri fokus pembicaraan; Intent yang menentukannya.
type Intent string

const (
	IntentConfirm      Intent = "confirm"      // "ya, benar"
	IntentRetraction   Intent = "retraction"   // "saya salah"
	IntentConfirmation Intent = "confirmation" // "apakah X Y"
	IntentDefinition   Intent = "definition"   // "apa itu X"
	IntentRecall       Intent = "recall"       // "sebut X", "ceritakan X"
	IntentQuestion     Intent = "question"     // pertanyaan umum lain
	IntentTeaching     Intent = "teaching"     // pernyataan biasa -- default
)

func classifyIntent(input string) Intent {
	switch {
	case isConfirmIntent(input):
		return IntentConfirm
	case isRetractionIntent(input):
		return IntentRetraction
	case isConfirmationQuery(input):
		return IntentConfirmation
	case isDefinitionQuery(input):
		return IntentDefinition
	case isRecallQuery(input):
		return IntentRecall
	case isQuestion(input):
		return IntentQuestion
	default:
		return IntentTeaching
	}
}

func isQuestion(input string) bool {
	s := strings.ToLower(strings.TrimSpace(input))
	return strings.HasSuffix(s, "?") ||
		strings.HasPrefix(s, "apa ") ||
		strings.HasPrefix(s, "apakah ") ||
		strings.HasPrefix(s, "siapa ") ||
		strings.HasPrefix(s, "mengapa ") ||
		strings.HasPrefix(s, "kenapa ") ||
		strings.HasPrefix(s, "bagaimana ")
}

// isRecallQuery HANYA untuk permintaan mengingat/menjelaskan eksplisit --
// "apa itu"/"siapa itu" sengaja TIDAK di sini lagi karena itu sudah jadi
// urusan isDefinitionQuery, dicek lebih dulu supaya tidak tabrakan.
func isRecallQuery(input string) bool {
	s := " " + strings.ToLower(strings.TrimSpace(input)) + " "
	triggers := []string{"sebut", "jelaskan", "ceritakan", "definisikan", "artinya", "maksud"}
	for _, w := range triggers {
		if strings.Contains(s, " "+w) {
			return true
		}
	}
	return false
}

func isDefinitionQuery(input string) bool {
	s := " " + strings.ToLower(strings.TrimSpace(input)) + " "
	triggers := []string{"apa itu", "siapa itu"}
	for _, w := range triggers {
		if strings.Contains(s, " "+w) {
			return true
		}
	}
	return false
}

func extractDefinitionTarget(input string) string {
	s := strings.ToLower(strings.TrimSpace(input))
	for _, prefix := range []string{"apa itu ", "siapa itu "} {
		if strings.HasPrefix(s, prefix) {
			return strings.TrimSpace(s[len(prefix):])
		}
	}
	return ""
}

// extractTeachingFocus mengambil kata konten PERTAMA dalam kalimat --
// itulah subjek yang sedang diajarkan user ("kucing adalah hewan" -> fokus
// = kucing), bukan node dengan aktivasi tertinggi di seluruh graph.
// extractTeachingFocus mengambil kata konten PERTAMA dalam kalimat -- tanpa
// melompati stopword. Supaya kalimat yang justru mengajarkan makna kata
// fungsi sendiri ("adalah ialah kata yang...") tetap benar fokusnya ke
// "adalah", bukan tergeser ke kata berikutnya.
func extractTeachingFocus(prompt string) string {
	for _, w := range strings.Fields(strings.ToLower(prompt)) {
		w = strings.TrimFunc(w, func(r rune) bool {
			return !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'))
		})
		if w == "" {
			continue
		}
		return w
	}
	return ""
}
func isConfirmIntent(input string) bool {
	s := normalizeForIntent(input)
	triggers := []string{"ya benar", "benar", "ingat itu", "betul"}
	for _, t := range triggers {
		if s == t {
			return true
		}
	}
	return false
}
func isRetractionIntent(input string) bool {
	s := normalizeForIntent(input)
	triggers := []string{"tidak saya salah", "saya salah", "lupakan itu", "saya bercanda", "bukan begitu"}
	for _, t := range triggers {
		if s == t {
			return true
		}
	}
	return false
}
func isConfirmationQuery(input string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(input)), "apakah ")
}

func extractConfirmationParts(input string) (subject string, claimWords []string) {
	s := strings.ToLower(strings.TrimSpace(input))
	s = strings.TrimPrefix(s, "apakah ")
	words := strings.Fields(s)
	if len(words) == 0 {
		return "", nil
	}
	return words[0], words[1:]
}
// normalizeForIntent membersihkan tanda baca ringan supaya perbandingan
// persis ("apakah kalimat ini SAMA DENGAN ucapan pendek X") tidak gagal
// cuma karena ada koma/titik.
func normalizeForIntent(input string) string {
	s := strings.ToLower(strings.TrimSpace(input))
	s = strings.Trim(s, ".,!?")
	s = strings.ReplaceAll(s, ",", "")
	return strings.Join(strings.Fields(s), " ")
}
