package learning

import (
	_ "embed"
	"strings"
)

//go:embed data/kata-dasar.txt
var rawDictionary string

var rootDictionary map[string]bool

func init() {
	rootDictionary = make(map[string]bool)
	for _, w := range strings.Split(rawDictionary, "\n") {
		w = strings.TrimSpace(w)
		if w != "" {
			rootDictionary[w] = true
		}
	}
}

// isKnownRoot mengecek apakah sebuah kata BENAR-BENAR ada di kamus kata dasar asli.
func isKnownRoot(word string) bool {
	return rootDictionary[word]
}

var knownPrefixes = []string{"menge", "meng", "meny", "mem", "men", "me", "ber", "ter", "per", "pe", "se", "ke", "di", "si"}
var knownSuffixes = []string{"kan", "lah", "kah", "nya", "an", "i"}

func isVowel(b byte) bool {
	switch b {
	case 'a', 'e', 'i', 'o', 'u':
		return true
	}
	return false
}

func restoreNasal(prefix, remainder string) string {
	if remainder == "" {
		return remainder
	}
	switch prefix {
	case "meny":
		return "s" + remainder
	case "mem":
		if isVowel(remainder[0]) {
			return "p" + remainder
		}
	case "men":
		if isVowel(remainder[0]) {
			return "t" + remainder
		}
	}
	return remainder
}

func splitOneLayer(word string) (prefix, root, suffix string) {
	root = word
	for _, p := range knownPrefixes {
		if strings.HasPrefix(root, p) && len(root)-len(p) >= 3 {
			prefix = p
			stripped := root[len(p):]
			root = restoreNasal(p, stripped)
			break
		}
	}
	for _, s := range knownSuffixes {
		if strings.HasSuffix(root, s) && len(root)-len(s) >= 3 {
			suffix = s
			root = root[:len(root)-len(s)]
			break
		}
	}
	return prefix, root, suffix
}

// SplitAffixChain mengupas imbuhan berlapis, TAPI hasil akhirnya WAJIB
// merupakan kata yang benar-benar ada di kamus. Kalau tidak, kita batalkan
// semua pemotongan dan kembalikan kata utuh apa adanya — ini yang mencegah
// kasus seperti "sebab"->"bab", "hewan"->"hew", "perintah"->"intah".
func SplitAffixChain(word string) (prefixes []string, suffixes []string, root string) {
	if isKnownRoot(word) {
		return nil, nil, word // sudah kata dasar asli, tidak perlu dipotong
	}
	current := word
	var p, s []string
	for i := 0; i < 4; i++ {
		if isKnownRoot(current) {
			break
		}
		pref, stripped, suf := splitOneLayer(current)
		if pref == "" && suf == "" {
			break
		}
		if pref != "" {
			p = append(p, pref)
		}
		if suf != "" {
			s = append(s, suf)
		}
		current = stripped
	}
	if !isKnownRoot(current) {
		// hasil akhirnya bukan kata dasar yang dikenal -> jangan percaya
		// hasil potongan ini, kembalikan kata aslinya utuh.
		return nil, nil, word
	}
	return p, s, current
}
