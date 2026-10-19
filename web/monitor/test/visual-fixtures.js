// Analytical image-plane fixtures. Independent motions are known test inputs,
// not depth/object labels made available to the production adapter.
function texture(x, y, seed) {
  let value = Math.imul(x+seed*521, 374761393)^Math.imul(y+seed*293, 668265263);
  value = Math.imul(value^(value>>>13), 1274126177);
  return 48+((value^(value>>>16))>>>0)%160;
}

export function visualFrame({width=120, height=120, dx=0, dy=0, regionDX=dx, regionDY=dy, brightness=0} = {}) {
  const frame = new Uint8ClampedArray(width*height*4);
  for (let y = 0; y < height; y++) for (let x = 0; x < width; x++) {
    const sx = x-regionDX, sy = y-regionDY;
    const local = sx >= 0 && sy >= 0 && sx < width/2 && sy < height/2;
    const value = brightness+(local ? texture(sx,sy,2) : texture(x-dx,y-dy,1));
    frame.set([value,value,value,255], 4*(y*width+x));
  }
  return frame;
}

export function uniformFrame(width=120, height=120, value=128) {
  const frame = new Uint8ClampedArray(width*height*4);
  for (let i = 0; i < frame.length; i += 4) frame.set([value,value,value,255],i);
  return frame;
}
