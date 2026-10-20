package runtime

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/project-horizon/horizon-core/services/ai/knowledge"
)

var ErrInspectionCursor = errors.New("invalid inspection cursor")
var ErrInspectionChanged = errors.New("inspection revision or runtime changed; start a new page view")

type InspectionPageRequest struct {
	Section string
	Order   string
	Limit   int
	Cursor  string
}
type inspectionCursor struct {
	Epoch    string `json:"epoch"`
	Revision uint64 `json:"revision"`
	Section  string `json:"section"`
	Order    string `json:"order"`
	Key      string `json:"key"`
}
type MonitorInspectionPage struct {
	Type            string                   `json:"type"`
	SchemaVersion   int                      `json:"schema_version"`
	BrainIdentity   string                   `json:"brain_identity"`
	StateRevision   uint64                   `json:"state_revision"`
	InspectionEpoch string                   `json:"inspection_epoch"`
	CapturedAt      string                   `json:"captured_at"`
	RuntimeCommit   string                   `json:"runtime_commit"`
	HashScope       string                   `json:"hash_scope"`
	Limit           int                      `json:"limit"`
	ByteLimit       int                      `json:"byte_limit"`
	Sections        []string                 `json:"sections"`
	Data            knowledge.InspectionData `json:"data"`
	NextCursor      string                   `json:"next_cursor,omitempty"`
	PageHash        string                   `json:"page_hash,omitempty"` // Last: browser verifies the original wire bytes.
}

func (r *BrainRuntime) inspectionEpochLocked() (string, error) {
	if r.inspectionEpoch == "" {
		var token [16]byte
		if _, err := rand.Read(token[:]); err != nil {
			return "", err
		}
		r.inspectionEpoch = hex.EncodeToString(token[:])
	}
	return r.inspectionEpoch, nil
}
func (r *BrainRuntime) MonitorInspectionPage(request InspectionPageRequest, commit string) (MonitorInspectionPage, error) {
	if r == nil || r.brain == nil {
		return MonitorInspectionPage{}, errors.New("brain runtime is not initialized")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.ensureWritableLocked(); err != nil {
		return MonitorInspectionPage{}, err
	}
	if request.Section == "" {
		request.Section = "neural_units"
	}
	if request.Order == "" {
		request.Order = "id"
	}
	if request.Limit == 0 {
		request.Limit = 128
	}
	epoch, err := r.inspectionEpochLocked()
	if err != nil {
		return MonitorInspectionPage{}, err
	}
	after := ""
	if request.Cursor != "" {
		if len(request.Cursor) > 8192 {
			return MonitorInspectionPage{}, ErrInspectionCursor
		}
		data, err := base64.RawURLEncoding.DecodeString(request.Cursor)
		if err != nil {
			return MonitorInspectionPage{}, ErrInspectionCursor
		}
		var cursor inspectionCursor
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&cursor); err != nil {
			return MonitorInspectionPage{}, ErrInspectionCursor
		}
		if decoder.Decode(new(any)) != io.EOF {
			return MonitorInspectionPage{}, ErrInspectionCursor
		}
		if cursor.Section != request.Section || cursor.Order != request.Order || cursor.Key == "" {
			return MonitorInspectionPage{}, ErrInspectionCursor
		}
		if cursor.Epoch != epoch || cursor.Revision != r.seq {
			return MonitorInspectionPage{}, ErrInspectionChanged
		}
		after = cursor.Key
	}
	data, err := r.brain.ReadInspectionPage(knowledge.InspectionRequest{Section: request.Section, After: after, Order: request.Order, Limit: request.Limit})
	if err != nil {
		return MonitorInspectionPage{}, err
	}
	next := ""
	if data.Next != "" {
		encoded, err := json.Marshal(inspectionCursor{epoch, r.seq, request.Section, request.Order, data.Next})
		if err != nil {
			return MonitorInspectionPage{}, err
		}
		next = base64.RawURLEncoding.EncodeToString(encoded)
		data.Next = ""
	}
	page := MonitorInspectionPage{Type: "brain.inspection.page", SchemaVersion: 1, BrainIdentity: BrainIdentity, StateRevision: r.seq, InspectionEpoch: epoch, CapturedAt: r.nowLocked().Format("2006-01-02T15:04:05.000Z07:00"), RuntimeCommit: commit, HashScope: "inspection_page", Limit: request.Limit, ByteLimit: knowledge.MaxInspectionBytes, Sections: append([]string(nil), knowledge.InspectionSections...), Data: data, NextCursor: next}
	encoded, err := json.Marshal(page)
	if err != nil {
		return MonitorInspectionPage{}, err
	}
	sum := sha256.Sum256(encoded)
	page.PageHash = "sha256:" + hex.EncodeToString(sum[:])
	encoded, err = json.Marshal(page)
	if err != nil {
		return MonitorInspectionPage{}, err
	}
	if len(encoded)+1 > knowledge.MaxInspectionBytes {
		return MonitorInspectionPage{}, knowledge.ErrInspectionTooLarge
	}
	return page, nil
}

type InspectionHandshake struct {
	Type            string `json:"type"`
	SchemaVersion   int    `json:"schema_version"`
	BrainIdentity   string `json:"brain_identity"`
	StateRevision   uint64 `json:"state_revision"`
	InspectionEpoch string `json:"inspection_epoch"`
	RuntimeCommit   string `json:"runtime_commit"`
	TelemetryMode   string `json:"telemetry_mode"`
	HashScope       string `json:"hash_scope"`
}

func (r *BrainRuntime) MonitorInspectionHandshake(commit string) (InspectionHandshake, error) {
	if r == nil || r.brain == nil {
		return InspectionHandshake{}, fmt.Errorf("brain runtime is not initialized")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.ensureWritableLocked(); err != nil {
		return InspectionHandshake{}, err
	}
	epoch, err := r.inspectionEpochLocked()
	if err != nil {
		return InspectionHandshake{}, err
	}
	return InspectionHandshake{Type: "brain.handshake", SchemaVersion: 2, BrainIdentity: BrainIdentity, StateRevision: r.seq, InspectionEpoch: epoch, RuntimeCommit: commit, TelemetryMode: "revisions", HashScope: "inspection_page"}, nil
}
