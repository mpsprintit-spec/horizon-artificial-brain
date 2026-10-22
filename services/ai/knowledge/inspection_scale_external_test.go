package knowledge_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"os"
	"reflect"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/services/ai/bridge"
	"github.com/project-horizon/horizon-core/services/ai/knowledge"
	brainruntime "github.com/project-horizon/horizon-core/services/ai/runtime"
)

func procMemory() map[string]uint64 {
	data, _ := os.ReadFile("/proc/self/status")
	result := map[string]uint64{}
	for _, line := range strings.Split(string(data), "\n") {
		f := strings.Fields(line)
		if len(f) >= 2 && (f[0] == "VmRSS:" || f[0] == "VmHWM:") {
			n, _ := strconv.ParseUint(f[1], 10, 64)
			result[strings.TrimSuffix(f[0], ":")] = n * 1024
		}
	}
	return result
}

// Explicit opt-in child process; ordinary root/CI tests never build large
// fixtures. Collect native API latency, allocation and Linux high-water RSS.
func TestInspectionScaleChild(t *testing.T) {
	mode := os.Getenv("HORIZON_INSPECTION_SCALE_MODE")
	if mode == "" {
		t.Skip("explicit scale probe only")
	}
	count, err := strconv.Atoi(os.Getenv("HORIZON_INSPECTION_SCALE_N"))
	if err != nil || count < 1 {
		t.Fatal("invalid fixture count")
	}
	fanout, _ := strconv.Atoi(os.Getenv("HORIZON_INSPECTION_SCALE_FANOUT"))
	if mode == "snapshot" && count > 1000000 {
		t.Skip("legacy full-object probe is capped at one million units in this 8 GiB environment")
	}
	debug.SetMemoryLimit(6 * 1024 * 1024 * 1024)
	setup := time.Now()
	brain := knowledge.NewInspectionScaleFixture(count, fanout)
	r := brainruntime.NewBrainRuntime(brain)
	if r == nil {
		t.Fatal("fixture rejected")
	}
	setupTime := time.Since(setup)
	runtime.GC()
	beforeRSS := procMemory()
	resetErr := os.WriteFile("/proc/self/clear_refs", []byte("5"), 0600)
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	start := time.Now()
	var held any
	var bytes int64
	var digest string
	switch mode {
	case "snapshot":
		snapshot, e := r.MonitorSnapshot()
		if e != nil {
			t.Fatal(e)
		}
		held = snapshot
		value := reflect.ValueOf(snapshot)
		for i := 0; i < value.NumField(); i++ {
			if rows, ok := value.Field(i).Interface().([]json.RawMessage); ok {
				for _, row := range rows {
					bytes += int64(len(row))
				}
			}
		}
		digest = snapshot.CanonicalStateHash
	case "page":
		server, e := bridge.New(r, bridge.Config{})
		if e != nil {
			t.Fatal(e)
		}
		response := httptest.NewRecorder()
		server.Handler().ServeHTTP(response, httptest.NewRequest("GET", "/v1/brain/inspection?section=neural_units&limit=256", nil))
		if response.Code != 200 {
			t.Fatalf("inspection status %d: %.300s", response.Code, response.Body.String())
		}
		held = response
		bytes = int64(response.Body.Len())
	case "hash":
		if hasher, ok := any(brain).(interface{ CanonicalHash() (string, error) }); ok {
			digest, err = hasher.CanonicalHash()
		} else {
			var data []byte
			data, err = brain.CanonicalJSON()
			sum := sha256.Sum256(data)
			digest = "sha256:" + hex.EncodeToString(sum[:])
			held = data
		}
		if err != nil {
			t.Fatal(err)
		}
	case "write":
		counter := &scaleCounter{}
		if writer, ok := any(brain).(interface{ WriteCanonicalJSON(io.Writer) error }); ok {
			err = writer.WriteCanonicalJSON(counter)
		} else {
			var data []byte
			data, err = brain.CanonicalJSON()
			_, _ = counter.Write(data)
			held = data
		}
		if err != nil {
			t.Fatal(err)
		}
		bytes = counter.n
	case "save", "checkpoint", "checkpoint_restore":
		path := t.TempDir() + "/state.json"
		if mode == "save" {
			err = r.SaveBrain(path)
		} else {
			err = r.Checkpoint(path)
		}
		if err != nil {
			t.Fatal(err)
		}
		stat, e := os.Stat(path)
		if e != nil {
			t.Fatal(e)
		}
		bytes = stat.Size()
		if mode == "checkpoint_restore" {
			recoveredBrain := knowledge.NewBrain()
			recovered := brainruntime.NewBrainRuntime(recoveredBrain)
			if err := recovered.RestoreCheckpoint(path); err != nil {
				t.Fatal(err)
			}
			hasher := any(recoveredBrain).(interface{ CanonicalHash() (string, error) })
			restoredHash, e := hasher.CanonicalHash()
			if e != nil {
				t.Fatal(e)
			}
			originalHash, e := any(brain).(interface{ CanonicalHash() (string, error) }).CanonicalHash()
			if e != nil || restoredHash != originalHash {
				t.Fatal("large checkpoint lost canonical state", e)
			}
			digest = restoredHash
			held = recovered
		}
	default:
		t.Fatal("unknown scale mode")
	}
	elapsed := time.Since(start)
	runtime.ReadMemStats(&after)
	rss := procMemory()
	runtime.KeepAlive(held)
	runtime.KeepAlive(r)
	result := map[string]any{"mode": mode, "units": count, "seed_synapses": count * fanout, "setup_seconds": setupTime.Seconds(), "seconds": elapsed.Seconds(), "allocated_bytes": after.TotalAlloc - before.TotalAlloc, "heap_before_bytes": before.HeapAlloc, "heap_after_bytes": after.HeapAlloc, "rss_before_bytes": beforeRSS["VmRSS"], "rss_after_bytes": rss["VmRSS"], "peak_rss_bytes": rss["VmHWM"], "peak_reset": resetErr == nil, "json_bytes": bytes, "canonical_hash": digest}
	data, _ := json.Marshal(result)
	fmt.Println("H18_SCALE " + string(data))
}

type scaleCounter struct{ n int64 }

func (c *scaleCounter) Write(p []byte) (int, error) { c.n += int64(len(p)); return len(p), nil }
