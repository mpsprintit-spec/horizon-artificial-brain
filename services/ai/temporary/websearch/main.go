package websearch

import "fmt"

// ============================================================================
// Horizon Learning
// ============================================================================

func LearnWord(word string) error {

	fmt.Println()
	fmt.Println("========================================================")
	fmt.Println("HORIZON KNOWLEDGE LEARNING")
	fmt.Println("========================================================")
	fmt.Println()

	// ======================================================
	// Check Local Knowledge
	// ======================================================

	if CheckKnowledge(word) {

		fmt.Println("Knowledge already exists.")
		fmt.Println()

		return nil

	}

	// ======================================================
	// Ask Search Permission
	// ======================================================

	if !AskSearchPermission(word) {

		fmt.Println()
		fmt.Println("Search cancelled by user.")
		return nil

	}

	// ======================================================
	// Internet
	// ======================================================

	if !CheckInternet() {

		return fmt.Errorf("no internet connection")

	}

	// ======================================================
	// Search Wikipedia
	// ======================================================

	body, err := SearchWikipedia(word)

	if err != nil {

		return err

	}

	// ======================================================
	// Parse Wikipedia
	// ======================================================

	result, err := ParseWikipedia(word, body)

	if err != nil {

		return err

	}

	// ======================================================
	// Validate Result
	// ======================================================

	if err := ValidateWikipedia(result); err != nil {

		return err

	}

	// ======================================================
	// Show Result
	// ======================================================

	ShowWikipediaResult(result)

	// ======================================================
	// User Validation
	// ======================================================

	if !AskValidation() {

		fmt.Println()
		fmt.Println("Information rejected by user.")
		return nil

	}

	// ======================================================
	// Dictionary Preview
	// ======================================================

	ShowDictionaryPreview(result)

	// ======================================================
	// Save Permission
	// ======================================================

	if !AskSaveKnowledge() {

		fmt.Println()
		fmt.Println("Knowledge not saved.")
		return nil

	}

	// ======================================================
	// Save
	// ======================================================

	if err := SaveKnowledge(result); err != nil {

		return err

	}

	// ======================================================
	// Finish
	// ======================================================

	ShowLearningFinished()

	return nil

}
