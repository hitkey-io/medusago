import { model, edge, start, step, io, decision, join, providerFlow, providerModel } from "./lib.mjs"
import { orderPaymentCollection, regionPaymentProvider, cartPaymentCollection } from "./shared.mjs"

const withChildren = (parent, children, name, rows) => ({
  nodes: [
    model("c", [0, 0.5], "PaymentCollection", "id = pay_col_1", "amount = 10000"),
    model("p", [1, 0.5], "Payment", "id = pay_1", "amount = 10000", "payment_collection_id = pay_col_1"),
    ...children,
  ],
  edges: [edge("p", "c"), ...children.map((n, i) => edge(n.id, "p", "", { fromSide: "l", toSide: i === 0 ? "t" : "b" }))],
})

export const pages = {
  "resources/commerce-modules/payment/links-to-other-modules": [orderPaymentCollection(), regionPaymentProvider()],
  "resources/commerce-modules/payment/payment": [
    withChildren(null, [model("a", [2, 0], "Capture", "id = capt_1", "amount = 4000", "payment_id = pay_1"), model("b", [2, 1], "Capture", "id = capt_2", "amount = 6000", "payment_id = pay_1")]),
    withChildren(null, [model("a", [2, 0], "Refund", "id = ref_1", "amount = 1000", "payment_id = pay_1"), model("b", [2, 1], "Refund", "id = ref_2", "amount = 2000", "payment_id = pay_1")]),
  ],
  "resources/commerce-modules/payment/payment-checkout-flow": [
    {
      title: "Payment in Checkout Flow",
      nodes: [
        start("s", [0, 0], "Start"),
        step("a", [1, 0], "Create payment collection", "(for cart)"),
        step("b", [2, 0], "Show customer", "payment providers"),
        io("c", [2, 1], "Customer selects", "payment provider"),
        step("d", [1, 1], "Create payment", "session"),
        step("e", [0, 1], "Initialize payment", "session with", "third-party provider"),
        io("f", [0, 2], "Customer places", "order"),
        step("g", [1, 2], "Authorize payment", "session"),
        decision("h", [1, 3], "Action required?"),
        start("end", [0, 3], "End"),
        step("i", [2, 3], "Handle required action", "with third-party", "provider"),
      ],
      edges: [
        edge("s", "a"),
        edge("a", "b"),
        edge("b", "c"),
        edge("c", "d"),
        edge("d", "e"),
        edge("e", "f"),
        edge("f", "g"),
        edge("g", "h"),
        edge("h", "end", "No"),
        edge("h", "i", "Yes"),
        edge("i", "g", "", { fromSide: "t", toSide: "r" }),
      ],
    },
  ],
  "resources/commerce-modules/payment/payment-collection": [
    {
      nodes: [
        model("c", [0, 0.5], "PaymentCollection", "id = pay_col_1", "amount = 10000"),
        model("s1", [1, 0], "PaymentSession", "id = payses_1", "amount = 4000", "provider_id = stripe", "payment_collection_id = pay_col_1"),
        model("s2", [1, 1], "PaymentSession", "id = payses_2", "amount = 6000", "provider_id = adyen", "payment_collection_id = pay_col_1"),
        model("p1", [2, 0], "Payment", "id = pay_1", "amount = 4000", "provider_id = stripe", "payment_session_id = payses_1", "payment_collection_id = pay_col_1"),
        model("p2", [2, 1], "Payment", "id = pay_2", "amount = 6000", "provider_id = adyen", "payment_session_id = payses_2", "payment_collection_id = pay_col_1"),
      ],
      edges: [edge("s1", "c", "", { fromSide: "l", toSide: "t" }), edge("s2", "c", "", { fromSide: "l", toSide: "b" }), edge("p1", "s1"), edge("p2", "s2")],
    },
    cartPaymentCollection(),
  ],
  "resources/commerce-modules/payment/payment-provider": [providerFlow("Payment Module Provider", "3rd Party Service"), providerModel("Payment", "PaymentProvider", "pp_")],
  "resources/commerce-modules/payment/payment-session": [
    {
      nodes: [
        model("c", [0, 0.5], "PaymentCollection", "id = pay_col_1", "amount = 10000"),
        model("s1", [1, 0], "PaymentSession", "id = payses_1", "provider_id = stripe", "payment_collection_id = pay_col_1"),
        model("s2", [1, 1], "PaymentSession", "id = payses_2", "provider_id = adyen", "payment_collection_id = pay_col_1"),
        model("p1", [2, 0], "PaymentProvider", "id = stripe"),
        model("p2", [2, 1], "PaymentProvider", "id = adyen"),
      ],
      edges: [edge("s1", "c", "", { fromSide: "l", toSide: "t" }), edge("s2", "c", "", { fromSide: "l", toSide: "b" }), edge("s1", "p1"), edge("s2", "p2")],
    },
  ],
  "resources/commerce-modules/payment/webhook-events": [
    {
      nodes: [
        start("a", [0, 0], "Receive webhook event"),
        step("b", [2, 0], "Execute", "getWebhookActionAndData"),
        io("c", [4, 0], "Return event", "action"),
        step("cap", [0, 1], "Capture associated", "payment session"),
        decision("d", [2, 1], "action = `captured`?"),
        join("j1", [3, 1]),
        decision("e", [4, 1], "action = `authorized`?"),
        join("j2", [2, 2]),
        step("auth", [4, 2], "Authorize associated", "payment session"),
        step("cc", [0, 3], "Complete cart"),
        decision("f", [2, 3], "Is cart completed?"),
        start("end", [2, 4], "End"),
      ],
      edges: [
        edge("a", "b"),
        edge("b", "c"),
        edge("c", "e", "", { fromSide: "b", toSide: "t" }),
        edge("e", "j1", "No"),
        edge("e", "auth", "Yes", { fromSide: "b", toSide: "t" }),
        edge("auth", "j1", "", { fromSide: "l", toSide: "b" }),
        edge("j1", "d", "", { fromSide: "l", toSide: "r" }),
        edge("d", "cap", "Yes"),
        edge("d", "j2", "No", { fromSide: "b", toSide: "t" }),
        edge("cap", "j2", "", { fromSide: "b", toSide: "l" }),
        edge("j2", "f", "", { fromSide: "b", toSide: "t" }),
        edge("f", "cc", "No"),
        edge("f", "end", "Yes", { fromSide: "b", toSide: "t" }),
        edge("cc", "end", "", { fromSide: "b", toSide: "l" }),
      ],
      gap: [60, 56],
    },
  ],
}
