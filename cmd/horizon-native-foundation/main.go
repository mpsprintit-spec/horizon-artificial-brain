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
	phase2 := flag.String("phase2", "", "Phase II encoder: identity or relational")
	credit := flag.String("credit", "", "Phase II credit arm: paired, unpaired, or paired-centered")
	epsilon := flag.Float64("epsilon", 0, "optional Phase II epsilon-greedy exploration")
	boundary := flag.String("credit-boundary", "", "evaluate frozen checkpoint around operation-cost boundary")
	retention := flag.String("retention-baseline", "", "frozen Phase II checkpoint for matched old-task evaluation")
	phase3 := flag.String("phase3-credit", "", "Phase III action, feature, unpaired or unmatched factual contrasts")
	phase3Depth := flag.Int("phase3-depth", 1, "maximum factual continuation depth, one or two")
	phase3Encoder := flag.String("phase3-encoder", "identity", "Phase III identity or relational readout ablation")
	noisy := flag.Bool("reward-offset", true, "Phase III shared exogenous reward offset")
	flag.Parse()
	if *retention != "" {
		seed, err := strconv.ParseInt(*seeds, 10, 64)
		if err != nil {
			panic(err)
		}
		result, err := nf.Phase2RetentionBaseline(*retention, seed, *stream, *cases)
		if err != nil {
			panic(err)
		}
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			panic(err)
		}
		fmt.Println(string(data))
		return
	}
	if *boundary != "" {
		result, err := nf.CreditBoundary(*boundary, *stream, *cases)
		if err != nil {
			panic(err)
		}
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			panic(err)
		}
		if err = os.MkdirAll(*dir, 0700); err != nil {
			panic(err)
		}
		file, err := os.OpenFile(*dir+"/boundary.json", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			panic(err)
		}
		if _, err = file.Write(data); err != nil {
			file.Close()
			panic(err)
		}
		if err = file.Close(); err != nil {
			panic(err)
		}
		fmt.Println(string(data))
		return
	}
	if *recovery {
		var err error
		if *phase2 != "" {
			err = nf.Phase2RecoveryAudit(*dir)
		} else {
			err = nf.RecoveryAudit(*dir)
		}
		if err != nil {
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

		if *phase3 != "" {
			result, err := nf.Phase3CreditExperiment(fmt.Sprintf("%s/seed%d", *dir, seed), seed, *phase3, *episodes, *cases, *stream, *noisy, *phase3Encoder, *phase3Depth)
			if err != nil {
				panic(err)
			}
			data, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				panic(err)
			}
			if err = os.WriteFile(fmt.Sprintf("%s/seed%d.json", *dir, seed), data, 0600); err != nil {
				panic(err)
			}
			fmt.Printf("phase3 seed=%d arm=%s return=%g regret=%g effectPrecision=%g seconds=%.2f\n", seed, *phase3, result.After.Return, result.Regret, result.EffectPrecision, result.Seconds)
			continue
		}
		if *credit != "" {
			if *credit != "paired" && *credit != "unpaired" && *credit != "paired-centered" {
				panic("invalid credit arm")
			}
			result, err := nf.Phase2CreditExperiment(fmt.Sprintf("%s/seed%d", *dir, seed), seed, *credit != "unpaired", *episodes, *cases, *stream, *credit == "paired-centered")
			if err != nil {
				panic(err)
			}
			data, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				panic(err)
			}
			if err = os.WriteFile(fmt.Sprintf("%s/seed%d.json", *dir, seed), data, 0600); err != nil {
				panic(err)
			}
			fmt.Printf("credit seed=%d arm=%s return=%g precision=%g nuisance=%g\n", seed, *credit, result.After.Return, result.Precision, result.NuisanceSensitivity)
			continue
		}
		if *phase2 != "" {
			if *phase2 != "identity" && *phase2 != "relational" {
				panic("invalid phase2 encoder")
			}
			result, err := nf.Phase2Experiment(fmt.Sprintf("%s/seed%d", *dir, seed), seed, *phase2, *episodes, *cases, *stream, *lambda, *epsilon)
			if err != nil {
				panic(err)
			}
			data, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				panic(err)
			}
			if err = os.WriteFile(fmt.Sprintf("%s/seed%d.json", *dir, seed), data, 0600); err != nil {
				panic(err)
			}
			fmt.Printf("seed=%d encoder=%s test=%g longer=%g permutation=%g seconds=%.2f\n", seed, *phase2, result.Evaluations["test"]["trained"].Return, result.Evaluations["longer"]["trained"].Return, result.Evaluations["permutation"]["trained"].Return, result.Duration)
			continue
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
