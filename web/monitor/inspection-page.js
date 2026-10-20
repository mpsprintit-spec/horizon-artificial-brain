// Verify the original JSON bytes, not a JS reserialization with different
// number/key/escaping rules. page_hash is the last field in this wire schema.
export async function verifyInspectionPage(text, digest) {
  const wire=text.trimEnd(), page=JSON.parse(wire);
  if(page.type!=="brain.inspection.page" || page.schema_version!==1 || page.hash_scope!=="inspection_page")throw new Error("invalid inspection page contract");
  if(!/^sha256:[0-9a-f]{64}$/.test(page.page_hash??""))throw new Error("invalid inspection page hash");
  const suffix=',"page_hash":"'+page.page_hash+'"}';
  if(!wire.endsWith(suffix))throw new Error("inspection page hash framing mismatch");
  const payload=wire.slice(0,-suffix.length)+"}";
  if(await digest(new TextEncoder().encode(payload))!==page.page_hash)throw new Error("inspection page hash mismatch");
  if(!Number.isSafeInteger(page.state_revision)||page.state_revision<0 || !page.inspection_epoch)throw new Error("invalid inspection revision");
  if(!Number.isSafeInteger(page.byte_limit)||page.byte_limit<1||new TextEncoder().encode(text).byteLength>page.byte_limit || page.byte_limit>512*1024)throw new Error("inspection page exceeds byte limit");
  if(!Number.isSafeInteger(page.limit)||page.limit<1||page.limit>512||!Array.isArray(page.data?.items)||page.data.items.length>page.limit)throw new Error("inspection page exceeds item limit");
  return page;
}

export async function readInspectionText(response) {
  const max=512*1024;
  if(Number(response.headers?.get("content-length"))>max)throw new Error("inspection response exceeds byte limit");
  if(!response.body?.getReader)return response.text();
  const reader=response.body.getReader(),chunks=[];let size=0;
  try {
    for(;;){const {done,value}=await reader.read();if(done)break;size+=value.byteLength;if(size>max){await reader.cancel();throw new Error("inspection response exceeds byte limit");}chunks.push(value);}
  }finally{reader.releaseLock();}
  const data=new Uint8Array(size);let offset=0;for(const chunk of chunks){data.set(chunk,offset);offset+=chunk.byteLength;}
  return new TextDecoder("utf-8",{fatal:true}).decode(data);
}

export function inspectionGraph(page) {
  if(page.data?.section!=="neural_units")throw new Error("graph requires a neural-unit inspection page");
  return {type:"brain.snapshot",brain_identity:page.brain_identity,state_revision:page.state_revision,captured_at:page.captured_at,
    canonical_state_hash:"",hash_scope:page.hash_scope,page_hash:page.page_hash,inspection_epoch:page.inspection_epoch,
    counts:page.data.counts,neural_units:page.data.items,synapses:page.data.related_synapses??[],populations:[],temporal_patterns:[],experience_traces:[],sensory_traces:[],episodes:[],bootstrap_experiences:[],provenance:[],
    brain_state:page.data.state_preview??{},inspection:page};
}
