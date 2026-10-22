import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { setTimeout as wait } from "node:timers/promises";
import { monitorHarness } from "../web/monitor/test/app-harness.js";

const [base,token,prefix,memoryPath]=process.argv.slice(2);
assert.ok(base&&token&&prefix&&memoryPath);
const replies=[];
const h=monitorHarness({epoch:Date.now()-1000,idPrefix:prefix,
  async fetch(request){const response=await fetch(request.url,{...request.settings,signal:AbortSignal.timeout(15000)});const body=await response.clone().json();assert.equal(response.status,202,JSON.stringify(body));replies.push(body);return response;},
  get:async(url,settings)=>fetch(url,{...settings,signal:AbortSignal.timeout(15000)}),
});
h.element("bridge").value=base;h.element("token").value=token;
const tick=()=>{h.clock.milliseconds=Date.now()-h.epoch;};
const sample=x=>({acceleration:{x,y:0,z:null},accelerationIncludingGravity:{x:0,y:0,z:9.81},rotationRate:{alpha:90,beta:0,gamma:null},interval:20});
await h.element("motionStart").onclick();await h.element("motionStart").onclick();
assert.equal(h.listeners.get("devicemotion").size,1);
for(const [i,x] of [8,8.1,-8].entries()){if(i)await wait(550);tick();await h.emitMotion(sample(x));}
assert.equal(replies.length,3);
const binding=(reply,id)=>reply.sensor_bindings.find(b=>b.field_id===id);
assert.deepEqual(binding(replies[0],"acceleration:x").population,binding(replies[1],"acceleration:x").population);
assert.notDeepEqual(binding(replies[0],"acceleration:x").population,binding(replies[2],"acceleration:x").population);
assert.deepEqual(binding(replies[0],"acceleration:y").population,binding(replies[2],"acceleration:y").population);
assert.ok(!replies[0].sensor_bindings.some(b=>b.field_id==="acceleration:z"));
const source=h.requests[0].body.source;
h.element("motionStop").onclick();await h.emitMotion(sample(8));assert.equal(replies.length,3);
await h.element("motionStart").onclick();tick();await h.emitMotion(sample(8));
assert.equal(replies.length,4);assert.notEqual(h.requests[3].body.source,source);
assert.deepEqual(binding(replies[0],"acceleration:x").population,binding(replies[3],"acceleration:x").population);
await h.element("cameraStart").onclick();tick();await [...h.intervals.values()][0]();
await h.element("audioStart").onclick();tick();await [...h.intervals.values()][1]();
tick();h.element("ioInput").value="naku melo";await h.element("ioSend").onclick();
tick();h.element("simModality").value="motion";h.element("simText").value=JSON.stringify({fields:[h.requests[0].body.fields[0]]});await h.element("simSend").onclick();
h.element("motionStop").onclick();h.element("cameraStop").onclick();h.element("audioStop").onclick();
assert.equal(replies.length,8);
assert.equal(h.intervals.size,0);assert.equal(h.listeners.get("devicemotion").size,0);
const headers={Authorization:"Bearer "+token};
const snapshot=await (await fetch(base+"/v1/brain/snapshot",{headers})).json();
const events=await (await fetch(base+"/v1/brain/events?after=0",{headers})).json();
const memory=JSON.parse(readFileSync(memoryPath,"utf8"));
for(const request of h.requests){
  const sensor=request.body;
  if(sensor.tokens){assert.deepEqual(sensor.tokens,["naku","melo"]);const event=events.events.find(e=>e.event?.observation?.event_id===sensor.event_id);assert.ok(event,"written observation missing from journal view");assert.deepEqual(event.event.observation.tokens,sensor.tokens);assert.deepEqual(event.event.stimulus,sensor.tokens);continue;}
  assert.equal(new URL(request.url).pathname,"/v1/sensor-observations");assert.equal(sensor.payload_hash,undefined);
  for(const state of [snapshot,memory]){
    const trace=state.sensory_traces.find(t=>t.sample_id===sensor.sample_id);assert.ok(trace);
    assert.equal(trace.source,sensor.source);assert.equal(trace.calibration_id,sensor.provenance.calibration_id);
    assert.equal(Date.parse(trace.timestamp),Date.parse(sensor.timestamp));
    assert.deepEqual(trace.parent_sample_ids??[],sensor.provenance.parent_sample_ids??[]);
    for(const field of sensor.fields){const stored=trace.fields.find(f=>f.field_id===field.field_id);assert.deepEqual(stored.values,field.values);assert.deepEqual(stored.coordinates??[],field.coordinates??[]);}
  }
}
console.log("Browser sensor smoke: production text/camera/audio/IMU/simulator handlers -> live Bridge -> physical reuse, lineage and persistence passed");
