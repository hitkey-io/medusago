import { model, edge, parentWithTwo, start, io, decision } from "./lib.mjs"
import { orderCart, cartPaymentCollection, cartPromotion } from "./shared.mjs"

export const pages = {
  "resources/commerce-modules/cart/concepts": [
    {
      nodes: [
        model("cart", [0, 1], "Cart", "id = cart_1", "shipping_address_id = caaddr_1", "billing_address_id = caaddr_2"),
        model("a1", [1, 0], "Address", "id = caaddr_1"),
        model("a2", [1, 2], "Address", "id = caaddr_2"),
      ],
      edges: [edge("cart", "a1", "Shipping Address", { fromSide: "t", toSide: "l" }), edge("cart", "a2", "Billing Address", { fromSide: "b", toSide: "l" })],
    },
  ],
  "resources/commerce-modules/cart/links-to-other-modules": [orderCart(), cartPaymentCollection(), cartPromotion()],
  "resources/commerce-modules/cart/promotions": [adjustments()],
  "resources/commerce-modules/cart/tax-lines": [
    taxLines(),
    taxCalculation("line item’s amount = 5000"),
  ],
}

function adjustments() {
  return {
    nodes: [
      model("cart", [0, 0.5], "Cart", "id = cart_1"),
      model("li", [1, 0], "LineItem", "id = cali_1", "cart_id = cart_1", "amount = 3000"),
      { ...model("lia", [2, 0], "LineItemAdjustment", "id = caliadj_1", "item_id = cali_1", "amount = 1500"), note: "50% off" },
      model("sm", [1, 1], "ShippingMethod", "id = casm_1", "cart_id = cart_1", "amount = 1000"),
      { ...model("sma", [2, 1], "ShippingMethodAdjustment", "id = casmadj_1", "amount = 1000", "shipping_method_id = casm_1"), noteBelow: "Free Shipping" },
    ],
    edges: [edge("li", "cart", "", { fromSide: "l", toSide: "t" }), edge("lia", "li"), edge("sm", "cart", "", { fromSide: "l", toSide: "b" }), edge("sma", "sm")],
  }
}

function taxLines() {
  return {
    nodes: [
      model("cart", [0, 0.5], "Cart", "id = cart_1"),
      model("li", [1, 0], "LineItem", "id = cali_1", "cart_id = cart_1"),
      model("lit", [2, 0], "LineItemTaxLine", "id = calitxl_1", "rate = 10", "item_id = cali_1"),
      model("sm", [1, 1], "ShippingMethod", "id = casm_1", "cart_id = cart_1"),
      model("smt", [2, 1], "ShippingMethodTaxLine", "id = casmtxl_1", "rate = 5", "shipping_method_id = casm_1"),
    ],
    edges: [edge("li", "cart", "", { fromSide: "l", toSide: "t" }), edge("lit", "li"), edge("sm", "cart", "", { fromSide: "l", toSide: "b" }), edge("smt", "sm")],
  }
}

export function taxCalculation(amountLine) {
  return {
    title: "Example of Calculating Tax Amount of Line Item",
    nodes: [
      start("a", [0, 0], "Calculating tax amount", "of line item"),
      io("b", [1, 0], "Calculate from", amountLine, "tax rate = 10"),
      io("c", [2, 0], "Result tax amount =", "500"),
      decision("d", [1, 1], "is_tax_inclusive", "enabled?"),
      start("e", [0, 1], "line item amount = 4500", "tax amount = 500", "total = 5000"),
      start("f", [2, 1], "line item amount = 5000", "tax amount = 500", "total = 5500"),
    ],
    edges: [edge("a", "b"), edge("b", "c"), edge("c", "d", "", { fromSide: "b", toSide: "t" }), edge("d", "e", "Yes"), edge("d", "f", "No")],
  }
}
