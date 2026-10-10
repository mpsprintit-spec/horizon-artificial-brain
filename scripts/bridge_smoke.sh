#!/usr/bin/env bash
set -euo pipefail

tmp_dir="$(mktemp -d)"
port="18765"
base="http://127.0.0.1:${port}"
token="ci-smoke-token"
event_id="ci-browser-observation-$(date +%s%N)"
pid=""

cleanup() {
  if [[ -n "${pid}" ]]; then kill "${pid}" 2>/dev/null || true; wait "${pid}" 2>/dev/null || true; fi
  rm -rf "${tmp_dir}"
}
trap cleanup EXIT

HORIZON_BRAIN_MEMORY_PATH="${tmp_dir}/brain_memory.json" \
HORIZON_EVENT_LOG_PATH="${tmp_dir}/events.jsonl" \
HORIZON_BRIDGE_ADDRESS="127.0.0.1:${port}" \
HORIZON_BRIDGE_TOKEN="${token}" \
HORIZON_MONITOR_ORIGINS="http://127.0.0.1:${port}" \
go run ./services/ai >"${tmp_dir}/runtime.log" 2>&1 &
pid=$!

for _ in {1..60}; do
  if curl -fsS "${base}/health" >"${tmp_dir}/health.json"; then break; fi
  sleep 1
done

test -s "${tmp_dir}/health.json"
before="$(curl -fsS "${base}/v1/brain/snapshot" -H "Authorization: Bearer ${token}")"
before_revision="$(jq -r '.state_revision' <<<"${before}")"
before_hash="$(jq -r '.canonical_state_hash' <<<"${before}")"

curl -fsS "${base}/v1/brain/handshake" -H "Authorization: Bearer ${token}" | jq -e '.type=="brain.handshake" and .canonical_state_hash != ""' >/dev/null
curl -fsS "${base}/v1/brain/events?after=0" -H "Authorization: Bearer ${token}" | jq -e '.events | type=="array"' >/dev/null

curl -fsS -X POST "${base}/v1/observations" \
  -H "Authorization: Bearer ${token}" \
  -H "Content-Type: application/json" \
  --data-binary @- <<JSON >/dev/null
{"schema_version":1,"brain_identity":"horizon-primary-brain","event_id":"${event_id}","session_id":"ci-browser-session","sequence":1,"timestamp":"2026-10-07T15:00:00Z","modality":"sensor","source":"ci-browser-sensor","provenance":{"source":"ci-browser-sensor","modality":"sensor","capture_device":"ci","adapter":"ci-integration","synthetic":false},"payload":{"bytes":[1,2,3],"kind":"raw-sensor"}}
JSON

after="$(curl -fsS "${base}/v1/brain/snapshot" -H "Authorization: Bearer ${token}")"
after_revision="$(jq -r '.state_revision' <<<"${after}")"
after_hash="$(jq -r '.canonical_state_hash' <<<"${after}")"
test "${after_revision}" -gt "${before_revision}"
test "${after_hash}" != "${before_hash}"
jq -e --arg id "${event_id}" '.events | any(.[]; .event.id == $id)' < <(curl -fsS "${base}/v1/brain/events?after=0" -H "Authorization: Bearer ${token}") >/dev/null
test -s "${tmp_dir}/brain_memory.json"
grep -q '"neural_units"' "${tmp_dir}/brain_memory.json"
grep -q '"synapses"' "${tmp_dir}/brain_memory.json"

echo "Bridge smoke: health, handshake, snapshot, observation, persistence and event replay passed"
echo "revision: ${before_revision} -> ${after_revision}"
echo "hash: ${before_hash} -> ${after_hash}"
