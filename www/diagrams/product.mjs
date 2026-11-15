import { model, inline, edge, start, decision, dotted, linkDiagram, topLinkDiagram } from "./lib.mjs"
import { productVariantInventoryItem } from "./shared.mjs"

export const pages = {
  "resources/commerce-modules/product/links-to-other-modules": [
    productVariantInventoryItem(),
    {
      nodes: [
        inline("v", [0, 0], "ProductVariant id = variant_1"),
        inline("p", [0, 1], "Product id = prod_1"),
        model("link", [1, 0], "product_variant_price_set", "variant_id = variant_1", "price_set_id = ps_1"),
        inline("s", [2, 0], "PriceSet id = ps_1"),
      ],
      edges: [edge("link", "v"), edge("link", "s"), edge("v", "p", "", { fromSide: "b", toSide: "t" })],
      groups: [
        { label: "Product Module", nodes: ["v", "p"] },
        { label: "MedusaGo Application", nodes: ["link"] },
        { label: "Pricing Module", nodes: ["s"] },
      ],
      gap: [70, 60],
    },
    linkDiagram({
      left: "Product",
      right: "Sales Channel",
      table: "product_sales_channel",
      rows: [{ inline: true, left: ["Product", "id = prod_1"], right: ["SalesChannel", "id = sc_1"], fields: ["product_id = prod_1", "sales_channel_id = sc_1"] }],
    }),
  ],
  "resources/commerce-modules/product/selling-products": [
    {
      nodes: [
        model("p1", [0, 0], "Product", "title: Shirt"),
        model("s", [2, 0], "ShippingProfile", "name: Default"),
        model("p2", [0, 1], "Product", "title: License Key"),
      ],
      edges: [edge("p1", "s", "Link", dotted)],
      groups: [
        { label: "Product Variants Require Shipping", nodes: ["p1", "s"] },
        { label: "Product Variant Don’t Require Shipping", nodes: ["p2"] },
      ],
      gap: [150, 70],
    },
    {
      nodes: [
        model("sp", [0, 0], "ShippingProfile", "name: Default"),
        model("p", [0, 1], "Product", "title: Video Game"),
        model("v1", [1, 0], "ProductVariant", "title: Physical"),
        model("v2", [1, 2], "ProductVariant", "title: Digital", "manage_inventory: true"),
        model("i", [2, 2], "InventoryItem", "requires_shipping: false"),
      ],
      edges: [edge("sp", "p", "Link", { ...dotted, fromSide: "b", toSide: "t" }), edge("v1", "p", "", { fromSide: "l", toSide: "r" }), edge("v2", "p", "", { fromSide: "l", toSide: "r" }), edge("v2", "i", "Link", dotted)],
      groups: [
        { label: "Variant Requires Shipping", nodes: ["v1"] },
        { label: "Variant Doesn’t Require Shipping", nodes: ["v2", "i"] },
      ],
      gap: [110, 60],
    },
    {
      nodes: [
        start("a", [0, 0], "Start shipping", "requirement check"),
        decision("b", [1, 0], "Product variant", "has inventory item?"),
        start("c", [2, 0], "Use inventory item’s", "requires_shipping value"),
        decision("d", [1, 1], "Product has", "shipping profile?"),
        start("e", [0, 1], "Variant doesn’t require", "shipping"),
        start("f", [2, 1], "Variant requires shipping"),
      ],
      edges: [edge("a", "b"), edge("b", "c", "Yes"), edge("b", "d", "No"), edge("d", "e", "No"), edge("d", "f", "Yes")],
    },
  ],
  "resources/commerce-modules/product/variant-inventory": [
    productVariantInventoryItem(),
    topLinkDiagram({
      table: "product_variant_inventory_item",
      fields: ["variant_id = variant_1", "inventory_item_id = iitem_1"],
      a: { module: "Product", nodes: [model("v", [0, 0], "ProductVariant", "id = variant_1")] },
      b: { module: "Inventory", nodes: [model("i", [0, 0], "InventoryItem", "id = iitem_1"), model("lv", [0, 0], "InventoryLevel", "id = ilev_1", "inventory_item_id = iitem_1", "stocked_quantity = 10")] },
      edges: [edge("link", "v", "", { fromSide: "b", toSide: "r" }), edge("link", "i", "", { fromSide: "b", toSide: "l" }), edge("lv", "i", "", { fromSide: "t", toSide: "b" })],
    }),
    {
      nodes: [
        model("i", [0, 0], "InventoryItem", "id = iitem_1"),
        model("lv", [0, 1], "InventoryLevel", "id = ilev_1", "inventory_item_id = iitem_1", "stocked_quantity = 10", "location_id = sloc_1"),
        model("a", [2, 0], "StockLocationAddress", "id = laddr_1", "country_code = us"),
        model("l", [2, 1], "StockLocation", "id = sloc_1", "address_id = laddr_1"),
      ],
      edges: [edge("lv", "i", "", { fromSide: "t", toSide: "b" }), edge("l", "a", "", { fromSide: "t", toSide: "b" }), edge("lv", "l", "Read-only link", { dashed: true })],
      groups: [
        { label: "Inventory Module", nodes: ["i", "lv"] },
        { label: "Stock Location Module", nodes: ["a", "l"] },
      ],
      gap: [140, 80],
    },
    {
      nodes: [
        model("i", [0, 0], "InventoryItem", "id = iitem_1"),
        model("lv", [0, 1], "InventoryLevel", "id = ilev_1", "inventory_item_id = iitem_1", "stocked_quantity = 10", "location_id = sloc_1"),
        model("a", [1, 0], "StockLocationAddress", "id = laddr_1", "country_code = us"),
        model("l", [1, 1], "StockLocation", "id = sloc_1", "address_id = laddr_1"),
        model("sc", [2, 0], "SalesChannel", "id = sc_1"),
        model("link", [2, 1], "sales_channel_stock_location", "sales_channel_id = sc_1", "stock_location_id = sloc_1"),
      ],
      edges: [
        edge("lv", "i", "", { fromSide: "t", toSide: "b" }),
        edge("l", "a", "", { fromSide: "t", toSide: "b" }),
        edge("lv", "l", "Read-only link", { dashed: true }),
        edge("link", "sc", "", { fromSide: "t", toSide: "b" }),
        edge("link", "l"),
      ],
      groups: [
        { label: "Inventory Module", nodes: ["i", "lv"] },
        { label: "Stock Location Module", nodes: ["a", "l"] },
        { label: "Sales Channel Module", nodes: ["sc"] },
        { label: "MedusaGo Application", nodes: ["link"] },
      ],
      gap: [110, 90],
    },
  ],
}
