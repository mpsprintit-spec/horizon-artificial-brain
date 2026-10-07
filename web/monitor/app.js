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
let snapshotRefresh = Promise.resolve();

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
  const sessionId=sessionStorage.getItem("horizon.session")||"browser-"+crypto.randomUUID();
  sessionStorage.setItem("horizon.session",sessionId);
  const eventId=nextId(modality);
  return {schema_version:1,brain_identity:"horizon-primary-brain",event_id:eventId,session_id:sessionId,sequence:sessionSequence,timestamp:new Date().toISOString(),modality,source,provenance:{source,modality,capture_device:extra.capture_device||"",adapter:"github-pages-browser",synthetic:Boolean(extra.synthetic)},payload,...extra.payloadMeta};
}
async function sendEnvelope(envelope, uiStatus) {
  try { uiStatus.textContent="captured"; const response=await postObservation(envelope); uiStatus.textContent="acknowledged r"+response.state_revision; }
  catch(error) { uiStatus.textContent="failed: "+error.message; throw error; }
}
function render() {
  const s=store.snapshot; if(!s)return;
  $("brain").textContent=s.brain_identity||"unavailable"; $("revision").textContent=String(s.state_revision); $("hash").textContent=s.canonical_state_hash||"unavailable";
  $("units").textContent=String(s.counts?.neural_units??"unavailable"); $("populations").textContent=String(s.counts?.populations??"unavailable"); $("synapses").textContent=String(s.counts?.synapses??"unavailable");
  $("prediction").textContent=JSON.stringify(s.brain_state?.prediction||{},null,2); $("error").textContent=JSON.stringify(s.brain_state?.error||{},null,2); $("memory").textContent=JSON.stringify(s.brain_state?.memory||{},null,2); $("plasticity").textContent=JSON.stringify(s.brain_state?.plasticity||{},null,2); $("curiosity").textContent=JSON.stringify(s.brain_state?.curiosity||{},null,2); $("provenance").textContent=JSON.stringify({provenance:s.provenance||[],episodes:s.episodes||[],bootstrap_experiences:s.bootstrap_experiences||[]},null,2);
  renderGraph(s); renderEvents(); renderInspectableState(s);
}
function renderGraph(s) {
  const svg=$("brainGraph");
  while(svg.firstChild)svg.removeChild(svg.firstChild);

  const NS="http://www.w3.org/2000/svg";
  const units=s.neural_units||[], synapses=s.synapses||[], populations=s.populations||[];
  const W=1000,H=650, pos=new Map(), populationByUnit=new Map();

  // Visual shell: human-brain silhouette. It is presentation geometry only;
  // canonical neurons and synapses below still come exclusively from BrainRuntime.
  const shell=document.createElementNS(NS,"g");
  shell.classList.add("brain-shell");

  const left=document.createElementNS(NS,"path");
  left.setAttribute("d","M487 105 C430 55 330 52 235 78 C140 104 77 178 66 274 C54 374 92 470 172 523 C226 559 302 565 359 542 C404 525 445 494 487 454 Z");
  left.classList.add("brain-hemisphere");
  shell.appendChild(left);

  const right=document.createElementNS(NS,"path");
  right.setAttribute("d","M513 105 C570 55 670 52 765 78 C860 104 923 178 934 274 C946 374 908 470 828 523 C774 559 698 565 641 542 C596 525 555 494 513 454 Z");
  right.classList.add("brain-hemisphere");
  shell.appendChild(right);

  const fissure=document.createElementNS(NS,"path");
  fissure.setAttribute("d","M500 103 C482 180 493 242 500 305 C507 370 518 421 500 468");
  fissure.classList.add("brain-fissure");
  shell.appendChild(fissure);

  const cerebellum=document.createElementNS(NS,"path");
  cerebellum.setAttribute("d","M650 486 C700 455 785 462 826 506 C852 533 849 578 812 600 C756 632 676 610 647 570 C633 551 632 514 650 486 Z");
  cerebellum.classList.add("brain-cerebellum");
  shell.appendChild(cerebellum);

  const brainstem=document.createElementNS(NS,"path");
  brainstem.setAttribute("d","M613 515 C604 548 604 581 622 617 C632 637 651 639 661 623 C673 604 667 568 650 526");
  brainstem.classList.add("brain-stem");
  shell.appendChild(brainstem);
  svg.appendChild(shell);

  // Assign each actual unit to its actual population membership.
  populations.forEach((p,pi)=>{
    for(const member of (p.units||[])){
      const id=Number(member.node_id??member.id);
      if(Number.isFinite(id) && !populationByUnit.has(id)) populationByUnit.set(id,pi);
    }
  });

  const populationCenters=new Map();
  const slotsPerSide=Math.max(1,Math.ceil(populations.length/2));
  const cols=Math.min(6,Math.max(3,Math.ceil(Math.sqrt(slotsPerSide))));
  const rows=Math.ceil(slotsPerSide/cols);

  function slotCenter(pi){
    const side=pi%2===0?-1:1;
    const slot=Math.floor(pi/2);
    const col=slot%cols;
    const row=Math.floor(slot/cols);
    const x=500+side*(88+col*58);
    const y=130+row*(Math.min(330,rows>1?330/(rows-1):0));
    return {x,y};
  }

  populations.forEach((p,pi)=>populationCenters.set(pi,slotCenter(pi)));

  function brainClamp(x,y){
    const dx=(x-500)/425, dy=(y-315)/245;
    const d=dx*dx+dy*dy;
    if(d<=0.88)return {x,y};
    const k=Math.sqrt(0.88/d);
    return {x:500+(x-500)*k,y:315+(y-315)*k};
  }

  // Place members in compact deterministic clusters around their real population.
  const membersByPopulation=new Map();
  units.forEach(u=>{
    const id=Number(u.id);
    const pi=populationByUnit.get(id);
    if(pi!==undefined){
      if(!membersByPopulation.has(pi))membersByPopulation.set(pi,[]);
      membersByPopulation.get(pi).push(u);
    }
  });

  for(const [pi,members] of membersByPopulation){
    const center=populationCenters.get(pi);
    const radius=Math.min(38,12+Math.sqrt(members.length)*4.2);
    members.forEach((u,i)=>{
      const angle=i*2.399963229728653;
      const r=radius*Math.sqrt((i+1)/members.length);
      const p=brainClamp(center.x+Math.cos(angle)*r,center.y+Math.sin(angle)*r);
      pos.set(Number(u.id),p);
    });
  }

  // Units without population membership remain real units; give them a deterministic
  // interior position rather than inventing a synthetic connection or node.
  const orphaned=units.filter(u=>!pos.has(Number(u.id)));
  orphaned.forEach((u,i)=>{
    const angle=i*2.399963229728653;
    const r=55+Math.sqrt(i)*7;
    pos.set(Number(u.id),brainClamp(500+Math.cos(angle)*Math.min(350,r),315+Math.sin(angle)*Math.min(205,r)));
  });

  const active=new Set();
  const latestEvent=store.events[store.events.length-1];
  const latestDelta=latestEvent?.state_delta||latestEvent?.StateDelta||{};
  for(const id of latestDelta.added_node_ids||latestDelta.AddedNodeIDs||[])active.add(Number(id));
  for(const id of Object.keys(latestDelta.activation_delta||latestDelta.ActivationDelta||{}))active.add(Number(id));
  const plasticityEvent=Boolean(latestEvent&&Object.keys(latestEvent.plasticity||{}).length);

  // Population boundaries follow actual membership positions.
  for(const [pi,members] of membersByPopulation){
    const ps=members.map(u=>pos.get(Number(u.id))).filter(Boolean);
    if(!ps.length)continue;
    const center=ps.reduce((a,b)=>({x:a.x+b.x,y:a.y+b.y}),{x:0,y:0});
    center.x/=ps.length; center.y/=ps.length;
    const radius=Math.min(62,Math.max(18,Math.sqrt(ps.length)*6+10));
    const ring=document.createElementNS(NS,"circle");
    ring.setAttribute("cx",center.x); ring.setAttribute("cy",center.y); ring.setAttribute("r",radius);
    ring.classList.add("population");
    svg.appendChild(ring);
  }

  // Every rendered edge is an actual canonical synapse.
  for(const e of synapses){
    const a=pos.get(Number(e.source_id)),b=pos.get(Number(e.target_id));
    if(!a||!b)continue;
    const line=document.createElementNS(NS,"line");
    const weight=Number(e.weight||0);
    line.setAttribute("x1",a.x); line.setAttribute("y1",a.y);
    line.setAttribute("x2",b.x); line.setAttribute("y2",b.y);
    line.setAttribute("stroke-width",String(Math.max(.6,Math.min(6,.7+Math.abs(weight)*4))));
    line.classList.add("edge");
    if(e.inhibitory)line.classList.add("inhibitory");
    if(active.has(Number(e.source_id))||active.has(Number(e.target_id)))line.classList.add("edge-active");
    if(plasticityEvent)line.classList.add("plasticity");
    line.addEventListener("click",()=>inspect(e,"synapse"));
    svg.appendChild(line);
  }

  // Actual neurons are drawn last so the relationship network stays readable.
  for(const u of units){
    const id=Number(u.id),p=pos.get(id);
    if(!p)continue;
    const c=document.createElementNS(NS,"circle");
    const activation=Number(u.activation||0);
    c.setAttribute("cx",p.x); c.setAttribute("cy",p.y);
    c.setAttribute("r",String(2.5+Math.min(8,Math.abs(activation)*8)));
    c.setAttribute("fill","hsl("+Math.max(0,Math.min(120,120-activation*120))+" 72% 58%)");
    c.classList.add("node");
    if(active.has(id))c.classList.add("pulse");
    if(selected===id)c.classList.add("selected");
    c.addEventListener("click",()=>inspect(u,"neural_unit"));
    svg.appendChild(c);
  }

  $("graphState").textContent=store.events.length
    ? "brain-shaped canonical graph · last runtime event r"+store.events[store.events.length-1].state_revision
    : "brain-shaped canonical graph · no runtime event";
}
function inspect(entity,type){selected=Number(entity.id||entity.node_id||entity.source_id||0);$("inspector").textContent=JSON.stringify({type,entity},null,2);render();}
function renderEvents(){const root=$("events");root.textContent="";for(const e of [...store.events].reverse()){const div=document.createElement("div");div.className="event"+(Object.keys(e.plasticity||{}).length?" plasticity-event":"");div.textContent=String(e.type)+" revision="+String(e.state_revision)+" timestamp="+String(e.timestamp)+" hash="+String(e.event_hash||"unavailable");root.appendChild(div);}}
function renderInspectableState(s){
  const root=$("inspectList"); if(!root)return; root.textContent="";
  const add=(label,value,type)=>{const button=document.createElement("button");button.className="inspectItem";button.textContent=label;button.onclick=()=>{selected=null;$("inspector").textContent=JSON.stringify({type,value},null,2)};root.appendChild(button);};
  (s.populations||[]).forEach((value,i)=>add("population "+i,value,"population"));
  (s.temporal_patterns||[]).forEach((value,i)=>add("temporal trace "+i,value,"temporal_trace"));
  (s.experience_traces||[]).forEach((value,i)=>add("experience trace "+i,value,"experience_trace"));
  (s.provenance||[]).forEach((value,i)=>add("provenance "+i,value,"provenance"));
  add("prediction state",s.brain_state?.prediction||{},"prediction_error");
  add("plasticity state",s.brain_state?.plasticity||{},"plasticity");
}
async function refreshSnapshotForEvent(event){
  snapshotRefresh=snapshotRefresh.then(async()=>{
    const snapshot=await get("/v1/brain/snapshot");
    if(Number(snapshot.state_revision)<Number(event.state_revision))throw new Error("snapshot revision is behind telemetry event");
    if(Number(snapshot.state_revision)===Number(event.state_revision) && event.canonical_state_hash && snapshot.canonical_state_hash!==event.canonical_state_hash)throw new Error("telemetry canonical hash mismatch");
    store.setSnapshot(snapshot); render();
  });
  return snapshotRefresh;
}
async function resync(){store.setStatus("resyncing");try{const s=await get("/v1/brain/snapshot");store.setSnapshot(s);$("streamState").textContent="snapshot resynced";openStream();}catch(error){status("ERROR","error");$("source").textContent=error.message;}}
function openStream(){if(socket)socket.close();if(!baseUrl())return;store.clearForReconnect();socket=new WebSocket(bridgeWsUrl());socket.onopen=()=>{store.setStatus("connected");$("streamState").textContent="live WebSocket";status("CONNECTED","connected");};socket.onmessage=async event=>{try{const payload=JSON.parse(event.data);if(payload.type==="brain.handshake"){$("commit").textContent=payload.runtime_commit||"unavailable";return;}const result=store.applyEvent(payload);if(result==="gap")await resync();else if(result==="applied")await refreshSnapshotForEvent(payload);else render();}catch(error){store.setStatus("error");$("streamState").textContent=error.message;status("ERROR","error");}};socket.onerror=()=>{store.setStatus("error");$("streamState").textContent="stream error";};socket.onclose=()=>{if(store.status!=="resyncing"){store.setStatus("stale");$("streamState").textContent="reconnecting";reconnectTimer=setTimeout(openStream,2000);}};}
async function connect(){if(!baseUrl())return;sessionStorage.setItem("horizon.bridge",baseUrl());if($("token").value.trim())sessionStorage.setItem("horizon.token",$("token").value.trim());status("CONNECTING","stale");try{const handshake=await get("/v1/brain/handshake");$("commit").textContent=handshake.runtime_commit||"unavailable";const snapshot=await get("/v1/brain/snapshot");if(handshake.canonical_state_hash!==snapshot.canonical_state_hash||handshake.state_revision!==snapshot.state_revision)throw new Error("handshake/snapshot revision or hash mismatch");store.setSnapshot(snapshot);$("source").textContent="Horizon Bridge canonical runtime";render();openStream();}catch(error){store.setStatus("error");status("ERROR","error");$("source").textContent=error.message;}}
function disconnect(){if(reconnectTimer)clearTimeout(reconnectTimer);if(socket)socket.close();store.setStatus("disconnected");status("DISCONNECTED","disconnected");}
$("connect").onclick=connect;$("disconnect").onclick=disconnect;store.subscribe(render);

$("simSend").onclick=async()=>{const text=$("simText").value;if(!text)return;const payload=rawPayload(text);const envelope=makeEnvelope($("simModality").value,"browser-simulator",{raw:text},{synthetic:true,payloadMeta:{payload_hash:await hashBuffer(payload)}});await sendEnvelope(envelope,$("simStatus"));};
$("cameraStart").onclick=async()=>{try{cameraStream=await navigator.mediaDevices.getUserMedia({video:true});$("video").srcObject=cameraStream;const canvas=document.createElement("canvas"),ctx=canvas.getContext("2d");cameraTimer=setInterval(async()=>{canvas.width=$("video").videoWidth||320;canvas.height=$("video").videoHeight||240;ctx.drawImage($("video"),0,0,canvas.width,canvas.height);const blob=await new Promise(resolve=>canvas.toBlob(resolve,"image/jpeg",.65));if(!blob)return;const data=await blob.arrayBuffer();const envelope=makeEnvelope("vision","github-pages-camera",{encoding:"image/jpeg;base64",data:binaryToBase64(data)},{capture_device:"browser-camera",payloadMeta:{payload_hash:await hashBuffer(data)}});sendEnvelope(envelope,$("cameraStatus")).catch(()=>{});},1000);$("cameraStatus").textContent="captured";}catch(error){$("cameraStatus").textContent="failed: "+error.message;}};
$("cameraStop").onclick=()=>{if(cameraTimer)clearInterval(cameraTimer);if(cameraStream)cameraStream.getTracks().forEach(t=>t.stop());cameraStream=null;$("cameraStatus").textContent="stopped";};
$("audioStart").onclick=async()=>{try{audioStream=await navigator.mediaDevices.getUserMedia({audio:true});audioContext=new AudioContext();const source=audioContext.createMediaStreamSource(audioStream);analyser=audioContext.createAnalyser();analyser.fftSize=256;source.connect(analyser);const draw=()=>{if(!analyser)return;const data=new Uint8Array(analyser.fftSize);analyser.getByteTimeDomainData(data);const ctx=$("wave").getContext("2d");ctx.clearRect(0,0,360,80);ctx.beginPath();data.forEach((v,i)=>{const x=i*360/data.length,y=(v/255)*80;i?ctx.lineTo(x,y):ctx.moveTo(x,y);});ctx.stroke();waveFrame=requestAnimationFrame(draw);};draw();recorder=new MediaRecorder(audioStream);recorder.ondataavailable=async e=>{if(!e.data.size)return;const data=await e.data.arrayBuffer();const envelope=makeEnvelope("audio","github-pages-microphone",{encoding:e.data.type,data:binaryToBase64(data)},{capture_device:"browser-microphone",payloadMeta:{payload_hash:await hashBuffer(data)}});sendEnvelope(envelope,$("audioStatus")).catch(()=>{});};recorder.start(1000);$("audioStatus").textContent="captured";}catch(error){$("audioStatus").textContent="failed: "+error.message;}};
$("audioStop").onclick=()=>{if(recorder)recorder.stop();if(audioStream)audioStream.getTracks().forEach(t=>t.stop());if(audioContext)audioContext.close();if(waveFrame)cancelAnimationFrame(waveFrame);recorder=null;audioStream=null;$("audioStatus").textContent="stopped";};
let lastMotionSent=0;
$("motionStart").onclick=async()=>{try{if(typeof DeviceMotionEvent!=="undefined"&&typeof DeviceMotionEvent.requestPermission==="function"){const p=await DeviceMotionEvent.requestPermission();if(p!=="granted")throw new Error("motion permission denied");}const handler=e=>{const now=performance.now();if(now-lastMotionSent<250)return;lastMotionSent=now;const payload={acceleration:e.acceleration,accelerationIncludingGravity:e.accelerationIncludingGravity,rotationRate:e.rotationRate,interval:e.interval};$("motionData").textContent=JSON.stringify(payload,null,2);const envelope=makeEnvelope("motion","github-pages-motion",payload,{capture_device:"browser-motion"});sendEnvelope(envelope,$("motionStatus")).catch(()=>{});};window.addEventListener("devicemotion",handler);$("motionStatus").textContent="captured";}catch(error){$("motionStatus").textContent="failed: "+error.message;}};

status("DISCONNECTED","disconnected");
