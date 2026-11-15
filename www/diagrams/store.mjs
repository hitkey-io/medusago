import { model, edge } from "./lib.mjs"

export const pages = {
  "resources/commerce-modules/store/locales": [
    {
      nodes: [
        model("s", [1, 0], "Store", "id = store_123"),
        model("a", [0, 1], "StoreLocale", "locale_code = en-US", "store_id = store_123"),
        model("b", [2, 1], "StoreLocale", "locale_code = fr-FR", "store_id = store_123"),
      ],
      edges: [edge("a", "s", "", { fromSide: "t", toSide: "l" }), edge("b", "s", "", { fromSide: "t", toSide: "r" })],
    },
  ],
}
