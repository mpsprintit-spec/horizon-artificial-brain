import test from "node:test";
import assert from "node:assert/strict";
import { extractVisionFields } from "../sensor-adapters.js";
import { visualFrame, uniformFrame } from "./visual-fixtures.js";

const flow = (fields, factor) => fields.filter(field => field.kind.startsWith("light.flow."+factor+"."));
const atPatch = (fields, factor, patch=0) => flow(fields,factor).find(field => field.group_id === "patch-"+patch);
const measure = (options={}, previousOptions={}) => {
  const first = extractVisionFields(visualFrame(previousOptions),120,120);
  return extractVisionFields(visualFrame(options),120,120,first.state,1e9);
};

test("textured stationary input has supported zero flow; uniform input stays unknown", () => {
  const stationary = measure();
  assert.equal(flow(stationary.fields,"displacement").length,16);
  assert.ok(flow(stationary.fields,"displacement").every(field=>field.values.every(value=>value===0)));
  const first = extractVisionFields(uniformFrame(),120,120);
  const unknown = extractVisionFields(uniformFrame(120,120,140),120,120,first.state,1e9);
  assert.equal(flow(unknown.fields,"displacement").length,0);
  assert.equal(flow(unknown.fields,"relative").length,0);
  assert.equal(flow(unknown.fields,"quality").length,16);
  assert.ok(flow(unknown.fields,"quality").every(field=>field.values[5]===0));
});

test("translation is measured in both directions and common image motion has zero residual flow", () => {
  for (const [dx,dy] of [[3,2],[-3,-2]]) {
    const result = measure({dx,dy});
    assert.equal(flow(result.fields,"displacement").length,16);
    for (const field of flow(result.fields,"displacement")) assert.deepEqual(field.values,[dx/6,dy/6]);
    assert.ok(flow(result.fields,"relative").every(field=>field.values.every(value=>value===0)));
    assert.ok(flow(result.fields,"quality").every(field=>field.values[5]===1 && field.values[6]===1));
    assert.equal(result.fields.length,128);
  }
});

test("differential image motion changes sign when measured region motions are reversed", () => {
  const a = measure({dx:1,regionDX:4}), b = measure({dx:4,regionDX:1});
  assert.deepEqual(atPatch(a.fields,"displacement").values,[4/6,0]);
  assert.deepEqual(atPatch(b.fields,"displacement").values,[1/6,0]);
  assert.deepEqual(atPatch(a.fields,"relative").values,[3/12,0]);
  assert.deepEqual(atPatch(b.fields,"relative").values,[-3/12,0]);
  assert.deepEqual(atPatch(a.fields,"relative",15).values,[0,0]);
  assert.ok(a.fields.every(field=>!/(distance|depth|near|far|object)/.test(field.kind)));
});

test("additive light change does not masquerade as image displacement", () => {
  const result = measure({brightness:20});
  assert.equal(flow(result.fields,"displacement").length,16);
  assert.ok(flow(result.fields,"displacement").every(field=>field.values.every(value=>value===0)));
  assert.ok(result.fields.filter(field=>field.kind === "light.temporal-delta-v1").every(field=>field.values[0]>0));
});

test("periodic texture does not invent a unique correspondence", () => {
  const frame = uniformFrame();
  for(let y=0;y<120;y++) for(let x=0;x<120;x++) {
    const value = x%4 < 2 ? 50 : 200;
    frame.set([value,value,value,255],4*(y*120+x));
  }
  const first = extractVisionFields(frame,120,120);
  const second = extractVisionFields(frame,120,120,first.state,1e9);
  assert.equal(flow(second.fields,"displacement").length,0);
  assert.ok(flow(second.fields,"quality").every(field=>field.values[2]===0 && field.values[5]===0));
});

test("lost texture and motion outside the search radius remain uncertain", () => {
  const first = extractVisionFields(visualFrame(),120,120), occluded = visualFrame();
  for(let y=0;y<24;y++) for(let x=0;x<24;x++) occluded.set([240,240,240,255],4*(y*120+x));
  const second = extractVisionFields(occluded,120,120,first.state,1e9);
  assert.equal(atPatch(second.fields,"quality").values[5],0);
  assert.equal(atPatch(second.fields,"displacement"),undefined);
  const outside = measure({dx:12});
  assert.equal(flow(outside.fields,"displacement").length,0);
  const boundary = measure({dx:6});
  assert.equal(flow(boundary.fields,"displacement").length,0);
});

test("missing temporal support or changed geometry resets correspondence; output stays bounded", () => {
  const first = extractVisionFields(visualFrame(),120,120), saved = [...first.state.luminance];
  for (const elapsed of [0,-1,NaN,1.1,61e9]) {
    const result = extractVisionFields(visualFrame({dx:1}),120,120,first.state,elapsed);
    assert.equal(flow(result.fields,"quality").length,0);
    assert.equal(result.fields.filter(field=>field.kind==="light.temporal-delta-v1").length,0);
  }
  const resized = extractVisionFields(visualFrame({width:80,height:80}),80,80,first.state,1e9);
  assert.equal(resized.fields.length,64);
  const result = extractVisionFields(visualFrame({dx:2}),120,120,first.state,999000000);
  assert.ok(flow(result.fields,"quality").every(field=>field.kind.includes("120x120")));
  assert.ok(result.fields.length<=128 && new Set(result.fields.map(field=>field.field_id)).size===result.fields.length);
  assert.ok(result.fields.every(field=>field.values.every(value=>Number.isFinite(value)&&value>=-1&&value<=1)));
  assert.ok(flow(result.fields,"quality").every(field=>field.temporal_offset_ns===-999000000 && field.temporal_scale_ns===999000000));
  assert.deepEqual([...first.state.luminance],saved);
  assert.equal(Object.hasOwn(result.fields[0],"luminance"),false);
});

test("isolated supported motion does not establish a shared image-motion reference", () => {
  const firstPixels=uniformFrame(), secondPixels=uniformFrame(), texture=visualFrame();
  for (let y=0;y<30;y++) for(let x=0;x<30;x++) {
    const pixel=texture.subarray(4*(y*120+x),4*(y*120+x)+4);
    firstPixels.set(pixel,4*(y*120+x));
    secondPixels.set(pixel,4*(y*120+x+2));
  }
  const first=extractVisionFields(firstPixels,120,120);
  const second=extractVisionFields(secondPixels,120,120,first.state,1e9);
  assert.equal(flow(second.fields,"displacement").length,1);
  assert.deepEqual(atPatch(second.fields,"displacement").values,[2/6,0]);
  assert.equal(flow(second.fields,"relative").length,0);
  assert.ok(flow(second.fields,"quality").every(field=>field.values[6]===0));
});
