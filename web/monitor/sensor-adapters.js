// Physical measurement adapters only. They assign no object, word, emotion,
// metric distance or semantic identity. Returned values are relative sensor
// coordinates; downstream cognition owns learning and interpretation.
import { extractVisualCorrespondence } from "./visual-correspondence.js";

const bounded = value => Math.max(-1, Math.min(1, value));
const luma = (r, g, b) => .2126*r + .7152*g + .0722*b;

export function extractVisionFields(rgba, width, height, previous = null, elapsedNS = 0) {
  if (!Number.isInteger(width) || !Number.isInteger(height) || width < 4 || height < 4 || rgba.length !== width*height*4) throw new Error("invalid RGBA sensor frame");
  const fields = [], state = [], luminance = new Float32Array(width*height);
  const old = previous?.width === width && previous?.height === height && previous.state?.length === 16 ? previous.state : null;
  for (let gy = 0; gy < 4; gy++) for (let gx = 0; gx < 4; gx++) {
    const x0 = Math.floor(gx*width/4), x1 = Math.floor((gx+1)*width/4);
    const y0 = Math.floor(gy*height/4), y1 = Math.floor((gy+1)*height/4);
    let red = 0, green = 0, blue = 0, dx = 0, dy = 0, count = 0;
    for (let y = y0; y < y1; y++) for (let x = x0; x < x1; x++) {
      const i = 4*(y*width+x), r = rgba[i]/255, g = rgba[i+1]/255, b = rgba[i+2]/255;
      const base = luma(r, g, b);
      luminance[y*width+x] = base;
      red += r; green += g; blue += b; count++;
      if (x+1 < width) dx += luma(rgba[i+4]/255, rgba[i+5]/255, rgba[i+6]/255)-base;
      if (y+1 < height) { const j = i+4*width; dy += luma(rgba[j]/255, rgba[j+1]/255, rgba[j+2]/255)-base; }
    }
    const rgb = [red/count, green/count, blue/count], intensity = luma(...rgb);
    const region = gy*4+gx, group = "patch-"+region;
    const coordinates = [2*x0/width-1, 2*y0/height-1, 2*x1/width-1, 2*y1/height-1];
    const add = (kind, values, timing = {}) => fields.push({field_id:group+":"+kind, group_id:group, kind, values:values.map(bounded), coordinates:[...coordinates], ...timing});
    add("light.intensity.luma-v1", [2*intensity-1]);
    add("light.rgb-v1", rgb.map(v => 2*v-1));
    add("light.gradient-v1", [dx/count, dy/count]);
    add("spatial.position-v1", [(coordinates[0]+coordinates[2])/2, (coordinates[1]+coordinates[3])/2]);
    if (old && Number.isSafeInteger(elapsedNS) && elapsedNS > 0 && elapsedNS <= 60e9) {
      add("light.temporal-delta-v1", rgb.map((v, c) => v-old[region][c]), {temporal_offset_ns:-elapsedNS, temporal_scale_ns:elapsedNS});
    }
    state.push(rgb);
  }
  if (old) fields.push(...extractVisualCorrespondence(previous.luminance, luminance, width, height, elapsedNS));
  return {fields, state:{width, height, state, luminance}};
}

// Fixed physical frequency coordinates (125..2000 Hz) and a fixed -80..0 dB
// spectral-shape range relative to RMS. Goertzel measures bands without an ASR,
// classifier, pitch label or arbitrary hash. Windows retain capture timing.
export function extractAudioFields(samples, sampleRate) {
  if (!(sampleRate >= 8000 && sampleRate <= 192000) || samples.length < 64 || samples.length/sampleRate > 60) throw new Error("invalid acoustic analysis window");
  for (const value of samples) if (!Number.isFinite(value) || Math.abs(value) > 1) throw new Error("invalid acoustic sample");
  const fields = [], groups = 8;
  let previousRMS = null;
  for (let group = 0; group < groups; group++) {
    const start = Math.floor(group*samples.length/groups), end = Math.floor((group+1)*samples.length/groups), count = end-start;
    let sum = 0;
    for (let i = start; i < end; i++) sum += samples[i]*samples[i];
    const rms = Math.sqrt(sum/count), spectrum = [];
    for (let band = 1; band <= 16; band++) {
      const frequency = band*125, coefficient = 2*Math.cos(2*Math.PI*frequency/sampleRate);
      let q1 = 0, q2 = 0;
      for (let i = start; i < end; i++) {
        const window = .5-.5*Math.cos(2*Math.PI*(i-start)/Math.max(1, count-1));
        const q = window*samples[i] + coefficient*q1-q2; q2 = q1; q1 = q;
      }
      const amplitude = 4*Math.sqrt(Math.max(0, q1*q1+q2*q2-coefficient*q1*q2))/count;
      const relative = rms > 1e-8 ? amplitude/(Math.SQRT2*rms) : 0;
      const db = 20*Math.log10(Math.max(relative, 1e-4));
      spectrum.push(bounded(2*(db+80)/80-1));
    }
    const groupID = "window-"+group, offset = -Math.round((samples.length-start)/sampleRate*1e9);
    const scale = Math.round(count/sampleRate*1e9);
    const add = (kind, values) => fields.push({field_id:groupID+":"+kind, group_id:groupID, kind, values, temporal_offset_ns:offset, temporal_scale_ns:scale});
    add("audio.rms-v1", [bounded(2*rms-1)]);
    add("audio.spectral-shape.125-2000hz-v1", spectrum);
    if (previousRMS !== null) add("audio.onset-delta-v1", [bounded(rms-previousRMS)]);
    previousRMS = rms;
  }
  return fields;
}
