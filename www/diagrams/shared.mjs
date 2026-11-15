// Pictures that appear on more than one page.
import { linkDiagram, topLinkDiagram, model, inline, edge } from "./lib.mjs"

export const orderCart = () =>
  linkDiagram({
    left: "Cart",
    right: "Order",
    table: "order_cart",
    rows: [{ left: ["Cart", "id = cart_123"], right: ["Order", "id = order_1"], fields: ["order_id = order_1", "cart_id = cart_1"] }],
  })

export const cartPromotion = () =>
  linkDiagram({
    left: "Cart",
    right: "Promotion",
    table: "cart_promotion",
    rows: [{ left: ["Cart", "id = cart_1"], right: ["Promotion", "id = promo_1"], fields: ["cart_id = cart_1", "promotion_id = promo_1"] }],
  })

export const orderPromotion = () =>
  linkDiagram({
    left: "Order",
    right: "Promotion",
    table: "order_promotion",
    rows: [{ left: ["Order", "id = order_1"], right: ["Promotion", "id = promo_1"], fields: ["order_id = order_1", "promotion_id = promo_1"] }],
  })

export const orderFulfillment = () =>
  linkDiagram({
    left: "Order",
    right: "Fulfillment",
    table: "order_fulfillment",
    rows: [{ left: ["Order", "id = order_1"], right: ["Fulfillment", "id = ful_1"], fields: ["order_id = order_1", "fulfillment_id = ful_1"] }],
  })

export const orderPaymentCollection = () =>
  linkDiagram({
    left: "Order",
    right: "Payment",
    rows: [
      { table: "order_payment_collection", left: ["Order", "id = order_1"], right: ["PaymentCollection", "id = paycol_1"], fields: ["order_id = order_1", "payment_collection_id = paycol_1"] },
      { table: "order_claim_payment_collection", left: ["OrderClaim", "id = claim_1"], right: ["PaymentCollection", "id = paycol_2"], fields: ["claim_id = claim_1", "payment_collection_id = paycol_2"] },
      { table: "order_exchange_payment_collection", left: ["OrderExchange", "id = oexc_1"], right: ["PaymentCollection", "id = paycol_3"], fields: ["exchange_id = oexc_1", "payment_collection_id = paycol_3"] },
    ],
  })

export const cartPaymentCollection = () => ({
  nodes: [
    model("m", [1, 0], "cart_payment_collection", "cart_id = cart_1", "payment_collection_id = pay_col_1"),
    model("l", [0, 1], "Cart", "id = cart_1"),
    model("r", [2, 1], "PaymentCollection", "id = pay_col_1"),
  ],
  edges: [edge("m", "l", "", { fromSide: "b", toSide: "r" }), edge("m", "r", "", { fromSide: "b", toSide: "l" })],
  groups: [
    { label: "MedusaGo Application", nodes: ["m"] },
    { label: "Cart Module", nodes: ["l"] },
    { label: "Payment Module", nodes: ["r"] },
  ],
  gap: [90, 90],
})

export const regionPaymentProvider = () =>
  linkDiagram({
    left: "Payment",
    right: "Region",
    table: "region_payment_provider",
    rows: [{ left: ["PaymentProvider", "id = stripe"], right: ["Region", "id = reg_1"], fields: ["region_id = reg_1", "payment_provider_id = stripe"] }],
  })

export const salesChannelStockLocation = () =>
  linkDiagram({
    left: "Sales Channel",
    right: "Stock Location",
    table: "sales_channel_stock_location",
    rows: [{ left: ["SalesChannel", "id = sc_1"], right: ["StockLocation", "id = sloc_1"], fields: ["sales_channel_id = sc_1", "stock_location_id = sloc_1"] }],
  })

export const locationFulfillmentSet = () =>
  linkDiagram({
    left: "Fulfillment",
    right: "Stock Location",
    table: "location_fulfillment_set",
    rows: [{ left: ["FulfillmentSet", "id = fuset_1"], right: ["StockLocation", "id = sloc_1"], fields: ["fulfillment_set_id = fuset_1", "stock_location_id = sloc_1"] }],
  })

export const locationFulfillmentProvider = () =>
  linkDiagram({
    left: "Fulfillment",
    right: "Stock Location",
    table: "location_fulfillment_provider",
    rows: [{ left: ["FulfillmentProvider", "id = serpro_1"], right: ["StockLocation", "id = sloc_1"], fields: ["fulfillment_provider_id = serpro_1", "stock_location_id = sloc_1"] }],
  })

export const shippingOptionPriceSet = () =>
  linkDiagram({
    left: "Pricing",
    right: "Fulfillment",
    table: "shipping_option_price_set",
    rows: [{ left: ["PriceSet", "id = pset_1"], right: ["ShippingOption", "id = so_1"], fields: ["price_set_id = pset_1", "shipping_option_id = so_1"] }],
  })

export const publishableKeySalesChannel = () =>
  linkDiagram({
    left: "Sales Channel",
    right: "API Key",
    table: "publishable_api_key_sales_channel",
    rows: [
      { left: ["SalesChannel", "id = sc_1"], right: ["ApiKey", "id = apk_1"], fields: ["publishable_key_id = apk_1", "sales_channel_id = sc_1"] },
      { left: ["SalesChannel", "id = sc_2"], right: ["ApiKey", "id = apk_2"], fields: ["publishable_key_id = apk_2", "sales_channel_id = sc_2"] },
    ],
  })

export const productVariantInventoryItem = () =>
  topLinkDiagram({
    table: "product_variant_inventory_item",
    fields: ["variant_id = variant_1", "inventory_item_id = iitem_1"],
    a: { module: "Product", nodes: [inline("v", [0, 0], "ProductVariant id = variant_1"), inline("p", [0, 0], "Product id = prod_1")] },
    b: { module: "Inventory", nodes: [model("i", [0, 0], "InventoryItem", "id = iitem_1")] },
    edges: [
      edge("link", "v", "", { fromSide: "b", toSide: "r" }),
      edge("link", "i", "", { fromSide: "b", toSide: "l" }),
      edge("v", "p", "", { fromSide: "b", toSide: "t" }),
    ],
  })
