import test from "node:test";
import assert from "node:assert/strict";
import { createSnapshotScheduler } from "../snapshot-scheduler.js";
import { deferred, flush, monitorHarness } from "./app-harness.js";

function clock() {
  let now=0,id=0;
  const timers=new Map();
  return {timers,now:()=>now,schedule:(callback,delay)=>{timers.set(++id,{callback,at:now+delay});return id;},cancel:key=>timers.delete(key),
    async advance(value){now=value;for(const [key,timer] of [...timers])if(timer.at<=now){timers.delete(key);timer.callback();}await flush();}};
}
const event=state_revision=>({state_revision});
const snapshot=state_revision=>({state_revision,canonical_state_hash:"sha256:"+state_revision});

test("bursts share reads; full snapshots start at most once a second and never overlap",async()=>{
  const c=clock(),applied=[],responses=[];
  const scheduler=createSnapshotScheduler({...c,load:()=>{const d=deferred();responses.push(d);return d.promise;},apply:s=>applied.push(s.state_revision)});
  const first=scheduler.request(event(1));
  for(let revision=2;revision<=100;revision++)assert.equal(scheduler.request(event(revision)),first);
  assert.equal(c.timers.size,1);
  await c.advance(0);assert.equal(responses.length,1);
  const next=scheduler.request(event(101));
  await c.advance(2000);assert.equal(responses.length,1,"overlapping full snapshot");
  responses[0].resolve(snapshot(100));await first;
  await c.advance(2000);assert.equal(responses.length,2);
  responses[1].resolve(snapshot(101));await next;
  assert.deepEqual(applied,[100,101]);
  const last=scheduler.request(event(102));
  await c.advance(2999);assert.equal(responses.length,2);
  await c.advance(3000);assert.equal(responses.length,3);
  responses[2].resolve(snapshot(102));await last;
  assert.equal(c.timers.size,0,"clean monitor keeps polling full graphs");
});

test("failed reads and integrity mismatches do not poison future refreshes",async()=>{
  const c=clock();let result=snapshot(1);const applied=[];
  const scheduler=createSnapshotScheduler({...c,load:async()=>result,apply:s=>applied.push(s)});
  const mismatch=scheduler.request({state_revision:1,canonical_state_hash:"sha256:wrong"});
  const rejected=assert.rejects(mismatch,/canonical hash mismatch/);
  await c.advance(0);await rejected;assert.equal(applied.length,0);
  const behind=scheduler.request(event(3));const behindRejected=assert.rejects(behind,/behind telemetry event/);
  await c.advance(1000);await behindRejected;
  result=snapshot(4);const retry=scheduler.request(event(4));await c.advance(2000);await retry;
  assert.equal(applied.length,1);
});

test("reconnect discards old in-flight graph and cancels pending work",async()=>{
  const c=clock(),old=deferred(),applied=[];let loads=0;
  const scheduler=createSnapshotScheduler({...c,load:()=>++loads===1?old.promise:Promise.resolve(snapshot(1)),apply:s=>applied.push(s.state_revision)});
  const first=scheduler.request(event(9));await c.advance(0);
  const queued=scheduler.request(event(10));scheduler.reset();assert.equal(await queued,null);
  const current=scheduler.request(event(1));old.resolve(snapshot(10));assert.equal(await first,null);
  await c.advance(0);await current;assert.deepEqual(applied,[1]);
  const pending=scheduler.request(event(2));scheduler.reset();assert.equal(await pending,null);assert.equal(c.timers.size,0);
});

test("production connect negotiates revisions only when advertised",async()=>{
  for(const supported of [false,true]) {
    const sockets=[];
    class Store {snapshot=null;revision=1;status="disconnected";subscribe(){}setSnapshot(){}resetForConnection(){}setStatus(status){this.status=status;}clearForReconnect(){} }
    class Socket {constructor(url){sockets.push(url);}close(){}}
    const h=monitorHarness({HorizonStore:Store,WebSocket:Socket,get:async url=>({ok:true,json:async()=>url.endsWith("handshake")?{state_revision:1,canonical_state_hash:"sha256:1",capabilities:supported?["revision_stream"]:[]}:{state_revision:1,canonical_state_hash:"sha256:1"}})});
    await h.connect();
    assert.equal(sockets.length,1,h.element("source").textContent);
    assert.equal(new URL(sockets[0]).searchParams.get("telemetry"),supported?"revisions":null);
    h.disconnect();
  }
});
