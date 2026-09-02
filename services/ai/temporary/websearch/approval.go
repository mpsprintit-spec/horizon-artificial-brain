package websearch

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

var input = bufio.NewReader(os.Stdin)

// ============================================================================
// Ask Search Permission
// ============================================================================

func AskSearchPermission(word string) bool {

	fmt.Println()
	fmt.Println("========================================================")
	fmt.Println("HORIZON WEBSEARCH")
	fmt.Println("========================================================")
	fmt.Println()

	fmt.Printf("Knowledge \"%s\" belum tersedia.\n\n", word)

	fmt.Println("Apakah Horizon boleh mencari informasi ini")
	fmt.Println("di Wikipedia Indonesia?")
	fmt.Println()

	fmt.Print("[Y] Ya  [N] Tidak : ")

	answer, _ := input.ReadString('\n')
	answer = strings.TrimSpace(strings.ToUpper(answer))

	return answer == AnswerYes

}

// ============================================================================
// Ask Validation
// ============================================================================

func AskValidation() bool {

	fmt.Println()
	fmt.Println("========================================================")
	fmt.Println("VALIDASI INFORMASI")
	fmt.Println("========================================================")
	fmt.Println()

	fmt.Println("Silakan baca hasil pencarian di atas.")
	fmt.Println("Jika informasi sudah benar pilih [Y].")
	fmt.Println("Jika menurut Anda salah pilih [N].")
	fmt.Println()

	fmt.Print("[Y] Valid  [N] Tolak : ")

	answer, _ := input.ReadString('\n')
	answer = strings.TrimSpace(strings.ToUpper(answer))

	return answer == AnswerYes

}

// ============================================================================
// Ask Next Word
// ============================================================================

func AskNextWord(word string) bool {

	fmt.Println()
	fmt.Println("========================================================")
	fmt.Println("KATA BARU DITEMUKAN")
	fmt.Println("========================================================")
	fmt.Println()

	fmt.Printf("Kata : %s\n\n", word)

	fmt.Println("Kata ini belum ada di Horizon Knowledge.")
	fmt.Println("Apakah Horizon boleh mencarinya?")
	fmt.Println()

	fmt.Print("[Y] Ya  [N] Tidak : ")

	answer, _ := input.ReadString('\n')
	answer = strings.TrimSpace(strings.ToUpper(answer))

	return answer == AnswerYes

}

// ============================================================================
// Ask Save
// ============================================================================

func AskSaveKnowledge() bool {

	fmt.Println()
	fmt.Println("========================================================")
	fmt.Println("SIMPAN KNOWLEDGE")
	fmt.Println("========================================================")
	fmt.Println()

	fmt.Println("Dictionary Horizon sudah selesai dibuat.")
	fmt.Println("Apakah ingin menyimpannya?")
	fmt.Println()

	fmt.Print("[Y] Simpan  [N] Batal : ")

	answer, _ := input.ReadString('\n')
	answer = strings.TrimSpace(strings.ToUpper(answer))

	return answer == AnswerYes

}

// ============================================================================
// Ask Continue Learning
// ============================================================================

func AskContinueLearning() bool {

	fmt.Println()
	fmt.Println("========================================================")
	fmt.Println("LANJUTKAN PEMBELAJARAN")
	fmt.Println("========================================================")
	fmt.Println()

	fmt.Print("[Y] Lanjut  [N] Selesai : ")

	answer, _ := input.ReadString('\n')
	answer = strings.TrimSpace(strings.ToUpper(answer))

	return answer == AnswerYes

}
