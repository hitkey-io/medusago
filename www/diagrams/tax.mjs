import { model, edge, providerFlow, providerModel } from "./lib.mjs"

export const pages = {
  "resources/commerce-modules/tax/tax-provider": [providerFlow("Tax Module Provider", "3rd Party Service"), providerModel("Tax", "TaxProvider", "tp_")],
  "resources/commerce-modules/tax/tax-rates-and-rules": [
    {
      nodes: [
        model("g", [0, 0.5], "TaxRegion", "id = txreg_1"),
        model("a", [1, 0], "TaxRate", "id = txr_1", "is_default = true", "tax_region_id = txreg_1"),
        model("b", [1, 1], "TaxRate", "id = txr_2", "is_default = false", "tax_region_id = txreg_1"),
        model("r", [2, 1], "TaxRateRule", "id = txrule_1", "tax_rate_id = txr_2", "reference_id = ptyp_1", "reference = product_type"),
      ],
      edges: [edge("a", "g", "", { fromSide: "l", toSide: "t" }), edge("b", "g", "", { fromSide: "l", toSide: "b" }), edge("r", "b")],
    },
  ],
}
