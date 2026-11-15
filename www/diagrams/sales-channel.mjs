import { linkDiagram } from "./lib.mjs"
import { publishableKeySalesChannel, salesChannelStockLocation } from "./shared.mjs"

export const pages = {
  "resources/commerce-modules/sales-channel/links-to-other-modules": [
    publishableKeySalesChannel(),
    linkDiagram({
      left: "Product",
      right: "Sales Channel",
      table: "product_sales_channel",
      rows: [{ inline: true, left: ["Product", "id = prod_1"], right: ["SalesChannel", "id = sc_1"], fields: ["product_id = prod_1", "sales_channel_id = sc_1"] }],
    }),
    salesChannelStockLocation(),
  ],
}
