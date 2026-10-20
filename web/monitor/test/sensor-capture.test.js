import test from "node:test";
import assert from "node:assert/strict";
import vm from "node:vm";
import { readFileSync } from "node:fs";
import { extractVisionFields, extractAudioFields, browserSensorProfile } from "../sensor-adapters.js";
import { visualFrame } from "./visual-fixtures.js";

// Execute the production event handlers with deterministic media devices.
// This verifies capture -> feature extraction -> HTTP wiring without granting
// device access in a test runner or depending on a real camera/microphone.
test("browser camera and microphone submit local numeric fields and stop cleanly", async () => {
  const elements=new Map(), intervals=new Map(), requests=[], tracks=[], storage=new Map();
  let nextTimer=0, nextID=0, imageSequence=0, capturedAt=1000;
  const drawing={clearRect(){},beginPath(){},moveTo(){},lineTo(){},stroke(){},drawImage(){},getImageData(_x,_y,w,h){
    return {data:visualFrame({width:w,height:h,dx:imageSequence++})};
  }};
  const element=id=>{
    if(!elements.has(id))elements.set(id,{value:"http://bridge.test",videoWidth:16,videoHeight:16,getContext:()=>drawing});
    return elements.get(id);
  };
  class AudioContextMock {
    sampleRate=8000;
    createMediaStreamSource(){return {connect(){}};}
    createAnalyser(){return {fftSize:0,getFloatTimeDomainData(data){for(let i=0;i<data.length;i++)data[i]=.3*Math.sin(2*Math.PI*500*i/8000);}};}
    async resume(){}
    async close(){}
  }
  const context={
    HorizonStore:class{snapshot=null;subscribe(){}},extractVisionFields,extractAudioFields,browserSensorProfile,
    document:{getElementById:element,createElement:()=>({getContext:()=>drawing})},
    sessionStorage:{getItem:key=>storage.get(key)??null,setItem:(key,value)=>storage.set(key,value)},
    navigator:{mediaDevices:{async getUserMedia(){const track={stopped:false,stop(){this.stopped=true;}};tracks.push(track);return {getTracks:()=>[track]};}}},
    AudioContext:AudioContextMock,performance:{now:()=>capturedAt},crypto:{randomUUID:()=>"fixture-"+(++nextID)},
    setInterval:callback=>{intervals.set(++nextTimer,callback);return nextTimer;},clearInterval:id=>intervals.delete(id),
    requestAnimationFrame:()=>1,cancelAnimationFrame(){},
    async fetch(url,options){requests.push({url,sensor:JSON.parse(options.body)});return {ok:true,json:async()=>({state_revision:requests.length})};},
    Float32Array,Uint8ClampedArray,TextEncoder,URL,console,
  };
  const source=readFileSync(new URL("../app.js",import.meta.url),"utf8").replace(/^import .*;\n/gm,"");
  vm.runInNewContext(source,context,{filename:"production-monitor-app.js"});
  await element("cameraStart").onclick();
  assert.equal(intervals.size,1);
  await [...intervals.values()][0]();
  capturedAt = 2000;
  await [...intervals.values()][0]();
  await element("audioStart").onclick();
  assert.equal(intervals.size,2);
  await [...intervals.values()][1]();
  assert.equal(requests.length,3);
  for(const {url,sensor} of requests){
    assert.equal(url,"http://bridge.test/v1/sensor-observations");
    assert.ok(sensor.sample_id&&sensor.timestamp);
    assert.ok(sensor.fields.length>1&&sensor.fields.length<=128);
    assert.equal(sensor.payload,undefined);
    assert.equal(sensor.tokens,undefined);
    assert.equal(sensor.payload_hash,undefined);
    assert.ok(sensor.fields.every(field=>field.field_id&&field.group_id&&field.values.length>0));
  }
  assert.equal(requests[0].sensor.modality,"vision");
  assert.equal(requests[1].sensor.modality,"vision");
  assert.equal(requests[2].sensor.modality,"audio");
  assert.ok(requests[0].sensor.source.startsWith("github-pages-camera:"+storage.get("horizon.session")+":"));
  assert.equal(requests[0].sensor.source,requests[1].sensor.source);
  assert.ok(requests[2].sensor.source.startsWith("github-pages-microphone:"+storage.get("horizon.session")+":"));
  assert.deepEqual(requests[1].sensor.provenance.parent_sample_ids,[requests[0].sensor.sample_id]);
  assert.equal(requests[1].sensor.fields.length,128);
  assert.equal(requests[1].sensor.provenance.adapter,"browser-light-motion-v2");
  assert.match(element("cameraStatus").textContent,/motion evidence 16\/16/);
  const displacement=requests[1].sensor.fields.filter(field=>field.kind.startsWith("light.flow.displacement."));
  assert.equal(displacement.length,16);
  assert.ok(displacement.every(field=>field.values[0]===1/6&&field.values[1]===0));
  assert.equal(requests[1].sensor.luminance,undefined);
  element("cameraStop").onclick();element("audioStop").onclick();
  assert.equal(intervals.size,0);
  assert.ok(tracks.every(track=>track.stopped));
});
