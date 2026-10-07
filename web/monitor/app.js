(() => {
  const $ = id => document.getElementById(id);
  const bridgeUrl = $("bridgeUrl"), token = $("token"), status = $("status");
  const graph = $("brainGraph");
  let snapshot = null, selected = null, timer = null;

  bridgeUrl.value = sessionStorage.getItem("horizon.bridge") || "";
  token.value = sessionStorage.getItem("horizon.token") || "";

  function setStatus(text, cls) {
    status.textContent = text;
    status.className = "status " + cls;
  }
  function baseUrl() {
    return bridgeUrl.value.trim().replace(/\/$/, "");
  }
  function headers() {
    const h = {};
    const t = token.value.trim();
    if (t) h.Authorization = "Bearer " + t;
    return h;
  }
  async function get(path) {
    const response = await fetch(baseUrl() + path, {headers: headers(), cache: "no-store"});
    const body = await response.json().catch(() => ({}));
    if (!response.ok) throw new Error(body.error || ("HTTP " + response.status));
    return body;
  }
  function text(id, value) { $(id).textContent = value ?? "unavailable"; }
  function renderState(s) {
    text("brainIdentity", s.brain_identity);
    text("revision", s.state_revision);
    text("hash", s.canonical_state_hash);
    text("captured", s.captured_at);
    text("units", s.counts.neural_units);
    text("populations", s.counts.populations);
    text("synapses", s.counts.synapses);
    text("lastUpdate", "Canonical snapshot revision " + s.state_revision + " captured " + s.captured_at);
    text("prediction", JSON.stringify(s.brain_state.prediction || {}, null, 2));
    text("error", JSON.stringify(s.brain_state.error || {}, null, 2));
    text("memory", JSON.stringify(s.brain_state.memory || {}, null, 2));
    text("plasticity", JSON.stringify(s.brain_state.plasticity || {}, null, 2));
    text("curiosity", JSON.stringify(s.brain_state.curiosity || {}, null, 2));
    text("selfModel", JSON.stringify(s.brain_state.self_model || {}, null, 2));
  }
  function unitMap() {
    const map = new Map();
    for (const raw of snapshot.neural_units || []) map.set(Number(raw.id), raw);
    return map;
  }
  function renderGraph(s) {
    while (graph.firstChild) graph.removeChild(graph.firstChild);
    const units = s.neural_units || [], synapses = s.synapses || [];
    if (!units.length) {
      $("graphNote").textContent = "No neural units recorded";
      return;
    }
    const W = 1000, H = 620, cx = W / 2, cy = H / 2, radius = Math.min(W,H) * .37;
    const positions = new Map();
    units.forEach((u, i) => {
      const a = units.length === 1 ? 0 : (2 * Math.PI * i / units.length);
      positions.set(Number(u.id), {x: cx + radius * Math.cos(a), y: cy + radius * Math.sin(a)});
    });
    const edges = document.createDocumentFragment();
    const map = unitMap();
    for (const edge of synapses) {
      const a = positions.get(Number(edge.source_id)), b = positions.get(Number(edge.target_id));
      if (!a || !b) continue;
      const line = document.createElementNS("http://www.w3.org/2000/svg","line");
      line.setAttribute("x1", a.x); line.setAttribute("y1", a.y);
      line.setAttribute("x2", b.x); line.setAttribute("y2", b.y);
      line.classList.add("edge");
      if (edge.inhibitory) line.classList.add("inhibitory");
      line.setAttribute("stroke-width", String(Math.max(1, Math.min(5, Math.abs(Number(edge.weight || 0)) * 3))));
      edges.appendChild(line);
    }
    graph.appendChild(edges);
    for (const u of units) {
      const p = positions.get(Number(u.id)); if (!p) continue;
      const circle = document.createElementNS("http://www.w3.org/2000/svg","circle");
      const activation = Number(u.activation || 0);
      circle.setAttribute("cx", p.x); circle.setAttribute("cy", p.y);
      circle.setAttribute("r", String(3 + Math.min(9, Math.abs(activation) * 9)));
      circle.classList.add("node"); if (activation <= 0) circle.classList.add("idle");
      if (selected === Number(u.id)) circle.classList.add("selected");
      circle.setAttribute("fill", "hsl(" + String(Math.max(0, Math.min(120, 120 - activation * 120))) + " 65% 55%)");
      circle.addEventListener("click", () => { selected = Number(u.id); renderInspector(map.get(selected)); renderGraph(snapshot); });
      graph.appendChild(circle);
    }
    $("graphNote").textContent = units.length + " units / " + synapses.length + " synapses from canonical snapshot";
  }
  function renderInspector(u) {
    if (!u) { $("inspector").textContent = "Select a neural unit."; return; }
    $("inspector").textContent = JSON.stringify(u, null, 2);
  }
  async function load() {
    if (!baseUrl()) { setStatus("DISCONNECTED","disconnected"); return; }
    try {
      const handshake = await get("/v1/brain/handshake");
      setStatus("CONNECTED","connected");
      text("brainIdentity", handshake.brain_identity);
      text("revision", handshake.state_revision);
      text("hash", handshake.canonical_state_hash);
      const next = await get("/v1/brain/snapshot");
      if (!snapshot || next.state_revision !== snapshot.state_revision || next.canonical_state_hash !== snapshot.canonical_state_hash) {
        snapshot = next;
        renderState(snapshot);
        renderGraph(snapshot);
        renderInspector(selected ? unitMap().get(selected) : null);
      }
      sessionStorage.setItem("horizon.bridge", baseUrl());
      if (token.value.trim()) sessionStorage.setItem("horizon.token", token.value.trim());
    } catch (error) {
      setStatus("DISCONNECTED","disconnected");
      $("source").textContent = String(error.message || error);
    }
  }
  $("connect").addEventListener("click", () => {
    if (timer) clearInterval(timer);
    load();
    timer = setInterval(load, 5000);
  });
  setStatus("DISCONNECTED","disconnected");
})();
