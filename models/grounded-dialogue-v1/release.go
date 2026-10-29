package groundeddialogue

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// ProfileFiles identifies serving artifacts, not their semantic quality. The
// canonical runtime still authenticates the checkpoint and journal on recovery.
type ProfileFiles struct {
	Checkpoint string
	Journal    string
	Codec      string
}

func (f ProfileFiles) Fingerprint() string {
	data := "horizon-serving-profile-v1\n" + f.Checkpoint + "\n" + f.Journal + "\n" + f.Codec + "\n"
	sum := sha256.Sum256([]byte(data))
	return hex.EncodeToString(sum[:])
}

type Release struct {
	Files         ProfileFiles
	Sequence      uint64
	TrainedOn     string
	Communication CommunicationProfile
}

// CommunicationProfile is a publisher's bounded evaluation description tied to
// the serving artifact hashes, never teacher answers or inference parameters.
type CommunicationProfile struct {
	Adapter        string   `json:"adapter"`
	VerifiedSkills []string `json:"verified_skills"`
	Examples       []string `json:"examples"`
	Limits         []string `json:"limits"`
}

// BundledRelease describes the actual embedded training artifacts. A different
// fingerprint means different files; it does not establish a better/newer Brain.
func BundledRelease() (Release, error) {
	var manifest struct {
		Version       string               `json:"version"`
		Sequence      uint64               `json:"sequence"`
		Communication CommunicationProfile `json:"communication"`
		Provenance    struct {
			TrainedOn string `json:"trained_on"`
		} `json:"provenance"`
		Files []struct {
			Target string `json:"target"`
			SHA256 string `json:"unpacked_sha256"`
		} `json:"files"`
	}
	raw, err := assets.ReadFile("manifest.json")
	if err != nil {
		return Release{}, err
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return Release{}, err
	}
	if manifest.Version != "grounded-dialogue-bundle-v1" {
		return Release{}, fmt.Errorf("unsupported dialogue bundle")
	}
	files := map[string]string{}
	for _, entry := range manifest.Files {
		if entry.Target != "language-brain.json" && entry.Target != "events.jsonl" && entry.Target != "codec.json" {
			continue
		}
		digest, err := hex.DecodeString(entry.SHA256)
		if err != nil || len(digest) != sha256.Size || files[entry.Target] != "" {
			return Release{}, fmt.Errorf("invalid serving identity")
		}
		files[entry.Target] = entry.SHA256
	}
	if len(files) != 3 {
		return Release{}, fmt.Errorf("incomplete serving identity")
	}
	return Release{Files: ProfileFiles{Checkpoint: files["language-brain.json"], Journal: files["events.jsonl"], Codec: files["codec.json"]}, Sequence: manifest.Sequence, TrainedOn: manifest.Provenance.TrainedOn, Communication: manifest.Communication}, nil
}
