import { HorizonStore } from "./lib/horizon-store.js";
import { extractVisionFields, extractAudioFields, extractMotionFields, browserSensorProfile } from "./sensor-adapters.js";

const $ = id => document.getElementById(id);
const store = new HorizonStore();
let socket = null;
let sessionSequence = 0;
let selected = null;
let reconnectTimer = null;
let cameraStream = null;
let cameraTimer = null;
let audioStream = null;
let audioTimer = null;
let cameraGeneration = 0;
let audioGeneration = 0;
let motionGeneration = 0;
let motionHandler = null;
let observationQueue = Promise.resolve();
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
function browserSessionId() {
  let id = sessionStorage.getItem("horizon.session");
  if (!id) { id = "browser-"+crypto.randomUUID(); sessionStorage.setItem("horizon.session", id); }
  return id;
}
function bridgeWsUrl() {
  const u=new URL(baseUrl()); u.protocol=u.protocol==="https:"?"wss:":"ws:"; u.pathname="/v1/brain/stream"; u.searchParams.set("after",String(store.revision));
  const token=$("token").value.trim(); if(token) u.searchParams.set("token",token); return u.toString();
}
async function get(path) {
  const response=await fetch(baseUrl()+path,{headers:authHeaders(),cache:"no-store"});
  const body=await response.json().catch(()=>({})); if(!response.ok) throw new Error(body.error||("HTTP "+response.status)); return body;
}
async function postObservation(envelope) {
  return enqueueObservation(async()=>{
    const payloadBytes=rawPayload(JSON.stringify(envelope.payload??null));
    envelope.payload_hash=await hashBuffer(payloadBytes);
    envelope.payload_bytes=payloadBytes.byteLength;
    const response=await fetch(baseUrl()+"/v1/observations",{method:"POST",headers:{"Content-Type":"application/json",...authHeaders()},body:JSON.stringify(envelope)});
    const body=await response.json().catch(()=>({})); if(!response.ok) throw new Error(body.error||("HTTP "+response.status)); return body;
  });
}
function enqueueObservation(send) {
  const result=observationQueue.then(send,send);
  observationQueue=result.catch(()=>{});
  return result;
}
function makeSensorObservation(modality, source, fields, capture) {
  return {modality, source:source+":"+browserSessionId()+":"+capture.id, timestamp:capture.timestamp, sample_id:nextId(modality), fields, confidence:.5,
    provenance:{...browserSensorProfile(modality), parent_sample_ids:[...(capture.parents ?? [])]}};
}
function postSensorObservation(sensor, current) {
  const send = async () => {
    if (!current()) throw new Error("capture stopped");
    const response = await fetch(baseUrl()+"/v1/sensor-observations", {method:"POST", headers:{"Content-Type":"application/json", ...authHeaders()}, body:JSON.stringify(sensor)});
    const body = await response.json().catch(() => ({}));
    if (!response.ok) throw new Error(body.error || ("HTTP "+response.status));
    return body;
  };
  return enqueueObservation(send);
}
function rawPayload(value) { return new TextEncoder().encode(value); }
async function hashBuffer(buffer) { const hash=await crypto.subtle.digest("SHA-256",buffer); return "sha256:"+Array.from(new Uint8Array(hash)).map(x=>x.toString(16).padStart(2,"0")).join(""); }
function makeEnvelope(modality, source, payload, extra={}) {
  const sessionId=browserSessionId();
  const eventId=nextId(modality);
  const tokens = modality === "text" || modality === "language" ? String(payload.raw ?? "").trim().split(/\s+/u).filter(Boolean) : [];
  return {schema_version:1,brain_identity:"horizon-primary-brain",event_id:eventId,session_id:sessionId,sequence:sessionSequence,timestamp:new Date().toISOString(),modality,source,provenance:{source,modality,capture_device:extra.capture_device||"",adapter:"github-pages-browser",synthetic:Boolean(extra.synthetic)},payload,...(tokens.length ? {tokens} : {}),...extra.payloadMeta};
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
  const units=s.neural_units||[],synapses=s.synapses||[];
  // Build a stable, topology-driven 3D layout rather than arranging populations in spirals.
  const ids=units.map(u=>Number(u.id)).filter(Number.isFinite), idSet=new Set(ids);
  const edges=synapses.map(e=>[Number(e.source_id),Number(e.target_id)]).filter(([a,b])=>idSet.has(a)&&idSet.has(b)&&a!==b);
  const layoutKey=ids.join(",")+"|"+edges.map(e=>e[0]+">"+e[1]).join(",");
  if(brain3D.layoutKey!==layoutKey){
    let seed=2166136261;for(let i=0;i<layoutKey.length;i++){seed^=layoutKey.charCodeAt(i);seed=Math.imul(seed,16777619);}
    const rand=()=>{seed^=seed<<13;seed^=seed>>>17;seed^=seed<<5;return (seed>>>0)/4294967296;};
    const points=new Map(ids.map(id=>[id,[(rand()-.5)*1.8,(rand()-.5)*1.8,(rand()-.5)*1.8]]));
    const degree=new Map(ids.map(id=>[id,0]));
    for(const [a,b] of edges){degree.set(a,(degree.get(a)||0)+1);degree.set(b,(degree.get(b)||0)+1);}
    const iterations=28, repulsion=.0018, spring=.012, target=.16;
    for(let it=0;it<iterations;it++){
      const forces=new Map(ids.map(id=>[id,[0,0,0]])),cool=.9*(1-it/iterations)+.1;
      // Repel nodes to prevent overlap; use softened inverse-square forces.
      for(let i=0;i<ids.length;i++)for(let j=i+1;j<ids.length;j++){
        const a=points.get(ids[i]),b=points.get(ids[j]),dx=a[0]-b[0],dy=a[1]-b[1],dz=a[2]-b[2];
        const d2=dx*dx+dy*dy+dz*dz+.012, f=repulsion/d2;
        const fa=forces.get(ids[i]),fb=forces.get(ids[j]);
        fa[0]+=dx*f;fa[1]+=dy*f;fa[2]+=dz*f;fb[0]-=dx*f;fb[1]-=dy*f;fb[2]-=dz*f;
      }
      // Connected neurons attract each other, with stronger pull for higher-degree nodes.
      for(const [aId,bId] of edges){
        const a=points.get(aId),b=points.get(bId),dx=b[0]-a[0],dy=b[1]-a[1],dz=b[2]-a[2],d=Math.sqrt(dx*dx+dy*dy+dz*dz)+1e-6;
        const f=(d-target)*spring,fa=forces.get(aId),fb=forces.get(bId);
        fa[0]+=dx/d*f;fa[1]+=dy/d*f;fa[2]+=dz/d*f;fb[0]-=dx/d*f;fb[1]-=dy/d*f;fb[2]-=dz/d*f;
      }
      for(const id of ids){const p=points.get(id),f=forces.get(id),mag=Math.sqrt(f[0]*f[0]+f[1]*f[1]+f[2]*f[2])+1e-6,step=Math.min(.035,.035/mag)*cool;p[0]+=f[0]*step;p[1]+=f[1]*step;p[2]+=f[2]*step;}
    }
    // Normalize the organic layout into the camera volume without imposing a shape.
    const vals=[...points.values()],maxAbs=Math.max(.01,...vals.flatMap(p=>p.map(Math.abs)));
    for(const p of vals)for(let k=0;k<3;k++)p[k]=p[k]/maxAbs*.92;
    brain3D.layoutKey=layoutKey;brain3D.layoutPositions=points;
  }
  const pos=brain3D.layoutPositions||new Map();
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
  (s.sensory_traces||[]).forEach((value,i)=>add("sensor trace "+i,value,"sensor_trace"));
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


$("ioSend").onclick=async()=>{const text=$("ioInput").value.trim();if(!text)return;$("ioStatus").textContent="sending...";try{const env=makeEnvelope("text","web-test-input",{raw:text},{synthetic:true}),response=await postObservation(env);$("ioStatus").textContent="accepted revision "+response.state_revision;$("ioInput").value="";await refreshSnapshotForEvent({state_revision:Number(response.state_revision),canonical_state_hash:response.canonical_state_hash});}catch(error){$("ioStatus").textContent="failed: "+error.message;}};
$("simSend").onclick=async()=>{
  const text=$("simText").value;if(!text.trim())return;
  try {
    const modality=$("simModality").value;
    if (modality === "text" || modality === "language") {
      await sendEnvelope(makeEnvelope(modality,"browser-simulator",{raw:text},{synthetic:true}),$("simStatus"));
      return;
    }
    const timestamp=new Date().toISOString(), input=JSON.parse(text);
    if (!input || Array.isArray(input) || typeof input !== "object") throw new Error("use numeric sensor JSON");
    const allowed=new Set(["fields","channel_states","spatial_coordinates","temporal_delta","confidence"]);
    if (Object.keys(input).some(key=>!allowed.has(key))) throw new Error("use numeric fields or channel arrays");
    if (![input.fields,input.channel_states,input.spatial_coordinates,input.temporal_delta].some(group=>Array.isArray(group)&&group.length)) throw new Error("numeric sensor state is required");
    const sensor={...input,modality,source:"browser-simulator:"+browserSessionId(),timestamp,sample_id:nextId("synthetic"),confidence:input.confidence??.5,
      provenance:{adapter:"browser-numeric-simulator-v1",calibration_id:"synthetic-normalized-v1"}};
    const response=await postSensorObservation(sensor,()=>true);
    $("simStatus").textContent="synthetic acknowledged r"+response.state_revision;
  } catch(error) { $("simStatus").textContent="failed: "+error.message; }
};
function stopCamera() {
  cameraGeneration++;
  if (cameraTimer) clearInterval(cameraTimer);
  if (cameraStream) cameraStream.getTracks().forEach(track => track.stop());
  cameraTimer = null; cameraStream = null; $("video").srcObject = null;
}
$("cameraStart").onclick=async()=>{
  stopCamera();
  const generation = cameraGeneration;
  const captureID = nextId("camera-capture");
  try {
    const stream = await navigator.mediaDevices.getUserMedia({video:true});
    if (generation !== cameraGeneration) { stream.getTracks().forEach(track => track.stop()); return; }
    cameraStream = stream; $("video").srcObject = stream;
    const canvas = document.createElement("canvas"), ctx = canvas.getContext("2d");
    let busy = false, previous = null, previousAt = 0, parent = null;
    cameraTimer = setInterval(async () => {
      if (busy || generation !== cameraGeneration || !$("video").videoWidth) return;
      busy = true;
      try {
        const ratio = Math.min(160/$("video").videoWidth, 120/$("video").videoHeight);
        canvas.width = Math.max(4, Math.round($("video").videoWidth*ratio));
        canvas.height = Math.max(4, Math.round($("video").videoHeight*ratio));
        ctx.drawImage($("video"), 0, 0, canvas.width, canvas.height);
        const capturedAt = performance.now(), timestamp = new Date().toISOString(), elapsedNS = previous ? Math.round((capturedAt-previousAt)*1e6) : 0;
        const measured = extractVisionFields(ctx.getImageData(0, 0, canvas.width, canvas.height).data, canvas.width, canvas.height, previous, elapsedNS);
        const sensor = makeSensorObservation("vision", "github-pages-camera", measured.fields, {id:captureID,timestamp,parents:parent ? [parent] : []});
        const response = await postSensorObservation(sensor, () => generation === cameraGeneration);
        if (generation === cameraGeneration) {
          previous = measured.state; previousAt = capturedAt; parent = sensor.sample_id;
          const quality = measured.fields.filter(field => field.kind.startsWith("light.flow.quality."));
          const supported = quality.filter(field => field.values[5]===1).length;
          $("cameraStatus").textContent = "acknowledged r"+response.state_revision+" · motion evidence "+(quality.length ? supported+"/"+quality.length : "pending");
        }
      } catch (error) { if (generation === cameraGeneration) $("cameraStatus").textContent = "failed: "+error.message; }
      finally { busy = false; }
    }, 1000);
    $("cameraStatus").textContent = "capturing";
  } catch (error) { if (generation === cameraGeneration) { stopCamera(); $("cameraStatus").textContent = "failed: "+error.message; } }
};
$("cameraStop").onclick=()=>{stopCamera(); $("cameraStatus").textContent="stopped";};
function stopAudio() {
  audioGeneration++;
  if (audioTimer) clearInterval(audioTimer);
  if (audioStream) audioStream.getTracks().forEach(track => track.stop());
  if (audioContext) void audioContext.close();
  if (waveFrame) cancelAnimationFrame(waveFrame);
  audioTimer = null; audioStream = null; audioContext = null; analyser = null; waveFrame = null;
}
$("audioStart").onclick=async()=>{
  stopAudio();
  const generation = audioGeneration;
  const captureID = nextId("audio-capture");
  try {
    const stream = await navigator.mediaDevices.getUserMedia({audio:true});
    if (generation !== audioGeneration) { stream.getTracks().forEach(track => track.stop()); return; }
    audioStream = stream; audioContext = new AudioContext();
    const context = audioContext, source = context.createMediaStreamSource(stream);
    analyser = context.createAnalyser(); analyser.fftSize = 16384; source.connect(analyser);
    const capture = analyser;
    await context.resume();
    if (generation !== audioGeneration) return;
    const draw = () => {
      if (generation !== audioGeneration) return;
      const data = new Float32Array(capture.fftSize); capture.getFloatTimeDomainData(data);
      const ctx = $("wave").getContext("2d"); ctx.clearRect(0,0,360,80); ctx.beginPath();
      for (let point = 0; point < 360; point++) {
        const value = data[Math.floor(point*data.length/360)], y = (1-value)*40;
        point ? ctx.lineTo(point,y) : ctx.moveTo(point,y);
      }
      ctx.stroke(); waveFrame = requestAnimationFrame(draw);
    };
    draw();
    let busy = false, parent = null;
    audioTimer = setInterval(async () => {
      if (busy || generation !== audioGeneration) return;
      busy = true;
      try {
        const samples = new Float32Array(capture.fftSize); capture.getFloatTimeDomainData(samples);
        const timestamp = new Date().toISOString();
        const fields = extractAudioFields(samples, context.sampleRate);
        const sensor = makeSensorObservation("audio", "github-pages-microphone", fields, {id:captureID,timestamp,parents:parent ? [parent] : []});
        const response = await postSensorObservation(sensor, () => generation === audioGeneration);
        if (generation === audioGeneration) { parent = sensor.sample_id; $("audioStatus").textContent = "acknowledged r"+response.state_revision; }
      } catch (error) { if (generation === audioGeneration) $("audioStatus").textContent = "failed: "+error.message; }
      finally { busy = false; }
    }, 1000);
    $("audioStatus").textContent = "capturing";
  } catch (error) { if (generation === audioGeneration) { stopAudio(); $("audioStatus").textContent = "failed: "+error.message; } }
};
$("audioStop").onclick=()=>{stopAudio(); $("audioStatus").textContent="stopped";};
function stopMotion() {
  motionGeneration++;
  if (motionHandler) window.removeEventListener("devicemotion",motionHandler);
  motionHandler=null;
}
$("motionStart").onclick=async()=>{
  stopMotion();
  const generation=motionGeneration,captureID=nextId("motion-capture");
  try {
    if (typeof DeviceMotionEvent === "undefined") throw new Error("motion sensor unavailable");
    if (typeof DeviceMotionEvent.requestPermission === "function") {
      const permission=await DeviceMotionEvent.requestPermission();
      if (generation !== motionGeneration) return;
      if (permission !== "granted") throw new Error("motion permission denied");
    }
    if (generation !== motionGeneration) return;
    let busy=false,lastSent=-Infinity,parent=null;
    motionHandler=async event=>{
      const now=performance.now();
      if (generation !== motionGeneration || busy || now-lastSent<500) return;
      busy=true;
      try {
        const timestamp=new Date().toISOString(),fields=extractMotionFields(event);
        const sensor=makeSensorObservation("motion","github-pages-motion",fields,{id:captureID,timestamp,parents:parent ? [parent] : []});
        $("motionData").textContent=JSON.stringify({acceleration:event.acceleration,accelerationIncludingGravity:event.accelerationIncludingGravity,rotationRate:event.rotationRate,interval:event.interval},null,2);
        lastSent=now;
        const response=await postSensorObservation(sensor,()=>generation===motionGeneration);
        if (generation === motionGeneration) {parent=sensor.sample_id;$("motionStatus").textContent="acknowledged r"+response.state_revision;}
      } catch(error) {if (generation === motionGeneration) $("motionStatus").textContent="failed: "+error.message;}
      finally {busy=false;}
    };
    window.addEventListener("devicemotion",motionHandler);
    $("motionStatus").textContent="capturing";
  } catch(error) {if (generation === motionGeneration) {stopMotion();$("motionStatus").textContent="failed: "+error.message;}}
};
$("motionStop").onclick=()=>{stopMotion();$("motionStatus").textContent="stopped";};

status("DISCONNECTED","disconnected");
