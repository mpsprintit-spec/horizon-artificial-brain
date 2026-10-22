import { extractMotionFields, extractAudioFields, extractVisionFields, browserSensorProfile } from "../sensor-adapters.js";

// Deterministic physical measurements, never semantic targets. The committed
// JSON is consumed by Go tests and checked against the live adapters in Node.
export function browserSensorFixtures() {
  const sensors=[];
  const add=(modality,id,source,fields,parent=null)=>{
    sensors.push({modality,source,timestamp:new Date(Date.UTC(2026,9,20,11,0,sensors.length)).toISOString(),sample_id:id,fields,confidence:.5,
      provenance:{...browserSensorProfile(modality),parent_sample_ids:parent?[parent]:[]}});
  };
  const motion=x=>extractMotionFields({acceleration:{x,y:0,z:null},accelerationIncludingGravity:{x:0,y:0,z:9.81},rotationRate:{alpha:90,beta:0,gamma:null},interval:20});
  add("motion","motion-a","browser-motion:fixture:capture-1",motion(8));
  add("motion","motion-near","browser-motion:fixture:capture-1",motion(8.1),"motion-a");
  add("motion","motion-b","browser-motion:fixture:capture-1",motion(-8),"motion-near");
  add("motion","motion-restarted","browser-motion:fixture:capture-2",motion(8));
  const tone=amplitudes=>Float32Array.from({length:8192},(_,i)=>amplitudes[Math.floor(i/1024)]*Math.sin(2*Math.PI*500*i/8000));
  add("audio","audio-a","browser-microphone:fixture:capture-1",extractAudioFields(tone(new Array(8).fill(.2)),8000));
  add("audio","audio-b","browser-microphone:fixture:capture-1",extractAudioFields(tone([.8,.2,.8,.2,.8,.2,.8,.2]),8000),"audio-a");
  const rgba=new Uint8ClampedArray(16*16*4);for(let i=0;i<rgba.length;i+=4)rgba.set([255,0,0,255],i);
  const first=extractVisionFields(rgba,16,16);
  add("vision","vision-a","browser-camera:fixture:capture-1",first.fields);
  for(let y=0;y<4;y++)for(let x=0;x<4;x++)rgba.set([255,255,255,255],4*(y*16+x));
  add("vision","vision-b","browser-camera:fixture:capture-1",extractVisionFields(rgba,16,16,first.state,1e9).fields,"vision-a");
  return sensors;
}
