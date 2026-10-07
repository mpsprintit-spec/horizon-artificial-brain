import test from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
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

test("production monitor has no demo snapshot or random-number path", () => {
  const app = readFileSync(new URL("../app.js", import.meta.url), "utf8");
  assert.equal(app.includes("demoSnapshot"), false);
  assert.equal(app.includes("Math.random"), false);
  for (const required of [
    'getUserMedia({video:true})',
    'getUserMedia({audio:true})',
    'DeviceMotionEvent',
    '/v1/brain/handshake',
    '/v1/brain/snapshot',
    '/v1/brain/stream',
    '/v1/observations',
    'neural_units',
    'synapses',
    'canonical_state_hash'
  ]) assert.ok(app.includes(required), "missing production monitor path: "+required);
});

test("canonical hash mismatches are rejected", () => {
  const store = new HorizonStore();
  store.setSnapshot({
    type:"brain.snapshot", brain_identity:"horizon-primary-brain", state_revision:4,
    captured_at:"", canonical_state_hash:"sha256:expected", counts:{}, neural_units:[],
    populations:[], synapses:[], temporal_patterns:[], experience_traces:[], brain_state:{}
  });
  assert.throws(() => store.applyEvent({
    type:"process",brain_identity:"horizon-primary-brain",state_revision:5,
    timestamp:"",canonical_state_hash:"sha256:other"
  }), /canonical state hash mismatch/);
});
