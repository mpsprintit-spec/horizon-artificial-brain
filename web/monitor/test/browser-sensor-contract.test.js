import test from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { browserSensorFixtures } from "./browser-sensor-fixtures.js";

test("Go fixtures contain actual production browser measurements",()=>{
  const committed=JSON.parse(readFileSync(new URL("../../../services/ai/runtime/testdata/browser_sensors_v1.json",import.meta.url),"utf8"));
  const measured=browserSensorFixtures();assert.equal(measured.length,committed.length);
  for(let i=0;i<measured.length;i++){
    const {fields:a,...metadataA}=measured[i],{fields:b,...metadataB}=committed[i];
    assert.deepEqual(metadataA,metadataB);assert.equal(a.length,b.length);
    for(let j=0;j<a.length;j++){
      const {values:x,...fieldA}=a[j],{values:y,...fieldB}=b[j];assert.deepEqual(fieldA,fieldB);assert.equal(x.length,y.length);
      // Transcendental audio math may differ by a few ulps across Node/V8
      // builds. This is far below sensor resolution; identities/timing stay exact.
      assert.ok(x.every((v,k)=>Math.abs(v-y[k])<=1e-12),"physical values changed");
    }
  }
});
