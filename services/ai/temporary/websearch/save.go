package websearch

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/learning"
)

// SaveKnowledge converts approved transient text into neural memory and discards
// the source sentences instead of writing dictionary definitions.
func SaveKnowledge(result *WikipediaResult) error {
	fmt.Println()
	fmt.Println("========================================================")
	fmt.Println("HORIZON SAVE NEURAL MEMORY")
	fmt.Println("========================================================")
	fmt.Println()

	memory := knowledge.NewKnowledgeBase()
	learner := learning.NewLearningUnit(memory)
	learner.Assimilate(result.Word, 0.9, 0.7)
	for _, text := range []string{result.Title, result.Description, result.Summary} {
		for _, sentence := range strings.FieldsFunc(text, func(r rune) bool { return r == '.' || r == '!' || r == '?' || r == '\n' || r == '\r' }) {
			learner.Assimilate(sentence, 0.6, 0.7)
		}
	}

	filename := filepath.Join("services", "ai", "brain_memory.json")
	fmt.Println("File      : ", filename)
	fmt.Println("Writing neural graph file...")
	if err := memory.Save(filename); err != nil {
		return err
	}
	fmt.Println("Neural memory successfully saved.")
	return nil
}
