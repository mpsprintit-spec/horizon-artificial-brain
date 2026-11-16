package runtime

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func assertNoRetiredCognitiveAPI(t *testing.T, rt *BrainRuntime) {
	t.Helper()
	for _, name := range []string{"DNF", "ApplyCognitiveModel", "PreviewCognitiveModel", "CognitiveModelParameters", "CognitiveBMPSWeights", "PreviewCoordinatedV5", "PreviewPhase2V5", "PreviewPhase3DiagnosticV5", "PreviewPhase4V5", "PreviewPhase4DiagnosticV5", "PreviewPhase4TerminalFeaturesV5"} {
		if _, exists := reflect.TypeOf(rt).MethodByName(name); exists {
			t.Fatalf("runtime still exports retired API %s", name)
		}
	}
	if _, exists := reflect.TypeOf(knowledge.BrainState{}).FieldByName("CognitiveModel"); exists {
		t.Fatal("BrainState still stores a retired controller")
	}
}

func TestRetiredCognitiveAPIsAreAbsent(t *testing.T) {
	assertNoRetiredCognitiveAPI(t, NewBrainRuntime(nil))
}

func TestRetiredCognitiveProductionDependencyClosure(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	retired := []string{"thinking", "decision", "language", "context", "understanding", "hfcc"}
	for _, name := range retired {
		matches, err := filepath.Glob(filepath.Join(root, "services", "ai", name, "*.go"))
		if err != nil || len(matches) != 0 {
			t.Fatalf("retired package %s still has source: %v %v", name, matches, err)
		}
	}
	for _, path := range []string{"cmd/horizon-metareasoning-v5/main.go", "services/ai/core/classify.go", "services/ai/core/legacy_mode.go", "services/ai/core/task.go", "services/ai/cluster.go"} {
		if _, err := os.Stat(filepath.Join(root, path)); !os.IsNotExist(err) {
			t.Fatalf("retired implementation still present at %s: %v", path, err)
		}
	}
	// This query checks imports, not VCS stamping of main executables. Tests
	// may run from a checkout without accessible Git administrative metadata.
	cmd := exec.Command("go", "list", "-buildvcs=false", "-deps", "./services/ai", "./services/ai/cmd/horizon-cli", "./cmd/horizon-chat", "./cmd/horizon-dialogue", "./cmd/horizon-hnl")
	cmd.Dir = root
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("production dependency graph: %v\n%s", err, output)
	}
	for _, dependency := range strings.Fields(string(output)) {
		for _, name := range retired {
			prefix := "github.com/project-horizon/horizon-core/services/ai/" + name
			if dependency == prefix || strings.HasPrefix(dependency, prefix+"/") {
				t.Fatalf("production still imports retired controller package %s", dependency)
			}
		}
	}
}
