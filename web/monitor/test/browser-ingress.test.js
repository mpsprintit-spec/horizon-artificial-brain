import test from "node:test";
import assert from "node:assert/strict";
import { monitorHarness, deferred, flush } from "./app-harness.js";
import { extractVisionFields, extractAudioFields } from "../sensor-adapters.js";

const sample=(x=8)=>({acceleration:{x,y:0,z:null},rotationRate:{alpha:90,beta:0,gamma:null},interval:20});
test("motion uses numeric ingress, one listener, bounded sends and acknowledged parents",async()=>{
  const pending=deferred();let hold=true;
  const h=monitorHarness({fetch:async(_request,reply)=>{if(hold){hold=false;await pending.promise;}return reply();}});
  await h.element("motionStart").onclick();await h.element("motionStart").onclick();
  assert.equal(h.listeners.get("devicemotion").size,1);
  const first=h.emitMotion(sample());await flush();
  h.clock.milliseconds+=300;await h.emitMotion(sample(8.1));
  assert.equal(h.requests.length,1);
  pending.resolve();await first;
  h.clock.milliseconds+=300;await h.emitMotion(sample(8.1));
  assert.equal(h.requests.length,2);
  for(const request of h.requests){assert.equal(new URL(request.url).pathname,"/v1/sensor-observations");assert.equal(request.body.tokens,undefined);assert.equal(request.body.payload_hash,undefined);assert.ok(request.body.fields.length>0);}
  assert.equal(h.requests[0].body.provenance.adapter,"browser-device-imu-v1");
  assert.deepEqual(h.requests[1].body.provenance.parent_sample_ids,[h.requests[0].body.sample_id]);
  assert.equal(Date.parse(h.requests[1].body.timestamp)-Date.parse(h.requests[0].body.timestamp),600);
  assert.deepEqual(h.requests[0].body.fields.find(f=>f.field_id==="reported-sample-interval").values,[20/1020]);
  const source=h.requests[0].body.source;
  h.element("motionStop").onclick();await h.emitMotion(sample());assert.equal(h.requests.length,2);
  await h.element("motionStart").onclick();await h.emitMotion(sample());
  assert.notEqual(h.requests[2].body.source,source);assert.deepEqual(h.requests[2].body.provenance.parent_sample_ids,[]);
});

test("stop cancels motion queued behind another capture and supersedes pending permission",async()=>{
  const blocked=deferred();let hold=true;
  const h=monitorHarness({fetch:async(_request,reply)=>{if(hold){hold=false;await blocked.promise;}return reply();}});
  await h.element("cameraStart").onclick();const camera=[...h.intervals.values()][0]();await flush();
  await h.element("motionStart").onclick();const motion=h.emitMotion(sample());await flush();
  h.element("motionStop").onclick();blocked.resolve();await Promise.all([camera,motion]);
  assert.equal(h.requests.length,1);assert.equal(h.element("motionStatus").textContent,"stopped");
  const permission=deferred();
  const p=monitorHarness({DeviceMotionEvent:class{static requestPermission(){return permission.promise;}}});
  const enabling=p.element("motionStart").onclick();p.element("motionStop").onclick();permission.resolve("granted");await enabling;
  assert.equal(p.listeners.get("devicemotion")?.size??0,0);assert.equal(p.element("motionStatus").textContent,"stopped");
});

test("permission failure, unavailable samples and failed sends do not create false parents",async()=>{
  const denied=monitorHarness({DeviceMotionEvent:class{static async requestPermission(){return "denied";}}});
  await denied.element("motionStart").onclick();assert.equal(denied.listeners.get("devicemotion")?.size??0,0);
  let fail=true;const h=monitorHarness({fetch:async(_request,reply)=>fail?{ok:false,json:async()=>({error:"offline"})}:reply()});
  await h.element("motionStart").onclick();await h.emitMotion({acceleration:null,rotationRate:null,interval:20});assert.equal(h.requests.length,0);
  await h.emitMotion(sample());fail=false;h.clock.milliseconds+=600;await h.emitMotion(sample());
  assert.equal(h.requests.length,2);assert.deepEqual(h.requests[1].body.provenance.parent_sample_ids,[]);
});

test("capture timestamps precede analysis and acquisition restarts separate physical streams",async()=>{
  const h=monitorHarness({extractVisionFields(clock,...args){const out=extractVisionFields(...args);clock.milliseconds+=100;return out;},extractAudioFields(clock,...args){const out=extractAudioFields(...args);clock.milliseconds+=100;return out;}});
  await h.element("cameraStart").onclick();await [...h.intervals.values()][0]();
  assert.equal(Date.parse(h.requests[0].body.timestamp),h.epoch+1000);
  const source=h.requests[0].body.source;
  await h.element("cameraStart").onclick();await [...h.intervals.values()][0]();
  assert.notEqual(h.requests[1].body.source,source);assert.deepEqual(h.requests[1].body.provenance.parent_sample_ids,[]);
  await h.element("audioStart").onclick();const at=h.clock.milliseconds;await [...h.intervals.values()][1]();
  assert.equal(Date.parse(h.requests[2].body.timestamp),h.epoch+at);
  assert.match(h.requests[2].body.provenance.calibration_id,/rms-relative/);
});

test("both text controls send ordered written tokens and hashes remain payload provenance",async()=>{
  const h=monitorHarness();h.element("ioInput").value="  Naku\tmelo\n🙂 ";await h.element("ioSend").onclick();
  h.element("simModality").value="text";h.element("simText").value="melo Naku";await h.element("simSend").onclick();
  assert.deepEqual(h.requests.map(r=>r.body.tokens),[["Naku","melo","🙂"],["melo","Naku"]]);
  assert.ok(h.requests.every(r=>new URL(r.url).pathname==="/v1/observations"&&r.body.payload_hash.startsWith("sha256:")));
});

test("written input waits behind captured sensors in timestamp order",async()=>{
  const pending=deferred();let first=true;
  const h=monitorHarness({fetch:async(_request,reply)=>{if(first){first=false;await pending.promise;}return reply();}});
  await h.element("cameraStart").onclick();const camera=[...h.intervals.values()][0]();await flush();
  h.clock.milliseconds+=600;await h.element("motionStart").onclick();const motion=h.emitMotion(sample());
  h.clock.milliseconds+=100;h.element("ioInput").value="naku melo";const written=h.element("ioSend").onclick();
  await flush();assert.equal(h.requests.length,1);
  pending.resolve();await Promise.all([camera,motion,written]);
  assert.deepEqual(h.requests.map(r=>r.body.modality),["vision","motion","text"]);
  const times=h.requests.map(r=>Date.parse(r.body.timestamp));assert.ok(times.every((t,i)=>i===0||t>=times[i-1]));
});

test("non-text simulator accepts numeric JSON instead of hashing arbitrary labels",async()=>{
  const h=monitorHarness();h.element("simModality").value="motion";
  for(const text of ["walking",'{"label":"walking"}',"[]",'{"fields":[]}']){h.element("simText").value=text;await h.element("simSend").onclick();assert.equal(h.requests.length,0);}
  h.element("simText").value='{"channel_states":[0.2,0.8]}';await h.element("simSend").onclick();
  assert.equal(h.requests.length,1);const request=h.requests[0];assert.equal(new URL(request.url).pathname,"/v1/sensor-observations");assert.deepEqual(request.body.channel_states,[.2,.8]);assert.equal(request.body.payload,undefined);assert.equal(request.body.provenance.calibration_id,"synthetic-normalized-v1");
});
