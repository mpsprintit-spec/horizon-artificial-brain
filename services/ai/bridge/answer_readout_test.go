package bridge

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/project-horizon/horizon-core/services/ai/runtime"
)

func TestUntrainedObservationResponsesKeepActivitySeparateFromAnswer(t *testing.T) {
	for _, scenario := range []struct{ path, body string }{
		{"/v1/observations", `{"event_id":"new-text","source":"test","modality":"text","timestamp":"2026-10-21T05:00:00Z","tokens":["naku"]}`},
		{"/v1/sensor-observations", `{"modality":"vision","source":"test","sample_id":"new-frame","timestamp":"2026-10-21T05:00:00Z","confidence":0.5,"fields":[{"field_id":"rgb","group_id":"camera","kind":"light.rgb-v1","values":[0.2,0.4,0.6]}]}`},
	} {
		t.Run(scenario.path, func(t *testing.T) {
			owner, err := runtime.NewRuntimeSupervisor(runtime.NewBrainRuntime(nil))
			if err != nil {
				t.Fatal(err)
			}
			defer owner.Close()
			server, err := NewManaged(owner, Config{Token: "readout-test"})
			if err != nil {
				t.Fatal(err)
			}
			req := httptest.NewRequest(http.MethodPost, scenario.path, strings.NewReader(scenario.body))
			req.Header.Set("Authorization", "Bearer readout-test")
			res := httptest.NewRecorder()
			server.Handler().ServeHTTP(res, req)
			if res.Code != http.StatusAccepted {
				t.Fatalf("status=%d: %s", res.Code, res.Body.String())
			}
			var response struct {
				Active []uint64      `json:"active_node_ids"`
				Answer answerReadout `json:"answer"`
			}
			if err := json.Unmarshal(res.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if len(response.Active) == 0 || response.Answer.Status != runtime.AnswerUninterpreted || len(response.Answer.NodeIDs) != 0 || response.Answer.Confidence != 0 || response.Answer.Uncertainty != 1 {
				t.Fatalf("untrained activity became an answer: %s", res.Body.String())
			}
			var raw map[string]json.RawMessage
			if err := json.Unmarshal(res.Body.Bytes(), &raw); err != nil {
				t.Fatal(err)
			}
			for _, forbidden := range []string{"text", "label", "prediction_assessment", "alternatives"} {
				if strings.Contains(string(raw["answer"]), `"`+forbidden+`"`) {
					t.Fatalf("transport included %q in numeric readout", forbidden)
				}
			}
		})
	}
}
