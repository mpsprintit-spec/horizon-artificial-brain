package main

import (
	"encoding/json"
	"flag"
	"fmt"
	nf "github.com/project-horizon/horizon-core/internal/nativefoundation"
	"os"
	"strconv"
	"strings"
)

func main() {
	dir := flag.String("out", "native-foundation-results", "output directory")
	seeds := flag.String("seeds", "37", "independent model seeds")
	episodes := flag.Int("train", 2000, "training episodes")
	cases := flag.Int("test", 256, "cases per evaluation distribution")
	lambda := flag.Float64("lambda", 1, "offline Sarsa lambda")
	stream := flag.Int64("stream", 200000, "held-out task stream")
	recovery := flag.Bool("recovery-audit", false, "preserve and verify a 64-episode training journal plus 5-episode suffix")
	flag.Parse()
	if *recovery {
		if err := nf.RecoveryAudit(*dir); err != nil {
			panic(err)
		}
		return
	}
	if *episodes < 1 || *cases < 1 {
		panic("train and test counts must be positive")
	}
	if e := os.MkdirAll(*dir, 0700); e != nil {
		panic(e)
	}
	for _, s := range strings.Split(*seeds, ",") {
		seed, e := strconv.ParseInt(s, 10, 64)
		if e != nil {
			panic(e)
		}
		result, e := nf.Experiment(fmt.Sprintf("%s/seed%d", *dir, seed), seed, *lambda, *episodes, *cases, *stream)
		if e != nil {
			panic(e)
		}
		data, e := json.MarshalIndent(result, "", "  ")
		if e != nil {
			panic(e)
		}
		path := fmt.Sprintf("%s/seed%d.json", *dir, seed)
		if e = os.WriteFile(path, data, 0600); e != nil {
			panic(e)
		}
		fmt.Printf("seed=%d lambda=%g return=%g untrained=%g changes=%d seconds=%.2f\n", seed, *lambda, result.Evaluations["test"]["trained"].Return, result.Evaluations["test"]["untrained"].Return, result.ParameterChanges, result.DurationSeconds)
	}
}
