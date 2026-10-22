import test from "node:test";
import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { readFileSync } from "node:fs";
import { verifyInspectionPage, inspectionGraph, readInspectionText } from "../inspection-page.js";
import { monitorHarness } from "./app-harness.js";

const wire=readFileSync(new URL("../testdata/inspection_page_v1.json",import.meta.url),"utf8");
const digest=data=>Promise.resolve("sha256:"+createHash("sha256").update(data).digest("hex"));

test("actual Go wire bytes verify despite HTML escaping and float formatting",async()=>{
  const page=await verifyInspectionPage(wire,digest);
  assert.equal(page.data.items.length,1);assert.ok(wire.includes("\\u003c"));
  const graph=inspectionGraph(page);
  assert.equal(graph.canonical_state_hash,"");assert.equal(graph.hash_scope,"inspection_page");
  assert.equal(graph.neural_units.length,1);assert.equal(graph.counts.neural_units,2);
  assert.ok(graph.inspection.next_cursor);assert.equal(graph.inspection_epoch,"wire-fixture-v1");
  await assert.rejects(verifyInspectionPage(wire.replace('"runtime_commit":"wire-fixture"','"runtime_commit":"changed"'),digest),/hash mismatch/);
  await assert.rejects(verifyInspectionPage(JSON.stringify(JSON.parse(wire)),digest),/hash mismatch/);
});

test("bounded response reader cancels an oversized stream",async()=>{
  let cancelled=false;
  const body=new ReadableStream({start(controller){controller.enqueue(new Uint8Array(300000));controller.enqueue(new Uint8Array(300000));},cancel(){cancelled=true;}});
  await assert.rejects(readInspectionText({body}),/byte limit/);assert.equal(cancelled,true);
  await assert.rejects(readInspectionText({headers:new Map([["content-length","9999999"]])}),/byte limit/);
  assert.equal(await readInspectionText(new Response(wire)),wire);
});

test("production connect uses bounded HTTP and WebSocket modes without full graph reads",async()=>{
  const paths=[],sockets=[];
  class Store{snapshot=null;revision=0;status="disconnected";subscribe(){}resetForConnection(){}setSnapshot(s){this.revision=s.state_revision;this.latest=s;}setStatus(s){this.status=s;}clearForReconnect(){}}
  class Socket{constructor(url){this.url=url;sockets.push(this);}close(){this.onclose?.();}}
  const h=monitorHarness({inspection:true,HorizonStore:Store,WebSocket:Socket,get:async url=>{paths.push(new URL(url).pathname);return new Response(wire);}});
  await h.connect();assert.equal(sockets.length,1,h.element("source").textContent);
  const url=new URL(sockets[0].url);assert.equal(url.searchParams.get("telemetry"),"revisions");assert.equal(url.searchParams.get("inspection"),"bounded");
  assert.deepEqual(paths,["/v1/brain/inspection"]);
  assert.match(h.element("source").textContent,/bounded/);
  h.disconnect();assert.equal(h.element("status").textContent,"DISCONNECTED");
});

test("invalid bounded integrity fails instead of falling back to a full snapshot",async()=>{
  const paths=[];
  class Store{snapshot=null;subscribe(){}setStatus(){}resetForConnection(){}}
  const h=monitorHarness({inspection:true,HorizonStore:Store,get:async url=>{paths.push(new URL(url).pathname);return new Response(wire.replace('"runtime_commit":"wire-fixture"','"runtime_commit":"tampered"'));}});
  await h.connect();assert.deepEqual(paths,["/v1/brain/inspection"]);assert.match(h.element("source").textContent,/hash mismatch/);
});
