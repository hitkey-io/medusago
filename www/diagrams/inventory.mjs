import { model, inline, edge, start, step, decision, join, linkDiagram, topLinkDiagram } from "./lib.mjs"
import { productVariantInventoryItem } from "./shared.mjs"

const link = (variant, item, sku, qty) => model(`m${item}`, [1, item - 1], "LinkProductVariantInventoryItem", `variant_id = variant_${variant}`, `inventory_item_id = iitem_${item}`, ...(qty ? [`required_quantity = ${qty}`] : []))
const invItem = (item, sku) => model(`i${item}`, [2, item - 1], "InventoryItem", `id = iitem_${item}`, `sku = ${sku}`)

export const pages = {
  "resources/commerce-modules/inventory/concepts": [
    {
      title: "Inventory Module",
      nodes: [
        { ...inline("l1", [0, 0], "InventoryLevel") },
        inline("l2", [0, 1], "InventoryLevel"),
        inline("i", [1, 0.5], "InventoryItem"),
        inline("r1", [2, 0], "ReservationItem"),
        inline("r2", [2, 1], "ReservationItem"),
      ],
      edges: [
        edge("l1", "i", "", { fromSide: "b", toSide: "l" }),
        edge("l2", "i", "", { fromSide: "t", toSide: "l", dir: "--" }),
        edge("r1", "i", "", { fromSide: "b", toSide: "r" }),
        edge("r2", "i", "", { fromSide: "t", toSide: "r", dir: "--" }),
      ],
      gap: [90, 90],
    },
  ],
  "resources/commerce-modules/inventory/inventory-in-flows": [
    {
      title: "Inventory Module in Product Variant Creation Flow",
      nodes: [start("a", [0, 0], "Product variant is created"), decision("b", [1, 0], "is", "manage_inventory", "enabled?"), start("c", [2, 0], "Create inventory item"), start("d", [1, 1], "End")],
      edges: [edge("a", "b"), edge("b", "c", "Yes"), edge("b", "d", "No")],
    },
    {
      title: "Inventory Module in Add to Cart Flow",
      nodes: [
        start("a", [0, 0], "Add product variant to cart"),
        decision("b", [1, 0], "is", "manage_inventory", "enabled?"),
        decision("c", [2, 0], "Inventory item has", "sufficient stock", "quantity?"),
        step("d", [3, 0], "Create line item in cart"),
        start("e", [1, 1], "Product variant is considered", "in stock"),
        start("f", [2, 1], "Don’t add variant to cart"),
      ],
      edges: [edge("a", "b"), edge("b", "c", "Yes"), edge("c", "d", "Yes"), edge("b", "e", "No"), edge("c", "f", "No")],
    },
    {
      title: "Inventory Module in Order Placed Flow",
      nodes: [start("a", [0, 0], "Order is placed"), step("b", [1, 0], "Find product variants having", "manage_inventory enabled"), start("c", [2, 0], "Create reservation items for the", "ordered quantities")],
      edges: [edge("a", "b"), edge("b", "c")],
    },
    {
      title: "Inventory Module in Order Fulfillment Flow",
      nodes: [
        start("a", [0, 0], "Item in order is fulfilled"),
        decision("b", [1, 0], "is variant’s", "manage_inventory", "enabled?"),
        step("c", [2, 0], "Subtract inventory level’s", "reserved_quantity from stocked_quantity"),
        start("d", [1, 1], "End"),
        step("e", [2, 1], "Reset the inventory level’s", "reserved_quantity"),
        start("f", [2, 2], "Delete associated", "reservation item"),
      ],
      edges: [edge("a", "b"), edge("b", "c", "Yes"), edge("b", "d", "No"), edge("c", "e"), edge("e", "f")],
    },
    {
      title: "Inventory Module in Order Return Flow",
      nodes: [
        start("a", [0, 0], "Item in order is returned"),
        decision("b", [1, 0], "is variant’s", "manage_inventory", "enabled?"),
        start("c", [2, 0], "Increment inventory level’s", "stocked_quantity with", "returned quantity"),
        start("d", [1, 1], "End"),
      ],
      edges: [edge("a", "b"), edge("b", "c", "Yes"), edge("b", "d", "No")],
    },
  ],
  "resources/commerce-modules/inventory/inventory-kit": [
    {
      nodes: [model("v", [0, 1], "ProductVariant", "id = variant_1", "sku = BICYCLE"), link(1, 1, "", 1), link(1, 2, "", 2), link(1, 3, "", 1), invItem(1, "FRAME"), invItem(2, "WHEEL"), invItem(3, "SEAT")],
      edges: [
        edge("m1", "v", "", { fromSide: "l", toSide: "t" }),
        edge("m2", "v"),
        edge("m3", "v", "", { fromSide: "l", toSide: "b" }),
        edge("m1", "i1"),
        edge("m2", "i2"),
        edge("m3", "i3"),
      ],
    },
    {
      nodes: [
        model("v1", [0, 0], "ProductVariant", "id = variant_1", "sku = SHIRT"),
        model("v2", [0, 1], "ProductVariant", "id = variant_2", "sku = PANTS"),
        model("v3", [0, 2], "ProductVariant", "id = variant_3", "sku = SHOES"),
        ...[1, 2, 3].map((n) => link(n, n, "")),
        invItem(1, "SHIRT"),
        invItem(2, "PANTS"),
        invItem(3, "SHOES"),
      ],
      edges: [1, 2, 3].flatMap((n) => [edge(`m${n}`, `v${n}`), edge(`m${n}`, `i${n}`)]),
    },
    {
      nodes: [model("v", [0, 1], "ProductVariant", "id = variant_4", "sku = BUNDLE"), ...[1, 2, 3].map((n) => model(`m${n}`, [1, n - 1], "LinkProductVariantInventoryItem", "variant_id = variant_4", `inventory_item_id = iitem_${n}`)), invItem(1, "SHIRT"), invItem(2, "PANTS"), invItem(3, "SHOES")],
      edges: [
        edge("m1", "v", "", { fromSide: "l", toSide: "t" }),
        edge("m2", "v"),
        edge("m3", "v", "", { fromSide: "l", toSide: "b" }),
        edge("m1", "i1"),
        edge("m2", "i2"),
        edge("m3", "i3"),
      ],
    },
  ],
  "resources/commerce-modules/inventory/links-to-other-modules": [productVariantInventoryItem()],
}
