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
  $("prediction").textContent=JSON.stringify(s.brain_state?.prediction||{},null,2); $("error").textContent=JSON.stringify(s.brain_state?.error||{},null,2); $("memory").textContent=JSON.stringify(s.brain_state?.memory||{},null,2); $("plasticity").textContent=JSON.stringify(s.brain_state?.plasticity||{},null,2); $("curiosity").textContent=JSON.stringify(s.brain_state?.curiosity||{},null,2); $("provenance").textContent=JSON.stringify({provenance:s.provenance||[],episodes:s.episodes||[],bootstrap_experiences:s.bootstrap_experiences||[]},null,2); renderCognitivePanel(s);
  renderGraph(s); renderEvents(); renderInspectableState(s);
}

function renderGraph(s) {
  const canvas=$("brainGraph"); if(!canvas)return;
  if(!brain3D)initBrain3D(canvas);
  drawBrain3D(s);
}
let brain3D=null;
function initBrain3D(canvas){
  const gl=canvas.getContext("webgl",{antialias:true,alpha:false})||canvas.getContext("experimental-webgl");
  if(!gl){$("graphState").textContent="WebGL unavailable";return;}
  const nodeVS='attribute vec3 p;attribute float size;attribute vec3 color;uniform mat4 mvp;varying vec3 c;void main(){gl_Position=mvp*vec4(p,1.0);gl_PointSize=size;c=color;}';
  const nodeFS='precision mediump float;varying vec3 c;void main(){vec2 q=gl_PointCoord-vec2(.5);if(dot(q,q)>.25)discard;gl_FragColor=vec4(c,1.0);}';
  const lineVS='attribute vec3 p;uniform mat4 mvp;void main(){gl_Position=mvp*vec4(p,1.0);}';
  const lineFS='precision mediump float;uniform float alpha;void main(){gl_FragColor=vec4(.36,.70,.74,alpha);}';
  const compile=(type,src)=>{const sh=gl.createShader(type);gl.shaderSource(sh,src);gl.compileShader(sh);if(!gl.getShaderParameter(sh,gl.COMPILE_STATUS))throw new Error(gl.getShaderInfoLog(sh));return sh;};
  const program=(vs,fs)=>{const p=gl.createProgram();gl.attachShader(p,compile(gl.VERTEX_SHADER,vs));gl.attachShader(p,compile(gl.FRAGMENT_SHADER,fs));gl.linkProgram(p);if(!gl.getProgramParameter(p,gl.LINK_STATUS))throw new Error(gl.getProgramInfoLog(p));return p;};
  brain3D={gl,canvas,nodeProgram:program(nodeVS,nodeFS),lineProgram:program(lineVS,lineFS),yaw:.15,pitch:-.08,zoom:3.2,drag:false,lastX:0,lastY:0,projected:[]};
  canvas.addEventListener("pointerdown",e=>{brain3D.drag=true;brain3D.lastX=e.clientX;brain3D.lastY=e.clientY;canvas.setPointerCapture(e.pointerId);});
  canvas.addEventListener("pointermove",e=>{if(!brain3D.drag)return;brain3D.yaw+=(e.clientX-brain3D.lastX)*.008;brain3D.pitch+=(e.clientY-brain3D.lastY)*.008;brain3D.pitch=Math.max(-1.3,Math.min(1.3,brain3D.pitch));brain3D.lastX=e.clientX;brain3D.lastY=e.clientY;if(store.snapshot)drawBrain3D(store.snapshot);});
  canvas.addEventListener("pointerup",e=>{brain3D.drag=false;canvas.releasePointerCapture(e.pointerId);});
  canvas.addEventListener("wheel",e=>{e.preventDefault();brain3D.zoom=Math.max(1.5,Math.min(6,brain3D.zoom+e.deltaY*.002));if(store.snapshot)drawBrain3D(store.snapshot);},{passive:false});
  canvas.addEventListener("click",e=>{
    if(!brain3D.projected.length)return;
    const rect=canvas.getBoundingClientRect(),x=(e.clientX-rect.left)*canvas.width/rect.width,y=(rect.bottom-e.clientY)*canvas.height/rect.height;
    let best=null,dist=18; for(const p of brain3D.projected){const d=Math.hypot(p.x-x,p.y-y);if(d<dist){dist=d;best=p;}}
    if(best)inspect(best.entity,"neural_unit");
  });
}
function matPerspective(fovy,aspect,near,far){const f=1/Math.tan(fovy/2),nf=1/(near-far);return [f/aspect,0,0,0,0,f,0,0,0,0,(far+near)*nf,-1,0,0,2*far*near*nf,0];}
function matMul(a,b){const o=new Array(16).fill(0);for(let col=0;col<4;col++)for(let row=0;row<4;row++)for(let k=0;k<4;k++)o[col*4+row]+=a[k*4+row]*b[col*4+k];return o;}
function matRot(yaw,pitch){const cy=Math.cos(yaw),sy=Math.sin(yaw),cp=Math.cos(pitch),sp=Math.sin(pitch);return [cy,sy*sp,-sy*cp,0,0,cp,sp,0,sy,-cy*sp,cy*cp,0,0,0,0,1];}
function matTranslate(z){return [1,0,0,0,0,1,0,0,0,0,1,0,0,0,z,1];}
function drawBrain3D(s){
  if(!brain3D)return;
  const {gl,canvas}=brain3D,w0=canvas.clientWidth,h0=canvas.clientHeight,dpr=Math.min(2,window.devicePixelRatio||1),w=Math.max(1,Math.floor(w0*dpr)),h=Math.max(1,Math.floor(h0*dpr));
  if(canvas.width!==w||canvas.height!==h){canvas.width=w;canvas.height=h;}
  gl.viewport(0,0,w,h);gl.clearColor(.018,.035,.04,1);gl.clear(gl.COLOR_BUFFER_BIT|gl.DEPTH_BUFFER_BIT);gl.enable(gl.DEPTH_TEST);gl.enable(gl.BLEND);gl.blendFunc(gl.SRC_ALPHA,gl.ONE_MINUS_SRC_ALPHA);
  const mvp=matMul(matPerspective(1,w/h,.1,100),matMul(matTranslate(-brain3D.zoom),matRot(brain3D.yaw,brain3D.pitch)));
  const units=s.neural_units||[],synapses=s.synapses||[],pops=s.populations||[],byUnit=new Map();
  pops.forEach((p,pi)=>(p.units||[]).forEach(m=>{const id=Number(m.node_id??m.id);if(Number.isFinite(id)&&!byUnit.has(id))byUnit.set(id,pi);}));
  const centers=new Map(),pc=Math.max(1,Math.ceil(pops.length/2));
  pops.forEach((p,pi)=>{const side=pi%2?-1:1,slot=Math.floor(pi/2),u=slot/Math.max(1,pc-1),theta=(slot*2.3999632297)%6.28318,r=.35+.72*Math.sqrt(Math.min(1,u));centers.set(pi,[side*(.18+r*.62*Math.cos(theta)),.48*Math.sin(theta),.34*Math.cos(theta)*side]);});
  const pos=new Map(),groups=new Map();
  units.forEach(u=>{const pi=byUnit.get(Number(u.id));if(pi!==undefined){if(!groups.has(pi))groups.set(pi,[]);groups.get(pi).push(u);}});
  for(const [pi,members] of groups){const c=centers.get(pi)||[0,0,0],rad=Math.min(.22,.06+Math.sqrt(members.length)*.025);members.forEach((u,i)=>{const a=i*2.3999632297,r=rad*Math.sqrt((i+1)/members.length);pos.set(Number(u.id),[c[0]+Math.cos(a)*r,c[1]+Math.sin(a)*r*.72,c[2]+Math.sin(a*1.7)*r*.75]);});}
  units.filter(u=>!pos.has(Number(u.id))).forEach((u,i)=>{const a=i*2.3999632297,r=.2+Math.sqrt(i)*.015;pos.set(Number(u.id),[Math.cos(a)*Math.min(.9,r),Math.sin(a)*.55*Math.min(.9,r),Math.sin(a*.7)*.35]);});
  const active=new Set(),ev=store.events[store.events.length-1],delta=ev?.state_delta||ev?.StateDelta||{};
  for(const id of delta.added_node_ids||delta.AddedNodeIDs||[])active.add(Number(id));
  for(const id of Object.keys(delta.activation_delta||delta.ActivationDelta||{}))active.add(Number(id));
  const nodeData=[],projected=[];
  const project=p=>{const q=[mvp[0]*p[0]+mvp[1]*p[1]+mvp[2]*p[2]+mvp[3],mvp[4]*p[0]+mvp[5]*p[1]+mvp[6]*p[2]+mvp[7],mvp[8]*p[0]+mvp[9]*p[1]+mvp[10]*p[2]+mvp[11],mvp[12]*p[0]+mvp[13]*p[1]+mvp[14]*p[2]+mvp[15]];return q[3]>0?{x:(q[0]/q[3]*.5+.5)*w,y:(q[1]/q[3]*.5+.5)*h}:null;};
  for(const u of units){const p=pos.get(Number(u.id));if(!p)continue;const a=Math.max(0,Number(u.activation||0)),hot=active.has(Number(u.id));nodeData.push(...p,2.8+a*8+(hot?4:0),hot?1:.25+a*.65,hot?1:.72,.82);const q=project(p);if(q)projected.push({x:q.x,y:q.y,entity:u});}
  brain3D.projected=projected;
  const lineData=[];for(const e of synapses){const a=pos.get(Number(e.source_id)),b=pos.get(Number(e.target_id));if(a&&b)lineData.push(...a,...b);}
  const draw=(program,data,mode,stride,attrs)=>{const b=gl.createBuffer();gl.bindBuffer(gl.ARRAY_BUFFER,b);gl.bufferData(gl.ARRAY_BUFFER,new Float32Array(data),gl.DYNAMIC_DRAW);gl.useProgram(program);if(program===brain3D.lineProgram){gl.uniform1f(gl.getUniformLocation(program,"alpha"),.72);}let off=0;attrs.forEach(a=>{const loc=gl.getAttribLocation(program,a.name);if(loc>=0){gl.enableVertexAttribArray(loc);gl.vertexAttribPointer(loc,a.size,gl.FLOAT,false,stride*4,off);}off+=a.size*4;});gl.uniformMatrix4fv(gl.getUniformLocation(program,"mvp"),false,new Float32Array(mvp));gl.drawArrays(mode,0,data.length/stride);gl.deleteBuffer(b);};
  draw(brain3D.nodeProgram,nodeData,gl.POINTS,7,[{name:"p",size:3},{name:"size",size:1},{name:"color",size:3}]);
  draw(brain3D.lineProgram,lineData,gl.LINES,3,[{name:"p",size:3}]);
  const renderedConnections=lineData.length/6;
  $("graphState").textContent=`3D canonical brain · ${units.length} neurons · ${synapses.length} synapses · ${renderedConnections} rendered connections · drag to rotate · wheel to zoom`;
}

function inspect(entity,type){selected=Number(entity.id||entity.node_id||entity.source_id||0);$("inspector").textContent=JSON.stringify({type,entity},null,2);render();}
function renderEvents(){const root=$("events");root.textContent="";for(const e of [...store.events].reverse()){const div=document.createElement("div");div.className="event"+(Object.keys(e.plasticity||{}).length?" plasticity-event":"");div.textContent=String(e.type)+" revision="+String(e.state_revision)+" timestamp="+String(e.timestamp)+" hash="+String(e.event_hash||"unavailable");root.appendChild(div);}}

function renderCognitivePanel(s){
  const last=store.events[store.events.length-1],obs=last?.observation||last?.Observation;
  $("ioOutput").textContent=JSON.stringify({runtime_event:last?.type||null,state_revision:s.state_revision,canonical_state_hash:s.canonical_state_hash,observation:obs||null},null,2);
  const active=(s.neural_units||[]).filter(u=>Number(u.activation||0)>.5).sort((a,b)=>Number(b.activation||0)-Number(a.activation||0)).slice(0,12);
  $("thoughtState").textContent=JSON.stringify({status:"neural-state view",note:"Kata hanya ditampilkan sebagai pikiran aktual jika runtime mengekspos aktivasi simbolik. Saat ini panel menunjukkan keadaan neural aktif, prediksi, error, dan plasticity.",active_neural_units:active.map(u=>({id:u.id,activation:u.activation})),prediction:s.brain_state?.prediction||{},error:s.brain_state?.error||{},plasticity:s.brain_state?.plasticity||{},communication_bootstrap_words:s.brain_state?.memory?.communication_lexicon_words||0,state_revision:s.state_revision},null,2);
}
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


$("ioSend").onclick=async()=>{const text=$("ioInput").value.trim();if(!text)return;$("ioStatus").textContent="sending...";try{const payload=rawPayload(text),env=makeEnvelope("text","web-test-input",{raw:text},{synthetic:true,payloadMeta:{payload_hash:await hashBuffer(payload)}}),response=await postObservation(env);$("ioStatus").textContent="accepted revision "+response.state_revision;$("ioInput").value="";await refreshSnapshotForEvent({state_revision:Number(response.state_revision),canonical_state_hash:response.canonical_state_hash});}catch(error){$("ioStatus").textContent="failed: "+error.message;}};
$("simSend").onclick=async()=>{const text=$("simText").value;if(!text)return;const payload=rawPayload(text);const envelope=makeEnvelope($("simModality").value,"browser-simulator",{raw:text},{synthetic:true,payloadMeta:{payload_hash:await hashBuffer(payload)}});await sendEnvelope(envelope,$("simStatus"));};
$("cameraStart").onclick=async()=>{try{cameraStream=await navigator.mediaDevices.getUserMedia({video:true});$("video").srcObject=cameraStream;const canvas=document.createElement("canvas"),ctx=canvas.getContext("2d");cameraTimer=setInterval(async()=>{canvas.width=$("video").videoWidth||320;canvas.height=$("video").videoHeight||240;ctx.drawImage($("video"),0,0,canvas.width,canvas.height);const blob=await new Promise(resolve=>canvas.toBlob(resolve,"image/jpeg",.65));if(!blob)return;const data=await blob.arrayBuffer();const envelope=makeEnvelope("vision","github-pages-camera",{encoding:"image/jpeg;base64",data:binaryToBase64(data)},{capture_device:"browser-camera",payloadMeta:{payload_hash:await hashBuffer(data)}});sendEnvelope(envelope,$("cameraStatus")).catch(()=>{});},1000);$("cameraStatus").textContent="captured";}catch(error){$("cameraStatus").textContent="failed: "+error.message;}};
$("cameraStop").onclick=()=>{if(cameraTimer)clearInterval(cameraTimer);if(cameraStream)cameraStream.getTracks().forEach(t=>t.stop());cameraStream=null;$("cameraStatus").textContent="stopped";};
$("audioStart").onclick=async()=>{try{audioStream=await navigator.mediaDevices.getUserMedia({audio:true});audioContext=new AudioContext();const source=audioContext.createMediaStreamSource(audioStream);analyser=audioContext.createAnalyser();analyser.fftSize=256;source.connect(analyser);const draw=()=>{if(!analyser)return;const data=new Uint8Array(analyser.fftSize);analyser.getByteTimeDomainData(data);const ctx=$("wave").getContext("2d");ctx.clearRect(0,0,360,80);ctx.beginPath();data.forEach((v,i)=>{const x=i*360/data.length,y=(v/255)*80;i?ctx.lineTo(x,y):ctx.moveTo(x,y);});ctx.stroke();waveFrame=requestAnimationFrame(draw);};draw();recorder=new MediaRecorder(audioStream);recorder.ondataavailable=async e=>{if(!e.data.size)return;const data=await e.data.arrayBuffer();const envelope=makeEnvelope("audio","github-pages-microphone",{encoding:e.data.type,data:binaryToBase64(data)},{capture_device:"browser-microphone",payloadMeta:{payload_hash:await hashBuffer(data)}});sendEnvelope(envelope,$("audioStatus")).catch(()=>{});};recorder.start(1000);$("audioStatus").textContent="captured";}catch(error){$("audioStatus").textContent="failed: "+error.message;}};
$("audioStop").onclick=()=>{if(recorder)recorder.stop();if(audioStream)audioStream.getTracks().forEach(t=>t.stop());if(audioContext)audioContext.close();if(waveFrame)cancelAnimationFrame(waveFrame);recorder=null;audioStream=null;$("audioStatus").textContent="stopped";};
let lastMotionSent=0;
$("motionStart").onclick=async()=>{try{if(typeof DeviceMotionEvent!=="undefined"&&typeof DeviceMotionEvent.requestPermission==="function"){const p=await DeviceMotionEvent.requestPermission();if(p!=="granted")throw new Error("motion permission denied");}const handler=e=>{const now=performance.now();if(now-lastMotionSent<250)return;lastMotionSent=now;const payload={acceleration:e.acceleration,accelerationIncludingGravity:e.accelerationIncludingGravity,rotationRate:e.rotationRate,interval:e.interval};$("motionData").textContent=JSON.stringify(payload,null,2);const envelope=makeEnvelope("motion","github-pages-motion",payload,{capture_device:"browser-motion"});sendEnvelope(envelope,$("motionStatus")).catch(()=>{});};window.addEventListener("devicemotion",handler);$("motionStatus").textContent="captured";}catch(error){$("motionStatus").textContent="failed: "+error.message;}};

status("DISCONNECTED","disconnected");
