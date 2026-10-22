import test from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { extractVisionFields, extractAudioFields } from "../sensor-adapters.js";

const frame = (r, g, b) => {
  const rgba = new Uint8ClampedArray(16*16*4);
  for (let i = 0; i < rgba.length; i += 4) rgba.set([r,g,b,255],i);
  return rgba;
};

test("camera preserves local groups and independently measured physical factors", () => {
  const first = extractVisionFields(frame(255,0,0),16,16);
  assert.equal(first.fields.length,64);
  const rgb = first.fields.filter(f=>f.kind === "light.rgb-v1");
  assert.equal(new Set(rgb.map(f=>f.group_id)).size,16);
  assert.deepEqual(rgb[0].values,[1,-1,-1]);
  assert.deepEqual(rgb[0].values,rgb[15].values);
  assert.notDeepEqual(rgb[0].coordinates,rgb[15].coordinates);
  const intensity = first.fields.find(f=>f.kind === "light.intensity.luma-v1");
  assert.ok(Math.abs(intensity.values[0]-(2*.2126-1)) < 1e-10);
  const repeated = extractVisionFields(frame(255,0,0),16,16,first.state,1e9);
  const deltas = repeated.fields.filter(f=>f.kind === "light.temporal-delta-v1");
  assert.equal(deltas.length,16);
  assert.ok(deltas.every(f=>f.values.every(v=>v===0)));
  assert.ok(deltas.every(f=>f.temporal_offset_ns===-1e9 && f.temporal_scale_ns===1e9));
});

test("a local light change preserves unaffected field values", () => {
  const original = frame(0,0,0), first = extractVisionFields(original,16,16);
  const changed = new Uint8ClampedArray(original);
  for (let y=0;y<4;y++) for(let x=0;x<4;x++) changed.set([255,255,255,255],4*(y*16+x));
  const second = extractVisionFields(changed,16,16,first.state,1e9);
  const byID = new Map(first.fields.map(f=>[f.field_id,f]));
  for (const field of second.fields.filter(f=>f.kind === "light.rgb-v1" && f.group_id !== "patch-0")) assert.deepEqual(field.values,byID.get(field.field_id).values);
  assert.deepEqual(second.fields.find(f=>f.field_id === "patch-0:light.temporal-delta-v1").values,[1,1,1]);
});

test("equal frequency with different envelopes shares spectral shape and preserves rhythm", () => {
  const rate=8000, count=8192;
  const signal = amplitude => Float32Array.from({length:count},(_,i)=>amplitude[Math.floor(i/(count/8))]*Math.sin(2*Math.PI*500*i/rate));
  const a=extractAudioFields(signal([.2,.2,.2,.2,.2,.2,.2,.2]),rate);
  const b=extractAudioFields(signal([.8,.2,.8,.2,.8,.2,.8,.2]),rate);
  const shape = fields=>fields.filter(f=>f.kind.startsWith("audio.spectral-shape"));
  for(let i=0;i<8;i++) {
    const left=shape(a)[i].values,right=shape(b)[i].values;
    assert.equal(left.indexOf(Math.max(...left)),3);
    assert.ok(left.every((value,j)=>Math.abs(value-right[j])<1e-5));
  }
  assert.notDeepEqual(a.filter(f=>f.kind==="audio.rms-v1").map(f=>f.values),b.filter(f=>f.kind==="audio.rms-v1").map(f=>f.values));
  assert.notDeepEqual(a.filter(f=>f.kind==="audio.onset-delta-v1").map(f=>f.values),b.filter(f=>f.kind==="audio.onset-delta-v1").map(f=>f.values));
  assert.ok(b.every(f=>f.temporal_offset_ns+f.temporal_scale_ns<=0));
  assert.ok(b.every(f=>f.values.every(v=>Number.isFinite(v)&&v>=-1&&v<=1)));
});

test("physical browser capture uses numeric endpoint and bounded windows", () => {
  const app=readFileSync(new URL("../app.js",import.meta.url),"utf8");
  assert.ok(app.includes("/v1/sensor-observations"));
  assert.ok(app.includes("getImageData"));
  assert.ok(app.includes("getFloatTimeDomainData"));
  assert.equal(app.includes("MediaRecorder"),false);
  assert.equal(app.includes("toBlob"),false);
  assert.equal(app.includes("image/jpeg"),false);
  assert.equal(app.includes("binaryToBase64"),false);
  assert.throws(()=>extractAudioFields([NaN, ...new Array(64).fill(0)],8000));
  assert.throws(()=>extractVisionFields([],16,16));
});
