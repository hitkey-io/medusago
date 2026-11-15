import { model, inline, edge, start, step, decision } from "./lib.mjs"
import { shippingOptionPriceSet } from "./shared.mjs"

export const pages = {
  "resources/commerce-modules/pricing/concepts": [
    {
      nodes: [
        model("s", [0, 0.5], "PriceSet", "id = ps_1"),
        model("a", [1, 0], "Price", "id = price_1", "amount = 400", "price_set_id = ps_1", "currency_code = usd"),
        model("b", [1, 1], "Price", "id = price_2", "amount = 300", "price_set_id = ps_1", "currency_code = eur"),
      ],
      edges: [edge("s", "a", "", { fromSide: "t", toSide: "l" }), edge("s", "b", "", { fromSide: "b", toSide: "l" })],
    },
  ],
  "resources/commerce-modules/pricing/links-to-other-modules": [
    shippingOptionPriceSet(),
    {
      nodes: [
        inline("ps", [0.5, 0], "PriceSet id = ps_1"),
        model("p1", [0, 1], "Price", "id = price_1", "amount = 1000", "price_set_id = ps_1", "currency_code = usd"),
        model("p2", [1, 1], "Price", "id = price_2", "amount = 1200", "price_set_id = ps_1", "currency_code = eur"),
        model("link", [2, 0], "product_variant_price_set", "variant_id = variant_1", "price_set_id = ps_1"),
        inline("pv", [2, 1], "ProductVariant id = variant_1"),
        inline("pr", [2, 2], "Product id = prod_1"),
      ],
      edges: [
        edge("link", "ps"),
        edge("p1", "ps", "", { fromSide: "t", toSide: "b" }),
        edge("p2", "ps", "", { fromSide: "t", toSide: "b" }),
        edge("link", "pv", "", { fromSide: "b", toSide: "t" }),
        edge("pv", "pr", "", { fromSide: "b", toSide: "t" }),
      ],
      groups: [
        { label: "Pricing Module", nodes: ["ps", "p1", "p2"] },
        { label: "MedusaGo Application", nodes: ["link"] },
        { label: "Product Module", nodes: ["pv", "pr"] },
      ],
      gap: [70, 80],
    },
  ],
  "resources/commerce-modules/pricing/price-calculation": [
    {
      nodes: [
        start("a", [0, 0], "Select original price"),
        decision("b", [1, 0], "Type of calculated price’s", "price list is `override`?"),
        start("c", [2, 0], "Set to calculated price"),
        decision("d", [1, 1], "Context doesn’t", "have rules?"),
        start("e", [2, 1], "Set to price of same currency", "with no rules (default)"),
        decision("f", [1, 2], "There’s a price matching all", "rules?"),
        start("g", [2, 2], "Select price matching all rules"),
        step("h", [0, 2], "Sort prices by the", "number of matching rules"),
        start("i", [0, 1], "Select price with", "highest number of", "matching rules"),
      ],
      edges: [edge("a", "b"), edge("b", "c", "Yes"), edge("b", "d", "No"), edge("d", "e", "Yes"), edge("d", "f", "No"), edge("f", "g", "Yes"), edge("f", "h", "No"), edge("h", "i")],
    },
  ],
  "resources/commerce-modules/pricing/price-rules": [
    {
      nodes: [model("p", [0, 0], "Price", "id = price_1", "rules_count = 1", "price_set_id = ps_1"), model("r", [1, 0], "PriceRule", "value = 10557", "attribute = zip_code", "price_id = price_1")],
      edges: [edge("r", "p")],
    },
    {
      nodes: [
        model("r1", [0, 0], "PriceRule", "value = 33775", "attribute = zip_code", "price_id = price_1"),
        model("p", [1, 0], "Price", "id = price_1", "rules_count = 2"),
        model("r2", [2, 0], "PriceRule", "value = reg_123", "attribute = region_id", "price_id = price_1"),
      ],
      edges: [edge("p", "r1"), edge("p", "r2")],
    },
    {
      nodes: [
        model("p", [1, 0], "Price", "id = price_1", "price_list_id = pl_1", "price_set_id = ps_1"),
        model("s", [2, 0], "PriceSet", "id = ps_1"),
        model("l", [1, 1], "PriceList", "id = pl_1", "rules_count = 2"),
        model("r1", [0, 1], "PriceListRule", "id = plr_2", "value = reg_123", "attribute = region_id", "price_list_id = pl_1"),
        model("r2", [2, 1], "PriceListRule", "id = plr_1", "value = 10557", "attribute = zip_code", "price_list_id = pl_1"),
      ],
      edges: [edge("p", "s"), edge("p", "l"), edge("r1", "l"), edge("r2", "l")],
    },
  ],
}
