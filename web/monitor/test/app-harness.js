import vm from "node:vm";
import { readFileSync } from "node:fs";
import { webcrypto } from "node:crypto";
import * as adapters from "../sensor-adapters.js";
import { verifyInspectionPage, inspectionGraph, readInspectionText } from "../inspection-page.js";
import { createSnapshotScheduler } from "../snapshot-scheduler.js";

// Execute production handlers. Only device, clock, transport and DOM surfaces
// are deterministic fakes; sensor extraction and send/stop logic are real.
export function monitorHarness(options = {}) {
  const elements=new Map(), intervals=new Map(), listeners=new Map(), requests=[], tracks=[], storage=new Map();
  const clock={milliseconds:1000}, epoch=options.epoch??Date.parse("2026-10-20T11:00:00Z");
  let nextID=0,nextTimer=0;
  const drawing={clearRect(){},beginPath(){},moveTo(){},lineTo(){},stroke(){},drawImage(){},getImageData(_x,_y,w,h){
    const data=new Uint8ClampedArray(w*h*4);for(let i=0;i<data.length;i+=4)data.set([100,50,20,255],i);return {data};
  }};
  const element=id=>{
    if(!elements.has(id))elements.set(id,{value:id==="bridge"?"http://bridge.test":"",textContent:"",videoWidth:16,videoHeight:16,getContext:()=>drawing});
    return elements.get(id);
  };
  class AudioContextMock {
    sampleRate=8000;
    createMediaStreamSource(){return {connect(){}};}
    createAnalyser(){return {fftSize:0,getFloatTimeDomainData(data){for(let i=0;i<data.length;i++)data[i]=.3*Math.sin(2*Math.PI*500*i/8000);}};}
    async resume(){} async close(){}
  }
  class ClockDate extends Date {constructor(...args){super(...(args.length?args:[epoch+clock.milliseconds]));}static now(){return epoch+clock.milliseconds;}}
  const reply=()=>({ok:true,json:async()=>({state_revision:requests.length,canonical_state_hash:"fixture"})});
  const context={...adapters,createSnapshotScheduler,verifyInspectionPage,inspectionGraph,readInspectionText,
    HorizonStore:options.HorizonStore??class{snapshot=null;revision=0;subscribe(){}setSnapshot(){}resetForConnection(){}},
    document:{getElementById:element,createElement:()=>({getContext:()=>drawing})},
    sessionStorage:{getItem:key=>storage.get(key)??null,setItem:(key,value)=>storage.set(key,value)},
    navigator:{mediaDevices:{async getUserMedia(){const track={stopped:false,stop(){this.stopped=true;}};tracks.push(track);return {getTracks:()=>[track]};}}},
    window:{addEventListener(name,handler){if(!listeners.has(name))listeners.set(name,new Set());listeners.get(name).add(handler);},removeEventListener(name,handler){listeners.get(name)?.delete(handler);}},
    DeviceMotionEvent:options.DeviceMotionEvent??class{},AudioContext:AudioContextMock,
    performance:{now:()=>clock.milliseconds},Date:ClockDate,
    crypto:{randomUUID:()=>(options.idPrefix??"fixture")+"-"+(++nextID),subtle:webcrypto.subtle},
    setInterval:callback=>{intervals.set(++nextTimer,callback);return nextTimer;},clearInterval:id=>intervals.delete(id),
    requestAnimationFrame:()=>1,cancelAnimationFrame(){},
    setTimeout,clearTimeout,WebSocket:options.WebSocket,
    async fetch(url,settings){
      if(!settings?.body){
        if(url.includes("/v1/brain/inspection")&&!options.inspection)return {ok:false,status:404,text:async()=>"404 page not found\n"};
        return options.get ? options.get(url,settings) : reply();
      }
      const request={url,body:JSON.parse(settings.body),settings};requests.push(request);
      return options.fetch ? options.fetch(request,reply) : reply();
    },
    Float32Array,Uint8ClampedArray,TextEncoder,URL,URLSearchParams,console,
  };
  if(options.extractVisionFields)context.extractVisionFields=(...args)=>options.extractVisionFields(clock,...args);
  if(options.extractAudioFields)context.extractAudioFields=(...args)=>options.extractAudioFields(clock,...args);
  vm.runInNewContext(readFileSync(new URL("../app.js",import.meta.url),"utf8").replace(/^import .*;\n/gm,""),context,{filename:"production-monitor-app.js"});
  return {element,clock,requests,listeners,intervals,tracks,storage,epoch,
    connect:()=>context.connect(),disconnect:()=>context.disconnect(),
    async emitMotion(event){await Promise.all([...(listeners.get("devicemotion")??[])].map(handler=>handler(event)));},
  };
}

export function deferred() {let resolve;const promise=new Promise(r=>{resolve=r;});return {promise,resolve};}
export const flush=()=>new Promise(resolve=>setImmediate(resolve));
