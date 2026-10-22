import test from "node:test";
import assert from "node:assert/strict";
import { extractMotionFields } from "../sensor-adapters.js";

export const motionSample = (x=8) => ({acceleration:{x,y:0,z:null},accelerationIncludingGravity:{x:0,y:0,z:9.81},rotationRate:{alpha:90,beta:0,gamma:null},interval:20});

test("motion preserves physical channels, missing axes, and reported interval",()=>{
  const fields=extractMotionFields(motionSample());
  assert.equal(fields.length,8);
  const byID=new Map(fields.map(f=>[f.field_id,f]));
  assert.deepEqual(byID.get("acceleration:x").values,[1/3]);
  assert.deepEqual(byID.get("acceleration:y").values,[0]);
  assert.deepEqual(byID.get("acceleration:x").coordinates,[1,0,0]);
  assert.deepEqual(byID.get("acceleration:y").coordinates,[0,1,0]);
  assert.equal(byID.has("acceleration:z"),false);
  assert.equal(byID.has("rotationRate:gamma"),false);
  assert.deepEqual(byID.get("rotationRate:alpha").values,[1/3]);
  assert.equal(byID.get("rotationRate:alpha").coordinates,undefined);
  assert.deepEqual(byID.get("reported-sample-interval").values,[20/1020]);
  assert.notEqual(byID.get("acceleration:x").kind,byID.get("accelerationIncludingGravity:x").kind);
  assert.notEqual(byID.get("acceleration:x").kind,byID.get("acceleration:y").kind);
  assert.notEqual(byID.get("rotationRate:alpha").kind,byID.get("rotationRate:beta").kind);
  assert.ok(fields.every(f=>f.temporal_scale_ns===undefined&&f.temporal_offset_ns===undefined));
});

test("small physical perturbations stay local and continuous without clipping ordinary measurements",()=>{
  const a=extractMotionFields(motionSample()),b=extractMotionFields(motionSample(8.1));
  assert.ok(Math.abs(b[0].values[0]-a[0].values[0])<.003);
  assert.deepEqual(a.slice(1),b.slice(1));
  for(const x of [-1e6,-100,-16,0,16,100,1e6]){
    const value=extractMotionFields({acceleration:{x}})[0].values[0];
    assert.ok(Number.isFinite(value)&&Math.abs(value)<1);
    assert.ok(Math.abs(16*value/(1-Math.abs(value))-x)<Math.max(1e-9,Math.abs(x)*1e-10));
  }
  const aCopy=extractMotionFields(motionSample());a[0].coordinates[0]=-1;assert.deepEqual(aCopy[0].coordinates,[1,0,0]);
});

test("unavailable motion is distinct from a measured zero and invalid input is rejected",()=>{
  for(const event of [null,{}, {acceleration:null,rotationRate:null,interval:20},{acceleration:{x:NaN}},{rotationRate:{beta:Infinity}},{acceleration:{x:"1"}},{acceleration:[]},{acceleration:{x:0},interval:-1}])assert.throws(()=>extractMotionFields(event));
  assert.deepEqual(extractMotionFields({acceleration:{x:0}})[0].values,[0]);
  assert.equal(extractMotionFields({acceleration:{x:0},interval:0}).length,1);
});
