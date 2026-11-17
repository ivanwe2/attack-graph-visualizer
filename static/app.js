// App: runs an algorithm through the Go API and replays its steps.
// The backend decides everything; this file only draws step `pos`.
import * as editor from './editor.js';
import { presets, randomGraph } from './graphs.js';

const $ = (id) => document.getElementById(id);
const cy = editor.cy;
const NAMES = { kruskal: 'Крускал', prim: 'Прим', dijkstra: 'Дейкстра' };
const PALETTE = ['#fde68a', '#a7f3d0', '#bfdbfe', '#fbcfe8', '#ddd6fe', '#fed7aa', '#99f6e4', '#e9d5ff', '#fecaca', '#d9f99d'];

let run = null; // { algorithm, start, result, nodes, edges } - nodes/edges are snapshots taken at run time
let pos = -1;   // index of the current step
let timer = null;

// ---------- API ----------

async function callApi(algorithm, start) {
  const res = await fetch('/api/run', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ graph: editor.getGraph(), algorithm, start }),
  });
  const data = await res.json();
  if (!res.ok) throw new Error(data.error || res.statusText);
  return data;
}

function edgeName(i, edges = run.edges) {
  const e = edges[i];
  return `${e.source().id()}-${e.target().id()} (${e.data('weight')})`;
}

// ---------- run / replay ----------

async function startRun() {
  stop();
  clearRun();
  if (cy.nodes().length === 0) return status('Графът е празен – кликнете върху платното, за да добавите върхове.', 'error');
  const algorithm = $('algorithm').value;
  const start = $('start').value;
  try {
    const result = await callApi(algorithm, start);
    run = { algorithm, start, result, nodes: cy.nodes(), edges: cy.edges() };
  } catch (err) {
    return status('Грешка: ' + err.message, 'error');
  }
  $('root').value = run.algorithm === 'kruskal' ? cy.nodes()[0].id() : start;
  $('checks-label').textContent = algorithm === 'dijkstra' ? 'Проверки „вече обходен“' : 'Проверки за цикъл';
  $('list-title').textContent = algorithm === 'kruskal' ? 'Сортирани ребра' : 'Кандидат-ребра (граница)';
  buildLog();
  go(0);
  play();
}

function clearRun() {
  stop();
  run = null;
  pos = -1;
  cy.batch(() => {
    cy.elements().removeClass('accepted rejected current frontier tree');
    cy.nodes().forEach((n) => n.removeData('color').data('label', n.id()));
  });
  $('log').replaceChildren();
  $('edge-list').replaceChildren();
  $('step-label').textContent = 'Няма изпълнение';
  for (const id of ['st-total', 'st-examined', 'st-checks', 'st-comp']) $(id).textContent = '–';
  $('banner').hidden = true;
  drawTree();
}

const lastIndex = () => (run ? run.result.steps.length - 1 : -1);
const delay = () => 2200 - 200 * Number($('speed').value); // speed 1..10 -> 2000..200 ms

function go(i) {
  if (!run) return;
  pos = Math.max(0, Math.min(i, lastIndex()));
  render();
}

function play() {
  if (!run) return;
  if (pos >= lastIndex()) go(0);
  $('play').textContent = '⏸';
  timer = setTimeout(function tick() {
    go(pos + 1);
    if (pos < lastIndex()) timer = setTimeout(tick, delay());
    else stop();
  }, delay());
}

function stop() {
  clearTimeout(timer);
  timer = null;
  $('play').textContent = '▶';
}

// render draws the state after steps[0..pos]. Accepted/rejected edges build up
// over the steps; components, frontier and distances are taken from step pos.
function render() {
  const steps = run.result.steps;
  const step = steps[pos];
  const accepted = new Set();
  const rejected = new Set();
  const inTree = new Set();
  for (let i = 0; i <= pos; i++) {
    const s = steps[i];
    if (s.kind === 'accept') accepted.add(s.edge);
    if (s.kind === 'reject' || s.kind === 'skip') rejected.add(s.edge);
    if (s.vertex && (s.kind === 'visit' || s.kind === 'accept')) inTree.add(s.vertex);
  }
  const frontier = new Set(step.frontier || []);

  cy.batch(() => {
    run.edges.forEach((e, i) => {
      e.removeClass('accepted rejected current frontier');
      if (accepted.has(i)) e.addClass('accepted');
      else if (rejected.has(i)) e.addClass('rejected');
      else if (frontier.has(i)) e.addClass('frontier');
      if (i === step.edge) e.addClass('current');
    });
    const colors = componentColors(step.comp);
    run.nodes.forEach((n, i) => {
      n.toggleClass('tree', inTree.has(n.id()));
      if (colors[i]) n.data('color', colors[i]);
      else n.removeData('color');
      const d = step.dist ? step.dist[n.id()] : undefined;
      n.data('label', d === undefined ? n.id() : `${n.id()}\n${d}`);
    });
  });

  const done = pos === lastIndex();
  const comps = step.comp ? new Set(step.comp).size : done ? run.result.components : '…';
  $('st-total').textContent = step.total;
  $('st-examined').textContent = step.examined;
  $('st-checks').textContent = step.cycleChecks;
  $('st-comp').textContent = comps;
  $('step-label').textContent = `Стъпка ${pos + 1} / ${steps.length}`;
  status(step.message, step.kind);

  // Say clearly, for the whole replay, that a disconnected graph gives a forest.
  const forest = run.result.components > 1;
  $('banner').hidden = !forest;
  if (forest) {
    $('banner').textContent = run.algorithm === 'dijkstra'
      ? 'Графът е несвързан: някои върхове са недостижими от началния.'
      : `Графът е несвързан: резултатът е минимална покриваща гора от ${run.result.components} дървета, а не едно дърво.`;
  }

  renderLog();
  renderEdgeList(accepted, rejected);
  drawTree();
}

// componentColors gives every union-find component with 2+ vertices its own color.
function componentColors(comp) {
  if (!comp) return [];
  const size = {};
  comp.forEach((root) => (size[root] = (size[root] || 0) + 1));
  const colorOf = {};
  let next = 0;
  return comp.map((root) => {
    if (size[root] < 2) return null;
    if (!(root in colorOf)) colorOf[root] = PALETTE[next++ % PALETTE.length];
    return colorOf[root];
  });
}

function status(text, kind = '') {
  const el = $('status');
  el.textContent = text;
  el.className = 'status ' + kind;
}

// ---------- side panels ----------

function buildLog() {
  const items = run.result.steps.map((s, i) => {
    const li = document.createElement('li');
    li.textContent = s.message;
    li.className = s.kind;
    li.onclick = () => { stop(); go(i); };
    return li;
  });
  $('log').replaceChildren(...items);
}

function renderLog() {
  const items = $('log').children;
  for (let i = 0; i < items.length; i++) {
    items[i].hidden = i > pos;
    items[i].classList.toggle('current', i === pos);
  }
  keepVisible($('log'), items[pos]);
}

// keepVisible scrolls only the list itself (scrollIntoView would also scroll the page).
function keepVisible(list, item) {
  if (!item) return;
  const l = list.getBoundingClientRect();
  const i = item.getBoundingClientRect();
  if (i.top < l.top) list.scrollTop -= l.top - i.top;
  else if (i.bottom > l.bottom) list.scrollTop += i.bottom - l.bottom;
}

function renderEdgeList(accepted, rejected) {
  const step = run.result.steps[pos];
  let edges;
  let examinedSoFar = new Set();
  if (run.algorithm === 'kruskal') {
    edges = run.result.steps[0].order;
    for (let i = 0; i <= pos; i++) if (run.result.steps[i].edge >= 0) examinedSoFar.add(run.result.steps[i].edge);
  } else {
    edges = step.frontier || [];
  }
  const done = pos === lastIndex();
  const items = edges.map((e) => {
    const li = document.createElement('li');
    li.textContent = edgeName(e);
    if (accepted.has(e)) li.className = 'accept';
    else if (rejected.has(e)) li.className = 'reject';
    else if (run.algorithm === 'kruskal' && done && !examinedSoFar.has(e)) li.className = 'skipped';
    if (e === step.edge) li.classList.add('current');
    return li;
  });
  if (items.length === 0) {
    const li = document.createElement('li');
    li.className = 'empty';
    li.textContent = run.algorithm === 'kruskal' ? 'Няма ребра.' : 'Няма кандидати.';
    items.push(li);
  }
  $('edge-list').replaceChildren(...items);
  keepVisible($('edge-list'), $('edge-list').querySelector('.current'));
}

// ---------- tree view ----------

const treeCy = cytoscape({
  container: $('tree'),
  autounselectify: true,
  boxSelectionEnabled: false,
  minZoom: 0.2,
  maxZoom: 1.5, // keep small trees from being blown up to fill the panel
  style: [
    { selector: 'node', style: {
      'background-color': '#bbf7d0', 'border-width': 2, 'border-color': '#15803d', label: 'data(id)',
      'text-valign': 'center', 'font-size': 12, 'font-weight': 600, width: 30, height: 30,
    } },
    { selector: 'node.root', style: { 'border-width': 4, 'border-color': '#f59e0b' } },
    { selector: 'edge', style: {
      width: 3, 'line-color': '#16a34a', label: 'data(weight)', 'font-size': 12,
      'text-background-color': '#ffffff', 'text-background-opacity': 1, 'text-background-padding': 2,
    } },
  ],
});
new ResizeObserver(() => treeCy.resize()).observe(treeCy.container());

// drawTree shows the finished tree (or forest) as a rooted tree, once the
// replay has reached the last step.
function drawTree() {
  treeCy.elements().remove();
  const ready = run && pos === lastIndex();
  $('tree-hint').hidden = ready;
  if (!ready) return;
  treeCy.resize(); // hiding the hint just made the panel taller; the layout needs the new size

  treeCy.add(run.nodes.map((n) => ({ group: 'nodes', data: { id: n.id() } })));
  treeCy.add(run.result.treeEdges.map((i) => {
    const e = run.edges[i];
    return { group: 'edges', data: { id: 't' + i, source: e.source().id(), target: e.target().id(), weight: e.data('weight') } };
  }));

  // One root per tree: the chosen root in its own tree, elsewhere the tree's
  // first vertex in graph order.
  const chosen = $('root').value;
  const ids = run.nodes.map((n) => n.id());
  const rootIds = treeCy.elements().components().map((c) => {
    const has = (id) => c.nodes().some((n) => n.id() === id);
    return has(chosen) ? chosen : ids.find(has);
  });
  const roots = treeCy.nodes().filter((n) => rootIds.includes(n.id())).addClass('root');
  treeCy.layout({ name: 'breadthfirst', roots, spacingFactor: 1.5, padding: 12, animate: false }).run();
}

// ---------- compare ----------

async function compare() {
  stop();
  if (cy.nodes().length === 0) return status('Графът е празен.', 'error');
  const start = $('start').value;
  let k, p;
  try {
    [k, p] = await Promise.all([callApi('kruskal', ''), callApi('prim', start)]);
  } catch (err) {
    return status('Грешка: ' + err.message, 'error');
  }
  const edges = cy.edges();
  const same = k.totalWeight === p.totalWeight;
  const sameSet = [...k.treeEdges].sort().join() === [...p.treeEdges].sort().join();

  $('cmp-kruskal').textContent = k.totalWeight;
  $('cmp-prim').textContent = p.totalWeight;
  $('cmp-prim-label').textContent = `Прим (от ${start})`;
  $('cmp-verdict').textContent = (same ? '✓ Теглата съвпадат' : '✗ Теглата се различават!')
    + (sameSet ? ' – избрани са същите ребра, но в различен ред.' : ' – при равни тегла са избрани различни ребра.')
    + (k.components > 1 ? ` Графът е несвързан: гора от ${k.components} дървета.` : '');
  $('cmp-verdict').className = same ? 'ok' : 'bad';

  const rows = [];
  for (let i = 0; i < Math.max(k.treeEdges.length, p.treeEdges.length); i++) {
    const tr = document.createElement('tr');
    for (const text of [i + 1, k.treeEdges[i] === undefined ? '' : edgeName(k.treeEdges[i], edges),
                        p.treeEdges[i] === undefined ? '' : edgeName(p.treeEdges[i], edges)]) {
      const td = document.createElement('td');
      td.textContent = text;
      tr.append(td);
    }
    rows.push(tr);
  }
  $('cmp-rows').replaceChildren(...rows);
  $('compare-panel').hidden = false;
}

// ---------- graph controls ----------

// refreshVertexSelects fills the start/root pickers, keeping the old choice if possible.
function refreshVertexSelects() {
  const ids = cy.nodes().map((n) => n.id());
  for (const sel of [$('start'), $('root')]) {
    const old = sel.value;
    sel.replaceChildren(...ids.map((id) => new Option(id, id)));
    if (ids.includes(old)) sel.value = old;
  }
}

editor.onChange(() => {
  clearRun();
  $('compare-panel').hidden = true;
  refreshVertexSelects();
  status(`Граф с ${cy.nodes().length} върха и ${cy.edges().length} ребра. Изберете алгоритъм и натиснете „Стартирай“.`);
});

function loadOrReport(graph) {
  try {
    editor.loadGraph(graph);
  } catch (err) {
    status('Невалиден граф: ' + err.message, 'error');
  }
}

for (const [key, p] of Object.entries(presets)) $('preset').add(new Option(p.name, key));
$('load-preset').onclick = () => loadOrReport(presets[$('preset').value].graph);
$('random').onclick = () => {
  const n = Math.max(1, Math.min(40, Number($('rnd-n').value) || 8));
  const density = Math.max(0, Math.min(1, Number($('rnd-density').value)));
  loadOrReport(randomGraph(n, density, Number($('rnd-seed').value) || 0, $('rnd-connected').checked));
};
$('layout').onclick = () => editor.relayout();
$('clear').onclick = () => loadOrReport({ nodes: [], edges: [] });
$('export').onclick = () => editor.exportJSON();
$('import').onclick = () => $('import-file').click();
$('import-file').onchange = async (e) => {
  const file = e.target.files[0];
  e.target.value = ''; // allow importing the same file again
  if (!file) return;
  try {
    editor.loadGraph(JSON.parse(await file.text()));
  } catch (err) {
    status('Невалиден файл: ' + err.message, 'error');
  }
};

// ---------- algorithm + replay controls ----------

$('run').onclick = startRun;
$('compare').onclick = compare;
$('first').onclick = () => { stop(); go(0); };
$('back').onclick = () => { stop(); go(pos - 1); };
$('forward').onclick = () => { stop(); go(pos + 1); };
$('last').onclick = () => { stop(); go(lastIndex()); };
$('play').onclick = () => (timer ? stop() : play());
$('root').onchange = drawTree;
$('algorithm').onchange = () => { clearRun(); status(`Избран алгоритъм: ${NAMES[$('algorithm').value]}. Натиснете „Стартирай“.`); };

// Keyboard: ← → step, space play/pause (unless the user is typing in a field).
document.addEventListener('keydown', (e) => {
  if (e.target.closest('input, select, textarea') || !run) return;
  if (e.key === 'ArrowRight') { stop(); go(pos + 1); }
  else if (e.key === 'ArrowLeft') { stop(); go(pos - 1); }
  else if (e.key === ' ' && !e.target.closest('button')) { e.preventDefault(); timer ? stop() : play(); }
});

loadOrReport(presets.classic.graph);
