// Preset graphs and a seeded random generator. A graph has the same JSON shape
// the API uses: { nodes: [{id, x, y}], edges: [{source, target, weight}] }.

function g(nodes, edges) {
  return {
    nodes: nodes.map(([id, x, y]) => ({ id, x, y })),
    edges: edges.map(([source, target, weight]) => ({ source, target, weight })),
  };
}

export const presets = {
  classic: {
    name: 'Класически (МПД = 39)',
    graph: g(
      [['A', 80, 80], ['B', 260, 40], ['C', 440, 80], ['D', 120, 240], ['E', 300, 220], ['F', 180, 380], ['G', 420, 360]],
      [['A', 'B', 7], ['A', 'D', 5], ['B', 'C', 8], ['B', 'D', 9], ['B', 'E', 7], ['C', 'E', 5],
       ['D', 'E', 15], ['D', 'F', 6], ['E', 'F', 8], ['E', 'G', 9], ['F', 'G', 11]],
    ),
  },
  ties: {
    name: 'Равни тегла (3×3)',
    graph: g(
      [['A', 0, 0], ['B', 160, 0], ['C', 320, 0], ['D', 0, 160], ['E', 160, 160], ['F', 320, 160],
       ['G', 0, 320], ['H', 160, 320], ['I', 320, 320]],
      [['A', 'B', 1], ['B', 'C', 2], ['D', 'E', 2], ['E', 'F', 1], ['G', 'H', 1], ['H', 'I', 2],
       ['A', 'D', 2], ['B', 'E', 1], ['C', 'F', 2], ['D', 'G', 1], ['E', 'H', 2], ['F', 'I', 1]],
    ),
  },
  disconnected: {
    name: 'Несвързан граф',
    graph: g(
      [['A', 40, 60], ['B', 220, 40], ['C', 240, 220], ['D', 60, 240],
       ['E', 420, 60], ['F', 560, 160], ['G', 420, 260], ['H', 600, 330]],
      [['A', 'B', 4], ['B', 'C', 2], ['C', 'D', 5], ['A', 'D', 3], ['A', 'C', 6],
       ['E', 'F', 1], ['F', 'G', 7], ['E', 'G', 2]],
    ),
  },
};

// Vertex names: A..Z, then AA, AB, ... (like spreadsheet columns).
export function vertexName(i) {
  let name = '';
  for (i += 1; i > 0; i = Math.floor((i - 1) / 26)) {
    name = String.fromCharCode(65 + ((i - 1) % 26)) + name;
  }
  return name;
}

// mulberry32 is a tiny seeded random number generator: same seed, same graph.
function mulberry32(seed) {
  return function () {
    seed = (seed + 0x6d2b79f5) | 0;
    let t = Math.imul(seed ^ (seed >>> 15), 1 | seed);
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}

// randomGraph places n vertices on a circle. Every pair becomes an edge with
// probability `density`. If `connected`, a random spanning tree is added first.
export function randomGraph(n, density, seed, connected) {
  const rand = mulberry32(seed);
  const radius = 60 + n * 22;
  const nodes = [];
  for (let i = 0; i < n; i++) {
    const angle = (2 * Math.PI * i) / n - Math.PI / 2;
    nodes.push({ id: vertexName(i), x: Math.round(radius * Math.cos(angle)), y: Math.round(radius * Math.sin(angle)) });
  }

  const edges = [];
  const used = new Set();
  const addEdge = (i, j) => {
    const key = Math.min(i, j) + '-' + Math.max(i, j);
    if (i === j || used.has(key)) return;
    used.add(key);
    edges.push({ source: nodes[i].id, target: nodes[j].id, weight: 1 + Math.floor(rand() * 15) });
  };

  if (connected) {
    for (let i = 1; i < n; i++) addEdge(i, Math.floor(rand() * i)); // join i to an earlier vertex
  }
  for (let i = 0; i < n; i++) {
    for (let j = i + 1; j < n; j++) {
      if (rand() < density) addEdge(i, j);
    }
  }
  return { nodes, edges };
}
