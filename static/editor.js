// Graph editor: a Cytoscape instance plus the editing gestures.
//   tap on empty canvas      -> new vertex
//   tap vertex, then another -> new edge (asks for the weight)
//   tap edge                 -> change its weight
//   right-click / Delete     -> remove a vertex or edge
import { vertexName } from './graphs.js';

const style = [
  { selector: 'node', style: {
    'background-color': '#ffffff', 'border-width': 2, 'border-color': '#334155',
    label: 'data(label)', 'text-valign': 'center', 'text-halign': 'center', 'text-wrap': 'wrap',
    'font-size': 13, 'font-weight': 600, color: '#0f172a', width: 38, height: 38,
  } },
  { selector: 'node[color]', style: { 'background-color': 'data(color)' } },
  { selector: 'node.tree', style: { 'background-color': '#bbf7d0', 'border-color': '#15803d' } },
  { selector: 'node.picked', style: { 'border-width': 4, 'border-color': '#f59e0b' } },
  { selector: 'edge', style: {
    width: 2.5, 'line-color': '#94a3b8', 'curve-style': 'straight',
    label: 'data(weight)', 'font-size': 12, color: '#334155',
    'text-background-color': '#ffffff', 'text-background-opacity': 1, 'text-background-padding': 2,
  } },
  { selector: 'edge.frontier', style: { 'line-color': '#2563eb', 'line-style': 'dashed', width: 3.5 } },
  { selector: 'edge.accepted', style: { 'line-color': '#16a34a', width: 5 } },
  { selector: 'edge.rejected', style: { 'line-color': '#dc2626', 'line-style': 'dashed', width: 2.5, opacity: 0.75 } },
  { selector: 'edge.current', style: {
    width: 7, 'z-index': 10, opacity: 1,
    'underlay-color': '#f59e0b', 'underlay-opacity': 0.45, 'underlay-padding': 5,
  } },
  { selector: 'edge.current.accepted', style: { 'line-color': '#ea580c' } },
];

export const cy = cytoscape({
  container: document.getElementById('cy'),
  style,
  minZoom: 0.3,
  maxZoom: 3,
  autounselectify: true, // we track the "picked" vertex ourselves
  boxSelectionEnabled: false,
});

// Cytoscape only notices window resizes. The panel also changes size when the
// banner or a two-line status appears, so tell Cytoscape about that too.
new ResizeObserver(() => cy.resize()).observe(cy.container());

let changeListener = () => {};
let picked = null; // first vertex of an edge being drawn
let edgeCounter = 0;

// onChange registers a callback for every structural edit (not for dragging).
export function onChange(fn) {
  changeListener = fn;
}

function changed() {
  changeListener();
}

function pick(node) {
  unpick();
  picked = node;
  node.addClass('picked');
}

function unpick() {
  if (picked) picked.removeClass('picked');
  picked = null;
}

function addVertex(position) {
  let i = 0;
  while (cy.getElementById(vertexName(i)).length > 0) i++; // first unused name
  const id = vertexName(i);
  cy.add({ group: 'nodes', data: { id, label: id }, position });
  changed();
}

function addEdge(source, target, weight) {
  edgeCounter++;
  cy.add({ group: 'edges', data: { id: 'e' + edgeCounter, source, target, weight } });
}

// askWeight returns an integer, or null if the user cancelled or typed nonsense.
function askWeight(question, current) {
  const answer = prompt(question, current);
  if (answer === null) return null;
  const w = Number(answer.trim());
  if (answer.trim() === '' || !Number.isInteger(w)) {
    alert('Теглото трябва да е цяло число.');
    return null;
  }
  return w;
}

function connect(a, b) {
  if (a.edgesWith(b).length > 0) {
    alert(`Вече има ребро ${a.id()}-${b.id()}.`);
    return;
  }
  const w = askWeight(`Тегло на реброто ${a.id()}-${b.id()}:`, 1 + Math.floor(Math.random() * 9));
  if (w === null) return;
  addEdge(a.id(), b.id(), w);
  changed();
}

function remove(ele) {
  if (ele === picked) unpick();
  ele.remove(); // removing a vertex also removes its edges
  changed();
}

cy.on('tap', (evt) => {
  if (evt.target !== cy) return; // only taps on the empty background
  if (picked) unpick(); // first tap on the background cancels a half-drawn edge
  else addVertex(evt.position);
});

cy.on('tap', 'node', (evt) => {
  const node = evt.target;
  if (!picked) pick(node);
  else if (picked === node) unpick();
  else {
    const from = picked;
    unpick();
    connect(from, node);
  }
});

cy.on('tap', 'edge', (evt) => {
  const edge = evt.target;
  const w = askWeight(`Ново тегло на ${edge.source().id()}-${edge.target().id()}:`, edge.data('weight'));
  if (w === null || w === edge.data('weight')) return;
  edge.data('weight', w);
  changed();
});

cy.on('cxttap', 'node, edge', (evt) => remove(evt.target));
cy.container().addEventListener('contextmenu', (e) => e.preventDefault());

document.addEventListener('keydown', (e) => {
  const typing = e.target.closest('input, select, textarea');
  if (picked && !typing && (e.key === 'Delete' || e.key === 'Backspace')) {
    e.preventDefault();
    remove(picked);
  }
});

// getGraph returns the graph in the API's JSON shape. Edge i of the result is
// cy.edges()[i], which is how the replay finds edges by index.
export function getGraph() {
  return {
    nodes: cy.nodes().map((n) => ({ id: n.id(), x: Math.round(n.position('x')), y: Math.round(n.position('y')) })),
    edges: cy.edges().map((e) => ({ source: e.source().id(), target: e.target().id(), weight: e.data('weight') })),
  };
}

// loadGraph replaces the whole graph; it throws on malformed input.
export function loadGraph(graph) {
  if (!graph || !Array.isArray(graph.nodes) || !Array.isArray(graph.edges)) {
    throw new Error('очаква се обект с полета "nodes" и "edges"');
  }
  const ids = new Set();
  for (const n of graph.nodes) {
    if (typeof n.id !== 'string' || n.id === '' || ids.has(n.id)) throw new Error(`невалиден или повтарящ се връх ${JSON.stringify(n.id)}`);
    ids.add(n.id);
  }
  for (const e of graph.edges) {
    if (!ids.has(e.source) || !ids.has(e.target)) throw new Error(`ребро към непознат връх: ${e.source}-${e.target}`);
    if (!Number.isInteger(e.weight)) throw new Error(`теглото на ${e.source}-${e.target} не е цяло число`);
  }

  unpick();
  cy.elements().remove();
  edgeCounter = 0;
  const hasPositions = graph.nodes.every((n) => Number.isFinite(n.x) && Number.isFinite(n.y));
  cy.add(graph.nodes.map((n) => ({ group: 'nodes', data: { id: n.id, label: n.id }, position: { x: n.x || 0, y: n.y || 0 } })));
  graph.edges.forEach((e) => addEdge(e.source, e.target, e.weight));
  if (!hasPositions) cy.layout({ name: 'circle' }).run();
  cy.fit(undefined, 40);
  changed();
}

// relayout spreads the vertices with Cytoscape's built-in force-directed layout.
export function relayout() {
  cy.layout({ name: 'cose', animate: false, randomize: false, nodeRepulsion: 9000, idealEdgeLength: 90 }).run();
  cy.fit(undefined, 40);
}

export function exportJSON() {
  const blob = new Blob([JSON.stringify(getGraph(), null, 2)], { type: 'application/json' });
  const a = document.createElement('a');
  a.href = URL.createObjectURL(blob);
  a.download = 'graph.json';
  a.click();
  setTimeout(() => URL.revokeObjectURL(a.href), 1000); // give the browser time to start the download
}
