// Helpers that describe the diagrams of the docs (rendered by src/components/Diagram.tsx).
//
// A diagram is a set of nodes placed on a grid (`at: [column, row]`, halves allowed), edges between them and optional dashed groups.
// A sequence diagram has actors and numbered steps. Every module file of this folder exports `pages`: the diagrams of a page in the order of
// the pictures of the upstream page (`null` where the picture is not a diagram), see scripts/import-upstream.mjs.

// a model with its fields: the title is the name, the rest are lines
export const model = (id, at, name, ...fields) => ({ id, at, title: name, lines: fields })
// a box with one line of text
export const inline = (id, at, text) => ({ id, at, kind: "box", plain: true, lines: [text] })
// parts of a flow chart (the text is plain, lines are the wrapped text)
export const start = (id, at, ...text) => ({ id, at, kind: "pill", plain: true, lines: text })
export const step = (id, at, ...text) => ({ id, at, kind: "box", plain: true, lines: text })
export const io = (id, at, ...text) => ({ id, at, kind: "io", plain: true, lines: text })
export const decision = (id, at, ...text) => ({ id, at, kind: "decision", plain: true, lines: text })
// where two paths of a flow meet
export const join = (id, at) => ({ id, at, kind: "join" })
export const edge = (from, to, label, opts = {}) => ({ from, to, ...(label ? { label } : {}), ...opts })
const dotted = { dashed: true, dir: "<->" }
export { dotted }

// "How a link between two modules is stored": the models of two modules, the link table that MedusaGo keeps between them, and arrows from
// the table to the models. `rows` repeats the same picture for several links; a row may have its own `table`.
export const linkDiagram = ({ left, right, table, rows }) => {
  const nodes = []
  const edges = []
  const leftIds = []
  const rightIds = []
  const midIds = []
  rows.forEach((row, i) => {
    const l = `l${i}`
    const m = `m${i}`
    const r = `r${i}`
    nodes.push(row.inline ? inline(l, [0, i], row.left.join(" ")) : model(l, [0, i], row.left[0], ...row.left.slice(1)))
    nodes.push(model(m, [1, i], row.table ?? table, ...row.fields))
    nodes.push(row.inline ? inline(r, [2, i], row.right.join(" ")) : model(r, [2, i], row.right[0], ...row.right.slice(1)))
    leftIds.push(l)
    midIds.push(m)
    rightIds.push(r)
    edges.push(edge(m, l), edge(m, r))
  })
  return {
    nodes,
    edges,
    groups: [
      { label: `${left} Module`, nodes: leftIds },
      { label: "MedusaGo Application", nodes: midIds },
      { label: `${right} Module`, nodes: rightIds },
    ],
  }
}

// A link table above two modules (the models of a module sit in a column under the table): `a` and `b` are the modules
// ({ module, nodes }), `edges` connect the table (id "link") and the nodes.
export const topLinkDiagram = ({ table, fields, a, b, edges, gap = [70, 90] }) => {
  const nodes = [model("link", [1, 0], table, ...fields)]
  const groups = [{ label: "MedusaGo Application", nodes: ["link"] }]
  for (const [side, col] of [
    [a, 0],
    [b, 2],
  ]) {
    side.nodes.forEach((n, i) => nodes.push({ ...n, at: [col, 1 + i] }))
    groups.push({ label: `${side.module} Module`, nodes: side.nodes.map((n) => n.id) })
  }
  return { nodes, edges, groups, gap }
}

// "Medusa -> provider -> third-party service" pictures of the providers of a module
export const providerFlow = (providerName, thirdParty = "3rd Party Service") => ({
  nodes: [start("a", [0, 0], "MedusaGo"), start("b", [1, 0], providerName), start("c", [2, 0], thirdParty)],
  edges: [edge("a", "b", "", { dir: "<->" }), edge("b", "c", "", { dir: "<->" })],
  gap: [70, 56],
})

// "The data model of a provider": the provider of a module with its services, and one row of the module per service
export const providerModel = (moduleName, rowName, idPrefix) => ({
  nodes: [
    {
      id: "p",
      at: [0, 0.5],
      title: `${moduleName} Module Provider (myProvider)`,
      lines: ["↳ service with identifier provider1", "↳ service with identifier provider2"],
    },
    model("r1", [1, 0], rowName, `id: ${idPrefix}provider1_myProvider`),
    model("r2", [1, 1], rowName, `id: ${idPrefix}provider2_myProvider`),
  ],
  edges: [edge("r1", "p", "", { fromSide: "l", toSide: "t" }), edge("r2", "p", "", { fromSide: "l", toSide: "b" })],
  gap: [80, 110],
})

export const sequence = (title, actors, steps, extra = {}) => ({
  title,
  type: "sequence",
  actors,
  numbered: true,
  steps: steps.map(([from, to, text, dashed]) => ({ from, to, text, ...(dashed ? { dashed: true } : {}) })),
  ...extra,
})

// the models of the module, a parent and two children that point to it (the same shape is used for many pictures)
export const parentWithTwo = (parent, a, b, { gap } = {}) => ({
  nodes: [
    model("p", [0, 0.5], ...parent),
    model("a", [1, 0], ...a),
    model("b", [1, 1], ...b),
  ],
  edges: [edge("a", "p", "", { fromSide: "l", toSide: "t" }), edge("b", "p", "", { fromSide: "l", toSide: "b" })],
  ...(gap ? { gap } : {}),
})
