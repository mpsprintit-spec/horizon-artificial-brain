import { HorizonStore } from "./lib/horizon-store.js";

const $ = id => document.getElementById(id);
const store = new HorizonStore();
let socket = null;
let sessionSequence = 0;
let selected = null;
let reconnectTimer = null;
let cameraStream = null;
let cameraTimer = null;
let audioStream = null;
let recorder = null;
let audioContext = null;
let analyser = null;
let waveFrame = null;

const savedBridge = sessionStorage.getItem("horizon.bridge");
const savedToken = sessionStorage.getItem("horizon.token");
if (savedBridge) $("bridge").value = savedBridge;
if (savedToken) $("token").value = savedToken;

function status(text, cls) { $("status").textContent=text; $("status").className="status "+cls; }
function baseUrl() { return $("bridge").value.trim().replace(/\/$/,""); }
function authHeaders() { const token=$("token").value.trim(); return token ? {Authorization:"Bearer "+token} : {}; }
function nextId(prefix) { sessionSequence += 1; return prefix+"-"+sessionSequence+"-"+crypto.randomUUID(); }
function bridgeWsUrl() {
  const u=new URL(baseUrl()); u.protocol=u.protocol==="https:"?"wss:":"ws:"; u.pathname="/v1/brain/stream"; u.searchParams.set("after",String(store.revision));
  const token=$("token").value.trim(); if(token) u.searchParams.set("token",token); return u.toString();
}
async function get(path) {
  const response=await fetch(baseUrl()+path,{headers:authHeaders(),cache:"no-store"});
  const body=await response.json().catch(()=>({})); if(!response.ok) throw new Error(body.error||("HTTP "+response.status)); return body;
}
async function postObservation(envelope) {
  const payloadBytes=rawPayload(JSON.stringify(envelope.payload??null));
  envelope.payload_hash=await hashBuffer(payloadBytes);
  envelope.payload_bytes=payloadBytes.byteLength;
  const response=await fetch(baseUrl()+"/v1/observations",{method:"POST",headers:{"Content-Type":"application/json",...authHeaders()},body:JSON.stringify(envelope)});
  const body=await response.json().catch(()=>({})); if(!response.ok) throw new Error(body.error||("HTTP "+response.status)); return body;
}
function binaryToBase64(buffer) { const bytes=new Uint8Array(buffer); let out=""; const step=0x8000; for(let i=0;i<bytes.length;i+=step)out+=String.fromCharCode(...bytes.subarray(i,Math.min(i+step,bytes.length))); return btoa(out); }
function rawPayload(value) { return new TextEncoder().encode(value); }
async function hashBuffer(buffer) { const hash=await crypto.subtle.digest("SHA-256",buffer); return "sha256:"+Array.from(new Uint8Array(hash)).map(x=>x.toString(16).padStart(2,"0")).join(""); }
function makeEnvelope(modality, source, payload, extra={}) {
  return {schema_version:1,brain_identity:"horizon-primary-brain",event_id:nextId(modality),session_id:sessionStorage.getItem("horizon.session")||"browser-"+crypto.randomUUID(),timestamp:new Date().toISOString(),modality,source,provenance:{source,modality,capture_device:extra.capture_device||"",adapter:"github-pages-browser",synthetic:Boolean(extra.synthetic)},payload,...extra.payloadMeta};
}
async function sendEnvelope(envelope, uiStatus) {
  try { uiStatus.textContent="captured"; const response=await postObservation(envelope); uiStatus.textContent="acknowledged r"+response.state_revision; }
  catch(error) { uiStatus.textContent="failed: "+error.message; throw error; }
}
function render() {
  const s=store.snapshot; if(!s)return;
  $("brain").textContent=s.brain_identity||"unavailable"; $("revision").textContent=String(s.state_revision); $("hash").textContent=s.canonical_state_hash||"unavailable";
  $("units").textContent=String(s.counts?.neural_units??"unavailable"); $("populations").textContent=String(s.counts?.populations??"unavailable"); $("synapses").textContent=String(s.counts?.synapses??"unavailable");
  $("prediction").textContent=JSON.stringify(s.brain_state?.prediction||{},null,2); $("error").textContent=JSON.stringify(s.brain_state?.error||{},null,2); $("memory").textContent=JSON.stringify(s.brain_state?.memory||{},null,2); $("plasticity").textContent=JSON.stringify(s.brain_state?.plasticity||{},null,2); $("curiosity").textContent=JSON.stringify(s.brain_state?.curiosity||{},null,2); $("provenance").textContent=JSON.stringify(s.brain_state?.self_model||{},null,2);
  renderGraph(s); renderEvents();
}
function renderGraph(s) {
  const svg=$("brainGraph"); while(svg.firstChild)svg.removeChild(svg.firstChild);
  const units=s.neural_units||[], synapses=s.synapses||[], populations=s.populations||[]; const pos=new Map(),cx=500,cy=325,rad=Math.min(270,Math.max(80,280-Math.min(units.length,200)*.35));
  units.forEach((u,i)=>{const a=units.length?2*Math.PI*i/units.length:0;pos.set(Number(u.id),{x:cx+rad*Math.cos(a),y:cy+rad*Math.sin(a)});});
  for(const p of populations){const ids=(p.units||[]).map(x=>Number(x.node_id)),ps=ids.map(id=>pos.get(id)).filter(Boolean);if(!ps.length)continue;const avg=ps.reduce((a,b)=>({x:a.x+b.x,y:a.y+b.y}),{x:0,y:0});avg.x/=ps.length;avg.y/=ps.length;const ring=document.createElementNS("http://www.w3.org/2000/svg","circle");ring.setAttribute("cx",avg.x);ring.setAttribute("cy",avg.y);ring.setAttribute("r",String(Math.min(120,35+ps.length*5)));ring.classList.add("population");svg.appendChild(ring);}
  for(const e of synapses){const a=pos.get(Number(e.source_id)),b=pos.get(Number(e.target_id));if(!a||!b)continue;const line=document.createElementNS("http://www.w3.org/2000/svg","line");line.setAttribute("x1",a.x);line.setAttribute("y1",a.y);line.setAttribute("x2",b.x);line.setAttribute("y2",b.y);line.setAttribute("stroke-width",String(Math.max(1,Math.min(8,Math.abs(Number(e.weight||0))*6))));line.classList.add("edge");if(e.inhibitory)line.classList.add("inhibitory");line.addEventListener("click",()=>inspect(e,"synapse"));svg.appendChild(line);}
  const active=new Set(); for(const event of store.events){const d=event.state_delta||event.StateDelta||{};for(const id of d.added_node_ids||d.AddedNodeIDs||[])active.add(Number(id));for(const id of Object.keys(d.activation_delta||d.ActivationDelta||{}))active.add(Number(id));}
  for(const u of units){const id=Number(u.id),p=pos.get(id);if(!p)continue;const c=document.createElementNS("http://www.w3.org/2000/svg","circle"),activation=Number(u.activation||0);c.setAttribute("cx",p.x);c.setAttribute("cy",p.y);c.setAttribute("r",String(3+Math.min(10,Math.abs(activation)*10)));c.setAttribute("fill","hsl("+Math.max(0,Math.min(120,120-activation*120))+" 65% 55%)");c.classList.add("node");if(active.has(id))c.classList.add("pulse");if(selected===id)c.classList.add("selected");c.addEventListener("click",()=>inspect(u,"neural_unit"));svg.appendChild(c);}
  $("graphState").textContent=store.events.length?"last runtime event r"+store.events[store.events.length-1].state_revision:"no runtime event";
}
function inspect(entity,type){selected=Number(entity.id||entity.node_id||entity.source_id||0);$("inspector").textContent=JSON.stringify({type,entity},null,2);render();}
function renderEvents(){const root=$("events");root.textContent="";for(const e of [...store.events].reverse()){const div=document.createElement("div");div.className="event";div.textContent=String(e.type)+" revision="+String(e.state_revision)+" timestamp="+String(e.timestamp)+" hash="+String(e.event_hash||"unavailable");root.appendChild(div);}}
async function resync(){store.setStatus("resyncing");try{const s=await get("/v1/brain/snapshot");store.setSnapshot(s);$("streamState").textContent="snapshot resynced";openStream();}catch(error){status("ERROR","error");$("source").textContent=error.message;}}
function openStream(){if(socket)socket.close();if(!baseUrl())return;store.clearForReconnect();socket=new WebSocket(bridgeWsUrl());socket.onopen=()=>{store.setStatus("connected");$("streamState").textContent="live WebSocket";status("CONNECTED","connected");};socket.onmessage=async event=>{try{const payload=JSON.parse(event.data);if(payload.type==="brain.handshake"){$("commit").textContent=payload.runtime_commit||"unavailable";return;}const result=store.applyEvent(payload);if(result==="gap")await resync();else render();}catch(error){store.setStatus("error");$("streamState").textContent=error.message;}};socket.onerror=()=>{store.setStatus("error");$("streamState").textContent="stream error";};socket.onclose=()=>{if(store.status!=="resyncing"){store.setStatus("stale");$("streamState").textContent="reconnecting";reconnectTimer=setTimeout(openStream,2000);}};}
async function connect(){if(!baseUrl())return;sessionStorage.setItem("horizon.bridge",baseUrl());if($("token").value.trim())sessionStorage.setItem("horizon.token",$("token").value.trim());status("CONNECTING","stale");try{const handshake=await get("/v1/brain/handshake");$("commit").textContent=handshake.runtime_commit||"unavailable";const snapshot=await get("/v1/brain/snapshot");if(handshake.canonical_state_hash!==snapshot.canonical_state_hash||handshake.state_revision!==snapshot.state_revision)throw new Error("handshake/snapshot revision or hash mismatch");store.setSnapshot(snapshot);$("source").textContent="Horizon Bridge canonical runtime";render();openStream();}catch(error){store.setStatus("error");status("ERROR","error");$("source").textContent=error.message;}}
function disconnect(){if(reconnectTimer)clearTimeout(reconnectTimer);if(socket)socket.close();store.setStatus("disconnected");status("DISCONNECTED","disconnected");}
$("connect").onclick=connect;$("disconnect").onclick=disconnect;store.subscribe(render);

$("simSend").onclick=async()=>{const text=$("simText").value;if(!text)return;const payload=rawPayload(text);const envelope=makeEnvelope($("simModality").value,"browser-simulator",{raw:text},{synthetic:true,payloadMeta:{payload_hash:await hashBuffer(payload)}});await sendEnvelope(envelope,$("simStatus"));};
$("cameraStart").onclick=async()=>{try{cameraStream=await navigator.mediaDevices.getUserMedia({video:true});$("video").srcObject=cameraStream;const canvas=document.createElement("canvas"),ctx=canvas.getContext("2d");cameraTimer=setInterval(async()=>{canvas.width=$("video").videoWidth||320;canvas.height=$("video").videoHeight||240;ctx.drawImage($("video"),0,0,canvas.width,canvas.height);const blob=await new Promise(resolve=>canvas.toBlob(resolve,"image/jpeg",.65));if(!blob)return;const data=await blob.arrayBuffer();const envelope=makeEnvelope("vision","github-pages-camera",{encoding:"image/jpeg;base64",data:binaryToBase64(data)},{capture_device:"browser-camera",payloadMeta:{payload_hash:await hashBuffer(data)}});sendEnvelope(envelope,$("cameraStatus")).catch(()=>{});},1000);$("cameraStatus").textContent="captured";}catch(error){$("cameraStatus").textContent="failed: "+error.message;}};
$("cameraStop").onclick=()=>{if(cameraTimer)clearInterval(cameraTimer);if(cameraStream)cameraStream.getTracks().forEach(t=>t.stop());cameraStream=null;$("cameraStatus").textContent="stopped";};
$("audioStart").onclick=async()=>{try{audioStream=await navigator.mediaDevices.getUserMedia({audio:true});audioContext=new AudioContext();const source=audioContext.createMediaStreamSource(audioStream);analyser=audioContext.createAnalyser();analyser.fftSize=256;source.connect(analyser);const draw=()=>{if(!analyser)return;const data=new Uint8Array(analyser.fftSize);analyser.getByteTimeDomainData(data);const ctx=$("wave").getContext("2d");ctx.clearRect(0,0,360,80);ctx.beginPath();data.forEach((v,i)=>{const x=i*360/data.length,y=(v/255)*80;i?ctx.lineTo(x,y):ctx.moveTo(x,y);});ctx.stroke();waveFrame=requestAnimationFrame(draw);};draw();recorder=new MediaRecorder(audioStream);recorder.ondataavailable=async e=>{if(!e.data.size)return;const data=await e.data.arrayBuffer();const envelope=makeEnvelope("audio","github-pages-microphone",{encoding:e.data.type,data:binaryToBase64(data)},{capture_device:"browser-microphone",payloadMeta:{payload_hash:await hashBuffer(data)}});sendEnvelope(envelope,$("audioStatus")).catch(()=>{});};recorder.start(1000);$("audioStatus").textContent="captured";}catch(error){$("audioStatus").textContent="failed: "+error.message;}};
$("audioStop").onclick=()=>{if(recorder)recorder.stop();if(audioStream)audioStream.getTracks().forEach(t=>t.stop());if(audioContext)audioContext.close();if(waveFrame)cancelAnimationFrame(waveFrame);recorder=null;audioStream=null;$("audioStatus").textContent="stopped";};
$("motionStart").onclick=async()=>{try{if(typeof DeviceMotionEvent!=="undefined"&&typeof DeviceMotionEvent.requestPermission==="function"){const p=await DeviceMotionEvent.requestPermission();if(p!=="granted")throw new Error("motion permission denied");}const handler=e=>{const payload={acceleration:e.acceleration,accelerationIncludingGravity:e.accelerationIncludingGravity,rotationRate:e.rotationRate,interval:e.interval};$("motionData").textContent=JSON.stringify(payload,null,2);const envelope=makeEnvelope("motion","github-pages-motion",payload,{capture_device:"browser-motion"});sendEnvelope(envelope,$("motionStatus")).catch(()=>{});};window.addEventListener("devicemotion",handler);$("motionStatus").textContent="captured";}catch(error){$("motionStatus").textContent="failed: "+error.message;}};

status("DISCONNECTED","disconnected");
