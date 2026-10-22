// Dirty revisions share one pending read. Full graph snapshots start at most
// once per interval and never overlap; event processing remains independent.
export function createSnapshotScheduler({load, apply, intervalMs=1000, now=()=>performance.now(), schedule=setTimeout, cancel=clearTimeout}) {
  let target=null, pending=null, timer=null, running=false, lastStart=-Infinity, generation=0;
  function arm() {
    if (running || timer!==null || !pending) return;
    timer=schedule(run,Math.max(0,lastStart+intervalMs-now()));
  }
  async function run() {
    timer=null; running=true; lastStart=now();
    const epoch=generation, wanted=target, waiting=pending;
    target=null; pending=null;
    try {
      const snapshot=await load();
      if (epoch!==generation) { waiting.resolve(null); return; }
      if (Number(snapshot.state_revision)<wanted.state_revision) throw new Error("snapshot revision is behind telemetry event");
      if (Number(snapshot.state_revision)===wanted.state_revision && wanted.canonical_state_hash && snapshot.canonical_state_hash!==wanted.canonical_state_hash) throw new Error("telemetry canonical hash mismatch");
      apply(snapshot);
      waiting.resolve(snapshot);
    } catch(error) {
      if (epoch===generation) waiting.reject(error); else waiting.resolve(null);
    } finally { running=false; arm(); }
  }
  return {
    request(event) {
      const revision=Number(event.state_revision);
      if (!Number.isSafeInteger(revision) || revision<0) return Promise.reject(new Error("invalid snapshot target revision"));
      if (!target || revision>target.state_revision) target={...event,state_revision:revision};
      else if (revision===target.state_revision && event.canonical_state_hash) target.canonical_state_hash=event.canonical_state_hash;
      if (!pending) {
        let resolve,reject;
        const promise=new Promise((ok,fail)=>{resolve=ok;reject=fail;});
        pending={promise,resolve,reject};
      }
      const result=pending.promise;
      arm();
      return result;
    },
    reset() {
      generation++; target=null; lastStart=-Infinity;
      if (timer!==null) cancel(timer);
      timer=null;
      pending?.resolve(null); pending=null;
    },
  };
}
