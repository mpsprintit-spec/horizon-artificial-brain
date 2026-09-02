package websearch

import (
	"fmt"
	"strings"
)

// ============================================================================
// Show Wikipedia Result
// ============================================================================

func ShowWikipediaResult(result *WikipediaResult) {

	fmt.Println()
	fmt.Println("========================================================")
	fmt.Println("               HORIZON WEBSEARCH RESULT")
	fmt.Println("========================================================")
	fmt.Println()

	fmt.Println("Word")
	fmt.Println("--------------------------------------------------------")
	fmt.Println(result.Word)
	fmt.Println()

	fmt.Println("Title")
	fmt.Println("--------------------------------------------------------")
	fmt.Println(result.Title)
	fmt.Println()

	fmt.Println("Description")
	fmt.Println("--------------------------------------------------------")
	fmt.Println(result.Description)
	fmt.Println()

	fmt.Println("Summary")
	fmt.Println("--------------------------------------------------------")
	fmt.Println(result.Summary)
	fmt.Println()

	fmt.Println("Source")
	fmt.Println("--------------------------------------------------------")
	fmt.Println("Wikipedia Indonesia")
	fmt.Println(result.URL)
	fmt.Println()

	fmt.Println("========================================================")
}

// ============================================================================
// Build Preview Dictionary
// ============================================================================

func BuildDictionaryPreview(result *WikipediaResult) string {

	var builder strings.Builder

	builder.WriteString("\n")
	builder.WriteString("========================================================\n")
	builder.WriteString("HORIZON DICTIONARY PREVIEW\n")
	builder.WriteString("========================================================\n\n")

	builder.WriteString("Lemma\n")
	builder.WriteString("----------------------------------------\n")
	builder.WriteString(result.Word)
	builder.WriteString("\n\n")

	builder.WriteString("Meaning\n")
	builder.WriteString("----------------------------------------\n")
	builder.WriteString(result.Summary)
	builder.WriteString("\n\n")

	builder.WriteString("Source\n")
	builder.WriteString("----------------------------------------\n")
	builder.WriteString(result.URL)
	builder.WriteString("\n\n")

	builder.WriteString("Status\n")
	builder.WriteString("----------------------------------------\n")
	builder.WriteString("READY TO SAVE\n")

	return builder.String()

}

// ============================================================================
// Show Dictionary Preview
// ============================================================================

func ShowDictionaryPreview(result *WikipediaResult) {

	fmt.Println(BuildDictionaryPreview(result))

}

// ============================================================================
// Show Learning Finished
// ============================================================================

func ShowLearningFinished() {

	fmt.Println()
	fmt.Println("========================================================")
	fmt.Println("Knowledge collection completed.")
	fmt.Println("Waiting for user confirmation.")
	fmt.Println("========================================================")
	fmt.Println()

}
