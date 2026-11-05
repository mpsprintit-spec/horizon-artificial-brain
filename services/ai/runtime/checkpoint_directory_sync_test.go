package runtime

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	k "github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func TestCheckpointDirectoryCommit(t *testing.T) {
	rt := NewBrainRuntime(k.NewBrain())
	dir := t.TempDir()
	name := filepath.Join(dir, "saved.json")
	for i := 0; i < 2; i++ {
		if err := rt.Checkpoint(name); err != nil {
			t.Fatal(err)
		}
		fresh := NewBrainRuntime(k.NewBrain())
		if err := fresh.RestoreCheckpoint(name); err != nil {
			t.Fatal(err)
		}
		if fresh.LastSequence() != rt.LastSequence() {
			t.Fatal("checkpoint sequence changed across replacement")
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 1 || entries[0].Name() != "saved.json" {
			t.Fatal("checkpoint replacement left a temporary file")
		}
	}
	t.Log("checkpoint directory", dir)
}

func TestCheckpointDirectoryCommitOrderAndErrors(t *testing.T) {
	for _, fault := range []bool{false, true} {
		t.Run(fmt.Sprint(fault), func(t *testing.T) {
			dir := t.TempDir()
			temp, name := filepath.Join(dir, "temporary"), filepath.Join(dir, "saved")
			if err := os.WriteFile(name, []byte("before"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(temp, []byte("after"), 0600); err != nil {
				t.Fatal(err)
			}
			called := 0
			want := errors.New("injected directory sync failure")
			err := commitCheckpointFile(temp, name, func(path string) error {
				called++
				if path != dir {
					t.Fatal("wrong checkpoint directory")
				}
				got, e := os.ReadFile(name)
				if e != nil || string(got) != "after" {
					t.Fatal("directory sync occurred before rename", e)
				}
				if _, e = os.Stat(temp); !os.IsNotExist(e) {
					t.Fatal("temporary path still present")
				}
				if fault {
					return want
				}
				return syncJournalDirectory(path)
			})
			if called != 1 {
				t.Fatal("directory sync not called once")
			}
			if fault {
				if !errors.Is(err, want) {
					t.Fatal("lost directory sync error", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
	dir := t.TempDir()
	called := false
	if err := commitCheckpointFile(filepath.Join(dir, "missing"), filepath.Join(dir, "saved"), func(string) error { called = true; return nil }); err == nil || called {
		t.Fatal("rename failure must stop before directory sync")
	}
}
