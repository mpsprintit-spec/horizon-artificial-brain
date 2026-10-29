// horizon-dialogue runs a saved internal V6 research language Brain offline.
// Teacher datasets and grammar labels are test-only and absent from this binary.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	"github.com/project-horizon/horizon-core/services/ai/runtime"
	"os"
)

func run() error {
	brain := flag.String("brain", "", "validated V6 checkpoint")
	codecFile := flag.String("codec", "", "neutral I/O alphabet metadata")
	requestFile := flag.String("request", "", "observed room and utterance JSON")
	replyFile := flag.String("reply", "", "optional full numeric trace JSON")
	flag.Parse()
	if *brain == "" || *codecFile == "" || *requestFile == "" {
		return fmt.Errorf("-brain, -codec and -request are required")
	}
	read := func(path string, target any) error {
		b, e := os.ReadFile(path)
		if e != nil {
			return e
		}
		return json.Unmarshal(b, target)
	}
	var codec runtime.DialogueCodec
	var request runtime.GroundedDialogueRequest
	if err := read(*codecFile, &codec); err != nil {
		return err
	}
	if err := read(*requestFile, &request); err != nil {
		return err
	}
	r := runtime.NewBrainRuntime(knowledge.NewBrain())
	if err := r.RestoreCheckpoint(*brain); err != nil {
		return err
	}
	out, err := r.ReadGroundedDialogue(codec, request)
	if err != nil {
		return err
	}
	if *replyFile != "" {
		b, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(*replyFile, append(b, '\n'), 0600); err != nil {
			return err
		}
	}
	fmt.Println(out.Text)
	if out.Text == "" {
		return fmt.Errorf("%s: %s", out.Status, out.Reason)
	}
	return nil
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
