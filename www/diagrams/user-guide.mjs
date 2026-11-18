import { model, edge } from "./lib.mjs"

// Pictures of the user guide that explain how products are composed (user-guide/products/create).
const kit = (name, parts) => {
  const nodes = [model("v", [0, (parts.length - 1) / 2], name, "Product Variant")]
  const edges = []
  parts.forEach((p, i) => {
    nodes.push(model(`i${i}`, [1, i], p, "Inventory"))
    edges.push(edge("v", `i${i}`, "", { fromSide: "r", toSide: "l" }))
  })
  return { nodes, edges, gap: [90, 24] }
}

export const pages = {
  "user-guide/products/create/bundle": [
    (() => {
      const d = kit("Camera Bundle", ["Camera", "Lens", "Camera Bag"])
      // every inventory item of the bundle belongs to a variant of its own
      ;["Camera", "Lens", "Camera Bag"].forEach((p, i) => {
        d.nodes.push(model(`p${i}`, [2, i], p, "Product Variant"))
        d.edges.push(edge(`i${i}`, `p${i}`, "", { fromSide: "r", toSide: "l" }))
      })
      return { title: "Bundle product", ...d }
    })(),
    null,
    null,
  ],
  "user-guide/products/create/multi-part": [
    { title: "Multi-part product", ...kit("Bicycle", ["Frame", "Wheels", "Handlebars"]) },
    null,
    null,
  ],
}
