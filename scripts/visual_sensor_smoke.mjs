import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { extractVisionFields } from "../web/monitor/sensor-adapters.js";
import { visualFrame, uniformFrame } from "../web/monitor/test/visual-fixtures.js";

const [base, token, prefix, memoryPath] = process.argv.slice(2);
assert.ok(base && token && prefix && memoryPath);
const source = "ci-rendered-visual-motion";
const headers = {Authorization:"Bearer "+token,"Content-Type":"application/json"};
const get = async path => {
  const response = await fetch(base+path,{headers,signal:AbortSignal.timeout(15000)});
  assert.ok(response.ok,"Bridge GET "+response.status);
  return response.json();
};
let previous = null, parent = null, previousAt = 0;
const sent = [];
for (const [index, rgba] of [visualFrame(),visualFrame({dx:1,regionDX:4}),uniformFrame()].entries()) {
  const capturedAt = performance.now();
  const elapsedNS = previous ? Math.max(1,Math.round((capturedAt-previousAt)*1e6)) : 0;
  const measured = extractVisionFields(rgba,120,120,previous,elapsedNS);
  const sensor = {modality:"vision",source,timestamp:new Date().toISOString(),sample_id:prefix+"-motion-"+index,confidence:.5,fields:measured.fields,
    provenance:{adapter:"ci-analytical-light-v1",calibration_id:"synthetic-image-plane-v1",parent_sample_ids:parent ? [parent] : []}};
  const response = await fetch(base+"/v1/sensor-observations",{method:"POST",headers,body:JSON.stringify(sensor),signal:AbortSignal.timeout(15000)});
  const body = await response.json();
  assert.equal(response.status,202,JSON.stringify(body));
  assert.equal(body.sensor_bindings.length,sensor.fields.length);
  for (const field of sensor.fields) {
    const binding = body.sensor_bindings.find(binding=>binding.field_id===field.field_id);
    assert.deepEqual(binding.values,field.values);
    assert.deepEqual(binding.coordinates,field.coordinates);
  }
  const displacement = body.sensor_bindings.filter(field=>field.kind.startsWith("light.flow.displacement."));
  if (index === 1) {
    assert.equal(displacement.length,16);
    const a = displacement.find(field=>field.group_id==="patch-0"), b = displacement.find(field=>field.group_id==="patch-15"), c = displacement.find(field=>field.group_id==="patch-14");
    assert.deepEqual(a.values,[4/6,0]);
    assert.deepEqual(b.values,[1/6,0]);
    assert.notDeepEqual(a.population,b.population);
    assert.deepEqual(b.population,c.population);
  } else assert.equal(displacement.length,0);
  if (index === 2) assert.ok(body.sensor_bindings.filter(field=>field.kind.startsWith("light.flow.quality.")).every(field=>field.values[5]===0));
  previous = measured.state; previousAt = capturedAt; parent = sensor.sample_id; sent.push(sensor);
}
const snapshot = await get("/v1/brain/snapshot");
const saved = JSON.parse(readFileSync(memoryPath,"utf8"));
const journalView = await get("/v1/brain/events?after=0");
for (const sensor of sent) {
  const event = journalView.events.find(event=>event.event?.sensor_observation?.sample_id===sensor.sample_id);
  assert.deepEqual(event.event.sensor_observation.fields,sensor.fields);
  for (const state of [snapshot,saved]) {
    const trace = state.sensory_traces.find(trace=>trace.sample_id===sensor.sample_id);
    assert.ok(trace,"committed sensor trace missing after persistence/monitor inspection");
    assert.equal(trace.fields.length,sensor.fields.length);
    assert.deepEqual(trace.parent_sample_ids??[],sensor.provenance.parent_sample_ids);
    assert.ok(trace.fields.every(field=>!Object.hasOwn(field,"luminance")));
  }
}
console.log("Visual sensor smoke: analytical pixels -> production adapter -> live Bridge -> scoped reuse, uncertainty, canonical memory and journal passed");
