// Bounded image-plane measurements, not an object tracker or depth decoder.
// A fixed search window makes large/ambiguous changes explicitly unsupported.
const searchRadius = 6;
const patchRadius = 3;
const minimumTexture = .015;
const maximumResidual = .08;
const minimumUniqueness = .1;

function patch(image, width, height, x, y) {
  if (x-patchRadius < 0 || y-patchRadius < 0 || x+patchRadius >= width || y+patchRadius >= height) return null;
  const values = [];
  let sum = 0, squared = 0;
  for (let dy = -patchRadius; dy <= patchRadius; dy++) for (let dx = -patchRadius; dx <= patchRadius; dx++) {
    const value = image[(y+dy)*width+x+dx];
    values.push(value); sum += value; squared += value*value;
  }
  const mean = sum/values.length;
  return {values:values.map(value => value-mean), texture:Math.sqrt(Math.max(0, squared/values.length-mean*mean))};
}

function match(template, image, width, height, x, y) {
  const candidates = [];
  for (let dy = -searchRadius; dy <= searchRadius; dy++) for (let dx = -searchRadius; dx <= searchRadius; dx++) {
    const target = patch(image, width, height, x+dx, y+dy);
    if (!target) continue;
    let error = 0;
    for (let i = 0; i < target.values.length; i++) error += Math.abs(target.values[i]-template.values[i]);
    candidates.push({dx, dy, error:error/target.values.length});
  }
  candidates.sort((a,b) => a.error-b.error || (a.dx*a.dx+a.dy*a.dy)-(b.dx*b.dx+b.dy*b.dy) || a.dy-b.dy || a.dx-b.dx);
  const best = candidates[0];
  if (!best) return null;
  // Adjacent pixels may describe one minimum. Compare against the best
  // competing minimum outside its one-pixel neighborhood.
  const competitor = candidates.find(candidate => Math.max(Math.abs(candidate.dx-best.dx), Math.abs(candidate.dy-best.dy)) > 1);
  const uniqueness = competitor ? (competitor.error-best.error)/Math.max(competitor.error, 1e-8) : 0;
  return {...best, uniqueness};
}

function measure(previous, current, width, height, x, y) {
  const result = {dx:0, dy:0, texture:0, residual:1, uniqueness:0, cycle:1, boundary:1, supported:false};
  // Retain a complete search neighborhood, including for the reverse match.
  const margin = searchRadius+patchRadius;
  if (x < margin || y < margin || x+margin >= width || y+margin >= height) return result;
  const template = patch(previous, width, height, x, y);
  result.texture = Math.min(1, 2*template.texture);
  if (template.texture < minimumTexture) return result;
  const forward = match(template, current, width, height, x, y);
  if (!forward) return result;
  result.residual = Math.min(1, forward.error);
  result.uniqueness = forward.uniqueness;
  result.boundary = Math.max(Math.abs(forward.dx), Math.abs(forward.dy))/searchRadius;
  if (forward.error > maximumResidual || forward.uniqueness < minimumUniqueness || result.boundary >= 1) return result;
  const targetX = x+forward.dx, targetY = y+forward.dy;
  if (targetX < margin || targetY < margin || targetX+margin >= width || targetY+margin >= height) { result.boundary = 1; return result; }
  const target = patch(current, width, height, x+forward.dx, y+forward.dy);
  const backward = match(target, previous, width, height, x+forward.dx, y+forward.dy);
  if (!backward) return result;
  const cycleError = Math.hypot(forward.dx+backward.dx, forward.dy+backward.dy);
  result.cycle = Math.min(1, cycleError/2);
  result.supported = target.texture >= minimumTexture && backward.error <= maximumResidual && backward.uniqueness >= minimumUniqueness && Math.max(Math.abs(backward.dx), Math.abs(backward.dy)) < searchRadius && cycleError <= 1;
  if (result.supported) { result.dx = forward.dx; result.dy = forward.dy; }
  return result;
}

const median = values => {
  const ordered = [...values].sort((a,b) => a-b), middle = Math.floor(ordered.length/2);
  return ordered.length%2 ? ordered[middle] : (ordered[middle-1]+ordered[middle])/2;
};

export function extractVisualCorrespondence(previous, current, width, height, elapsedNS) {
  if (!previous || previous.length !== width*height || current.length !== width*height || !Number.isSafeInteger(elapsedNS) || elapsedNS <= 0 || elapsedNS > 60e9) return [];
  const measured = [];
  for (let gy = 0; gy < 4; gy++) for (let gx = 0; gx < 4; gx++) {
    const x0 = Math.floor(gx*width/4), x1 = Math.floor((gx+1)*width/4);
    const y0 = Math.floor(gy*height/4), y1 = Math.floor((gy+1)*height/4);
    measured.push({gx, gy, coordinates:[2*x0/width-1, 2*y0/height-1, 2*x1/width-1, 2*y1/height-1], ...measure(previous, current, width, height, Math.floor((x0+x1)/2), Math.floor((y0+y1)/2))});
  }
  const supported = measured.filter(item => item.supported);
  const referenceAvailable = supported.length >= 4 && new Set(supported.map(item=>item.gx)).size >= 2 && new Set(supported.map(item=>item.gy)).size >= 2;
  const referenceX = referenceAvailable ? median(supported.map(item=>item.dx)) : 0;
  const referenceY = referenceAvailable ? median(supported.map(item=>item.dy)) : 0;
  const fields = [];
  for (const [index, item] of measured.entries()) {
    const group = "patch-"+index;
    const add = (factor, values) => {
      // Geometry is part of the measurement coordinate system: six pixels
      // at different image resolutions must not silently share a scope.
      const kind = "light.flow."+factor+".r6px."+width+"x"+height+"-v1";
      fields.push({field_id:group+":"+kind, group_id:group, kind, values, coordinates:[...item.coordinates], temporal_offset_ns:-elapsedNS, temporal_scale_ns:elapsedNS});
    };
    // quality: texture, residual, uniqueness, cycle penalty, range usage,
    // measurement support, and support of the shared image-motion reference.
    add("quality", [item.texture, item.residual, item.uniqueness, item.cycle, item.boundary, Number(item.supported), referenceAvailable ? supported.length/16 : 0]);
    if (!item.supported) continue; // Unknown correspondence is not zero motion.
    add("displacement", [item.dx/searchRadius, item.dy/searchRadius]);
    if (referenceAvailable) add("relative", [(item.dx-referenceX)/(2*searchRadius), (item.dy-referenceY)/(2*searchRadius)]);
  }
  return fields;
}
