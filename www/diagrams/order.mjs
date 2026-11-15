import { model, edge, start, step, io, linkDiagram } from "./lib.mjs"
import { orderCart, orderFulfillment, orderPaymentCollection, orderPromotion } from "./shared.mjs"
import { taxCalculation } from "./cart.mjs"

// an order, its items and shipping methods, and what is attached to them (adjustments, tax lines)
const orderParts = (itemExtra, methodExtra) => ({
  nodes: [
    model("o", [0, 0.5], "Order", "id = order_1"),
    model("oi", [1, 0], "OrderItem", "id = orditem_1", "item_id = ordli_1", "order_id = ord_1"),
    model("li", [2, 0], "OrderLineItem", "id = ordli_1", "unit_price = 3000"),
    itemExtra.node,
    model("os", [1, 1], "OrderShipping", "id = ordspmv_1", "order_id = ord_1", "shipping_method_id = ordsm_1"),
    model("sm", [2, 1], "OrderShippingMethod", "id = ordsm_1", "amount = 1000"),
    methodExtra.node,
  ],
  edges: [
    edge("oi", "o", "", { fromSide: "l", toSide: "t" }),
    edge("oi", "li"),
    edge(itemExtra.node.id, "li"),
    edge("os", "o", "", { fromSide: "l", toSide: "b" }),
    edge("os", "sm"),
    edge(methodExtra.node.id, "sm"),
  ],
})

export const pages = {
  "resources/commerce-modules/order/concepts": [
    {
      nodes: [
        model("a", [0, 0], "OrderItem", "id = orditem_1", "quantity = 1", "order_id = ord_1"),
        model("o", [1, 0], "Order", "id = ord_1"),
        model("b", [2, 0], "OrderItem", "id = orditem_2", "quantity = 3", "order_id = ord_1"),
      ],
      edges: [edge("a", "o"), edge("b", "o")],
    },
    {
      nodes: [
        model("o", [0, 0], "Order", "id = ord_1"),
        model("s", [1, 0], "OrderShipping", "id = ordspmv_2", "order_id = ord_1", "shipping_method_id = ordsm_1"),
        model("m", [2, 0], "ShippingMethod", "id = ordsm_1", "amount = 3000"),
      ],
      edges: [edge("s", "o"), edge("s", "m")],
    },
  ],
  "resources/commerce-modules/order/links-to-other-modules": [orderCart(), orderFulfillment(), returnFulfillment(), orderPaymentCollection(), orderPromotion()],
  "resources/commerce-modules/order/promotion-adjustments": [
    orderParts(
      { node: { ...model("lia", [3, 0], "OrderLineItemAdjustment", "id = ordliadj_1", "item_id = ordli_1", "amount = 1500"), note: "50% off" } },
      { node: { ...model("sma", [3, 1], "OrderShippingMethodAdjustment", "id = ordsmadj_1", "amount = 1000", "shipping_method_id = ordsm_1"), noteBelow: "Free Shipping" } },
    ),
  ],
  "resources/commerce-modules/order/return": [
    {
      title: "RMA Automation with Returns",
      nodes: [
        start("c", [0, 0], "Customer"),
        step("a", [1, 0], "Create Return"),
        io("r", [1, 1], "Return"),
        start("m", [0, 2], "Merchant"),
        step("v", [1, 2], "View Return"),
        step("d", [2, 2], "Mark as Received"),
      ],
      edges: [edge("c", "a"), edge("a", "r", "Output"), edge("r", "v", "Input"), edge("m", "v"), edge("v", "d")],
    },
  ],
  "resources/commerce-modules/order/tax-lines": [
    orderParts(
      { node: model("lit", [3, 0], "OrderLineItemTaxLine", "id = ordlitxl_1", "rate = 10", "item_id = ordli_1") },
      { node: model("smt", [3, 1], "OrderShippingMethodTaxLine", "id = ordsmtxl_1", "rate = 5", "shipping_method_id = ordsm_1") },
    ),
    taxCalculation("line item’s unit price = 5000"),
  ],
}

function returnFulfillment() {
  return linkDiagram({
    left: "Order",
    right: "Fulfillment",
    table: "return_fulfillment",
    rows: [{ left: ["Return", "id = return_1"], right: ["Fulfillment", "id = ful_1"], fields: ["return_id = return_1", "fulfillment_id = ful_1"] }],
  })
}
