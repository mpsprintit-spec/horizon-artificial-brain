import assert from "node:assert/strict";
import { verifyInspectionPage } from "../web/monitor/inspection-page.js";
import { createHash } from "node:crypto";
import { monitorHarness } from "../web/monitor/test/app-harness.js";

const [base,token]=process.argv.slice(2),headers={Authorization:"Bearer "+token};
const digest=data=>Promise.resolve("sha256:"+createHash("sha256").update(data).digest("hex"));
const response=await fetch(base+"/v1/brain/inspection?limit=8",{headers});assert.equal(response.status,200);
const page=await verifyInspectionPage(await response.text(),digest);
assert.ok(page.data.total>=page.data.items.length);assert.equal(page.data.items.length,8);
assert.equal(page.canonical_state_hash,undefined);assert.equal(page.hash_scope,"inspection_page");
const paths=[],sockets=[];
class Store{snapshot=null;revision=0;status="disconnected";subscribe(){}resetForConnection(){}setSnapshot(s){this.revision=s.state_revision;}setStatus(s){this.status=s;}clearForReconnect(){}}
class Socket{constructor(url){sockets.push(url);}close(){this.onclose?.();}}
const h=monitorHarness({inspection:true,HorizonStore:Store,WebSocket:Socket,get:async(url,settings)=>{paths.push(new URL(url).pathname);return fetch(url,settings);}});
h.element("bridge").value=base;h.element("token").value=token;
await h.connect();assert.equal(sockets.length,1,h.element("source").textContent);
assert.equal(new URL(sockets[0]).searchParams.get("inspection"),"bounded");
assert.deepEqual(paths,["/v1/brain/inspection"]);h.disconnect();
console.log("Inspection smoke: authenticated bounded pages, Go/JS wire digest and production bounded monitor connection passed");
