package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

func readCompositionFile(path string, value any) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, (1<<20)+1))
	if err != nil {
		return err
	}
	if len(data) > 1<<20 {
		return fmt.Errorf("composition file exceeds 1 MiB")
	}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return fmt.Errorf("composition file must contain one bounded JSON object")
	}
	return nil
}

func (c *cli) handleComposition(input string) bool {
	parts := strings.SplitN(input, " ", 3)
	var err error
	if len(parts) < 2 {
		return true
	}
	if strings.EqualFold(parts[1], "status") {
		state, statusErr := c.horizon.CompositionStatus()
		if statusErr != nil {
			err = statusErr
		} else if state == nil {
			fmt.Fprintln(c.out, "Komposisi: nonaktif")
		} else {
			fmt.Fprintf(c.out, "Komposisi: %s; mode usulan tindakan; batas kata=%d\n", state.Config.Version, knowledge.MaxCompositionWords)
		}
	} else if len(parts) != 3 || strings.TrimSpace(parts[2]) == "" {
		err = fmt.Errorf("pakai: composition initialize <backup-baru>, composition learn <exposure.json>, composition scene <scene.json>, composition status")
	} else {
		path := strings.TrimSpace(parts[2])
		switch strings.ToLower(parts[1]) {
		case "initialize":
			err = c.horizon.InitializeNeuralComposition(knowledge.DefaultCompositionConfig(), path, time.Now().UTC())
		case "learn":
			var exposure knowledge.CompositionExposure
			err = readCompositionFile(path, &exposure)
			if err == nil {
				_, err = c.horizon.LearnComposition(exposure, time.Now().UTC())
			}
		case "scene":
			var scene knowledge.CompositionScene
			err = readCompositionFile(path, &scene)
			if err == nil {
				_, err = c.horizon.ObserveCompositionScene("cli-scene-"+time.Now().UTC().Format(time.RFC3339Nano), "cli-numeric-simulator", scene)
			}
		default:
			err = fmt.Errorf("perintah komposisi tidak dikenal")
		}
		if err == nil {
			fmt.Fprintln(c.out, "Perubahan komposisi tersimpan pada runtime; tidak ada eksekusi perangkat.")
		}
	}
	if err != nil {
		fmt.Fprintf(c.out, "Komposisi gagal: %v\n", err)
	}
	return true
}
