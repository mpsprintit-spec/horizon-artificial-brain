import test from "node:test";
import assert from "node:assert/strict";
import { HorizonStore } from "../dist/lib/horizon-store.js";

test("revision gaps trigger resync", () => {
  const store = new HorizonStore();
  store.setSnapshot({
    type:"brain.snapshot", brain_identity:"horizon-primary-brain", state_revision:4,
    captured_at:"", canonical_state_hash:"sha256:x", counts:{}, neural_units:[],
    populations:[], synapses:[], temporal_patterns:[], experience_traces:[], brain_state:{}
  });
  assert.equal(store.applyEvent({type:"process",brain_identity:"horizon-primary-brain",state_revision:5,timestamp:""}),"applied");
  assert.equal(store.applyEvent({type:"process",brain_identity:"horizon-primary-brain",state_revision:7,timestamp:""}),"gap");
  assert.deepEqual(store.gap,{expected:6,received:7});
});
